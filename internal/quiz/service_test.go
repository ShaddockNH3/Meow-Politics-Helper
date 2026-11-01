package quiz

import (
	"testing"

	"quiz/internal/course"
	"quiz/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return &Service{Store: store.New(t.TempDir())}
}

func bankQuestion(number, text, answer string) course.Question {
	return course.Question{
		QuestionNumber:     number,
		QuestionType:       "单选题",
		QuestionText:       text,
		Options:            map[string]string{"A": "甲", "B": "乙", "C": "丙"},
		CorrectAnswer:      answer,
		OriginalChapterKey: "0",
	}
}

func TestRecordAnswerAndIncorrectBook(t *testing.T) {
	svc := newTestService(t)
	q := bankQuestion("1", "题干", "A")

	// 答对：不进错题本
	correct, err := svc.RecordAnswer("u1", "demo", q, "A")
	if err != nil || !correct {
		t.Fatalf("答对应判正确且无错误: correct=%v err=%v", correct, err)
	}
	book, _ := svc.IncorrectBook("u1", "demo")
	if len(book) != 0 {
		t.Fatalf("答对不应进错题本，实际 %d 条", len(book))
	}

	// 答错：进错题本一次
	correct, err = svc.RecordAnswer("u1", "demo", q, "B")
	if err != nil || correct {
		t.Fatalf("答错应判错误: correct=%v err=%v", correct, err)
	}
	// 重复答错同一题不重复记录
	_, _ = svc.RecordAnswer("u1", "demo", q, "C")
	book, err = svc.IncorrectBook("u1", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(book) != 1 {
		t.Fatalf("错题本应只有 1 条（去重），实际 %d", len(book))
	}
	if book[0].UserAnswer != "B" {
		t.Fatalf("应记录首次错误答案 B，实际 %s", book[0].UserAnswer)
	}

	// 统计也应更新
	stats := make(map[string]QuestionStat)
	if err := svc.Store.LoadJSON("u1", StatsFileName, &stats); err != nil {
		t.Fatal(err)
	}
	entry, ok := stats["0_1"]
	if !ok || entry.CorrectCount != 1 || entry.ErrorCount != 2 {
		t.Fatalf("统计不正确: %+v", entry)
	}
}

func TestDeleteIncorrect(t *testing.T) {
	svc := newTestService(t)
	q := bankQuestion("7", "另一题", "C")
	_, _ = svc.RecordAnswer("u1", "demo", q, "A")

	deleted, err := svc.DeleteIncorrect("u1", "demo", q.OriginalChapterKey, q.QuestionNumber)
	if err != nil || !deleted {
		t.Fatalf("删除错题应成功: deleted=%v err=%v", deleted, err)
	}
	book, _ := svc.IncorrectBook("u1", "demo")
	if len(book) != 0 {
		t.Fatalf("删除后错题本应为空，实际 %d", len(book))
	}
	if deleted, _ := svc.DeleteIncorrect("u1", "demo", q.OriginalChapterKey, q.QuestionNumber); deleted {
		t.Fatal("重复删除应返回 false")
	}
}

func TestPruneIncorrectForCourse(t *testing.T) {
	svc := newTestService(t)
	q1 := bankQuestion("1", "保留", "A")
	q2 := bankQuestion("2", "淘汰", "B")
	_, _ = svc.RecordAnswer("u1", "demo", q1, "X")
	_, _ = svc.RecordAnswer("u1", "demo", q2, "X")

	if err := svc.PruneIncorrectForCourse("u1", "demo", map[string][]course.Question{
		"0": {q1},
	}); err != nil {
		t.Fatal(err)
	}
	book, _ := svc.IncorrectBook("u1", "demo")
	if len(book) != 1 || book[0].QuestionText != "保留" {
		t.Fatalf("清理后应只剩保留的错题: %+v", book)
	}
}

func TestClearUserData(t *testing.T) {
	svc := newTestService(t)
	q := bankQuestion("1", "题干", "A")
	_, _ = svc.RecordAnswer("u1", "demo", q, "B")

	if err := svc.ClearUserData("u1"); err != nil {
		t.Fatal(err)
	}
	book, _ := svc.IncorrectBook("u1", "demo")
	if len(book) != 0 {
		t.Fatalf("清理后错题本应为空，实际 %d", len(book))
	}
	stats := make(map[string]QuestionStat)
	_ = svc.Store.LoadJSON("u1", StatsFileName, &stats)
	if len(stats) != 0 {
		t.Fatalf("清理后统计应为空，实际 %d", len(stats))
	}
}
