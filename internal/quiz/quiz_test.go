package quiz

import (
	"testing"

	"quiz/internal/course"
)

func sampleBank() *course.Bank {
	return &course.Bank{
		Key:        "demo",
		MaxChapter: 2,
		Chapters: map[string][]course.Question{
			"0": {
				{QuestionNumber: "1", QuestionText: "Q0-1", Options: map[string]string{"A": "a", "B": "b"}, CorrectAnswer: "A", OriginalChapterKey: "0", OriginalIndex: 0},
				{QuestionNumber: "2", QuestionText: "Q0-2", Options: map[string]string{"A": "a", "B": "b"}, CorrectAnswer: "B", OriginalChapterKey: "0", OriginalIndex: 1},
			},
			"1": {
				{QuestionNumber: "1", QuestionText: "Q1-1", Options: map[string]string{"A": "a", "B": "b"}, CorrectAnswer: "A", OriginalChapterKey: "1", OriginalIndex: 0},
			},
			"2": {}, // 空章节
		},
	}
}

func TestSelectQuestionsAllChapters(t *testing.T) {
	selected := SelectQuestions(sampleBank(), []string{"all"}, "sequential")
	if len(selected) != 3 {
		t.Fatalf("全部章节应选出 3 题，实际 %d", len(selected))
	}
	// 顺序应保持章节号升序
	if selected[0].QuestionText != "Q0-1" || selected[2].QuestionText != "Q1-1" {
		t.Fatalf("章节顺序不正确: %s -> %s", selected[0].QuestionText, selected[2].QuestionText)
	}
}

func TestSelectQuestionsSpecificChapter(t *testing.T) {
	selected := SelectQuestions(sampleBank(), []string{"1"}, "sequential")
	if len(selected) != 1 || selected[0].QuestionText != "Q1-1" {
		t.Fatalf("指定章节选题不正确: %+v", selected)
	}
	// 不存在的章节应被跳过（"5" 不存在，注意 "9" 是旧版"全部章节"的别名）
	if got := SelectQuestions(sampleBank(), []string{"5", "0"}, "sequential"); len(got) != 2 {
		t.Fatalf("无效章节应跳过，实际 %d 题", len(got))
	}
	// 旧版别名 "9" 等价于全部章节
	if got := SelectQuestions(sampleBank(), []string{"9"}, "sequential"); len(got) != 3 {
		t.Fatalf("别名 9 应等价于全部章节，实际 %d 题", len(got))
	}
}

func TestSelectQuestionsRandomStillFull(t *testing.T) {
	selected := SelectQuestions(sampleBank(), []string{"all"}, "random")
	if len(selected) != 3 {
		t.Fatalf("随机模式也应返回全部 3 题，实际 %d", len(selected))
	}
}

func TestParseQuizQuestionID(t *testing.T) {
	courseKey, chapter, index, err := ParseQuizQuestionID("quiz_custom_my_bank_3_12")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if courseKey != "custom_my_bank" || chapter != "3" || index != 12 {
		t.Fatalf("解析结果不正确: %s %s %d", courseKey, chapter, index)
	}
	if _, _, _, err := ParseQuizQuestionID("garbage"); err == nil {
		t.Fatal("非法 ID 应报错")
	}
}

func TestIsCorrect(t *testing.T) {
	q := course.Question{CorrectAnswer: "AB", Options: map[string]string{"A": "1", "B": "2"}}
	cases := map[string]bool{
		"AB":    true,
		"ba":    true, // 顺序与大小写无关
		"A B":   true,
		"A":     false,
		"ABC":   false,
		"":      false,
		"  ab ": true,
	}
	for answer, want := range cases {
		if got := IsCorrect(q, answer); got != want {
			t.Errorf("IsCorrect(%q) = %v, 期望 %v", answer, got, want)
		}
	}
}

func TestToOutputIDs(t *testing.T) {
	output := ToOutput(sampleBank().Chapters["0"], "demo")
	if output[0].QuizQuestionID != "quiz_demo_0_0" || output[1].DisplayNumber != 2 {
		t.Fatalf("ToOutput 结果不正确: %+v", output)
	}
}
