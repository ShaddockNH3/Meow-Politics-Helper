// Meow-Politics-Helper：本地刷题工具（原"喵喵学习小助手"）。
//
// main 只负责装配：构建各层依赖、注册路由、打开浏览器、启动服务。
package main

import (
	"embed"
	"io/fs"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"

	"quiz/internal/course"
	"quiz/internal/httpapi"
	"quiz/internal/quiz"
	"quiz/internal/service"
	"quiz/internal/session"
	"quiz/internal/store"
)

//go:embed clean_outputs/* quiz.html assets/*
var embeddedFS embed.FS

// builtinCourseSpecs 是内置题库清单。新增内置题库只需在这里加一行。
var builtinCourseSpecs = []course.BuiltinSpec{
	{Key: "maogai", Name: "2025上毛概康老师", SourceDir: "clean_outputs/maogao_202506_kang_outputs", MaxChapter: 8},
	{Key: "xigai_li", Name: "2024下习概李老师", SourceDir: "clean_outputs/xigai_202412_li_outputs", MaxChapter: 0},
	{Key: "xigai_yang", Name: "2025下习概杨老师", SourceDir: "clean_outputs/xigai_202512_yang_outputs", MaxChapter: 17},
}

func main() {
	// 数据层
	st := store.New("user_data")
	if err := st.EnsureDir(); err != nil {
		log.Fatalf("无法创建用户数据目录: %v", err)
	}

	// 题库层：内置题库（embed）+ 用户自定义题库共用一个注册表
	registry := course.NewRegistry()
	if err := course.LoadBuiltin(registry, embeddedFS, builtinCourseSpecs); err != nil {
		log.Fatalf("加载内置题库失败: %v", err)
	}

	// 业务层
	quizSvc := &quiz.Service{Store: st}
	banks := &service.BankService{Store: st, Registry: registry}
	sessions := session.NewManager()
	handlers := httpapi.NewHandlers(st, registry, quizSvc, banks, sessions)

	// 装配服务
	h := server.Default(server.WithHostPorts(httpapi.DefaultBindAddress))
	httpapi.RegisterRoutes(h, embeddedFS, handlers)

	log.Printf("喵喵学习小助手已启动，监听于 http://%s", httpapi.DefaultBindAddress)
	for _, addr := range httpapi.DetectLANAddresses(httpapi.ServerPort(httpapi.DefaultBindAddress)) {
		log.Printf("同一局域网可访问: %s (%s)", addr.URL, addr.Name)
	}

	go func() {
		time.Sleep(1 * time.Second) // 等待服务启动
		openBrowser("http://localhost:" + httpapi.ServerPort(httpapi.DefaultBindAddress))
	}()

	h.Spin()
}

// openBrowser 在服务启动后打开浏览器。
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		if isWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}

	log.Printf("正在尝试打开浏览器: %s %v", cmd, args)
	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("打开浏览器失败: %v", err)
	}
}

// isWSL 检查是否在 WSL 环境中运行。
func isWSL() bool {
	if data, err := exec.Command("cat", "/proc/version").Output(); err == nil {
		return strings.Contains(string(data), "WSL")
	}
	return false
}

// 静态资源目录校验：确保 embed 的 assets 可被子文件系统访问。
var _ fs.FS = embeddedFS
