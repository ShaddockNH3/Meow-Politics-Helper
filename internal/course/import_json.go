package course

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// readFileLimited 读取文件，最大 16 MiB，防止异常大文件拖垮内存。
func readFileLimited(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	const max = int64(16 << 20)
	if info.Size() > max {
		return nil, fmt.Errorf("文件超过大小限制 (%d MiB)", max>>20)
	}
	return os.ReadFile(path)
}

// DefaultGeneratePrompt 是推荐给用户拿去外部 AI（网页版或其他工具）使用的生成提示词。
// 生成的 JSON 可直接粘贴进「导入题库（粘贴 JSON）」。
const DefaultGeneratePrompt = `你是一个严谨的题库生成器。请从用户提供的材料中生成适合刷题的选择题。

重要边界：
- "参考资料"才是出题依据。
- 本提示词里的字段名、JSON 示例、输出要求都不是参考资料，绝对不能围绕 question_number、question_type、options、correct_answer 等格式字段出题。
- 如果参考资料是一篇文章、课文、讲义或摘录，题目必须考察该资料的内容本身，而不是考察 JSON 格式。

输出要求：
1. 只输出 JSON，不要 Markdown，不要解释，不要代码块。
2. 顶层必须是一个 JSON 数组。
3. 每个题目对象必须严格包含 question_number、question_type、question_text、options、correct_answer。
4. question_type 只能是 "单选题" 或 "多选题"。
5. options 是对象，键只能使用 A、B、C、D、E，值为非空选项文本。
6. correct_answer 是字符串：单选如 "A"，多选如 "AB"，必须只包含 options 中存在的键。
7. 不要编造材料中没有依据的事实。材料不足时少出题，不要硬凑。
8. 题目不要重复，选项不要重复。
9. 默认生成 20-30 题；如果用户指定题量，以用户要求为准。

示例输出：
[
  {
    "question_number": "AI-001",
    "question_type": "单选题",
    "question_text": "题干文本",
    "options": {
      "A": "选项A",
      "B": "选项B",
      "C": "选项C",
      "D": "选项D"
    },
    "correct_answer": "A"
  }
]`

// ParseQuestionsJSON 解析粘贴导入的题库 JSON 文本。
// 兼容多种来源格式：标准数组、被 Markdown 代码块包裹的数组、
// {"questions":[...]} / {"data":[...]} / {"items":[...]} 包装、
// 以及 question/answer 等宽松字段名。解析后统一清洗。
func ParseQuestionsJSON(text string) ([]Question, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}
	questions, err := decodeQuestions([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("内容不是合法题库 JSON: %w", err)
	}
	return NormalizeQuestionList(questions)
}

// NormalizeQuestionList 清洗题目列表：去空白、统一答案大小写、
// 剔除缺题干/选项不足/答案无效的题目、去重、补题号、规范题型。
func NormalizeQuestionList(questions []Question) ([]Question, error) {
	cleaned := make([]Question, 0, len(questions))
	seen := make(map[string]bool)
	for i, q := range questions {
		q.QuestionText = strings.TrimSpace(q.QuestionText)
		q.CorrectAnswer = strings.ToUpper(strings.TrimSpace(q.CorrectAnswer))
		if q.QuestionText == "" || len(q.Options) < 2 || q.CorrectAnswer == "" {
			continue
		}
		if seen[q.QuestionText] {
			continue
		}
		seen[q.QuestionText] = true
		options := make(map[string]string)
		for key, value := range q.Options {
			key = strings.ToUpper(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			if key == "" || value == "" {
				continue
			}
			options[key] = value
		}
		q.Options = options
		if len(q.Options) < 2 || !AnswerMatchesOptions(q.CorrectAnswer, q.Options) {
			continue
		}
		if q.QuestionNumber == "" {
			q.QuestionNumber = fmt.Sprintf("IMPORT-%03d", i+1)
		}
		if len([]rune(q.CorrectAnswer)) > 1 {
			q.QuestionType = "多选题"
		} else if q.QuestionType != "多选题" {
			q.QuestionType = "单选题"
		}
		cleaned = append(cleaned, q)
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("没有可用题目：请检查题干、选项和答案字段")
	}
	return cleaned, nil
}

// AnswerMatchesOptions 检查答案中的每个选项键都存在于选项里。
func AnswerMatchesOptions(answer string, options map[string]string) bool {
	for _, r := range answer {
		if _, ok := options[string(r)]; !ok {
			return false
		}
	}
	return true
}

// decodeQuestions 先按严格格式解析，失败后按宽松字段名逐题解析。
func decodeQuestions(data []byte) ([]Question, error) {
	var questions []Question
	if err := json.Unmarshal(data, &questions); err == nil {
		return questions, nil
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		var wrapper map[string]json.RawMessage
		if wrapperErr := json.Unmarshal(data, &wrapper); wrapperErr != nil {
			return nil, err
		}
		for _, key := range []string{"questions", "data", "items"} {
			if raw, ok := wrapper[key]; ok {
				if arrayErr := json.Unmarshal(raw, &rawItems); arrayErr != nil {
					return nil, arrayErr
				}
				break
			}
		}
	}
	questions = make([]Question, 0, len(rawItems))
	for _, raw := range rawItems {
		question, err := decodeFlexibleQuestion(raw)
		if err == nil {
			questions = append(questions, question)
		}
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("未找到可解析的题目对象")
	}
	return questions, nil
}

func decodeFlexibleQuestion(data []byte) (Question, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Question{}, err
	}
	question := Question{
		QuestionNumber: scalarString(raw["question_number"]),
		QuestionType:   scalarString(raw["question_type"]),
		QuestionText:   scalarString(raw["question_text"]),
		CorrectAnswer:  scalarString(raw["correct_answer"]),
		Options:        normalizeFlexibleOptions(raw["options"]),
	}
	if question.QuestionText == "" {
		question.QuestionText = scalarString(raw["question"])
	}
	if question.CorrectAnswer == "" {
		question.CorrectAnswer = scalarString(raw["answer"])
	}
	return question, nil
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%g", typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func normalizeFlexibleOptions(value any) map[string]string {
	options := make(map[string]string)
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if text := scalarString(item); text != "" {
				options[strings.ToUpper(key)] = text
			}
		}
	case []any:
		for index, item := range typed {
			if index >= 5 {
				break
			}
			if text := scalarString(item); text != "" {
				options[string(rune('A'+index))] = text
			}
		}
	}
	return options
}
