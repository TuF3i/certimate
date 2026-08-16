package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/certimate-go/certimate/internal/domain"
)

// adminRule 表示「管理员」判定：超级管理员，或 users 集合中 role=admin 的成员。
const adminRule = "@request.auth.collectionName = '_superusers' || (@request.auth.collectionName = 'users' && @request.auth.role = 'admin')"

// grantRule 表示「被授权用户」判定：管理员，或位于该工作流 grantedUsers 中的用户。
const grantRule = adminRule + " || (@request.auth.collectionName = 'users' && grantedUsers.id ?= @request.auth.id)"

// grantRuleByWorkflow 用于通过 workflowRef 关联派生可见性的集合（证书、运行记录、日志、输出）。
const grantRuleByWorkflow = adminRule + " || (@request.auth.collectionName = 'users' && workflowRef.grantedUsers.id ?= @request.auth.id)"

func init() {
	m.Register(func(app core.App) error {
		tracer := NewTracer("v0.5.1")
		tracer.Printf("go ...")

		// users 集合增加 role 字段（"user" / "admin"），默认 "user"。
		{
			collection, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("role") == nil {
				collection.Fields.Add(&core.TextField{
					Name:     "role",
					Max:      32,
					Pattern:  "^(user|admin)$",
					Required: true,
				})
				if err := app.Save(collection); err != nil {
					return err
				}
				tracer.Printf("collection 'users' field 'role' added")
			}
		}

		// 存量成员默认普通用户角色；admin@certimate.fun 是 _superusers（天然管理员），不在此集合。
		{
			collection, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			records, err := app.FindAllRecords(collection)
			if err != nil {
				return err
			}
			for _, record := range records {
				changed := false
				if record.GetString("role") == "" {
					record.Set("role", "user")
					changed = true
				}
				if changed {
					if err := app.Save(record); err != nil {
						return err
					}
				}
			}
		}

		// workflow 集合增加 grantedUsers 多选 relation（指向 users），用于工作流粒度授权。
		// 注意：RelationField.CollectionId 必须为目标集合的 ID（非名字）。
		{
			collection, err := app.FindCollectionByNameOrId(domain.CollectionNameWorkflow)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("grantedUsers") == nil {
				usersCollection, err := app.FindCollectionByNameOrId("users")
				if err != nil {
					return err
				}

				collection.Fields.Add(&core.RelationField{
					Name:         "grantedUsers",
					CollectionId: usersCollection.Id,
					MaxSelect:    999,
					MinSelect:    0,
				})
				if err := app.Save(collection); err != nil {
					return err
				}
				tracer.Printf("collection 'workflow' field 'grantedUsers' added")
			}
		}

		// oauth2_link 增加 targetCollection 字段，标识关联目标集合（"_superusers" / "users"）。
		{
			collection, err := app.FindCollectionByNameOrId(domain.CollectionNameOAuth2Link)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("targetCollection") == nil {
				collection.Fields.Add(&core.TextField{
					Name:     "targetCollection",
					Max:      64,
					Required: true,
				})
				if err := app.Save(collection); err != nil {
					return err
				}
				tracer.Printf("collection 'oauth2_link' field 'targetCollection' added")
			}

			// 存量绑定均为超级管理员，回填 targetCollection。
			records, err := app.FindAllRecords(collection)
			if err != nil {
				return err
			}
			for _, record := range records {
				if record.GetString("targetCollection") == "" {
					record.Set("targetCollection", core.CollectionNameSuperusers)
					if err := app.Save(record); err != nil {
						return err
					}
				}
			}
		}

		// 收紧集合访问规则：
		//   - access / settings（含预设模板）/ acme_accounts / oauth2_link：仅管理员
		//   - workflow：管理员可见全部；普通用户仅见被授权的工作流（只读）
		//   - certificate / workflow_run / workflow_logs / workflow_output：随 workflowRef 授权派生
		//   - users：管理员可管理；成员可查看/更新自己
		{
			adminOnly := []string{
				domain.CollectionNameAccess,
				domain.CollectionNameSettings,
				domain.CollectionNameACMEAccount,
				domain.CollectionNameOAuth2Link,
			}
			for _, name := range adminOnly {
				collection, err := app.FindCollectionByNameOrId(name)
				if err != nil {
					return err
				}
				if collection.ListRule == nil || *collection.ListRule != adminRule {
					collection.ListRule = types.Pointer(adminRule)
					collection.ViewRule = types.Pointer(adminRule)
					collection.CreateRule = types.Pointer(adminRule)
					collection.UpdateRule = types.Pointer(adminRule)
					collection.DeleteRule = types.Pointer(adminRule)
					if err := app.Save(collection); err != nil {
						return err
					}
					tracer.Printf("collection '%s' rules restricted to admins", name)
				}
			}

			workflowCol, err := app.FindCollectionByNameOrId(domain.CollectionNameWorkflow)
			if err != nil {
				return err
			}
			if workflowCol.ListRule == nil || *workflowCol.ListRule != grantRule {
				workflowCol.ListRule = types.Pointer(grantRule)
				workflowCol.ViewRule = types.Pointer(grantRule)
				workflowCol.CreateRule = types.Pointer(adminRule)
				workflowCol.UpdateRule = types.Pointer(adminRule)
				workflowCol.DeleteRule = types.Pointer(adminRule)
				if err := app.Save(workflowCol); err != nil {
					return err
				}
				tracer.Printf("collection 'workflow' rules updated (grant-based)")
			}

			derived := []string{
				domain.CollectionNameCertificate,
				domain.CollectionNameWorkflowRun,
				domain.CollectionNameWorkflowLog,
				domain.CollectionNameWorkflowOutput,
			}
			for _, name := range derived {
				collection, err := app.FindCollectionByNameOrId(name)
				if err != nil {
					return err
				}
				if collection.ListRule == nil || *collection.ListRule != grantRuleByWorkflow {
					collection.ListRule = types.Pointer(grantRuleByWorkflow)
					collection.ViewRule = types.Pointer(grantRuleByWorkflow)
					collection.CreateRule = types.Pointer(adminRule)
					collection.UpdateRule = types.Pointer(adminRule)
					collection.DeleteRule = types.Pointer(adminRule)
					if err := app.Save(collection); err != nil {
						return err
					}
					tracer.Printf("collection '%s' rules updated (grant-derived)", name)
				}
			}

			usersCol, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			usersCol.ListRule = types.Pointer(adminRule)
			usersCol.CreateRule = types.Pointer(adminRule)
			usersCol.DeleteRule = types.Pointer(adminRule)
			usersCol.ViewRule = types.Pointer("id = @request.auth.id || (" + adminRule + ")")
			usersCol.UpdateRule = types.Pointer("id = @request.auth.id || (" + adminRule + ")")
			if err := app.Save(usersCol); err != nil {
				return err
			}
			tracer.Printf("collection 'users' rules updated (role-aware)")
		}

		tracer.Printf("done")
		return nil
	}, func(app core.App) error {
		return errors.ErrUnsupported
	})
}
