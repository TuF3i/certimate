package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/certimate-go/certimate/internal/domain"
)

func init() {
	m.Register(func(app core.App) error {
		tracer := NewTracer("v0.5.0")
		tracer.Printf("go ...")

		// create `users` auth collection
		// 成员账号：仅超级管理员可创建/管理（规则保持默认 null = superuser only），
		// 成员自身通过 /api/collections/users/auth-with-password 登录。
		{
			if _, err := app.FindCollectionByNameOrId("users"); err == nil {
				tracer.Printf("collection 'users' already exists, skip")
			} else {
				col := core.NewAuthCollection("users")
				col.Fields.Add(&core.TextField{
					Name: "name",
					Max:  128,
				})
				// 成员可查看/更新自己的记录（含密码），但不能枚举其他用户；
				// 创建与删除仅超级管理员（规则保持 null = superuser only）。
				col.ViewRule = types.Pointer("id = @request.auth.id")
				col.UpdateRule = types.Pointer("id = @request.auth.id")
				if err := app.Save(col); err != nil {
					return err
				}
				tracer.Printf("collection 'users' created")
			}
		}

		// 多用户（数据共享）模式：将业务集合的访问规则放开为「任意已登录用户」。
		// settings 集合保持 superuser only（普通用户不可修改全局设置）。
		{
			rule := "@request.auth.id != ''"
			names := []string{
				domain.CollectionNameAccess,
				domain.CollectionNameCertificate,
				domain.CollectionNameWorkflow,
				domain.CollectionNameWorkflowRun,
				domain.CollectionNameWorkflowLog,
				domain.CollectionNameWorkflowOutput,
				domain.CollectionNameACMEAccount,
				domain.CollectionNameOAuth2Link,
			}
			for _, name := range names {
				collection, err := app.FindCollectionByNameOrId(name)
				if err != nil {
					return err
				}
				if collection.ListRule == nil || *collection.ListRule != rule {
					collection.ListRule = types.Pointer(rule)
					collection.ViewRule = types.Pointer(rule)
					collection.CreateRule = types.Pointer(rule)
					collection.UpdateRule = types.Pointer(rule)
					collection.DeleteRule = types.Pointer(rule)
					if err := app.Save(collection); err != nil {
						return err
					}
					tracer.Printf("collection '%s' rules opened to authenticated users", name)
				}
			}
		}

		tracer.Printf("done")
		return nil
	}, func(app core.App) error {
		return errors.ErrUnsupported
	})
}
