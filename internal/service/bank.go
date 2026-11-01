// Package service 提供自定义题库的业务操作：加载、导入 JSON、保存编辑、删除。
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"quiz/internal/course"
	"quiz/internal/store"
)

// customBankDirName 是用户数据目录下存放自定义题库的子目录。
// 历史原因沿用了旧版的目录名，避免老用户已有题库丢失。
const customBankDirName = "ai_question_banks"

// BankService 管理自定义题库。
type BankService struct {
	Store    *store.Store
	Registry *course.Registry
}

// CustomBankPath 返回自定义题库文件路径（位于用户目录的题库子目录下）。
func (s *BankService) CustomBankPath(userID, courseKey string) (string, error) {
	if err := s.Store.ValidateUserID(userID); err != nil {
		return "", err
	}
	return filepath.Join(s.Store.BaseDir(), userID, customBankDirName, filepath.Base(courseKey)+".json"), nil
}

// LoadUserCustomBanks 把用户的所有自定义题库注册进 registry。
func (s *BankService) LoadUserCustomBanks(userID string) error {
	dir := filepath.Join(s.Store.BaseDir(), userID, customBankDirName)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		bank, _, err := course.LoadCustomFromFile(path, entry.Name())
		if err != nil {
			log.Printf("读取自定义题库失败 %s: %v", path, err)
			continue
		}
		s.Registry.Register(bank)
	}
	return nil
}

// ImportTarget 描述一次导入的目标位置。
type ImportTarget struct {
	TargetMode   string // "new" 或 "existing"
	TargetCourse string
	CourseKey    string
	CourseName   string
	ChapterKey   string
	ChapterName  string
}

// ImportJSON 把解析好的题目写入自定义题库（新建题库或已有题库的新章节）。
// 返回题库元信息与最终写入的章节键。
func (s *BankService) ImportJSON(userID string, target ImportTarget, questions []course.Question) (*course.BankMeta, string, error) {
	cleaned, err := course.NormalizeQuestionList(questions)
	if err != nil {
		return nil, "", err
	}

	var meta course.BankMeta
	chapters := map[string][]course.Question{}
	now := time.Now().UTC().Format(time.RFC3339)

	if target.TargetMode == "existing" {
		if !strings.HasPrefix(target.TargetCourse, "custom_") {
			return nil, "", fmt.Errorf("请选择一个已有自定义题库")
		}
		loadedMeta, loadedBank, err := s.loadCustomPayload(userID, target.TargetCourse)
		if err != nil {
			return nil, "", fmt.Errorf("读取已有题库失败: %w", err)
		}
		meta = *loadedMeta
		if meta.ChapterNames == nil {
			meta.ChapterNames = map[string]string{}
		}
		for key, value := range loadedBank.Chapters {
			chapters[key] = value
		}
		chapterKey := normalizeChapterKey(target.ChapterKey, chapters)
		if _, exists := chapters[chapterKey]; exists {
			return nil, "", fmt.Errorf("章节 %s 已存在，请换一个章节号或选择自动新章节", chapterKey)
		}
		target.ChapterKey = chapterKey
	} else {
		courseName := strings.TrimSpace(target.CourseName)
		if courseName == "" {
			courseName = "自定义题库"
		}
		var courseKey string
		if strings.TrimSpace(target.CourseKey) == "" {
			courseKey = course.SanitizeKey(courseName)
		} else {
			courseKey = course.SanitizeKey(target.CourseKey)
		}
		meta = course.BankMeta{CourseKey: courseKey, CourseName: courseName, CreatedAt: now, ChapterNames: map[string]string{}}
		target.CourseKey = courseKey
		target.ChapterKey = normalizeChapterKey(target.ChapterKey, chapters)
	}

	chapterName := strings.TrimSpace(target.ChapterName)
	if chapterName == "" {
		chapterName = "章节 " + target.ChapterKey
	}
	chapters[target.ChapterKey] = cleaned
	meta.ChapterNames[target.ChapterKey] = chapterName
	meta.ChapterCount = len(chapters)
	meta.QuestionCount = countQuestions(chapters)
	meta.UpdatedAt = now
	if meta.CreatedAt == "" {
		meta.CreatedAt = now
	}
	if meta.CourseKey == "" {
		meta.CourseKey = target.CourseKey
	}

	if err := s.saveCustomBank(userID, target.CourseKey, meta, chapters); err != nil {
		return nil, "", err
	}
	s.Registry.Register(&course.Bank{
		Key:          target.CourseKey,
		Name:         meta.CourseName,
		Custom:       true,
		MaxChapter:   course.MaxChapterKey(chapters),
		ChapterNames: meta.ChapterNames,
		Chapters:     chapters,
	})
	return &meta, target.ChapterKey, nil
}

