package routes

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/certificate"
	"github.com/certimate-go/certimate/internal/notify"
	"github.com/certimate-go/certimate/internal/oauth2"
	"github.com/certimate-go/certimate/internal/repository"
	"github.com/certimate-go/certimate/internal/rest/handlers"
	"github.com/certimate-go/certimate/internal/statistics"
	"github.com/certimate-go/certimate/internal/workflow"
)

var (
	certificateSvc *certificate.CertificateService
	workflowSvc    *workflow.WorkflowService
	statisticsSvc  *statistics.StatisticsService
	notifySvc      *notify.NotifyService
)

func BindRouter(router *router.Router[*core.RequestEvent]) {
	accessRepo := repository.NewAccessRepository()
	workflowRepo := repository.NewWorkflowRepository()
	workflowRunRepo := repository.NewWorkflowRunRepository()
	acmeAccountRepo := repository.NewACMEAccountRepository()
	certificateRepo := repository.NewCertificateRepository()
	statisticsRepo := repository.NewStatisticsRepository()

	certificateSvc = certificate.NewCertificateService(acmeAccountRepo, certificateRepo)
	workflowSvc = workflow.NewWorkflowService(workflowRepo, workflowRunRepo)
	statisticsSvc = statistics.NewStatisticsService(statisticsRepo)
	notifySvc = notify.NewNotifyService(accessRepo)

	authApiGroup := router.Group("/api")
	authApiGroup.Bind(apis.RequireSuperuserAuth())
	handlers.NewCertificatesHandler(authApiGroup, certificateSvc)
	handlers.NewWorkflowsHandler(authApiGroup, workflowSvc)
	handlers.NewStatisticsHandler(authApiGroup, statisticsSvc)
	handlers.NewNotificationsHandler(authApiGroup, notifySvc)

	// OAuth2 公共路由不在 /api 的 superuser auth 之下注册，
	// 避免跳转与回调被 401。9669 /api/oauth2/* 下仍属于顶层 Router。
	oauth2Svc := oauth2.NewService()
	handlers.NewOAuth2Handler(router, oauth2Svc)
}
