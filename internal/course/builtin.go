package course

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"strconv"
	"strings"
	"time"
)

// BuiltinSpec 描述一个内置题库的来源。
type BuiltinSpec struct {
	Key        string // 课程键，如 "maogai"
	Name       string // 展示名，如 "2025上毛概康老师"
	SourceDir  string // embed FS 内的目录，如 "clean_outputs/maogao_202506_kang_outputs"
	MaxChapter int    // 最大章节号（章节文件为 <n>.json，从 0 开始）
}

// LoadBuiltin 从文件系统加载内置题库并注册到 registry。
// 缺失或损坏的章节文件会被跳过（初始化为空章节），不阻塞启动。
func LoadBuiltin(registry *Registry, fsys fs.FS, specs []BuiltinSpec) error {
	for _, spec := range specs {
		bank := &Bank{
			Key:        spec.Key,
			Name:       spec.Name,
			MaxChapter: spec.MaxChapter,
			Chapters:   make(map[string][]Question),
		}
		for i := 0; i <= spec.MaxChapter; i++ {
			chapterKey := strconv.Itoa(i)
			// embed FS 统一使用正斜杠路径，不能用 filepath.Join
			filePath := spec.SourceDir + "/" + chapterKey + ".json"
			questions, err := loadChapterFile(fsys, filePath)
			if err != nil {
				log.Printf("提示：章节 %s (%s) 题库文件 %s 不可用，已跳过: %v", chapterKey, spec.Key, filePath, err)
			}
			assignChapterIndices(chapterKey, questions)
			bank.Chapters[chapterKey] = questions
		}
		registry.Register(bank)
		log.Printf("内置题库「%s」加载完成，共 %d 题", bank.Name, bank.QuestionCount())
	}
	return nil
}

func loadChapterFile(fsys fs.FS, filePath string) ([]Question, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return []Question{}, err
	}
	var questions []Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return []Question{}, fmt.Errorf("解析 %s 失败: %w", filePath, err)
	}
	return questions, nil
}

func assignChapterIndices(chapterKey string, questions []Question) {
	for idx := range questions {
		questions[idx].OriginalChapterKey = chapterKey
		questions[idx].OriginalIndex = idx
	}
}

// LoadCustomFromFile 从用户数据目录读取一个自定义题库文件（{"meta":..., "chapters":...} 格式）。
// 文件名（不含扩展名）即题库键，比文件内 meta 更可信，以此为准。
func LoadCustomFromFile(path, fileName string) (*Bank, *BankMeta, error) {
	data, err := readFileLimited(path)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		Meta     BankMeta              `json:"meta"`
		Chapters map[string][]Question `json:"chapters"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, fmt.Errorf("解析自定义题库 %s 失败: %w", fileName, err)
	}
	key := strings.TrimSuffix(fileName, ".json")
	if payload.Chapters == nil {
		payload.Chapters = make(map[string][]Question)
	}
	bank := &Bank{
		Key:          key,
		Name:         payload.Meta.CourseName,
		Custom:       true,
		MaxChapter:   MaxChapterKey(payload.Chapters),
		ChapterNames: payload.Meta.ChapterNames,
		Chapters:     payload.Chapters,
	}
	return bank, &payload.Meta, nil
}

// SanitizeKey 把任意字符串清洗成合法的自定义题库键（custom_ 前缀）。
func SanitizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	key := strings.Trim(b.String(), "_")
	if key == "" {
		key = fmt.Sprintf("import_%d", time.Now().Unix())
	}
	if !strings.HasPrefix(key, "custom_") {
		key = "custom_" + key
	}
	return key
}
