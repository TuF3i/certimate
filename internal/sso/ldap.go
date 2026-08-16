package sso

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"github.com/certimate-go/certimate/internal/domain"
)

// ldapUser 表示 LDAP 认证成功的用户信息。
type ldapUser struct {
	DN    string
	Email string
	Name  string
}

// authenticateLDAPUser 使用服务账号搜索用户 DN，再用用户 DN 绑定验证密码。
// searchFilter 中 {{username}} 会被替换为登录用户名。
func (s *Service) authenticateLDAPUser(ctx context.Context, cfg *domain.SettingsContentForSSOLDAP, username, password string) (*ldapUser, error) {
	conn, err := dialLDAP(ctx, cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// 1. 服务账号绑定（用于搜索）
	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("failed to bind with service account: %w", err)
	}

	// 2. 搜索用户 DN
	escaped := ldap.EscapeFilter(username)
	filter := strings.ReplaceAll(cfg.SearchFilter, "{{username}}", escaped)
	searchReq := ldap.NewSearchRequest(
		cfg.SearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // sizeLimit
		0, // timeLimit
		false,
		filter,
		[]string{"dn", cfg.EmailAttribute, cfg.NameAttribute},
		nil,
	)

	searchResult, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}
	if len(searchResult.Entries) == 0 {
		return nil, fmt.Errorf("LDAP user %q not found", username)
	}
	entry := searchResult.Entries[0]

	// 3. 用户绑定验证密码
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	return &ldapUser{
		DN:    entry.DN,
		Email: entry.GetAttributeValue(cfg.EmailAttribute),
		Name:  entry.GetAttributeValue(cfg.NameAttribute),
	}, nil
}

// dialLDAP 建立 LDAP 连接；ldaps:// 使用 TLS。
func dialLDAP(ctx context.Context, serverURL string) (*ldap.Conn, error) {
	scheme := "ldap"
	if idx := strings.Index(serverURL, "://"); idx >= 0 {
		scheme = serverURL[:idx]
	}

	switch scheme {
	case "ldaps":
		return ldap.DialURL(serverURL, ldap.DialWithTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}))
	case "ldap":
		return ldap.DialURL(serverURL)
	default:
		return nil, fmt.Errorf("unsupported LDAP scheme %q (use ldap:// or ldaps://)", scheme)
	}
}
