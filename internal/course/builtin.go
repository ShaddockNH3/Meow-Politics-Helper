package course

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// bankKeyPattern 限制内置题库 key 只能是小写字母、数字、下划线和短横线。
var bankKeyPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// BankMetaFile 是题库目录下 meta.json 的格式，也是给贡献者的唯一配置点。
type BankMetaFile struct {
	Key          string            `json:"key"`                     // 题库标识，如 "maogai"（小写字母/数字/下划线/短横线）
	Name         string            `json:"name"`                    // 展示名，如 "2025上毛概康老师"
	MaxChapter   *int              `json:"max_chapter,omitempty"`   // 最大章节号；留空则自动扫描 0.json、1.json… 直到缺文件为止
	ChapterNames map[string]string `json:"chapter_names,omitempty"` // 可选：章节名映射 {"0": "导论", ...}
}

// LoadBanksFromFS 扫描 fsys 中 root 目录下的每个子目录，
// 读取其中的 meta.json 并加载题库（章节文件为 <章节号>.json，从 0 开始）。
//
// 新增内置题库无需改任何 Go 代码：在 clean_outputs/ 下新建目录，
// 放入 meta.json 和编号的章节 JSON 文件即可（欢迎直接提 PR）。
// 缺失或损坏的章节文件会被跳过（初始化为空章节），不阻塞启动。
func LoadBanksFromFS(registry *Registry, fsys fs.FS, root string) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return fmt.Errorf("无法读取题库目录 %s: %w", root, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := root + "/" + entry.Name()
		bank, err := loadBankDir(fsys, dirPath, entry.Name())
		if err != nil {
			log.Printf("跳过题库目录 %s: %v", dirPath, err)
			continue
		}
		registry.Register(bank)
		log.Printf("内置题库「%s」加载完成，共 %d 题（%d 章）", bank.Name, bank.QuestionCount(), len(bank.Chapters))
	}
	return nil
}

func loadBankDir(fsys fs.FS, dirPath, dirName string) (*Bank, error) {
	metaData, err := fs.ReadFile(fsys, dirPath+"/meta.json")
	if err != nil {
		return nil, fmt.Errorf("缺少 meta.json（每个题库目录必须有 meta.json 声明 key 和 name）: %w", err)
	}
	var meta BankMetaFile
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("解析 meta.json 失败: %w", err)
	}
	meta.Key = strings.TrimSpace(meta.Key)
	if !bankKeyPattern.MatchString(meta.Key) {
		return nil, fmt.Errorf("meta.json 的 key %q 不合法（需为小写字母/数字/下划线/短横线）", meta.Key)
	}
	if strings.TrimSpace(meta.Name) == "" {
		return nil, fmt.Errorf("meta.json 缺少 name")
	}

	chapters := make(map[string][]Question)
	maxChapter := -1
	if meta.MaxChapter != nil {
		maxChapter = *meta.MaxChapter
		if maxChapter < 0 {
			return nil, fmt.Errorf("max_chapter 不能为负数")
		}
	} else {
		// 自动扫描：从 0.json 开始，遇到第一个缺失的文件就停
		maxChapter = detectMaxChapter(fsys, dirPath)
		if maxChapter < 0 {
			return nil, fmt.Errorf("目录中没有找到任何章节文件（0.json、1.json…）")
		}
	}

	for i := 0; i <= maxChapter; i++ {
		chapterKey := strconv.Itoa(i)
		// embed FS 统一使用正斜杠路径，不能用 filepath.Join
		questions, err := loadChapterFile(fsys, dirPath+"/"+chapterKey+".json")
		if err != nil {
			log.Printf("提示：题库 %s 章节 %s 文件不可用，已跳过: %v", meta.Key, chapterKey, err)
		}
		assignChapterIndices(chapterKey, questions)
		chapters[chapterKey] = questions
	}

	return &Bank{
		Key:          meta.Key,
		Name:         meta.Name,
		MaxChapter:   maxChapter,
		ChapterNames: meta.ChapterNames,
		Chapters:     chapters,
	}, nil
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

// detectMaxChapter 逐个尝试章节文件，返回存在的最大章节号；一个都没有时返回 -1。
func detectMaxChapter(fsys fs.FS, dirPath string) int {
	for i := 0; ; i++ {
		if _, err := fs.ReadFile(fsys, dirPath+"/"+strconv.Itoa(i)+".json"); err != nil {
			return i - 1
		}
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
