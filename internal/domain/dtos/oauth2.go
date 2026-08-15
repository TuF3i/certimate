package dtos

type OAuth2Provider struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Enabled     bool     `json:"enabled"`
	RedirectURL string   `json:"redirectUrl,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	AuthURL     string   `json:"authUrl,omitempty"`
}

type OAuth2ListProvidersResp struct {
	Providers []OAuth2Provider `json:"providers"`
}

type OAuth2RedirectReq struct {
	Provider       string `json:"provider"       bind:"query"`
	RedirectURL    string `json:"redirectUrl,omitempty" bind:"query"`
}

type OAuth2CallbackReq struct {
	Provider    string `json:"provider"    bind:"query"`
	Code        string `json:"code"        bind:"query"`
	State       string `json:"state"       bind:"query"`
	RedirectURL string `json:"redirectUrl,omitempty" bind:"query"`
}

type OAuth2CallbackResp struct {
	Token     string         `json:"token"`
	Record    OAuth2Identity `json:"record"`
}

type OAuth2Identity struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username,omitempty"`
	Verified  bool   `json:"verified"`
	AvatarUrl string `json:"avatarUrl,omitempty"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}
