package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/certimate-go/certimate/internal/app"
	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/repository"
	"github.com/certimate-go/certimate/internal/settings"
)

const (
	oauth2StateTTL = 5 * time.Minute
)

// ResolvedProvider 表示运行时实际生效的 OAuth2 提供商配置：
// 在用户配置基础上回填默认的端点和字段。
type ResolvedProvider struct {
	Settings     domain.SettingsContentForOAuth2Provider
	Preset       *ProviderPreset
	SubjectField string
	EmailField   string
	NameField    string
	AvatarField  string
}

func ResolveProvider(settings domain.SettingsContentForOAuth2Provider) *ResolvedProvider {
	preset := LookupPreset(settings.Name)
	resolved := &ResolvedProvider{
		Settings: settings,
		Preset:   preset,
	}

	if settings.SubjectField != "" {
		resolved.SubjectField = settings.SubjectField
	} else if preset != nil && preset.SubjectField != "" {
		resolved.SubjectField = preset.SubjectField
	} else {
		resolved.SubjectField = "id"
	}

	if preset != nil {
		resolved.EmailField = preset.EmailField
		resolved.NameField = preset.NameField
		resolved.AvatarField = preset.AvatarField
	}

	// Endpoints 缺省时回填预设。
	if settings.AuthURL == "" && preset != nil {
		resolved.Settings.AuthURL = preset.AuthURL
	}
	if settings.TokenURL == "" && preset != nil {
		resolved.Settings.TokenURL = preset.TokenURL
	}
	if settings.UserInfoURL == "" && preset != nil {
		resolved.Settings.UserInfoURL = preset.UserInfoURL
	}
	if len(settings.Scopes) == 0 && preset != nil {
		resolved.Settings.Scopes = preset.Scopes
	}

	return resolved
}

// Service 提供 OAuth2 授权 URL 生成、回调处理与超级管理员关联服务。
type Service struct {
	linkRepo *repository.OAuth2LinkRepository
}

func NewService() *Service {
	return &Service{linkRepo: repository.NewOAuth2LinkRepository()}
}

// ListEnabledProviders 列出已启用的 OAuth2 提供商（脱敏，不含 clientSecret）。
func (s *Service) ListEnabledProviders(ctx context.Context) []domain.SettingsContentForOAuth2Provider {
	cfg := settings.GetGlobalSettingsForOAuth2()
	enabled := make([]domain.SettingsContentForOAuth2Provider, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		resolved := ResolveProvider(p)
		// 返回填充默认值后的配置，但剥离 secret。
		out := resolved.Settings
		out.ClientSecret = ""
		enabled = append(enabled, out)
	}
	return enabled
}

// GetProvider 根据名称查找已启用的 provider；未启用或缺失时返回错误。
func (s *Service) GetProvider(ctx context.Context, name string) (*ResolvedProvider, error) {
	cfg := settings.GetGlobalSettingsForOAuth2()
	for _, p := range cfg.Providers {
		if p.Name != name {
			continue
		}
		if !p.Enabled {
			return nil, fmt.Errorf("oauth2 provider %q is disabled", name)
		}
		if p.ClientID == "" || p.ClientSecret == "" {
			return nil, fmt.Errorf("oauth2 provider %q is missing credentials", name)
		}
		return ResolveProvider(p), nil
	}
	return nil, fmt.Errorf("oauth2 provider %q is not configured", name)
}

// BuildAuthorizeURL 生成跳转到 OAuth2 提供商的授权 URL，并写出一次性 state。
// state 由进程内 KV 存储（约 oauth2StateTTL 时间），_logout 后过期失效。
func (s *Service) BuildAuthorizeURL(ctx context.Context, providerName, redirectOverride string) (string, error) {
	provider, err := s.GetProvider(ctx, providerName)
	if err != nil {
		return "", err
	}

	redirectURL := provider.Settings.RedirectURL
	if redirectOverride != "" {
		redirectURL = redirectOverride
	}
	if redirectURL == "" {
		return "", fmt.Errorf("oauth2 provider %q is missing redirectUrl", providerName)
	}
	if provider.Settings.AuthURL == "" {
		// 自托管的 OAuth2/OIDC 提供商（如 Authentik）端点不在内设预设中，管理员必须显式填入。
		return "", fmt.Errorf("oauth2 provider %q is missing authUrl (required for self-hosted providers)", providerName)
	}

	scopes := provider.Settings.Scopes
	params := url.Values{}
	params.Set("client_id", provider.Settings.ClientID)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	if len(scopes) > 0 {
		sep := " "
		if provider.Settings.ScopesJoin != "" {
			sep = provider.Settings.ScopesJoin
		}
		params.Set("scope", strings.Join(scopes, sep))
	}

	state, err := s.issueState(providerName)
	if err != nil {
		return "", err
	}
	params.Set("state", state)

	authURL := provider.Settings.AuthURL
	if strings.Contains(authURL, "?") {
		authURL += "&" + params.Encode()
	} else {
		authURL += "?" + params.Encode()
	}
	return authURL, nil
}

