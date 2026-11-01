package httpapi

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"quiz/internal/quiz"
)

// InitSession 初始化用户会话：校验 ID、确保目录、建立内存会话。
func (h *Handlers) InitSession(ctx context.Context, c *app.RequestContext) {
	var req InitSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效的请求: user_id 是必需的。 "+err.Error())
		return
	}
	if err := h.Store.ValidateUserID(req.UserID); err != nil {
		badRequest(c, err.Error())
		return
	}

	userDir, _ := h.Store.UserDir(req.UserID)
	isNewUser := false
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		isNewUser = true
		if err := h.Store.EnsureUserDir(req.UserID); err != nil {
			log.Printf("错误: 为用户 %s 创建目录失败: %v", req.UserID, err)
			serverError(c, "无法初始化用户数据存储区")
			return
		}
	} else if err != nil {
		log.Printf("错误: 检查用户目录 %s 时发生错误: %v", userDir, err)
		serverError(c, "检查用户数据时出错")
		return
	}

	h.Sessions.Get(req.UserID)
	message := "用户会话已建立"
	if isNewUser {
		message = "新用户会话已创建并初始化成功"
		log.Printf("会话: 新用户 %s 首次使用，已创建用户目录: %s", req.UserID, userDir)
	}
	c.JSON(consts.StatusOK, utils.H{"user_id": req.UserID, "message": message, "is_new_user": isNewUser})
}

// QuickReviewStart 开始速刷模式：返回全部题目与答案。
func (h *Handlers) QuickReviewStart(ctx context.Context, c *app.RequestContext) {
	var req StartModeRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	bank, ok := h.Registry.Get(req.Course)
	if !ok {
		badRequest(c, "未知的题库: "+req.Course)
		return
	}

	selected := quiz.SelectQuestions(bank, req.ChapterChoice, req.OrderChoice)
	output := quiz.ToOutput(selected, req.Course)

	session := h.Sessions.Get(req.UserID)
	session.Lock()
	session.Mode = "review"
	session.Course = req.Course
	session.CurrentQuestions = output
	session.CurrentQuestionIndex = 0
	session.Unlock()

	log.Printf("用户 %s 开始速刷模式，课程: %s, 章节: %v, 顺序: %s, 返回 %d 题", req.UserID, req.Course, req.ChapterChoice, req.OrderChoice, len(output))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "速刷模式开始",
		"total_questions": len(output),
		"questions":       output,
	})
}

// GetNextQuestion 速刷模式下逐步返回下一题。
// 前端通常一次性拿到全部题目自行导航，此接口仅为兼容保留。
func (h *Handlers) GetNextQuestion(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	session := h.Sessions.Get(req.UserID)
	session.Lock()
	defer session.Unlock()

	if session.Mode != "review" {
		badRequest(c, "当前不处于速刷模式 (或会话模式不匹配)")
		return
	}
	session.CurrentQuestionIndex++
	if session.CurrentQuestionIndex >= len(session.CurrentQuestions) {
		c.JSON(consts.StatusOK, utils.H{"message": "速刷完成!", "quiz_completed": true, "question": nil})
		return
	}
	c.JSON(consts.StatusOK, utils.H{
		"question":       session.CurrentQuestions[session.CurrentQuestionIndex],
		"quiz_completed": false,
	})
}

// QuizStart 开始答题模式：返回全部题目与答案，判分在提交时由服务端完成。
func (h *Handlers) QuizStart(ctx context.Context, c *app.RequestContext) {
	var req StartModeRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	bank, ok := h.Registry.Get(req.Course)
	if !ok {
		badRequest(c, "未知的题库: "+req.Course)
		return
	}

	selected := quiz.SelectQuestions(bank, req.ChapterChoice, req.OrderChoice)
	output := quiz.ToOutput(selected, req.Course)

	session := h.Sessions.Get(req.UserID)
	session.Lock()
	session.Mode = "quiz"
	session.Course = req.Course
	session.Unlock()

	log.Printf("用户 %s 开始答题模式，课程: %s, 章节: %v, 顺序: %s, 返回 %d 题", req.UserID, req.Course, req.ChapterChoice, req.OrderChoice, len(output))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "答题模式开始",
		"total_questions": len(output),
		"questions":       output,
	})
}

