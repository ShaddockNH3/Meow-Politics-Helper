// Package quiz 包含刷题核心业务：选题、判分、错题本与答题统计。
package quiz

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"quiz/internal/course"
)

// OutputQuestion 是下发给前端的题目 DTO（与旧版 QuestionOutput 字段一致）。
type OutputQuestion struct {
	QuizQuestionID         string            `json:"quiz_question_id"`
	DisplayNumber          int               `json:"display_number"`
	Course                 string            `json:"course"`
	OriginalChapter        string            `json:"original_chapter"`
	OriginalQuestionNumber string            `json:"original_question_number"`
	QuestionType           string            `json:"question_type"`
	QuestionText           string            `json:"question_text"`
	Options                map[string]string `json:"options"`
	CorrectAnswer          string            `json:"correct_answer"`
}

// SelectQuestions 按章节选择和顺序要求从题库中筛题。
// chapterChoices 中的 "all"（或旧版别名 "9"）表示全部章节；
// orderChoice 为 "random"（或旧版别名 "1"）时随机打乱。
func SelectQuestions(bank *course.Bank, chapterChoices []string, orderChoice string) []course.Question {
	if bank == nil {
		return nil
	}
	var selected []course.Question

	isSelectAll := false
	for _, choice := range chapterChoices {
		if strings.ToLower(choice) == "all" || choice == "9" {
			isSelectAll = true
			break
		}
	}

	var chapterKeys []string
	if isSelectAll {
		for i := 0; i <= bank.MaxChapter; i++ {
			chapterKeys = append(chapterKeys, strconv.Itoa(i))
		}
	} else {
		for _, choice := range chapterChoices {
			if _, err := strconv.Atoi(choice); err != nil {
				continue
			}
			if _, ok := bank.Chapters[choice]; ok {
				chapterKeys = append(chapterKeys, choice)
			}
		}
	}

	for _, chapterKey := range chapterKeys {
		selected = append(selected, bank.Chapters[chapterKey]...)
	}

	if len(selected) == 0 {
		return []course.Question{}
	}
	if orderChoice == "random" || orderChoice == "1" {
		Shuffle(selected)
	}
	return selected
}

// Shuffle 随机打乱切片（线程安全）。
func Shuffle[T any](items []T) {
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
}

// ToOutput 把题目列表转换为 API 输出。
// QuizQuestionID 格式为 "quiz_<course>_<chapter>_<index>"，判分时据此反查原题。
func ToOutput(questions []course.Question, courseKey string) []OutputQuestion {
	output := make([]OutputQuestion, len(questions))
	for i, q := range questions {
		output[i] = OutputQuestion{
			QuizQuestionID:         fmt.Sprintf("quiz_%s_%s_%d", courseKey, q.OriginalChapterKey, q.OriginalIndex),
			DisplayNumber:          i + 1,
			Course:                 courseKey,
			OriginalChapter:        q.OriginalChapterKey,
			OriginalQuestionNumber: q.QuestionNumber,
			QuestionType:           q.QuestionType,
			QuestionText:           q.QuestionText,
			Options:                q.Options,
			CorrectAnswer:          q.CorrectAnswer,
		}
	}
	return output
}

// IncorrectToOutput 把错题记录转换为 API 输出。
// QuizQuestionID 格式为 "incorrect_<course>_<chapter>_<题号>_<列表索引>"。
func IncorrectToOutput(items []IncorrectQuestion, fallbackCourse string) []OutputQuestion {
	output := make([]OutputQuestion, len(items))
	for i, item := range items {
		courseKey := item.Course
		if courseKey == "" {
			courseKey = fallbackCourse
		}
		output[i] = OutputQuestion{
			QuizQuestionID:         fmt.Sprintf("incorrect_%s_%s_%s_%d", courseKey, item.OriginalChapter, item.QuestionNumber, i),
			DisplayNumber:          i + 1,
			Course:                 courseKey,
			OriginalChapter:        item.OriginalChapter,
			OriginalQuestionNumber: item.QuestionNumber,
			QuestionType:           item.QuestionType,
			QuestionText:           item.QuestionText,
			Options:                item.Options,
			CorrectAnswer:          item.CorrectAnswer,
		}
	}
	return output
}

// ParseQuizQuestionID 解析 "quiz_<course>_<chapter>_<index>" 格式的题目 ID。
// course 本身可含下划线（如 custom_xxx），因此章节取倒数第二段、索引取最后一段。
func ParseQuizQuestionID(id string) (courseKey, chapterKey string, index int, err error) {
	raw := strings.TrimPrefix(id, "quiz_")
	parts := strings.Split(raw, "_")
	if len(parts) < 3 {
		return "", "", 0, fmt.Errorf("无法从题目ID %s 解析原始题目信息", id)
	}
	courseKey = strings.Join(parts[:len(parts)-2], "_")
	chapterKey = parts[len(parts)-2]
	index, convErr := strconv.Atoi(parts[len(parts)-1])
	if convErr != nil {
		return "", "", 0, fmt.Errorf("题目ID %s 中的索引不合法", id)
	}
	return courseKey, chapterKey, index, nil
}

// IsCorrect 服务端判分：把用户答案与正确答案都按字母排序后比较，
// 因此 "BA" 和 "AB" 对多选题等价，大小写不敏感。
func IsCorrect(q course.Question, userAnswer string) bool {
	return normalizeAnswer(userAnswer) == normalizeAnswer(q.CorrectAnswer)
}

func normalizeAnswer(answer string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(answer)) {
		if r == ' ' || r == '\t' || r == ',' || r == '，' {
			continue
		}
		b.WriteRune(r)
	}
	letters := strings.Split(b.String(), "")
	sort.Strings(letters)
	return strings.Join(letters, "")
}
