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
	SettingsNameSSO                  = "sso"
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

// 表示单点登录（SSO）的全局设置数据结构。
// OIDC 与 LDAP 各支持一个提供者配置。
type SettingsContentForSSO struct {
	OIDC *SettingsContentForSSOOIDC `json:"oidc,omitempty"`
	LDAP *SettingsContentForSSOLDAP `json:"ldap,omitempty"`
}

// SettingsContentForSSOOIDC 表示标准 OIDC 提供者配置。
type SettingsContentForSSOOIDC struct {
	Enabled      bool     `json:"enabled"`
	DisplayName  string   `json:"displayName,omitempty"` // 登录页按钮展示名，零值时默认 "OIDC"
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	DiscoveryURL string   `json:"discoveryUrl"` // OIDC Discovery 端点，如 https://auth.example.com/.well-known/openid-configuration
	Scopes       []string `json:"scopes"`       // 零值时默认 ["openid", "email", "profile"]
	AutoCreate   bool     `json:"autoCreate"`   // 首次登录无绑定时是否自动创建普通用户
}

// SettingsContentForSSOLDAP 表示 LDAP 提供者配置。
type SettingsContentForSSOLDAP struct {
	Enabled        bool   `json:"enabled"`
	DisplayName    string `json:"displayName,omitempty"`    // 登录页表单标题，零值时默认 "LDAP"
	ServerURL      string `json:"serverUrl"`                // 如 ldap://host:389 或 ldaps://host:636
	BindDN         string `json:"bindDn"`                   // 用于搜索用户的服务账号 DN
	BindPassword   string `json:"bindPassword"`             // 服务账号密码
	SearchBase     string `json:"searchBase"`               // 用户搜索基 DN
	SearchFilter   string `json:"searchFilter"`             // 搜索过滤器，{{username}} 为登录用户名占位符，零值时默认 (uid={{username}})
	EmailAttribute string `json:"emailAttribute,omitempty"` // 邮箱属性名，零值时默认 "mail"
	NameAttribute  string `json:"nameAttribute,omitempty"`  // 显示名属性名，零值时默认 "displayName"
	AutoCreate     bool   `json:"autoCreate"`               // 首次登录无绑定时是否自动创建普通用户
}

func (c SettingsContent) AsSSO() *SettingsContentForSSO {
	content := &SettingsContentForSSO{}
	xmaps.Populate(c, content)

	if content.OIDC != nil {
		if content.OIDC.DisplayName == "" {
			content.OIDC.DisplayName = "OIDC"
		}
		if len(content.OIDC.Scopes) == 0 {
			content.OIDC.Scopes = []string{"openid", "email", "profile"}
		}
	}
	if content.LDAP != nil {
		if content.LDAP.DisplayName == "" {
			content.LDAP.DisplayName = "LDAP"
		}
		if content.LDAP.SearchFilter == "" {
			content.LDAP.SearchFilter = "(uid={{username}})"
		}
		if content.LDAP.EmailAttribute == "" {
			content.LDAP.EmailAttribute = "mail"
		}
		if content.LDAP.NameAttribute == "" {
			content.LDAP.NameAttribute = "displayName"
		}
	}

	return content
}
