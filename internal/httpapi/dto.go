// Package httpapi 收拢所有 HTTP handler 与路由注册。
package httpapi

import "quiz/internal/course"

// InitSessionRequest 初始化用户会话。
type InitSessionRequest struct {
	UserID string `json:"user_id" vd:"required,min=1,max=50"`
}

// StartModeRequest 开始速刷 / 答题模式。
type StartModeRequest struct {
	UserID        string   `json:"user_id" vd:"required"`
	Course        string   `json:"course" vd:"required"`
	ChapterChoice []string `json:"chapter_choice" vd:"required"`
	OrderChoice   string   `json:"order_choice" vd:"required"`
}

// GetNextQuestionRequest 按用户 ID 定位请求（速刷下一题 / 清理数据）。
type GetNextQuestionRequest struct {
	UserID string `json:"user_id" vd:"required"`
}

// StartIncorrectReviewRequest 开始错题回顾。
type StartIncorrectReviewRequest struct {
	UserID string `json:"user_id" vd:"required"`
	Course string `json:"course" vd:"required"`
}

// SubmitAnswerRequest 提交答案。正确性由服务端判分，
// 旧版前端传的 was_correct 字段会被忽略。
type SubmitAnswerRequest struct {
	UserID         string `json:"user_id" vd:"required"`
	QuizQuestionID string `json:"quiz_question_id" vd:"required"`
	UserAnswer     string `json:"user_answer" vd:"required"`
}

// DeleteIncorrectQuestionRequest 从错题本删除一题。
type DeleteIncorrectQuestionRequest struct {
	UserID                 string `json:"user_id" vd:"required"`
	Course                 string `json:"course"`
	OriginalChapter        string `json:"original_chapter" vd:"required"`
	OriginalQuestionNumber string `json:"original_question_number" vd:"required"`
}

// ImportJSONRequest 粘贴 JSON 导入题库。
type ImportJSONRequest struct {
	UserID       string `json:"user_id" vd:"required"`
	TargetMode   string `json:"target_mode"`
	TargetCourse string `json:"target_course"`
	CourseKey    string `json:"course_key"`
	CourseName   string `json:"course_name"`
	ChapterKey   string `json:"chapter_key"`
	ChapterName  string `json:"chapter_name"`
	JSONText     string `json:"json_text" vd:"required"`
}

// DeleteCustomCourseRequest 删除自定义题库。
type DeleteCustomCourseRequest struct {
	UserID string `json:"user_id" vd:"required"`
	Course string `json:"course" vd:"required"`
}

// SaveCustomCourseRequest 保存自定义题库编辑。
type SaveCustomCourseRequest struct {
	UserID       string                       `json:"user_id" vd:"required"`
	Course       string                       `json:"course" vd:"required"`
	CourseName   string                       `json:"course_name" vd:"required"`
	ChapterNames map[string]string            `json:"chapter_names"`
	Chapters     map[string][]course.Question `json:"chapters"`
	Questions    []course.Question            `json:"questions"`
}

// NetworkAddress 是一个可访问的网络地址。
type NetworkAddress struct {
	Name           string `json:"name"`
	InterfaceIndex int    `json:"interface_index"`
	IP             string `json:"ip"`
	URL            string `json:"url"`
}
