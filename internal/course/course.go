// Package course 定义题目、题库与课程注册表。
//
// 内置题库（embed 进二进制）和用户自定义题库统一为同一套 Bank 抽象，
// 由 Registry 集中管理，业务层不再感知两者的差异。
package course

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Question 是一道选择题的完整数据。
type Question struct {
	QuestionNumber     string            `json:"question_number"`
	QuestionType       string            `json:"question_type"` // "单选题" 或 "多选题"
	QuestionText       string            `json:"question_text"`
	Options            map[string]string `json:"options"`
	CorrectAnswer      string            `json:"correct_answer"` // 如 "A" 或 "AB"
	OriginalChapterKey string            `json:"-"`              // 内部使用：所属章节键
	OriginalIndex      int               `json:"-"`              // 内部使用：章节内索引
}

// BankMeta 是自定义题库文件的元信息，字段名与旧版保持兼容。
type BankMeta struct {
	CourseKey     string            `json:"course_key"`
	CourseName    string            `json:"course_name"`
	ChapterCount  int               `json:"chapter_count"`
	QuestionCount int               `json:"question_count"`
	ChapterNames  map[string]string `json:"chapter_names,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	UpdatedAt     string            `json:"updated_at,omitempty"`
}

// Bank 是一个题库（课程）：包含按章节组织的题目。
type Bank struct {
	Key          string
	Name         string
	Custom       bool
	MaxChapter   int
	ChapterNames map[string]string
	Chapters     map[string][]Question
}

// QuestionCount 返回题库总题数。
func (b *Bank) QuestionCount() int {
	total := 0
	for _, questions := range b.Chapters {
		total += len(questions)
	}
	return total
}

// Lookup 按章节键和索引查找题目。
func (b *Bank) Lookup(chapterKey string, index int) (Question, bool) {
	questions, ok := b.Chapters[chapterKey]
	if !ok || index < 0 || index >= len(questions) {
		return Question{}, false
	}
	return questions[index], true
}

// Registry 是线程安全的题库注册表。
type Registry struct {
	mu    sync.RWMutex
	banks map[string]*Bank
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{banks: make(map[string]*Bank)}
}

// Register 注册或替换一个题库，并统一为每道题补齐章节键与章节内索引
// （这两个字段是判分反查原题的依据，必须在注册时保证正确）。
func (r *Registry) Register(bank *Bank) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for chapterKey, questions := range bank.Chapters {
		for idx := range questions {
			questions[idx].OriginalChapterKey = chapterKey
			questions[idx].OriginalIndex = idx
		}
	}
	r.banks[bank.Key] = bank
}

// Get 按键查找题库。
func (r *Registry) Get(key string) (*Bank, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bank, ok := r.banks[key]
	return bank, ok
}

// Delete 删除一个题库。
func (r *Registry) Delete(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.banks, key)
}

// Info 是对外展示的题库摘要（API 输出格式）。
type Info struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	MaxChapter    int      `json:"max_chapter"`
	ChapterNames  []string `json:"chapter_names,omitempty"`
	Custom        bool     `json:"custom"`
	QuestionCount int      `json:"question_count"`
}

// List 返回全部题库摘要：内置在前，自定义按名称排序。
func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infos := make([]Info, 0, len(r.banks))
	for _, bank := range r.banks {
		infos = append(infos, Info{
			Key:           bank.Key,
			Name:          bank.Name,
			MaxChapter:    bank.MaxChapter,
			ChapterNames:  chapterNamesForInfo(bank),
			Custom:        bank.Custom,
			QuestionCount: bank.QuestionCount(),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Custom != infos[j].Custom {
			return !infos[i].Custom
		}
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// LookupQuestion 按 课程键/章节键/索引 全局查找题目，供判分使用。
func (r *Registry) LookupQuestion(courseKey, chapterKey string, index int) (Question, bool) {
	bank, ok := r.Get(courseKey)
	if !ok {
		return Question{}, false
	}
	return bank.Lookup(chapterKey, index)
}

// chapterNamesForInfo 把章节名映射展开成按章节号排序的数组。
func chapterNamesForInfo(bank *Bank) []string {
	if len(bank.ChapterNames) == 0 {
		return nil
	}
	names := make([]string, bank.MaxChapter+1)
	for i := 0; i <= bank.MaxChapter; i++ {
		key := strconv.Itoa(i)
		if name := strings.TrimSpace(bank.ChapterNames[key]); name != "" {
			names[i] = name
		}
	}
	return names
}

// ParseIndex 尽力把字符串解析为整数，失败返回 0。
func ParseIndex(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

// NextChapterKey 返回下一个可用章节键（现有最大章节号 + 1）。
func NextChapterKey(chapters map[string][]Question) string {
	max := -1
	for key := range chapters {
		if n := ParseIndex(key); n > max {
			max = n
		}
	}
	return fmt.Sprintf("%d", max+1)
}

// MaxChapterKey 返回现有最大章节号。
func MaxChapterKey(chapters map[string][]Question) int {
	max := 0
	for key := range chapters {
		if n := ParseIndex(key); n > max {
			max = n
		}
	}
	return max
}
