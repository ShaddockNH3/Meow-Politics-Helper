package quiz

import (
	"fmt"
	"log"
	"strings"
	"time"

	"quiz/internal/course"
	"quiz/internal/store"
)

// 用户数据文件名（与旧版保持兼容，老用户的错题本和统计不丢）。
const (
	StatsFileName = "question_stats.json"

	incorrectSuffix = "_incorrect_questions.json"
	deletedSuffix   = "_deleted_incorrect_questions.json"
)

// IncorrectFileName 返回课程对应的错题本文件名：<course>_incorrect_questions.json。
// 该命名恰好复现了旧版所有课程的文件名（maogai / xigai_li / xigai_yang / custom_*），
// 因此旧数据无需迁移。
func IncorrectFileName(courseKey string) string {
	return courseKey + incorrectSuffix
}

// IncorrectQuestion 是一条错题记录。
type IncorrectQuestion struct {
	Course          string            `json:"course,omitempty"`
	QuestionNumber  string            `json:"question_number"`
	QuestionType    string            `json:"question_type"`
	QuestionText    string            `json:"question_text"`
	Options         map[string]string `json:"options"`
	CorrectAnswer   string            `json:"correct_answer"`
	OriginalChapter string            `json:"original_chapter"`
	UserAnswer      string            `json:"user_answer,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
	DeletedAt       time.Time         `json:"deleted_at,omitempty"`
}

// QuestionStat 是一道题的答题统计。
type QuestionStat struct {
	OriginalChapterKey     string    `json:"original_chapter_key"`
	OriginalQuestionNumber string    `json:"original_question_number"`
	CorrectCount           int       `json:"correct_count"`
	ErrorCount             int       `json:"error_count"`
	LastAnswered           time.Time `json:"last_answered"`
}

// Service 持有持久化仓库，提供刷题相关的读写操作。
type Service struct {
	Store *store.Store
}

// RecordAnswer 记录一次答题：更新统计；答错则写入课程错题本（已答错过的题不重复记）。
// 判分在服务端完成，正确性以返回值告知调用方。
func (s *Service) RecordAnswer(userID, courseKey string, q course.Question, userAnswer string) (bool, error) {
	correct := IsCorrect(q, userAnswer)

	stats := make(map[string]QuestionStat)
	if err := s.Store.LoadJSON(userID, StatsFileName, &stats); err != nil {
		return correct, fmt.Errorf("加载用户统计数据失败: %w", err)
	}
	statKey := fmt.Sprintf("%s_%s", q.OriginalChapterKey, q.QuestionNumber)
	entry := stats[statKey]
	entry.OriginalChapterKey = q.OriginalChapterKey
	entry.OriginalQuestionNumber = q.QuestionNumber
	if correct {
		entry.CorrectCount++
	} else {
		entry.ErrorCount++
	}
	entry.LastAnswered = time.Now()
	stats[statKey] = entry
	if err := s.Store.SaveJSON(userID, StatsFileName, stats); err != nil {
		return correct, fmt.Errorf("保存用户统计数据失败: %w", err)
	}

	if !correct {
		if err := s.AddIncorrect(userID, courseKey, q, userAnswer); err != nil {
			return correct, err
		}
	}
	return correct, nil
}

// AddIncorrect 把一道题加入错题本；同一章节内相同题干的题不重复添加。
func (s *Service) AddIncorrect(userID, courseKey string, q course.Question, userAnswer string) error {
	book, err := s.IncorrectBook(userID, courseKey)
	if err != nil {
		return fmt.Errorf("加载用户错题本失败: %w", err)
	}
	for _, item := range book {
		if item.QuestionText == q.QuestionText && item.OriginalChapter == q.OriginalChapterKey {
			return nil
		}
	}
	book = append(book, IncorrectQuestion{
		Course:          courseKey,
		QuestionNumber:  q.QuestionNumber,
		QuestionType:    q.QuestionType,
		QuestionText:    q.QuestionText,
		Options:         q.Options,
		CorrectAnswer:   q.CorrectAnswer,
		OriginalChapter: q.OriginalChapterKey,
		UserAnswer:      userAnswer,
		Timestamp:       time.Now(),
	})
	return s.saveIncorrectBook(userID, courseKey, book)
}

// IncorrectBook 读取课程错题本。
func (s *Service) IncorrectBook(userID, courseKey string) ([]IncorrectQuestion, error) {
	book := []IncorrectQuestion{}
	if err := s.Store.LoadJSON(userID, IncorrectFileName(courseKey), &book); err != nil {
		return nil, err
	}
	return book, nil
}

func (s *Service) saveIncorrectBook(userID, courseKey string, book []IncorrectQuestion) error {
	return s.Store.SaveJSON(userID, IncorrectFileName(courseKey), book)
}

// DeleteIncorrect 从错题本删除一题（按章节 + 题号），并把记录归档到删除历史文件。
// 返回是否确实删除了题目。
func (s *Service) DeleteIncorrect(userID, courseKey, chapterKey, questionNumber string) (bool, error) {
	book, err := s.IncorrectBook(userID, courseKey)
	if err != nil {
		return false, err
	}
	if len(book) == 0 {
		return false, nil
	}

	updated := make([]IncorrectQuestion, 0, len(book))
	deleted := IncorrectQuestion{}
	found := false
	for _, item := range book {
		if item.OriginalChapter == chapterKey && item.QuestionNumber == questionNumber {
			found = true
			deleted = item
			deleted.DeletedAt = time.Now()
			continue
		}
		updated = append(updated, item)
	}
	if !found {
		return false, nil
	}
	if err := s.saveIncorrectBook(userID, courseKey, updated); err != nil {
		return false, err
	}

	history := []IncorrectQuestion{}
	historyFile := courseKey + deletedSuffix
	if err := s.Store.LoadJSON(userID, historyFile, &history); err != nil {
		log.Printf("警告：用户 %s 加载已删除错题历史失败: %v", userID, err)
	}
	history = append(history, deleted)
	if err := s.Store.SaveJSON(userID, historyFile, history); err != nil {
		log.Printf("警告：用户 %s 保存已删除错题历史失败: %v", userID, err)
	}
	return true, nil
}

// PruneIncorrectForCourse 在自定义题库被编辑后，把错题本中已不存在的题目清掉。
func (s *Service) PruneIncorrectForCourse(userID, courseKey string, chapters map[string][]course.Question) error {
	book, err := s.IncorrectBook(userID, courseKey)
	if err != nil || len(book) == 0 {
		return err
	}
	existing := make(map[string]bool)
	for chapterKey, questions := range chapters {
		for _, q := range questions {
			existing[chapterKey+"\x00"+q.QuestionNumber] = true
		}
	}
	filtered := make([]IncorrectQuestion, 0, len(book))
	for _, item := range book {
		if existing[item.OriginalChapter+"\x00"+item.QuestionNumber] {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == len(book) {
		return nil
	}
	return s.saveIncorrectBook(userID, courseKey, filtered)
}

// ClearUserData 清理用户的错题本与统计：相关文件重命名加时间戳备份，
// 遍历用户目录下所有 *_incorrect_questions.json（含旧版遗留文件名）。
func (s *Service) ClearUserData(userID string) error {
	files, err := s.Store.UserFiles(userID)
	if err != nil {
		return err
	}
	stamp := time.Now().Format("2006_01_02_15_04_05")
	for _, name := range files {
		if name == StatsFileName || (strings.HasSuffix(name, incorrectSuffix) && !strings.HasSuffix(name, deletedSuffix)) {
			if err := s.Store.RenameFile(userID, name, stamp); err != nil {
				log.Printf("错误：用户 %s 清理文件 %s 失败: %v", userID, name, err)
			} else {
				log.Printf("信息：用户 %s 的 %s 已备份清理。", userID, name)
			}
		}
	}
	return nil
}
