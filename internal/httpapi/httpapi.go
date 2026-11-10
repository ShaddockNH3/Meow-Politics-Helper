package httpapi

import (
	"context"
	"embed"
	"io/fs"
	"log"
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

// registerStaticRoutes 提供嵌入的前端构建产物（web/dist）。
// 前端是 Vite 打包的单页应用：/ 返回 index.html，/assets/* 返回静态资源，
// 其余路径回落到 index.html（SPA 行为）。
func registerStaticRoutes(h *server.Hertz, embeddedFS embed.FS) {
	distFS, err := fs.Sub(embeddedFS, "web/dist")
	if err != nil {
		log.Fatalf("无法访问嵌入的前端产物 web/dist（先执行 cd web && npm run build）: %v", err)
	}
	assetsFS, err := fs.Sub(distFS, "assets")
	if err != nil {
		log.Fatalf("无法访问前端资源目录 web/dist/assets: %v", err)
	}

	serveIndex := func(ctx context.Context, c *app.RequestContext) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		c.Data(consts.StatusOK, "text/html; charset=utf-8", data)
	}

	h.GET("/", serveIndex)

	h.GET("/assets/*filepath", func(ctx context.Context, c *app.RequestContext) {
		name := strings.TrimPrefix(c.Param("filepath"), "/")
		if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\\") {
			c.String(consts.StatusBadRequest, "Bad request")
			return
		}
		data, err := fs.ReadFile(assetsFS, name)
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		c.Data(consts.StatusOK, contentTypeOf(name), data)
	})

	// SPA 回落：未匹配到的路径（非 /api）统一交给前端
	h.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		path := string(c.Path())
		if strings.HasPrefix(path, "/api") {
			c.JSON(consts.StatusNotFound, utils.H{"error": "接口不存在"})
			return
		}
		serveIndex(ctx, c)
	})
}

func contentTypeOf(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(name, ".woff"):
		return "font/woff"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func badRequest(c *app.RequestContext, message string) {
	c.JSON(consts.StatusBadRequest, utils.H{"error": message})
}

func serverError(c *app.RequestContext, message string) {
	c.JSON(consts.StatusInternalServerError, utils.H{"error": message})
}
