package domain

import (
	xmaps "github.com/certimate-go/certimate/pkg/utils/maps"
)

const CollectionNameSettings = "settings"

type Settings struct {
	Meta
	Name    string          `db:"name"    json:"name"`
	Content SettingsContent `db:"content" json:"content"`
}

const (
	SettingsNameEmails               = "emails"
	SettingsNameNotificationTemplate = "notifyTemplate"
	SettingsNameScriptTemplate       = "scriptTemplate"
	SettingsNameSSLProvider          = "sslProvider"
	SettingsNamePersistence          = "persistence"
	SettingsNameOAuth2               = "oauth2"
)

type SettingsContent map[string]any

type SettingsContentForSSLProvider struct {
	Provider CAProviderType                    `json:"provider"`
	Configs  map[CAProviderType]map[string]any `json:"configs"`
	Timeout  int                               `json:"timeout"`
}

type SettingsContentForPersistence struct {
	CertificatesWarningDaysBeforeExpire int `json:"certificatesWarningDaysBeforeExpire"`
	CertificatesRetentionMaxDays        int `json:"certificatesRetentionMaxDays"`
	WorkflowRunsRetentionMaxDays        int `json:"workflowRunsRetentionMaxDays"`
}

func (c SettingsContent) AsSSLProvider() *SettingsContentForSSLProvider {
	content := &SettingsContentForSSLProvider{}
	xmaps.Populate(c, content)

	if content.Provider == "" {
		content.Provider = CAProviderTypeLetsEncrypt
	}

	if content.Timeout < 0 {
		content.Timeout = 0
	}

	return content
}

func (c SettingsContent) AsPersistence() *SettingsContentForPersistence {
	content := &SettingsContentForPersistence{}
	xmaps.Populate(c, content)

	if content.CertificatesWarningDaysBeforeExpire <= 0 {
		content.CertificatesWarningDaysBeforeExpire = 21
	}

	if content.CertificatesRetentionMaxDays < 0 {
		content.CertificatesRetentionMaxDays = 0
	}

	if content.WorkflowRunsRetentionMaxDays < 0 {
		content.WorkflowRunsRetentionMaxDays = 0
	}

	return content
}

// 表示 OAuth2 登录的全局设置数据结构。
type SettingsContentForOAuth2 struct {
	Providers []SettingsContentForOAuth2Provider `json:"providers"`
}

type SettingsContentForOAuth2Provider struct {
	Name             string   `json:"name"`        // 提供商标识符，如 "github"、"google"
	DisplayName      string   `json:"displayName"` // 展示名称，如 "GitHub"
	Enabled          bool     `json:"enabled"`
	ClientID         string   `json:"clientId"`
	ClientSecret     string   `json:"clientSecret"`
	Scopes           []string `json:"scopes"`
	RedirectURL      string   `json:"redirectUrl"`      // 完整的回调地址，形如 https://your-host/api/oauth2/callback?provider=github
	AuthURL          string   `json:"authUrl"`          // 授权端点；为空时按预设填充
	TokenURL         string   `json:"tokenUrl"`         // Token 端点；为空时按预设填充
	UserInfoURL      string   `json:"userInfoUrl"`      // UserInfo 端点；为空时按预设填充
	SubjectField     string   `json:"subjectField"`     // 从 UserInfo JSON 中读取的主键字段；零值时按预设填充
	ScopesJoin       string   `json:"scopesJoin"`       // 部分提供商（如微信）的 scope 使用空格或逗号连接，控制请求拼接分隔符
	AutoCreate       bool     `json:"autoCreate"`       // 若关联不存在，是否自动创建超级管理员账户（默认仅以管理员身份预置）
	AutoCreatePrefix string   `json:"autoCreatePrefix"` // 自动创建超级管理员账户时的邮箱前缀
}

func (c SettingsContent) AsOAuth2() *SettingsContentForOAuth2 {
	content := &SettingsContentForOAuth2{}
	xmaps.Populate(c, content)

	for i := range content.Providers {
		p := &content.Providers[i]
		if p.DisplayName == "" {
			p.DisplayName = p.Name
		}
	}

	return content
}
