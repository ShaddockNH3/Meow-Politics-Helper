package httpapi

import (
	"context"
	"log"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"quiz/internal/course"
	"quiz/internal/service"
)

// CoursesList 返回所有题库；带 user_id 时会先加载该用户的自定义题库。
func (h *Handlers) CoursesList(ctx context.Context, c *app.RequestContext) {
	if userID := string(c.Query("user_id")); userID != "" {
		_ = h.Banks.LoadUserCustomBanks(userID)
	}
	c.JSON(consts.StatusOK, utils.H{"courses": h.Registry.List()})
}

// ImportJSONQuestionBank 粘贴 JSON 导入为自定义题库（或已有题库的新章节）。
func (h *Handlers) ImportJSONQuestionBank(ctx context.Context, c *app.RequestContext) {
	var req ImportJSONRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	if err := h.Store.ValidateUserID(req.UserID); err != nil {
		badRequest(c, err.Error())
		return
	}
	questions, err := course.ParseQuestionsJSON(req.JSONText)
	if err != nil || len(questions) == 0 {
		badRequest(c, "JSON 题库不可用: "+err.Error())
		return
	}
	courseName := strings.TrimSpace(req.CourseName)
	if courseName == "" {
		courseName = "JSON 导入题库"
	}
	meta, chapterKey, err := h.Banks.ImportJSON(req.UserID, service.ImportTarget{
		TargetMode:   req.TargetMode,
		TargetCourse: req.TargetCourse,
		CourseKey:    req.CourseKey,
		CourseName:   courseName,
		ChapterKey:   req.ChapterKey,
		ChapterName:  req.ChapterName,
	}, questions)
	if err != nil {
		badRequest(c, "保存题库失败: "+err.Error())
		return
	}
	c.JSON(consts.StatusOK, utils.H{"message": "JSON 题库已导入", "course": meta, "chapter_key": chapterKey, "questions": len(questions)})
}

// CustomCourseDetail 返回自定义题库内容供编辑器使用。
func (h *Handlers) CustomCourseDetail(ctx context.Context, c *app.RequestContext) {
	userID := string(c.Query("user_id"))
	courseKey := string(c.Query("course"))
	if err := h.Store.ValidateUserID(userID); err != nil {
		badRequest(c, err.Error())
		return
	}
	if !strings.HasPrefix(courseKey, "custom_") {
		badRequest(c, "只能编辑自定义题库")
		return
	}
	meta, bank, err := h.Banks.LoadForEdit(userID, courseKey)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{"error": "题库不存在或无法读取"})
		return
	}
	c.JSON(consts.StatusOK, utils.H{"meta": meta, "chapters": bank.Chapters})
}

// SaveCustomCourse 保存自定义题库编辑，并同步清理错题本中已不存在的题目。
func (h *Handlers) SaveCustomCourse(ctx context.Context, c *app.RequestContext) {
	var req SaveCustomCourseRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	if err := h.Store.ValidateUserID(req.UserID); err != nil {
		badRequest(c, err.Error())
		return
	}
	if !strings.HasPrefix(req.Course, "custom_") {
		badRequest(c, "内置题库不能编辑")
		return
	}
	chapters := req.Chapters
	if len(chapters) == 0 && len(req.Questions) > 0 {
		chapters = map[string][]course.Question{"0": req.Questions}
	}
	if len(chapters) == 0 {
		badRequest(c, "题库至少需要保留一个章节")
		return
	}

	meta, err := h.Banks.SaveEditedBank(req.UserID, req.Course, req.CourseName, req.ChapterNames, chapters)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	// 编辑后题目可能被删，同步清理错题本
	if bank, ok := h.Registry.Get(req.Course); ok {
		if err := h.Quiz.PruneIncorrectForCourse(req.UserID, req.Course, bank.Chapters); err != nil {
			log.Printf("警告: 保存自定义题库后同步清理错题本失败，用户=%s 题库=%s: %v", req.UserID, req.Course, err)
		}
	}
	c.JSON(consts.StatusOK, utils.H{"message": "题库已保存", "questions": meta.QuestionCount})
}

// DeleteCustomCourse 删除自定义题库。
func (h *Handlers) DeleteCustomCourse(ctx context.Context, c *app.RequestContext) {
	var req DeleteCustomCourseRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	if err := h.Store.ValidateUserID(req.UserID); err != nil {
		badRequest(c, err.Error())
		return
	}
	if err := h.Banks.DeleteBank(req.UserID, req.Course); err != nil {
		if strings.Contains(err.Error(), "不能删除") {
			badRequest(c, err.Error())
			return
		}
		serverError(c, err.Error())
		return
	}
	c.JSON(consts.StatusOK, utils.H{"message": "题库已删除"})
}
