package oauth2

import (
	"github.com/certimate-go/certimate/internal/domain"
)

var ErrInvalidProvider = domain.NewError(400, "invalid oauth2 provider")

// 预设的通用 OAuth2 提供商端点信息。
// 仅作为默认值使用；管理员可在 Settings 中覆盖各端点与字段。
type ProviderPreset struct {
	Name         string
	DisplayName  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
	SubjectField string
	EmailField   string
	NameField    string
	AvatarField  string
}

func (p *ProviderPreset) ResolveSubjectField(custom string) string {
	if custom != "" {
		return custom
	}
	if p == nil {
		return "id"
	}
	if p.SubjectField != "" {
		return p.SubjectField
	}
	return "id"
}

var providerPresets = map[string]*ProviderPreset{
	"github": {
		Name:         "github",
		DisplayName:  "GitHub",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"read:user"},
		SubjectField: "id",
		EmailField:   "email",
		NameField:    "login",
		AvatarField:  "avatar_url",
	},
	"gitlab": {
		Name:         "gitlab",
		DisplayName:  "GitLab",
		AuthURL:      "https://gitlab.com/oauth/authorize",
		TokenURL:     "https://gitlab.com/oauth/token",
		UserInfoURL:  "https://gitlab.com/api/v4/user",
		Scopes:       []string{"read_user"},
		SubjectField: "id",
		EmailField:   "email",
		NameField:    "username",
		AvatarField:  "avatar_url",
	},
	"gitee": {
		Name:         "gitee",
		DisplayName:  "Gitee",
		AuthURL:      "https://gitee.com/oauth/authorize",
		TokenURL:     "https://gitee.com/oauth/token",
		UserInfoURL:  "https://gitee.com/api/v5/user",
		Scopes:       []string{"user_info"},
		SubjectField: "id",
		EmailField:   "email",
		NameField:    "login",
		AvatarField:  "avatar_url",
	},
	"google": {
		Name:         "google",
		DisplayName:  "Google",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		SubjectField: "sub",
		EmailField:   "email",
		NameField:    "name",
		AvatarField:  "picture",
	},
	"azuread": {
		Name:         "azuread",
		DisplayName:  "Microsoft (Azure AD)",
		AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		SubjectField: "sub",
		EmailField:   "email",
		NameField:    "name",
		AvatarField:  "picture",
	},
	"dingtalk": {
		Name:         "dingtalk",
		DisplayName:  "DingTalk",
		AuthURL:      "https://login.dingtalk.com/oauth2/auth",
		TokenURL:     "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
		UserInfoURL:  "https://api.dingtalk.com/v1.0/contact/users/me",
		Scopes:       []string{},
		SubjectField: "openId",
		EmailField:   "email",
		NameField:    "nick",
		AvatarField:  "avatarUrl",
	},
	// Authentik 是自托管的 identity provider，端点 URL 与部署实例 + 应用 slug 相关，
	// 因此此处只出厂 OIDC 标准的 scope 与字段映射；管理员必须显式填入 3 个端点。
	// 端点格式示例：https://<your-authentik-host>/application/o/<application-slug>/[authorize|token|userinfo]/
	"authentik": {
		Name:         "authentik",
		DisplayName:  "Authentik",
		Scopes:       []string{"openid", "email", "profile"},
		SubjectField: "sub",
		EmailField:   "email",
		NameField:    "preferred_username",
		AvatarField:  "picture",
	},
}

// LookupPreset 返回某 provider 的默认预设；不存在时返回 nil。
func LookupPreset(name string) *ProviderPreset {
	if p, ok := providerPresets[name]; ok {
		return p
	}
	return nil
}

// AllPresetNames 返回内置预设支持的 provider 列表。
func AllPresetNames() []string {
	names := make([]string, 0, len(providerPresets))
	for name := range providerPresets {
		names = append(names, name)
	}
	return names
}
