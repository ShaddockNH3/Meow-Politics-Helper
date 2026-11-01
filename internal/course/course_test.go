package course

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadBuiltin(t *testing.T) {
	fsys := fstest.MapFS{
		"clean_outputs/demo/0.json": &fstest.MapFile{Data: []byte(`[
			{"question_number":"1","question_type":"单选题","question_text":"题干A","options":{"A":"甲","B":"乙"},"correct_answer":"A"},
			{"question_number":"2","question_type":"单选题","question_text":"题干B","options":{"A":"甲","B":"乙"},"correct_answer":"B"}
		]`)},
		"clean_outputs/demo/1.json": &fstest.MapFile{Data: []byte(`[]`)},
		// 章节 2 缺失，应跳过而不是报错
	}
	registry := NewRegistry()
	err := LoadBuiltin(registry, fsys, []BuiltinSpec{
		{Key: "demo", Name: "演示题库", SourceDir: "clean_outputs/demo", MaxChapter: 2},
	})
	if err != nil {
		t.Fatalf("LoadBuiltin 失败: %v", err)
	}
	bank, ok := registry.Get("demo")
	if !ok {
		t.Fatal("题库应已注册")
	}
	if bank.QuestionCount() != 2 {
		t.Fatalf("应有 2 题，实际 %d", bank.QuestionCount())
	}
	if _, ok := bank.Chapters["2"]; !ok {
		t.Fatal("缺失章节也应初始化为空章节")
	}
	q, ok := bank.Lookup("0", 1)
	if !ok || q.QuestionNumber != "2" || q.OriginalChapterKey != "0" || q.OriginalIndex != 1 {
		t.Fatalf("Lookup 结果不正确: %+v ok=%v", q, ok)
	}
}

func TestParseQuestionsJSONVariants(t *testing.T) {
	// 标准 + 代码块包裹 + 宽松字段
	cases := []string{
		`[{"question_number":"1","question_type":"单选题","question_text":"题干","options":{"A":"甲","B":"乙"},"correct_answer":"A"}]`,
		"```json\n[{\"question_number\":\"1\",\"question_type\":\"单选题\",\"question_text\":\"题干\",\"options\":{\"A\":\"甲\",\"B\":\"乙\"},\"correct_answer\":\"A\"}]\n```",
		`[{"question":"题干","answer":"A","options":["甲","乙"]}]`,
		`{"questions":[{"question_number":"1","question_type":"单选题","question_text":"题干","options":{"A":"甲","B":"乙"},"correct_answer":"a"}]}`,
	}
	for i, text := range cases {
		questions, err := ParseQuestionsJSON(text)
		if err != nil {
			t.Fatalf("case %d 解析失败: %v", i, err)
		}
		if len(questions) != 1 {
			t.Fatalf("case %d 应解析出 1 题，实际 %d", i, len(questions))
		}
		if questions[0].CorrectAnswer != "A" {
			t.Fatalf("case %d 答案应为大写 A，实际 %s", i, questions[0].CorrectAnswer)
		}
	}
}

func TestParseQuestionsJSONRejectsInvalid(t *testing.T) {
	if _, err := ParseQuestionsJSON(`[{"question_text":"缺选项","correct_answer":"A"}]`); err == nil {
		t.Fatal("缺选项的题目应被全部剔除并报错")
	}
	if _, err := ParseQuestionsJSON(`[{"question_text":"答案不在选项里","options":{"A":"甲"},"correct_answer":"C"}]`); err == nil {
		t.Fatal("答案不在选项中的题目应被剔除并报错")
	}
}

func TestNormalizeQuestionListDedupAndType(t *testing.T) {
	questions, err := NormalizeQuestionList([]Question{
		{QuestionText: "多选", Options: map[string]string{"A": "1", "B": "2", "C": "3"}, CorrectAnswer: "cba"},
		{QuestionText: "多选", Options: map[string]string{"A": "1", "B": "2", "C": "3"}, CorrectAnswer: "ABC"}, // 重复题干应剔除
		{QuestionText: "单选", Options: map[string]string{"A": "1", "B": "2"}, CorrectAnswer: "b", QuestionType: "单选题"},
	})
	if err != nil {
		t.Fatalf("NormalizeQuestionList 失败: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("去重后应剩 2 题，实际 %d", len(questions))
	}
	if questions[0].QuestionType != "多选题" || questions[0].CorrectAnswer != "CBA" {
		t.Fatalf("多选题答案/类型不规范: %+v", questions[0])
	}
}

func TestSanitizeKey(t *testing.T) {
	if got := SanitizeKey("My Bank!"); !strings.HasPrefix(got, "custom_") || strings.Contains(got, " ") {
		t.Fatalf("SanitizeKey 结果不合法: %s", got)
	}
	if got := SanitizeKey("custom_already"); got != "custom_already" {
		t.Fatalf("已有前缀不应重复添加: %s", got)
	}
	if got := SanitizeKey("!!!"); !strings.HasPrefix(got, "custom_") {
		t.Fatalf("空结果应生成兜底键: %s", got)
	}
}