// HandleCallback 完成 OAuth2 回调：校验 state、换取 access_token、获取用户信息、
// 关联或（若允许）创建超级管理员，再返回 superuser 记录与新颁发的鉴权 token。
func (s *Service) HandleCallback(ctx context.Context, providerName, code, state, redirectOverride string) (*core.Record, string, error) {
	provider, err := s.GetProvider(ctx, providerName)
	if err != nil {
		return nil, "", err
	}

	if err := s.consumeState(state, providerName); err != nil {
		return nil, "", err
	}
	if code == "" {
		return nil, "", errors.New("missing authorization code")
	}
	// 对自托管提供商（如 Authentik），管理员可能遗漏了某个端点；提前给出明确错误，避免走到 HTTP 底层报错。
	if provider.Settings.TokenURL == "" {
		return nil, "", fmt.Errorf("oauth2 provider %q is missing tokenUrl", providerName)
	}
	if provider.Settings.UserInfoURL == "" {
		return nil, "", fmt.Errorf("oauth2 provider %q is missing userInfoUrl", providerName)
	}

	token, err := s.exchangeCode(ctx, provider, code, redirectOverride)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	profile, err := s.fetchUserInfo(ctx, provider, token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch userinfo: %w", err)
	}

	subject := stringField(profile, provider.SubjectField)
	if subject == "" {
		return nil, "", fmt.Errorf("userinfo does not contain subject field %q", provider.SubjectField)
	}

	superuser, err := s.resolveSuperuser(ctx, providerName, subject, profile, provider)
	if err != nil {
		return nil, "", err
	}

	tokenStr, err := superuser.NewAuthToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to issue auth token: %w", err)
	}

	return superuser, tokenStr, nil
}

// resolveSuperuser 根据 (provider, subject) 关联查找已绑定的超级管理员；
// 不存在时若 provider 允许 autoCreate 且无其他安全限制，则尝试创建一个超级管理员账户并绑定。
func (s *Service) resolveSuperuser(ctx context.Context, providerName, subject string, profile map[string]any, provider *ResolvedProvider) (*core.Record, error) {
	link, err := s.linkRepo.GetByProviderAndSubject(ctx, providerName, subject)
	if err != nil && !errors.Is(err, domain.ErrRecordNotFound) {
		return nil, err
	}
	if link != nil {
		rec, err := app.GetApp().FindRecordById("_superusers", link.SuperuserId)
		if err != nil {
			// 关联的 superuser 已被删除，清理失效链接后允许自动重建（若开启）。
			_ = s.linkRepo.Delete(ctx, link.Id)
		} else {
			// 刷新最近一次的 profile 快照。
			link.UserProfileEmail = stringField(profile, provider.EmailField)
			link.UserProfileName = stringField(profile, provider.NameField)
			link.UserProfileAvatar = stringField(profile, provider.AvatarField)
			_, _ = s.linkRepo.Save(ctx, link)
			return rec, nil
		}
	}

	// 关联不存在；尝试以 email 在已有 superuser 中匹配并绑定，避免重复账号。
	if email := stringField(profile, provider.EmailField); email != "" {
		existing, err := app.GetApp().FindFirstRecordByFilter("_superusers", "email={:email}", dbx.Params{"email": email})
		if err == nil && existing != nil {
			if err := s.bindLink(ctx, providerName, subject, existing.Id, profile, provider); err != nil {
				return nil, err
			}
			return existing, nil
		}
	}

	if !provider.Settings.AutoCreate {
		return nil, fmt.Errorf("no superuser linked to oauth2 provider %q ( administrators must first link it in settings )", providerName)
	}

	superuser, err := s.createSuperuser(ctx, providerName, subject, profile, provider)
	if err != nil {
		return nil, err
	}
	if err := s.bindLink(ctx, providerName, subject, superuser.Id, profile, provider); err != nil {
		return nil, err
	}
	return superuser, nil
}