// SubmitAnswer 答题模式提交答案：服务端判分并记录统计与错题。
func (h *Handlers) SubmitAnswer(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}

	courseKey, chapterKey, index, err := quiz.ParseQuizQuestionID(req.QuizQuestionID)
	if err != nil {
		serverError(c, err.Error())
		return
	}
	question, ok := h.Registry.LookupQuestion(courseKey, chapterKey, index)
	if !ok {
		log.Printf("错误: 找不到题目ID %s 对应的原始题目 (答题模式)", req.QuizQuestionID)
		serverError(c, "找不到原始题目")
		return
	}

	correct, err := h.Quiz.RecordAnswer(req.UserID, courseKey, question, req.UserAnswer)
	if err != nil {
		log.Printf("错误: 用户 %s 记录答案失败: %v", req.UserID, err)
		serverError(c, "记录答案失败: "+err.Error())
		return
	}
	c.JSON(consts.StatusOK, utils.H{
		"message":        "答案已记录",
		"correct":        correct,
		"correct_answer": question.CorrectAnswer,
	})
}

// IncorrectReviewStart 开始错题回顾：返回错题本中的全部错题（随机顺序）。
func (h *Handlers) IncorrectReviewStart(ctx context.Context, c *app.RequestContext) {
	var req StartIncorrectReviewRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	if _, ok := h.Registry.Get(req.Course); !ok {
		badRequest(c, "未知的题库: "+req.Course)
		return
	}

	book, err := h.Quiz.IncorrectBook(req.UserID, req.Course)
	if err != nil {
		log.Printf("用户 %s 加载 %s 课程错题本失败: %v", req.UserID, req.Course, err)
		serverError(c, "加载用户错题本失败")
		return
	}
	if len(book) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "错题簿是空的哦！太棒了！", "total_questions": 0, "questions": []quiz.OutputQuestion{}})
		return
	}

	quiz.Shuffle(book)
	output := quiz.IncorrectToOutput(book, req.Course)

	session := h.Sessions.Get(req.UserID)
	session.Lock()
	session.Mode = "incorrect_review"
	session.Course = req.Course
	session.Unlock()

	log.Printf("用户 %s 开始错题回顾模式, 返回 %d 题", req.UserID, len(output))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "错题回顾模式开始",
		"total_questions": len(output),
		"questions":       output,
	})
}

// SubmitIncorrectReviewAnswer 记录错题回顾中的一次作答（当前仅日志，不影响错题本）。
func (h *Handlers) SubmitIncorrectReviewAnswer(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	log.Printf("用户 %s 错题回顾提交: QID %s, 用户答案 %s (仅记录日志)", req.UserID, req.QuizQuestionID, req.UserAnswer)
	c.JSON(consts.StatusOK, utils.H{"message": "错题回顾答案已由服务器记录(日志)"})
}

// DeleteIncorrectQuestion 从错题本删除一题。
func (h *Handlers) DeleteIncorrectQuestion(ctx context.Context, c *app.RequestContext) {
	var req DeleteIncorrectQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	courseKey := req.Course
	if courseKey == "" {
		courseKey = "maogai"
	}
	if _, ok := h.Registry.Get(courseKey); !ok {
		badRequest(c, "未知的题库: "+courseKey)
		return
	}

	deleted, err := h.Quiz.DeleteIncorrect(req.UserID, courseKey, req.OriginalChapter, req.OriginalQuestionNumber)
	if err != nil {
		log.Printf("错误: 用户 %s 删除错题失败: %v", req.UserID, err)
		serverError(c, "删除错题失败")
		return
	}
	if deleted {
		log.Printf("信息: 用户 %s 从错题本中删除题目: 章节 %s, 题号 %s", req.UserID, req.OriginalChapter, req.OriginalQuestionNumber)
	} else {
		log.Printf("警告: 用户 %s 请求删除错题 (章节 %s, 题号 %s)，但错题本中未找到该题", req.UserID, req.OriginalChapter, req.OriginalQuestionNumber)
	}
	c.Status(consts.StatusNoContent)
}

// UserDataClear 清理用户数据（错题本与统计，均备份后清理）。
func (h *Handlers) UserDataClear(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		badRequest(c, "无效请求: "+err.Error())
		return
	}
	log.Printf("用户 %s 请求清理其数据...", req.UserID)
	if err := h.Quiz.ClearUserData(req.UserID); err != nil {
		serverError(c, "清理用户数据时发生错误")
		return
	}
	h.Sessions.Clear(req.UserID)
	c.JSON(consts.StatusOK, utils.H{"message": "用户数据（错题本和统计）已成功清理。"})
}
