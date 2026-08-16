package sso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/certimate-go/certimate/internal/app"
	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/repository"
	"github.com/certimate-go/certimate/internal/settings"
)

const (
	// ProviderOIDC 标识 OIDC 提供者。
	ProviderOIDC = "oidc"
	// ProviderLDAP 标识 LDAP 提供者。
	ProviderLDAP = "ldap"

	oidcStateTTL = 5 * time.Minute
)

// Service 提供单点登录（OIDC / LDAP）的认证服务。
type Service struct {
	linkRepo *repository.OAuth2LinkRepository
}

func NewService() *Service {
	return &Service{linkRepo: repository.NewOAuth2LinkRepository()}
}

// GetConfig 返回脱敏后的 SSO 配置（不含 clientSecret / bindPassword），供前端渲染。
func (s *Service) GetConfig(ctx context.Context) *domain.SettingsContentForSSO {
	cfg := settings.GetGlobalSettingsForSSO()

	out := &domain.SettingsContentForSSO{}
	if cfg.OIDC != nil {
		c := *cfg.OIDC
		c.ClientSecret = ""
		out.OIDC = &c
	}
	if cfg.LDAP != nil {
		c := *cfg.LDAP
		c.BindPassword = ""
		out.LDAP = &c
	}
	return out
}

// GetEnabledOIDC 返回已启用且凭据完整的 OIDC 配置；未配置或未启用时报错。
func (s *Service) GetEnabledOIDC(ctx context.Context) (*domain.SettingsContentForSSOOIDC, error) {
	cfg := settings.GetGlobalSettingsForSSO()
	if cfg.OIDC == nil || !cfg.OIDC.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled")
	}
	if cfg.OIDC.ClientID == "" || cfg.OIDC.ClientSecret == "" {
		return nil, fmt.Errorf("OIDC is missing client credentials")
	}
	if cfg.OIDC.DiscoveryURL == "" {
		return nil, fmt.Errorf("OIDC is missing discoveryUrl")
	}
	return cfg.OIDC, nil
}

// GetEnabledLDAP 返回已启用且凭据完整的 LDAP 配置；未配置或未启用时报错。
func (s *Service) GetEnabledLDAP(ctx context.Context) (*domain.SettingsContentForSSOLDAP, error) {
	cfg := settings.GetGlobalSettingsForSSO()
	if cfg.LDAP == nil || !cfg.LDAP.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}
	if cfg.LDAP.ServerURL == "" || cfg.LDAP.BindDN == "" || cfg.LDAP.BindPassword == "" || cfg.LDAP.SearchBase == "" {
		return nil, fmt.Errorf("LDAP configuration is incomplete")
	}
	return cfg.LDAP, nil
}

// BuildAuthorizeURL 生成跳转到 OIDC 提供商的授权 URL，并写出一次性 state。
// redirectURL 为统一的回调地址（由请求侧按当前访问地址计算）。
func (s *Service) BuildAuthorizeURL(ctx context.Context, oidcCfg *domain.SettingsContentForSSOOIDC, redirectURL string) (string, error) {
	if redirectURL == "" {
		return "", errors.New("OIDC redirect URL is empty")
	}

	discovery, err := s.fetchOIDCDiscovery(ctx, oidcCfg.DiscoveryURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}
	if discovery.AuthorizationEndpoint == "" {
		return "", errors.New("OIDC discovery does not contain authorization_endpoint")
	}

	state, err := s.issueState(ProviderOIDC)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("client_id", oidcCfg.ClientID)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(oidcCfg.Scopes, " "))
	params.Set("state", state)

	authURL := discovery.AuthorizationEndpoint
	if strings.Contains(authURL, "?") {
		authURL += "&" + params.Encode()
	} else {
		authURL += "?" + params.Encode()
	}
	return authURL, nil
}

// HandleOIDCCallback 完成 OIDC 回调：校验 state、用 code 换 access_token、拉取 userinfo、
// 关联或（若允许）自动创建普通用户，并返回账号记录与新颁发的鉴权 token。
func (s *Service) HandleOIDCCallback(ctx context.Context, code, state, redirectURL string) (*core.Record, string, error) {
	if redirectURL == "" {
		return nil, "", errors.New("OIDC redirect URL is empty")
	}

	oidcCfg, err := s.GetEnabledOIDC(ctx)
	if err != nil {
		return nil, "", err
	}
	if err := s.consumeState(state, ProviderOIDC); err != nil {
		return nil, "", err
	}
	if code == "" {
		return nil, "", errors.New("missing authorization code")
	}

	discovery, err := s.fetchOIDCDiscovery(ctx, oidcCfg.DiscoveryURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}

	accessToken, err := s.exchangeOIDCCode(ctx, oidcCfg.ClientID, oidcCfg.ClientSecret, discovery, code, redirectURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	profile, err := s.fetchOIDCUserInfo(ctx, discovery, accessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch userinfo: %w", err)
	}

	subject := stringField(profile, "sub")
	if subject == "" {
		return nil, "", errors.New("userinfo does not contain 'sub' claim")
	}

	account, err := s.resolveAccount(ctx, ProviderOIDC, subject, profile["email"], profile["name"], oidcCfg.AutoCreate, oidcCfg.AutoCreatePrefix)
	if err != nil {
		return nil, "", err
	}

	tokenStr, err := account.NewAuthToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to issue auth token: %w", err)
	}
	return account, tokenStr, nil
}