// SaveEditedBank 用前端编辑器提交的完整数据覆盖保存一个自定义题库。
func (s *BankService) SaveEditedBank(userID, courseKey, courseName string, chapterNames map[string]string, chapters map[string][]course.Question) (*course.BankMeta, error) {
	if !strings.HasPrefix(courseKey, "custom_") {
		return nil, fmt.Errorf("内置题库不能编辑")
	}
	if len(chapters) == 0 {
		return nil, fmt.Errorf("题库至少需要保留一个章节")
	}
	cleanedChapters := make(map[string][]course.Question, len(chapters))
	for key, questions := range chapters {
		cleaned, err := course.NormalizeQuestionList(questions)
		if err != nil {
			return nil, fmt.Errorf("章节 %s 不可用: %w", key, err)
		}
		cleanedChapters[fmt.Sprintf("%d", course.ParseIndex(key))] = cleaned
	}
	if chapterNames == nil {
		chapterNames = map[string]string{}
	}
	cleanedChapters, chapterNames = compactChapters(cleanedChapters, chapterNames)
	for key := range cleanedChapters {
		if strings.TrimSpace(chapterNames[key]) == "" {
			chapterNames[key] = "章节 " + key
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	meta := course.BankMeta{
		CourseKey:     courseKey,
		CourseName:    courseName,
		ChapterNames:  chapterNames,
		ChapterCount:  len(cleanedChapters),
		QuestionCount: countQuestions(cleanedChapters),
		UpdatedAt:     now,
		CreatedAt:     now,
	}
	// 保留原创建时间
	if old, _, err := s.loadCustomPayload(userID, courseKey); err == nil && old.CreatedAt != "" {
		meta.CreatedAt = old.CreatedAt
	}

	if err := s.saveCustomBank(userID, courseKey, meta, cleanedChapters); err != nil {
		return nil, err
	}
	s.Registry.Register(&course.Bank{
		Key:          courseKey,
		Name:         courseName,
		Custom:       true,
		MaxChapter:   course.MaxChapterKey(cleanedChapters),
		ChapterNames: meta.ChapterNames,
		Chapters:     cleanedChapters,
	})
	return &meta, nil
}

// DeleteBank 删除自定义题库文件并从注册表移除。内置题库不允许删除。
func (s *BankService) DeleteBank(userID, courseKey string) error {
	if !strings.HasPrefix(courseKey, "custom_") {
		return fmt.Errorf("内置题库不能删除")
	}
	path, err := s.CustomBankPath(userID, courseKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除题库文件失败: %w", err)
	}
	s.Registry.Delete(courseKey)
	return nil
}

// LoadForEdit 读取题库文件的元信息与内容，供编辑器使用。
func (s *BankService) LoadForEdit(userID, courseKey string) (*course.BankMeta, *course.Bank, error) {
	return s.loadCustomPayload(userID, courseKey)
}

// loadCustomPayload 读取题库文件的元信息与内容。
func (s *BankService) loadCustomPayload(userID, courseKey string) (*course.BankMeta, *course.Bank, error) {
	path, err := s.CustomBankPath(userID, courseKey)
	if err != nil {
		return nil, nil, err
	}
	bank, meta, err := course.LoadCustomFromFile(path, courseKey+".json")
	if err != nil {
		return nil, nil, err
	}
	return meta, bank, nil
}

func (s *BankService) saveCustomBank(userID, courseKey string, meta course.BankMeta, chapters map[string][]course.Question) error {
	if err := s.Store.ValidateUserID(userID); err != nil {
		return err
	}
	path, err := s.CustomBankPath(userID, courseKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Meta     course.BankMeta              `json:"meta"`
		Chapters map[string][]course.Question `json:"chapters"`
	}{Meta: meta, Chapters: chapters}, "", "  ")
	if err != nil {
		return err
	}
	// 原子写：临时文件 + rename
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func normalizeChapterKey(value string, chapters map[string][]course.Question) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "auto") {
		return course.NextChapterKey(chapters)
	}
	return fmt.Sprintf("%d", course.ParseIndex(value))
}

func countQuestions(chapters map[string][]course.Question) int {
	total := 0
	for _, questions := range chapters {
		total += len(questions)
	}
	return total
}

// compactChapters 把章节键重新压缩为连续的 0..n，并同步章节名。
func compactChapters(chapters map[string][]course.Question, chapterNames map[string]string) (map[string][]course.Question, map[string]string) {
	keys := make([]string, 0, len(chapters))
	for key := range chapters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return course.ParseIndex(keys[i]) < course.ParseIndex(keys[j])
	})
	compacted := make(map[string][]course.Question, len(chapters))
	compactedNames := make(map[string]string, len(chapters))
	for index, oldKey := range keys {
		newKey := fmt.Sprintf("%d", index)
		compacted[newKey] = chapters[oldKey]
		if name := strings.TrimSpace(chapterNames[oldKey]); name != "" {
			compactedNames[newKey] = name
		}
	}
	return compacted, compactedNames
}
