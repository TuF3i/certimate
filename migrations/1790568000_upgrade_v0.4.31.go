package migrations

import (
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/certimate-go/certimate/internal/domain"
)

func init() {
	m.Register(func(app core.App) error {
		tracer := NewTracer("v0.4.31")
		tracer.Printf("go ...")

		// create `oauth2_link` collection
		//   key: (provider, subjectId) -> superuserId
		{
			if _, err := app.FindCollectionByNameOrId(domain.CollectionNameOAuth2Link); err == nil {
				// 已存在，跳过
			} else {
				col := core.NewBaseCollection(domain.CollectionNameOAuth2Link)
				col.System = false

				col.Fields.Add(&core.TextField{
					Name:     "provider",
					Required: true,
					Min:      1,
					Max:      64,
					Pattern:  "^[a-zA-Z0-9_.-]+$",
				})
				col.Fields.Add(&core.TextField{
					Name:     "subjectId",
					Required: true,
					Min:      1,
					Max:      256,
				})
				col.Fields.Add(&core.TextField{
					Name:     "superuserId",
					Required: true,
					Min:      1,
					Max:      32,
				})
				col.Fields.Add(&core.TextField{
					Name: "userProfileEmail",
					Max:  320,
				})
				col.Fields.Add(&core.TextField{
					Name: "userProfileName",
					Max:  128,
				})
				col.Fields.Add(&core.TextField{
					Name: "userProfileAvatar",
					Max:  2048,
				})
				col.Fields.Add(&core.AutodateField{
					Name:     "created",
					OnCreate: true,
				})
				col.Fields.Add(&core.AutodateField{
					Name:     "updated",
					OnCreate: true,
					OnUpdate: true,
				})

				col.AddIndex("idx_oauth2_link_provider_subject", true, "provider, subjectId", "")
				col.AddIndex("idx_oauth2_link_provider_super", false, "provider, superuserId", "")

				if err := app.Save(col); err != nil {
					return err
				}

				tracer.Printf("collection '%s' created", domain.CollectionNameOAuth2Link)
			}
		}

		// pre-create empty `settings` row for `oauth2`
		{
			if _, err := app.FindFirstRecordByFilter(domain.CollectionNameSettings, "name={:name}", dbx.Params{"name": domain.SettingsNameOAuth2}); err != nil {
				settingsCollection, err := app.FindCollectionByNameOrId(domain.CollectionNameSettings)
				if err != nil {
					return err
				}

				record := core.NewRecord(settingsCollection)
				record.Set("name", domain.SettingsNameOAuth2)
				record.Set("content", domain.SettingsContent{"providers": []any{}})
				if err := app.Save(record); err != nil {
					return err
				}

				tracer.Printf("settings '%s' initialized", domain.SettingsNameOAuth2)
			}
		}

		tracer.Printf("done")
		return nil
	}, func(app core.App) error {
		return errors.ErrUnsupported
	})
}