func (s *Service) bindLink(ctx context.Context, providerName, subject, superuserId string, profile map[string]any, provider *ResolvedProvider) error {
	link := &domain.OAuth2Link{
		Provider:          providerName,
		SubjectId:         subject,
		SuperuserId:       superuserId,
		UserProfileEmail:  stringField(profile, provider.EmailField),
		UserProfileName:   stringField(profile, provider.NameField),
		UserProfileAvatar: stringField(profile, provider.AvatarField),
	}
	_, err := s.linkRepo.Save(ctx, link)
	return err
}

func (s *Service) createSuperuser(ctx context.Context, providerName, subject string, profile map[string]any, provider *ResolvedProvider) (*core.Record, error) {
	collection, err := app.GetApp().FindCollectionByNameOrId("_superusers")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)

	email := stringField(profile, provider.EmailField)
	prefix := provider.Settings.AutoCreatePrefix
	if prefix == "" {
		prefix = "oauth2"
	}
	if email == "" {
		// 没有邮箱时构造一个占位邮箱以满足 superuser 字段非空约束。
		email = fmt.Sprintf("%s+%s+%s@certimate.local", prefix, providerName, subject)
	}
	record.Set("email", email)
	// 生成一个用户与 admin 都不知道的随机强密码（管理员可后续修改）。
	// PocketBase 的密码字段会自动 hash 化。
	password, err := randomPassword(32)
	if err != nil {
		return nil, err
	}
	record.Set("password", password)
	record.Set("passwordConfirm", password)

	if err := app.GetApp().Save(record); err != nil {
		return nil, fmt.Errorf("failed to create superuser: %w", err)
	}
	return record, nil
}

// --- state store ---

func (s *Service) issueState(providerName string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	app.GetApp().Store().Set(stateKey(state), map[string]any{
		"provider": providerName,
		"expires":  time.Now().Add(oauth2StateTTL).Unix(),
	})
	return state, nil
}

func (s *Service) consumeState(state, providerName string) error {
	if state == "" {
		return errors.New("missing state")
	}
	key := stateKey(state)
	val := app.GetApp().Store().Get(key)
	if val == nil {
		return errors.New("invalid or expired oauth2 state")
	}
	m, ok := val.(map[string]any)
	if !ok {
		return errors.New("invalid oauth2 state payload")
	}
	if exp, _ := m["expires"].(int64); exp > 0 && time.Now().Unix() > exp {
		app.GetApp().Store().Remove(key)
		return errors.New("expired oauth2 state")
	}
	if p, _ := m["provider"].(string); p != providerName {
		return errors.New("oauth2 state does not match provider")
	}
	app.GetApp().Store().Remove(key)
	return nil
}

func stateKey(state string) string {
	return fmt.Sprintf("certimate|oauth2|state|%s", state)
}

// --- low-level helpers ---

func (s *Service) exchangeCode(ctx context.Context, provider *ResolvedProvider, code, redirectOverride string) (string, error) {
	redirectURL := provider.Settings.RedirectURL
	if redirectOverride != "" {
		redirectURL = redirectOverride
	}
	if redirectURL == "" {
		return "", errors.New("missing redirectUrl")
	}

	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("client_id", provider.Settings.ClientID)
	body.Set("client_secret", provider.Settings.ClientSecret)
	body.Set("redirect_uri", redirectURL)
	bodyStr := body.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.Settings.TokenURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// GitHub 默认返回 application/json when Accept 设为 json。
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("could not parse token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("token response does not contain access_token: %s", string(raw))
	}
	return parsed.AccessToken, nil
}

func (s *Service) fetchUserInfo(ctx context.Context, provider *ResolvedProvider, accessToken string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.Settings.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", resp.StatusCode, string(raw))
	}

	var profile map[string]any
	if err := json.Unmarshal(raw, &profile); err != nil {
		return nil, fmt.Errorf("could not parse userinfo response: %w", err)
	}
	return profile, nil
}

func stringField(m map[string]any, key string) string {
	if m == nil || key == "" {
		return ""
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case float64:
			return fmt.Sprintf("%v", t)
		case json.Number:
			return t.String()
		case bool:
			return fmt.Sprintf("%v", t)
		default:
			b, _ := json.Marshal(v)
			return string(b)
		}
	}
	return ""
}

func randomPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
