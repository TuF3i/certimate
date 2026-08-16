package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// oidcDiscovery 表示 OIDC Discovery 文档（仅取需要的字段）。
type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (s *Service) fetchOIDCDiscovery(ctx context.Context, discoveryURL string) (*oidcDiscovery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("discovery endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var doc oidcDiscovery
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("could not parse discovery document: %w", err)
	}
	if doc.TokenEndpoint == "" || doc.UserInfoEndpoint == "" {
		return nil, fmt.Errorf("discovery document is missing token_endpoint/userinfo_endpoint")
	}
	return &doc, nil
}

func (s *Service) exchangeOIDCCode(ctx context.Context, clientID, clientSecret string, discovery *oidcDiscovery, code, redirectURL string) (string, error) {
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("could not parse token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("token response does not contain access_token: %s", strings.TrimSpace(string(raw)))
	}
	return parsed.AccessToken, nil
}

func (s *Service) fetchOIDCUserInfo(ctx context.Context, discovery *oidcDiscovery, accessToken string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserInfoEndpoint, nil)
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
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var profile map[string]any
	if err := json.Unmarshal(raw, &profile); err != nil {
		return nil, fmt.Errorf("could not parse userinfo response: %w", err)
	}
	return profile, nil
}

// BuildRedirectURL 由请求侧计算统一的 OIDC 回调地址：<scheme>://<host>/api/sso/callback。
func BuildRedirectURL(scheme, host string) string {
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/api/sso/callback", scheme, host)
}

// stringField 从 JSON 对象中安全读取字符串字段。
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
