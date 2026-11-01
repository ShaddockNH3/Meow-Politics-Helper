package httpapi

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"quiz/internal/course"
	"quiz/internal/quiz"
	"quiz/internal/service"
	"quiz/internal/session"
	"quiz/internal/store"
)

// Handlers 聚合所有依赖，供各 handler 使用。
type Handlers struct {
	Store    *store.Store
	Registry *course.Registry
	Quiz     *quiz.Service
	Banks    *service.BankService
	Sessions *session.Manager
}

// NewHandlers 装配依赖。
func NewHandlers(st *store.Store, registry *course.Registry, quizSvc *quiz.Service, banks *service.BankService, sessions *session.Manager) *Handlers {
	return &Handlers{Store: st, Registry: registry, Quiz: quizSvc, Banks: banks, Sessions: sessions}
}

// RegisterRoutes 注册静态资源与全部 API 路由。
func RegisterRoutes(h *server.Hertz, embeddedFS embed.FS, handlers *Handlers) {
	registerStaticRoutes(h, embeddedFS)

	api := h.Group("/api")
	{
		api.GET("/network/info", handlers.NetworkInfo)

		api.GET("/courses", handlers.CoursesList)
		api.POST("/courses/delete", handlers.DeleteCustomCourse)
		api.POST("/courses/import-json", handlers.ImportJSONQuestionBank)
		api.GET("/courses/custom/detail", handlers.CustomCourseDetail)
		api.POST("/courses/custom/save", handlers.SaveCustomCourse)

		api.POST("/session/init", handlers.InitSession)

		api.POST("/review/start", handlers.QuickReviewStart)
		api.POST("/review/next", handlers.GetNextQuestion)

		api.POST("/quiz/start", handlers.QuizStart)
		api.POST("/quiz/submit_answer", handlers.SubmitAnswer)

		api.POST("/incorrect_questions/review/start", handlers.IncorrectReviewStart)
		api.POST("/incorrect_questions/review/submit_answer", handlers.SubmitIncorrectReviewAnswer)
		api.POST("/incorrect_questions/delete", handlers.DeleteIncorrectQuestion)

		api.POST("/user/data/clear", handlers.UserDataClear)
	}
}

// registerStaticRoutes 提供嵌入的 quiz.html 与 assets 静态文件。
func registerStaticRoutes(h *server.Hertz, embeddedFS embed.FS) {
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		data, err := embeddedFS.ReadFile("quiz.html")
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		c.Data(consts.StatusOK, "text/html; charset=utf-8", data)
	})

	assetsFS, err := fs.Sub(embeddedFS, "assets")
	if err != nil {
		log.Fatalf("无法访问嵌入的 assets 目录: %v", err)
	}
	h.GET("/assets/:file", func(ctx context.Context, c *app.RequestContext) {
		fileName := path.Clean(c.Param("file"))
		if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
			c.String(consts.StatusBadRequest, "Bad request")
			return
		}
		data, err := fs.ReadFile(assetsFS, fileName)
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(fileName, ".js"):
			contentType = "application/javascript; charset=utf-8"
		case strings.HasSuffix(fileName, ".css"):
			contentType = "text/css; charset=utf-8"
		}
		c.Data(consts.StatusOK, contentType, data)
	})
}

func badRequest(c *app.RequestContext, message string) {
	c.JSON(consts.StatusBadRequest, utils.H{"error": message})
}

func serverError(c *app.RequestContext, message string) {
	c.JSON(consts.StatusInternalServerError, utils.H{"error": message})
}
