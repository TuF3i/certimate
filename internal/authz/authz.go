package authz

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/certimate-go/certimate/internal/domain"
)

// CollectionNameUsers 普通成员账号集合名。
const CollectionNameUsers = "users"

// UserRoleAdmin 普通用户中的管理员角色。
const UserRoleAdmin = "admin"

// IsAdmin 判断认证记录是否为管理员：
// 超级管理员（_superusers），或 users 集合中 role=admin 的成员。
func IsAdmin(auth *core.Record) bool {
	if auth == nil {
		return false
	}
	if auth.IsSuperuser() {
		return true
	}
	return auth.Collection().Name == CollectionNameUsers && auth.GetString("role") == UserRoleAdmin
}

// IsMember 判断认证记录是否为普通成员（users 集合且非管理员）。
func IsMember(auth *core.Record) bool {
	if auth == nil {
		return false
	}
	return auth.Collection().Name == CollectionNameUsers && auth.GetString("role") != UserRoleAdmin
}

// CanAccessWorkflow 判断认证记录能否访问指定工作流：
// 管理员可见全部；普通成员仅当工作流 grantedUsers 包含其 id。
func CanAccessWorkflow(ctx context.Context, auth *core.Record, workflow *domain.Workflow) bool {
	if auth == nil {
		return false
	}
	if IsAdmin(auth) {
		return true
	}
	if workflow == nil {
		return false
	}
	for _, userId := range workflow.GrantedUsers {
		if userId == auth.Id {
			return true
		}
	}
	return false
}

// CanAccessCertificate 判断认证记录能否访问指定证书：
// 管理员可见全部；普通成员仅当证书归属的工作流被授权给该成员。
func CanAccessCertificate(ctx context.Context, auth *core.Record, certificate *domain.Certificate, workflow *domain.Workflow) bool {
	if auth == nil {
		return false
	}
	if IsAdmin(auth) {
		return true
	}
	if certificate == nil {
		return false
	}
	return CanAccessWorkflow(ctx, auth, workflow)
}

// ErrForbidden 表示无权限访问。
func ErrForbidden() error {
	return fmt.Errorf("forbidden")
}