// HandleLDAPLogin 完成 LDAP 用户名密码认证：绑定验证后关联或创建普通用户。
func (s *Service) HandleLDAPLogin(ctx context.Context, username, password string) (*core.Record, string, error) {
	ldapCfg, err := s.GetEnabledLDAP(ctx)
	if err != nil {
		return nil, "", err
	}
	if username == "" || password == "" {
		return nil, "", errors.New("missing username or password")
	}

	user, err := s.authenticateLDAPUser(ctx, ldapCfg, username, password)
	if err != nil {
		return nil, "", err
	}

	subject := user.DN
	if subject == "" {
		return nil, "", errors.New("LDAP user has empty DN")
	}

	account, err := s.resolveAccount(ctx, ProviderLDAP, subject, user.Email, user.Name, ldapCfg.AutoCreate, ldapCfg.AutoCreatePrefix)
	if err != nil {
		return nil, "", err
	}

	tokenStr, err := account.NewAuthToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to issue auth token: %w", err)
	}
	return account, tokenStr, nil
}

// resolveAccount 根据 (provider, subject) 关联查找已绑定的账号；
// 不存在时若允许 autoCreate，则按 email 匹配已有账号，否则创建普通用户。
func (s *Service) resolveAccount(ctx context.Context, providerName, subject string, emailAny, nameAny any, autoCreate bool, autoCreatePrefix string) (*core.Record, error) {
	email := fmt.Sprintf("%v", emailAny)
	if email == "<nil>" {
		email = ""
	}
	name := fmt.Sprintf("%v", nameAny)
	if name == "<nil>" {
		name = ""
	}

	link, err := s.linkRepo.GetByProviderAndSubject(ctx, providerName, subject)
	if err != nil && !errors.Is(err, domain.ErrRecordNotFound) {
		return nil, err
	}
	if link != nil {
		target := link.TargetCollection
		if target == "" {
			target = core.CollectionNameSuperusers
		}
		rec, err := app.GetApp().FindRecordById(target, link.SuperuserId)
		if err != nil {
			// 关联的账号已被删除，清理失效链接后允许自动重建（若开启）。
			_ = s.linkRepo.Delete(ctx, link.Id)
		} else {
			link.UserProfileEmail = email
			link.UserProfileName = name
			_, _ = s.linkRepo.Save(ctx, link)
			return rec, nil
		}
	}

	// 关联不存在；尝试以 email 在已有账号（先管理员、后成员）中匹配并绑定，避免重复账号。
	if email != "" {
		for _, collectionName := range []string{core.CollectionNameSuperusers, "users"} {
			existing, err := app.GetApp().FindFirstRecordByFilter(collectionName, "email={:email}", map[string]any{"email": email})
			if err == nil && existing != nil {
				if err := s.bindLink(ctx, providerName, subject, existing.Id, collectionName, email, name); err != nil {
					return nil, err
				}
				return existing, nil
			}
		}
	}

	if !autoCreate {
		return nil, fmt.Errorf("no account linked to %s ( an administrator must first link it in settings )", providerName)
	}

	// 自动创建的账号默认是普通用户（users 集合，role=user）。
	account, err := s.createUser(ctx, providerName, subject, email, name, autoCreatePrefix)
	if err != nil {
		return nil, err
	}
	if err := s.bindLink(ctx, providerName, subject, account.Id, "users", email, name); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *Service) bindLink(ctx context.Context, providerName, subject, accountId, targetCollection, email, name string) error {
	link := &domain.OAuth2Link{
		Provider:         providerName,
		SubjectId:        subject,
		TargetCollection: targetCollection,
		SuperuserId:      accountId,
		UserProfileEmail: email,
		UserProfileName:  name,
	}
	_, err := s.linkRepo.Save(ctx, link)
	return err
}

// createUser 创建普通用户账号（users 集合，role=user）。
func (s *Service) createUser(ctx context.Context, providerName, subject, email, name, prefix string) (*core.Record, error) {
	collection, err := app.GetApp().FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	if prefix == "" {
		prefix = "sso"
	}
	if email == "" {
		email = fmt.Sprintf("%s+%s+%s@certimate.local", prefix, providerName, subject)
	}
	record.Set("email", email)
	record.Set("name", name)
	record.Set("role", "user")

	password, err := randomPassword(32)
	if err != nil {
		return nil, err
	}
	record.Set("password", password)
	record.Set("passwordConfirm", password)

	if err := app.GetApp().Save(record); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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
		"expires":  time.Now().Add(oidcStateTTL).Unix(),
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
		return errors.New("invalid or expired sso state")
	}
	m, ok := val.(map[string]any)
	if !ok {
		return errors.New("invalid sso state payload")
	}
	if exp, _ := m["expires"].(int64); exp > 0 && time.Now().Unix() > exp {
		app.GetApp().Store().Remove(key)
		return errors.New("expired sso state")
	}
	if p, _ := m["provider"].(string); p != providerName {
		return errors.New("sso state does not match provider")
	}
	app.GetApp().Store().Remove(key)
	return nil
}

func stateKey(state string) string {
	return fmt.Sprintf("certimate|sso|state|%s", state)
}

func randomPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
