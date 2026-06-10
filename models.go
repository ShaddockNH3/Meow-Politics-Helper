package main

import (
	"sync"
	"time"
)

// --- 配置常量 ---
const (
	defaultBindAddress              = "0.0.0.0:8899"
	projectURL                      = "https://github.com/ShaddockNH3/Meow-Politics-Helper"
	maogaiQuestionSourceDir         = "clean_outputs/maogao_202506_kang_outputs"
	xigaiLiQuestionSourceDir        = "clean_outputs/xigai_202412_li_outputs"
	xigaiYangQuestionSourceDir      = "clean_outputs/xigai_202512_yang_outputs"
	maogaiMaxChapterIndex           = 8
	xigaiLiMaxChapterIndex          = 0  // 李老师的习概只有一个章节
	xigaiYangMaxChapterIndex        = 17 // 杨老师的习概包含导论(0) + 17章
	userDataBaseDir                 = "user_data"
	incorrectQuestionsFile          = "incorrect_questions.json"            // 默认(毛概)错题文件
	maogaiIncorrectQuestionsFile    = "maogai_incorrect_questions.json"     // 毛概错题文件
	xigaiIncorrectQuestionsFile     = "xigai_incorrect_questions.json"      // 兼容老版本的习概错题文件（保留以向后兼容）
	xigaiLiIncorrectQuestionsFile   = "xigai_li_incorrect_questions.json"   // 习概 李老师 错题文件
	xigaiYangIncorrectQuestionsFile = "xigai_yang_incorrect_questions.json" // 习概 杨老师 错题文件
	deleteIncorrectQuestionsFile    = "deleted_incorrect_questions.json"
	questionStatsFile               = "question_stats.json"
	aiConfigFile                    = "ai_config.json"
	aiQuestionBanksDir              = "ai_question_banks"
)

type CourseConfig struct {
	Key           string
	Name          string
	SourceDir     string
	MaxChapter    int
	ChapterNames  map[string]string
	Questions     map[string][]Question
	IncorrectFile string
}

type CourseInfo struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	MaxChapter    int      `json:"max_chapter"`
	ChapterNames  []string `json:"chapter_names,omitempty"`
	Custom        bool     `json:"custom"`
	QuestionCount int      `json:"question_count"`
}

type AIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	APIPath string `json:"api_path"`
	Model   string `json:"model"`
}

type AIConfigStatus struct {
	Configured bool   `json:"configured"`
	BaseURL    string `json:"base_url"`
	APIPath    string `json:"api_path"`
	Model      string `json:"model"`
	KeyPreview string `json:"key_preview,omitempty"`
}

type AIModelInfo struct {
	ID string `json:"id"`
}

