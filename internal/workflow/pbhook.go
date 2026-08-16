package workflow

import (
	"context"
	"slices"

	"github.com/pocketbase/pocketbase/core"

	"github.com/certimate-go/certimate/internal/app"
	"github.com/certimate-go/certimate/internal/authz"
	"github.com/certimate-go/certimate/internal/domain"
)

func registerWorkflowRecordEvents() {
	pb := app.GetApp()
	pb.OnRecordCreateRequest(domain.CollectionNameWorkflow).BindFunc(func(e *core.RecordRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if err := onWorkflowRecordCreateOrUpdate(e.Request.Context(), e.App, e.Record); err != nil {
			app.GetLogger().Error(err.Error())
			return err
		}

		return nil
	})
	pb.OnRecordUpdateRequest(domain.CollectionNameWorkflow).BindFunc(func(e *core.RecordRequestEvent) error {
		// 普通用户（非管理员）可以编辑被授权的工作流，但不得修改 grantedUsers（授权元数据），
		// 防止用户把自己加入其他工作流的授权列表实现自我提权。
		if e.Auth != nil && !authz.IsAdmin(e.Auth) {
			if original := e.Record.Original(); original != nil {
				if !slices.Equal(original.GetStringSlice("grantedUsers"), e.Record.GetStringSlice("grantedUsers")) {
					return e.ForbiddenError("Only admins can change workflow grants.", nil)
				}
			}
		}

		if err := e.Next(); err != nil {
			return err
		}

		if err := onWorkflowRecordCreateOrUpdate(e.Request.Context(), e.App, e.Record); err != nil {
			app.GetLogger().Error(err.Error())
			return err
		}

		return nil
	})
	pb.OnRecordDeleteRequest(domain.CollectionNameWorkflow).BindFunc(func(e *core.RecordRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if err := onWorkflowRecordDelete(e.Request.Context(), e.App, e.Record); err != nil {
			app.GetLogger().Error(err.Error())
			return err
		}

		return nil
	})
}

func onWorkflowRecordCreateOrUpdate(_ context.Context, _ core.App, record *core.Record) error {
	scheduler := app.GetScheduler()

	// 向数据库插入/更新时，同时更新定时任务
	enabled := record.GetBool("enabled")
	trigger := record.GetString("trigger")
	triggerCron := record.GetString("triggerCron")

	// 如果非定时触发或未启用，移除定时任务
	if !enabled || trigger != domain.WorkflowTriggerTypeScheduled.String() {
		scheduler.Remove(buildPbJobKey(record.Id))
		return nil
	}

	// 反之，重新添加定时任务
	if err := registerWorkflowJob(thisSvcInst(), record.Id, triggerCron); err != nil {
		return err
	}

	return nil
}

func onWorkflowRecordDelete(_ context.Context, _ core.App, record *core.Record) error {
	scheduler := app.GetScheduler()

	// 从数据库删除时，同时移除定时任务
	scheduler.Remove(buildPbJobKey(record.Id))

	return nil
}
