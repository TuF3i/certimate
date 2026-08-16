package dtos

import "github.com/certimate-go/certimate/internal/domain"

// SSOConfigResp 返回脱敏的 SSO 配置与统一回调地址。
type SSOConfigResp struct {
	Config       *domain.SettingsContentForSSO `json:"config"`
	OIDCCallback string                        `json:"oidcCallback"`
}

// SSOLdapLoginReq LDAP 登录请求。
type SSOLdapLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