type AIImportTask struct {
	ID          string              `json:"id"`
	UserID      string              `json:"-"`
	Status      string              `json:"status"`
	Message     string              `json:"message"`
	Error       string              `json:"error,omitempty"`
	Course      *AIQuestionBankMeta `json:"course,omitempty"`
	Questions   int                 `json:"questions,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
}

// --- 数据结构定义 ---
type Question struct {
	QuestionNumber     string            `json:"question_number"`
	QuestionType       string            `json:"question_type"`
	QuestionText       string            `json:"question_text"`
	Options            map[string]string `json:"options"`
	CorrectAnswer      string            `json:"correct_answer"`
	GlobalCorrectCount int               `json:"correct_count"` // 未来可能用于全局统计
	GlobalErrorCount   int               `json:"error_count"`   // 未来可能用于全局统计
	OriginalChapterKey string            `json:"-"`             // 内部使用，标记原始章节
	OriginalIndex      int               `json:"-"`             // 内部使用，标记在原始章节中的索引
}

type QuestionOutput struct {
	QuizQuestionID         string            `json:"quiz_question_id"`         // 在当前测验/回顾中的唯一ID
	DisplayNumber          int               `json:"display_number"`           // 在当前列表中的显示序号 (1-based)
	Course                 string            `json:"course"`                   // 题目所属课程/题库
	OriginalChapter        string            `json:"original_chapter"`         // 原始章节键
	OriginalQuestionNumber string            `json:"original_question_number"` // 原始题号
	QuestionType           string            `json:"question_type"`
	QuestionText           string            `json:"question_text"`
	Options                map[string]string `json:"options"`
	CorrectAnswer          string            `json:"correct_answer"` // 答案将始终包含
}

type UserIncorrectQuestion struct {
	Course          string            `json:"course,omitempty"`
	QuestionNumber  string            `json:"question_number"`
	QuestionType    string            `json:"question_type"`
	QuestionText    string            `json:"question_text"`
	Options         map[string]string `json:"options"`
	CorrectAnswer   string            `json:"correct_answer"`
	OriginalChapter string            `json:"original_chapter"`
	UserAnswer      string            `json:"user_answer,omitempty"` // 用户在答错时的答案
	Timestamp       time.Time         `json:"timestamp"`             // 答错的时间
	DeletedAt       time.Time         `json:"deleted_at,omitempty"`  // 题目被删除的时间
}

type UserQuestionStat struct {
	OriginalChapterKey     string    `json:"original_chapter_key"`
	OriginalQuestionNumber string    `json:"original_question_number"`
	CorrectCount           int       `json:"correct_count"`
	ErrorCount             int       `json:"error_count"`
	LastAnswered           time.Time `json:"last_answered"`
}

// UserSession 存储用户当前会话状态
type UserSession struct {
	UserID string
	// CurrentQuestions is used if frontend logic relies on server to step through questions,
	// e.g. for a simplified /api/review/next. If frontend receives all questions
	// from /start endpoints and manages navigation itself, this might be less critical
	// for those modes. For this iteration, we keep it for potential use with /api/review/next.
	CurrentQuestions     []QuestionOutput
	OriginalIncorrect    []UserIncorrectQuestion // Store the full incorrect questions for retrieval
	CurrentQuestionIndex int                     // Index for session.CurrentQuestions (e.g., /api/review/next)
	CurrentMode          string                  // "review", "quiz", "incorrect_review"
	CurrentCourse        string                  // "maogai", "xigai_li" 或 "xigai_yang" - 当前选择的课程
	mu                   sync.Mutex              // 保护会话内部数据
}

// --- 请求结构体 ---
type InitSessionRequest struct {
	UserID string `json:"user_id" vd:"required,min=1,max=50"` // UserID是必需的,长度1-50
}

type StartModeRequest struct {
	UserID        string   `json:"user_id" vd:"required"`
	Course        string   `json:"course" vd:"required"`         // "maogai", "xigai_li" 或 "xigai_yang"
	ChapterChoice []string `json:"chapter_choice" vd:"required"` // 例如 ["0", "1", "all"]
	OrderChoice   string   `json:"order_choice" vd:"required"`   // "sequential" 或 "random"
}

type GetNextQuestionRequest struct {
	UserID string `json:"user_id" vd:"required"`
}

type StartIncorrectReviewRequest struct {
	UserID string `json:"user_id" vd:"required"`
	Course string `json:"course" vd:"required"`
}

type SubmitAnswerRequest struct {
	UserID         string `json:"user_id" vd:"required"`
	QuizQuestionID string `json:"quiz_question_id" vd:"required"` // 题目在当前测验中的ID
	UserAnswer     string `json:"user_answer" vd:"required"`      // 用户选择的答案
	WasCorrect     bool   `json:"was_correct"`                    // 由前端判断并发送该答案是否正确
}

type DeleteIncorrectQuestionRequest struct {
	UserID                 string `json:"user_id" vd:"required"`
	Course                 string `json:"course"`
	OriginalChapter        string `json:"original_chapter" vd:"required"`
	OriginalQuestionNumber string `json:"original_question_number" vd:"required"`
}

type SaveAIConfigRequest struct {
	UserID  string `json:"user_id" vd:"required"`
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	APIPath string `json:"api_path"`
	Model   string `json:"model"`
}

type AIImportQuestionBankRequest struct {
	UserID       string `json:"user_id" vd:"required"`
	CourseKey    string `json:"course_key"`
	CourseName   string `json:"course_name"`
	TargetMode   string `json:"target_mode"`
	TargetCourse string `json:"target_course"`
	ChapterKey   string `json:"chapter_key"`
	ChapterName  string `json:"chapter_name"`
	SystemPrompt string `json:"system_prompt"`
	SourceText   string `json:"source_text"`
}

type JSONImportQuestionBankRequest struct {
	UserID       string `json:"user_id" vd:"required"`
	CourseKey    string `json:"course_key"`
	CourseName   string `json:"course_name"`
	TargetMode   string `json:"target_mode"`
	TargetCourse string `json:"target_course"`
	ChapterKey   string `json:"chapter_key"`
	ChapterName  string `json:"chapter_name"`
	JSONText     string `json:"json_text" vd:"required"`
}

type AIQuestionBankMeta struct {
	CourseKey     string            `json:"course_key"`
	CourseName    string            `json:"course_name"`
	ChapterCount  int               `json:"chapter_count"`
	QuestionCount int               `json:"question_count"`
	ChapterNames  map[string]string `json:"chapter_names,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at,omitempty"`
}

type DeleteCustomCourseRequest struct {
	UserID string `json:"user_id" vd:"required"`
	Course string `json:"course" vd:"required"`
}

type SaveCustomCourseRequest struct {
	UserID       string                `json:"user_id" vd:"required"`
	Course       string                `json:"course" vd:"required"`
	CourseName   string                `json:"course_name" vd:"required"`
	ChapterNames map[string]string     `json:"chapter_names"`
	Chapters     map[string][]Question `json:"chapters"`
	Questions    []Question            `json:"questions"`
}

type NetworkAddress struct {
	Name           string `json:"name"`
	InterfaceIndex int    `json:"interface_index"`
	IP             string `json:"ip"`
	URL            string `json:"url"`
}

type NetworkInfoResponse struct {
	Port        string           `json:"port"`
	LocalURL    string           `json:"local_url"`
	LANURLs     []NetworkAddress `json:"lan_urls"`
	LANURLCount int              `json:"lan_url_count"`
	Hint        string           `json:"hint"`
}
