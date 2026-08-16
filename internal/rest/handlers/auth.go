package handlers

import (
	"net/netip"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/app"
	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/rest/resp"
)

type authHandler struct{}

func NewAuthHandler(router *router.Router[*core.RequestEvent]) *authHandler {
	handler := &authHandler{}

	// 公开端点：统一登录入口，自动区分超级管理员（_superusers）与成员（users）。
	group := router.Group("/api/auth")
	group.POST("/login", handler.login)

	return handler
}

type authLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login 接受邮箱 + 密码，依次尝试 _superusers 与 users 集合，
// 命中即签发对应集合的标准 auth token。两个集合都失败时返回统一的错误信息，
// 避免泄露该邮箱是否已被注册。
func (h *authHandler) login(e *core.RequestEvent) error {
	req := &authLoginReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}
	if req.Username == "" || req.Password == "" {
		return resp.Err(e, domain.NewError(400, "invalid params"))
	}

	// 先尝试超级管理员，再尝试成员账号。
	for _, collectionName := range []string{core.CollectionNameSuperusers, "users"} {
		record, err := app.GetApp().FindAuthRecordByEmail(collectionName, strings.TrimSpace(req.Username))
		if err != nil || record == nil {
			continue
		}
		if !record.ValidatePassword(req.Password) {
			continue
		}

		// 与 PocketBase 内置登录一致：超级管理员若配置了 IP 白名单则校验来源 IP。
		if record.IsSuperuser() {
			allowedIPs := app.GetApp().Settings().SuperuserIPs
			if len(allowedIPs) > 0 && !isIPInList(allowedIPs, e.RealIP()) {
				return resp.Err(e, domain.NewError(403, "superuser IP is not whitelisted"))
			}
		}

		token, err := record.NewAuthToken()
		if err != nil {
			return resp.Err(e, err)
		}

		// 与标准登录响应一致：强制导出 email 字段。
		record.IgnoreEmailVisibility(true)

		return resp.Ok(e, map[string]any{
			"token":  token,
			"record": record,
		})
	}

	return resp.Err(e, domain.NewError(400, "invalid username or password"))
}

// isIPInList 检查指定 IP 是否属于列表中的单个 IP 或 CIDR 子网。
func isIPInList(ipsOrSubnets []string, ip string) bool {
	if len(ipsOrSubnets) == 0 || ip == "" {
		return false
	}

	searchAddr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	for _, item := range ipsOrSubnets {
		// subnet?
		if prefix, err := netip.ParsePrefix(item); err == nil {
			if prefix.Contains(searchAddr) {
				return true
			}
			continue
		}

		// individual ip?
		if addr, err := netip.ParseAddr(item); err == nil {
			if addr == searchAddr {
				return true
			}
			continue
		}
	}

	return false
}
