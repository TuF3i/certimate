package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/domain/dtos"
	"github.com/certimate-go/certimate/internal/rest/resp"
	"github.com/certimate-go/certimate/internal/sso"
)

type SSOHandler struct {
	service *sso.Service
}

func NewSSOHandler(router *router.Router[*core.RequestEvent], service *sso.Service) *SSOHandler {
	handler := &SSOHandler{service: service}

	group := router.Group("/api/sso")
	// 这些路由不应受 /api 的鉴权中间件保护（登录前必须可达）。
	group.GET("/config", handler.getConfig)
	group.GET("/redirect", handler.redirect)
	group.GET("/callback", handler.callback)
	group.POST("/ldap/login", handler.ldapLogin)

	return handler
}

// getConfig 返回脱敏后的 SSO 配置（OIDC + LDAP），供登录页与设置页使用。
func (h *SSOHandler) getConfig(e *core.RequestEvent) error {
	cfg := h.service.GetConfig(e.Request.Context())
	return resp.Ok(e, &dtos.SSOConfigResp{
		Config:       cfg,
		OIDCCallback: h.callbackURL(e),
	})
}

// redirect 触发 OIDC 授权跳转。
func (h *SSOHandler) redirect(e *core.RequestEvent) error {
	oidcCfg, err := h.service.GetEnabledOIDC(e.Request.Context())
	if err != nil {
		return resp.Err(e, err)
	}

	authURL, err := h.service.BuildAuthorizeURL(e.Request.Context(), oidcCfg, h.callbackURL(e))
	if err != nil {
		return resp.Err(e, err)
	}

	return e.Redirect(http.StatusTemporaryRedirect, authURL)
}

// callback 处理 OIDC 回调：颁发 token 后 307 跳回前端。
func (h *SSOHandler) callback(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	code := q.Get("code")
	state := q.Get("state")

	if state == "" {
		return resp.Err(e, domain.NewError(400, "invalid sso callback"))
	}

	account, token, err := h.service.HandleOIDCCallback(e.Request.Context(), code, state, h.callbackURL(e))
	if err != nil {
		return resp.Err(e, err)
	}

	redirectTarget := q.Get("returnUrl")
	if redirectTarget == "" {
		redirectTarget = "/login"
	}
	u, perr := url.Parse(redirectTarget)
	if perr != nil || (u.IsAbs() && u.Host != e.Request.URL.Host) {
		redirectTarget = "/login"
	}

	qp := url.Values{}
	qp.Set("sso_token", token)
	qp.Set("sso_email", account.Email())
	sep := "?"
	if strings.Contains(redirectTarget, "?") {
		sep = "&"
	}
	finalURL := redirectTarget + sep + qp.Encode()

	http.SetCookie(e.Response, &http.Cookie{
		Name:     "certimate_sso_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60,
	})

	return e.Redirect(http.StatusTemporaryRedirect, finalURL)
}

// ldapLogin 处理 LDAP 用户名密码登录，返回 { token, record }。
func (h *SSOHandler) ldapLogin(e *core.RequestEvent) error {
	req := &dtos.SSOLdapLoginReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}

	account, token, err := h.service.HandleLDAPLogin(e.Request.Context(), req.Username, req.Password)
	if err != nil {
		return resp.Err(e, domain.NewError(400, err.Error()))
	}

	account.IgnoreEmailVisibility(true)
	return resp.Ok(e, map[string]any{
		"token":  token,
		"record": account,
	})
}

// callbackURL 计算当前请求对应的 OIDC 回调地址。
func (h *SSOHandler) callbackURL(e *core.RequestEvent) string {
	scheme := e.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if e.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return sso.BuildRedirectURL(scheme, e.Request.Host)
}
