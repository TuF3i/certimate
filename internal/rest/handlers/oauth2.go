package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/domain/dtos"
	"github.com/certimate-go/certimate/internal/oauth2"
	"github.com/certimate-go/certimate/internal/rest/resp"
)

type OAuth2Handler struct {
	service *oauth2.Service
}

func NewOAuth2Handler(router *router.Router[*core.RequestEvent], service *oauth2.Service) *OAuth2Handler {
	handler := &OAuth2Handler{service: service}

	group := router.Group("/api/oauth2")
	// 这些路由不应该被 /api 的 RequireSuperuserAuth 中间件保护，
	// 在不绑定鉴权 hook 的 group 下注册。
	group.GET("/providers", handler.listProviders)
	group.GET("/redirect", handler.redirect)
	group.GET("/callback", handler.callback)

	return handler
}

// listProviders 列出已启用的 OAuth2 提供商（无需鉴权），供登录页展示按钮。
func (h *OAuth2Handler) listProviders(e *core.RequestEvent) error {
	list := h.service.ListEnabledProviders(e.Request.Context())
	out := make([]dtos.OAuth2Provider, 0, len(list))
	for _, p := range list {
		out = append(out, dtos.OAuth2Provider{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Enabled:     p.Enabled,
			RedirectURL: p.RedirectURL,
			Scopes:      p.Scopes,
			AuthURL:     p.AuthURL,
		})
	}

	return resp.Ok(e, &dtos.OAuth2ListProvidersResp{Providers: out})
}

// redirect 处理登录页跳转到 OAuth2 提供商的请求。
func (h *OAuth2Handler) redirect(e *core.RequestEvent) error {
	providerName := e.Request.URL.Query().Get("provider")
	if providerName == "" {
		return resp.Err(e, oauth2.ErrInvalidProvider)
	}
	redirectURL := e.Request.URL.Query().Get("redirectUrl")

	authURL, err := h.service.BuildAuthorizeURL(e.Request.Context(), providerName, redirectURL)
	if err != nil {
		return resp.Err(e, err)
	}

	return e.Redirect(http.StatusTemporaryRedirect, authURL)
}

// callback 处理 OAuth2 提供商的回调：领取 code+state，颁发 PocketBase 超级管理员 token，
// 然后以 302 跳回前端并附带 token/provider/provider 等查询参数。
func (h *OAuth2Handler) callback(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	providerName := q.Get("provider")
	code := q.Get("code")
	state := q.Get("state")
	redirectURL := q.Get("redirectUrl")

	if providerName == "" || state == "" {
		return resp.Err(e, oauth2.ErrInvalidProvider)
	}

	superuser, token, err := h.service.HandleCallback(e.Request.Context(), providerName, code, state, redirectURL)
	if err != nil {
		return resp.Err(e, err)
	}

	// 把 token 写入 HttpOnly cookie，这样前端在跳转后可以读取却又不易被脚本读取。
	// 为兼容 PocketBase 前端 SDK 的 authStore.save 流程，我们同时也通过 query 返回 record 元数据。
	record := dtos.OAuth2Identity{
		Id:       superuser.Id,
		Email:    superuser.Email(),
		Verified: superuser.GetBool("verified"),
		Username: superuser.GetString("username"),
		Created:  superuser.GetDateTime("created").String(),
		Updated:  superuser.GetDateTime("updated").String(),
	}
	if avatar := superuser.GetString("avatar"); avatar != "" {
		record.AvatarUrl = avatar
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
	qp.Set("oauth2_token", token)
	qp.Set("oauth2_email", record.Email)
	qp.Set("oauth2_provider", providerName)
	sep := "?"
	if strings.Contains(redirectTarget, "?") {
		sep = "&"
	}
	finalURL := redirectTarget + sep + qp.Encode()

	// 同时写一份 HttpOnly cookie，前端可通过它免 query 暴露完成登录。
	http.SetCookie(e.Response, &http.Cookie{
		Name:     "certimate_oauth2_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60,
	})

	return e.Redirect(http.StatusTemporaryRedirect, finalURL)
}
