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
		tracer := NewTracer("v0.5.3")
		tracer.Printf("go ...")

		// 工作流集合：UpdateRule 放开为授权可见即允许编辑（grantRule），
		// 创建/删除仍仅限管理员；grantedUsers 的修改由服务端 hook 单独保护。
		{
			collection, err := app.FindCollectionByNameOrId(domain.CollectionNameWorkflow)
			if err != nil {
				return err
			}
			if collection.UpdateRule == nil || *collection.UpdateRule != grantRule {
				collection.UpdateRule = types.Pointer(grantRule)
				if err := app.Save(collection); err != nil {
					return err
				}
				tracer.Printf("collection 'workflow' updateRule opened to granted users")
			}
		}

		tracer.Printf("done")
		return nil
	}, func(app core.App) error {
		return errors.ErrUnsupported
	})
}
