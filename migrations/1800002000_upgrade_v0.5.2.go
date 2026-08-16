package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/certimate-go/certimate/internal/domain"
)

func init() {
	m.Register(func(app core.App) error {
		tracer := NewTracer("v0.5.2")
		tracer.Printf("go ...")

		// settings 集合：OAuth2 设置项更名为「单点登录」（name: oauth2 -> sso），
		// 旧的多提供者配置结构已废弃，content 重置为空结构。
		{
			oldRecord, err := app.FindFirstRecordByFilter(domain.CollectionNameSettings, "name={:name}", map[string]any{"name": "oauth2"})
			if err == nil && oldRecord != nil {
				// 若已存在 sso 记录则删除旧的 oauth2 记录，否则改名复用。
				if _, err := app.FindFirstRecordByFilter(domain.CollectionNameSettings, "name={:name}", map[string]any{"name": "sso"}); err == nil {
					if err := app.Delete(oldRecord); err != nil {
						return err
					}
					tracer.Printf("old settings 'oauth2' record removed")
				} else {
					oldRecord.Set("name", "sso")
					oldRecord.Set("content", domain.SettingsContent{})
					if err := app.Save(oldRecord); err != nil {
						return err
					}
					tracer.Printf("settings 'oauth2' renamed to 'sso'")
				}
			} else {
				// 全新部署：预置空 sso 记录。
				settingsCollection, err := app.FindCollectionByNameOrId(domain.CollectionNameSettings)
				if err != nil {
					return err
				}
				if _, err := app.FindFirstRecordByFilter(domain.CollectionNameSettings, "name={:name}", map[string]any{"name": "sso"}); err != nil {
					record := core.NewRecord(settingsCollection)
					record.Set("name", "sso")
					record.Set("content", domain.SettingsContent{})
					if err := app.Save(record); err != nil {
						return err
					}
					tracer.Printf("settings 'sso' initialized")
				}
			}
		}

		tracer.Printf("done")
		return nil
	}, func(app core.App) error {
		return errors.ErrUnsupported
	})
}
