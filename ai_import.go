package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const defaultAIBaseURL = "https://api.openai.com/v1"
const defaultAIAPIPath = "/chat/completions"
const defaultAIModel = "gpt-4.1-mini"
const defaultAISystemPrompt = `你是一个严谨的题库生成器。请从用户提供的材料中生成适合刷题的选择题。

重要边界：
- “参考资料”才是出题依据。
- 本提示词里的字段名、JSON 示例、输出要求都不是参考资料，绝对不能围绕 question_number、question_type、options、correct_answer、correct_count、error_count 等格式字段出题。
- 如果参考资料是一篇文章、课文、讲义或摘录，题目必须考察该资料的内容本身，而不是考察 JSON 格式。

输出要求：
1. 只输出 JSON，不要 Markdown，不要解释，不要代码块。
2. 顶层必须是一个 JSON 数组。
3. 每个题目对象必须严格包含这些字段，并与示例字段名完全一致：
   - question_number: 字符串，例如 "AI-001"
   - question_type: 只能是 "单选题" 或 "多选题"
   - question_text: 字符串，题干应完整、清晰、可独立理解
   - options: 对象，键只能使用 "A"、"B"、"C"、"D"、"E"，值为选项文本
   - correct_answer: 字符串，单选如 "A"，多选如 "AB"，必须只包含 options 中存在的选项键
   - correct_count: 数字，固定为 0
   - error_count: 数字，固定为 0
4. 不要生成判断题、填空题、简答题。
5. 不要编造材料中没有依据的事实。材料不足时少出题，不要硬凑。
6. 多选题必须至少有两个正确选项；单选题只能有一个正确选项。
7. 题目之间不要重复，选项之间不要重复。
8. 默认生成 20-30 题；如果用户要求题量，以用户要求为准。

示例输出格式：
[
  {
    "question_number": "AI-001",
    "question_type": "单选题",
    "question_text": "题干文本",
    "options": {"A":"选项A","B":"选项B","C":"选项C","D":"选项D"},
    "correct_answer": "A",
    "correct_count": 0,
    "error_count": 0
  }
]`

var (
	aiTasks   = make(map[string]*AIImportTask)
	aiTasksMu sync.RWMutex
)

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func loadAIConfig(userID string) (AIConfig, error) {
	var cfg AIConfig
	if err := loadUserJSONData(userID, aiConfigFile, &cfg); err != nil {
		return cfg, err
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultAIBaseURL
	}
	if cfg.APIPath == "" {
		cfg.APIPath = defaultAIAPIPath
	}
	if cfg.Model == "" {
		cfg.Model = defaultAIModel
	}
	return cfg, nil
}

func saveAIConfig(userID string, cfg AIConfig) error {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultAIBaseURL
	}
	cfg.APIPath = normalizeAPIPath(cfg.APIPath)
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultAIModel
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	return saveUserJSONData(userID, aiConfigFile, cfg)
}

func normalizeAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultAIAPIPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func AIConfigStatusHandler(ctx context.Context, c *app.RequestContext) {
	userID := string(c.Query("user_id"))
	if err := validateUserID(userID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	cfg, err := loadAIConfig(userID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "读取 AI 配置失败"})
		return
	}
	c.JSON(consts.StatusOK, AIConfigStatus{Configured: cfg.APIKey != "", BaseURL: cfg.BaseURL, APIPath: cfg.APIPath, Model: cfg.Model, KeyPreview: maskAPIKey(cfg.APIKey)})
}

func SaveAIConfigHandler(ctx context.Context, c *app.RequestContext) {
	var req SaveAIConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	if err := validateUserID(req.UserID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	old, _ := loadAIConfig(req.UserID)
	cfg := AIConfig{APIKey: req.APIKey, BaseURL: req.BaseURL, APIPath: req.APIPath, Model: req.Model}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = old.APIKey
	}
	if err := saveAIConfig(req.UserID, cfg); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存 AI 配置失败"})
		return
	}
	c.JSON(consts.StatusOK, utils.H{"message": "AI 配置已保存", "key_preview": maskAPIKey(cfg.APIKey)})
}

func AIModelsHandler(ctx context.Context, c *app.RequestContext) {
	userID := string(c.Query("user_id"))
	if err := validateUserID(userID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	cfg, err := loadAIConfig(userID)
	if err != nil || cfg.APIKey == "" {
		c.JSON(consts.StatusOK, utils.H{"models": []AIModelInfo{{ID: defaultAIModel}}, "message": "未配置 API Key，先使用默认模型"})
		return
	}
	models, err := fetchAIModels(cfg)
	if err != nil || len(models) == 0 {
		message := "模型列表获取失败，保留当前模型"
		if err != nil {
			message += ": " + err.Error()
		}
		c.JSON(consts.StatusOK, utils.H{"models": []AIModelInfo{{ID: cfg.Model}}, "message": message})
		return
	}
	c.JSON(consts.StatusOK, utils.H{"models": models})
}

func fetchAIModels(cfg AIConfig) ([]AIModelInfo, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fetchAIModelsWithPowerShell(cfg, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("模型列表接口返回 %d: %s", resp.StatusCode, string(data))
	}
	return parseModelList(data)
}

func fetchAIModelsWithPowerShell(cfg AIConfig, originalErr error) ([]AIModelInfo, error) {
	if runtime.GOOS != "windows" {
		return nil, originalErr
	}
	url := strings.TrimRight(cfg.BaseURL, "/") + "/models"
	script := `$ErrorActionPreference='Stop'; $headers=@{Authorization='Bearer ' + $env:AI_API_KEY}; (Invoke-WebRequest -Uri $env:AI_MODELS_URL -Headers $headers -UseBasicParsing -TimeoutSec 30).Content`
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Env = append(os.Environ(), "AI_API_KEY="+cfg.APIKey, "AI_MODELS_URL="+url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Go HTTP 请求失败: %v；PowerShell fallback 也失败: %v: %s", originalErr, err, strings.TrimSpace(string(output)))
	}
	return parseModelList(output)
}

func parseModelList(data []byte) ([]AIModelInfo, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	models := make([]AIModelInfo, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.ID == "" {
			continue
		}
		models = append(models, AIModelInfo{ID: item.ID})
	}
	return models, nil
}

func AIImportQuestionBankHandler(ctx context.Context, c *app.RequestContext) {
	userID := c.PostForm("user_id")
	if err := validateUserID(userID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	cfg, err := loadAIConfig(userID)
	if err != nil || cfg.APIKey == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "请先保存 AI API Key"})
		return
	}

	courseName := strings.TrimSpace(c.PostForm("course_name"))
	if courseName == "" {
		courseName = "AI 生成题库"
	}
	targetMode := strings.TrimSpace(c.PostForm("target_mode"))
	targetCourse := strings.TrimSpace(c.PostForm("target_course"))
	chapterKey := strings.TrimSpace(c.PostForm("chapter_key"))
	courseKey := sanitizeCourseKey(c.PostForm("course_key"))
	chapterName := strings.TrimSpace(c.PostForm("chapter_name"))
	if chapterName == "" {
		chapterName = "AI 导入"
	}
	sourceText := strings.TrimSpace(c.PostForm("source_text"))
	systemPrompt := strings.TrimSpace(c.PostForm("system_prompt"))
	if systemPrompt == "" {
		systemPrompt = defaultAISystemPrompt
	}

	if fh, err := c.FormFile("file"); err == nil && fh != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "文件上传暂未开放，当前请复制材料文字，或使用 JSON 导入题库。PDF/DOCX/图片解析会作为未来计划处理。"})
		return
	}

	task := createAIImportTask(userID)
	c.JSON(consts.StatusAccepted, utils.H{"message": "题库生成已在后台开始", "task_id": task.ID})

	go runAIImportTask(task.ID, cfg, userID, targetMode, targetCourse, courseKey, courseName, chapterKey, chapterName, systemPrompt, sourceText)
}

func createAIImportTask(userID string) *AIImportTask {
	task := &AIImportTask{ID: fmt.Sprintf("ai_%d", time.Now().UnixNano()), UserID: userID, Status: "running", Message: "正在生成题库", CreatedAt: time.Now()}
	aiTasksMu.Lock()
	aiTasks[task.ID] = task
	aiTasksMu.Unlock()
	return task
}

func updateAIImportTask(taskID string, update func(*AIImportTask)) {
	aiTasksMu.Lock()
	defer aiTasksMu.Unlock()
	if task, ok := aiTasks[taskID]; ok {
		update(task)
	}
}

func runAIImportTask(taskID string, cfg AIConfig, userID, targetMode, targetCourse, courseKey, courseName, chapterKey, chapterName, systemPrompt, sourceText string) {
	questions, err := generateQuestionsWithAI(cfg, systemPrompt, sourceText)
	if err != nil {
		completeAIImportTask(taskID, "failed", "生成失败", err.Error(), nil, 0)
		return
	}
	if len(questions) == 0 {
		completeAIImportTask(taskID, "failed", "生成失败", "AI 没有生成可用题目，请换一份材料或补充文字说明", nil, 0)
		return
	}
	meta, _, savedChapterKey, err := saveQuestionsToCustomBank(userID, targetMode, targetCourse, courseKey, courseName, chapterKey, chapterName, questions)
	if err != nil {
		completeAIImportTask(taskID, "failed", "保存失败", "保存生成题库失败: "+err.Error(), nil, 0)
		return
	}
	completeAIImportTask(taskID, "completed", fmt.Sprintf("题库已生成到章节 %s", savedChapterKey), "", &meta, len(questions))
}

func completeAIImportTask(taskID, status, message, errText string, meta *AIQuestionBankMeta, questions int) {
	now := time.Now()
	updateAIImportTask(taskID, func(task *AIImportTask) {
		task.Status = status
		task.Message = message
		task.Error = errText
		task.Course = meta
		task.Questions = questions
		task.CompletedAt = &now
	})
}

func AIImportTaskStatusHandler(ctx context.Context, c *app.RequestContext) {
	taskID := string(c.Query("task_id"))
	userID := string(c.Query("user_id"))
	aiTasksMu.RLock()
	task, ok := aiTasks[taskID]
	aiTasksMu.RUnlock()
	if !ok || task.UserID != userID {
		c.JSON(consts.StatusNotFound, utils.H{"error": "任务不存在"})
		return
	}
	c.JSON(consts.StatusOK, task)
}

func JSONImportQuestionBankHandler(ctx context.Context, c *app.RequestContext) {
	var req JSONImportQuestionBankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	if err := validateUserID(req.UserID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	questions, err := parseAIQuestions(req.JSONText)
	if err != nil || len(questions) == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{"error": fmt.Sprintf("JSON 题库不可用: %v", err)})
		return
	}
	courseName := strings.TrimSpace(req.CourseName)
	if courseName == "" {
		courseName = "JSON 导入题库"
	}
	meta, _, chapterKey, err := saveQuestionsToCustomBank(req.UserID, req.TargetMode, req.TargetCourse, req.CourseKey, courseName, req.ChapterKey, req.ChapterName, questions)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "保存题库失败: " + err.Error()})
		return
	}
	c.JSON(consts.StatusOK, utils.H{"message": "JSON 题库已导入", "course": meta, "chapter_key": chapterKey, "questions": len(questions)})
}

func saveCustomQuestionBank(userID string, meta AIQuestionBankMeta, chapters map[string][]Question) error {
	if err := ensureUserDir(userID); err != nil {
		return err
	}
	dir := customQuestionBankDir(userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	payload := struct {
		Meta     AIQuestionBankMeta    `json:"meta"`
		Chapters map[string][]Question `json:"chapters"`
	}{Meta: meta, Chapters: chapters}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, meta.CourseKey+".json"), data, 0600)
}

func generateQuestionsWithAI(cfg AIConfig, prompt, sourceText string) ([]Question, error) {
	if strings.TrimSpace(sourceText) == "" {
		return nil, fmt.Errorf("请先提供参考资料文字，不要空材料生成题库")
	}
	if strings.Contains(cfg.APIPath, "chat/completions") {
		return generateQuestionsWithChatCompletions(cfg, prompt, sourceText)
	}
	return generateQuestionsWithResponses(cfg, prompt, sourceText)
}

func buildMaterialPrompt(sourceText string) string {
	var b strings.Builder
	b.WriteString("请严格只依据下面的参考资料生成题目；不得使用参考资料之外的主题、知识点或例子。\n")
	b.WriteString("每一道题的题干或选项都必须能在参考资料中找到直接依据。\n")
	b.WriteString("如果输出内容的主题与参考资料主题不一致，就是严重错误。\n")
	b.WriteString("如果参考资料不足以生成 20-30 题，就按实际可生成数量输出，禁止硬凑。\n\n")
	b.WriteString("参考资料开始：\n")
	b.WriteString(strings.TrimSpace(sourceText))
	b.WriteString("\n参考资料结束。")
	b.WriteString("\n\n再次确认：只输出 JSON 数组，题目必须来自上面的参考资料。")
	return b.String()
}

func generateQuestionsWithResponses(cfg AIConfig, prompt, sourceText string) ([]Question, error) {
	content := []map[string]any{{"type": "input_text", "text": buildMaterialPrompt(sourceText)}}
	body := map[string]any{"model": cfg.Model, "input": []map[string]any{{"role": "system", "content": []map[string]any{{"type": "input_text", "text": prompt}}}, {"role": "user", "content": content}}}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+cfg.APIPath, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return sendAIRequestAndParse(req, cfg.APIKey, data)
}

func generateQuestionsWithChatCompletions(cfg AIConfig, prompt, sourceText string) ([]Question, error) {
	userText := buildMaterialPrompt(sourceText)
	userText = buildChatCompletionUserPrompt(prompt, userText)
	body := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是题库生成器，只输出 JSON。"},
			{"role": "user", "content": userText},
		},
	}
	data, _ := json.Marshal(body)
	endpoint := strings.TrimRight(cfg.BaseURL, "/") + cfg.APIPath
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	questions, err := sendAIRequestAndParse(req, cfg.APIKey, data)
	if err == nil {
		return questions, nil
	}
	compatBody := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是题库生成器，只输出 JSON。"},
			{"role": "user", "content": userText},
		},
	}
	compatData, _ := json.Marshal(compatBody)
	compatReq, reqErr := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(compatData))
	if reqErr != nil {
		return nil, err
	}
	compatQuestions, compatErr := sendAIRequestAndParse(compatReq, cfg.APIKey, compatData)
	if compatErr == nil {
		return compatQuestions, nil
	}
	return nil, fmt.Errorf("标准请求失败: %v；兼容请求也失败: %v", err, compatErr)
}

func buildChatCompletionUserPrompt(formatPrompt, materialPrompt string) string {
	return materialPrompt + "\n\n输出格式要求如下。注意：下面只是格式要求，不是参考资料，不能围绕字段名或 JSON 结构出题。\n" + formatPrompt
}

func sendAIRequestAndParse(req *http.Request, apiKey string, data []byte) ([]Question, error) {
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		respData, fallbackErr := postAIWithFallback(req.URL.String(), apiKey, data)
		if fallbackErr != nil {
			return nil, fmt.Errorf("Go HTTP 请求 AI 服务失败: %w；备用请求也失败: %v", err, fallbackErr)
		}
		return parseAIResponseOrRepair(req.URL.String(), apiKey, data, respData)
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI 服务返回错误 %d: %s", resp.StatusCode, friendlyAIError(respData))
	}
	return parseAIResponseOrRepair(req.URL.String(), apiKey, data, respData)
}

func friendlyAIError(respData []byte) string {
	raw := strings.TrimSpace(string(respData))
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &payload); err == nil && payload.Error.Message != "" {
		message := payload.Error.Message
		if strings.Contains(strings.ToLower(message), "supported api model names") || strings.Contains(strings.ToLower(message), "model") && strings.Contains(strings.ToLower(message), "passed") {
			return message + "。请在 AI 配置里把模型切换为该服务支持的模型，例如 deepseek-v4-pro 或 deepseek-v4-flash，然后重新生成。"
		}
		return message
	}
	return raw
}

func postAIWithFallback(url, apiKey string, data []byte) ([]byte, error) {
	respData, psErr := postAIWithPowerShell(url, apiKey, data)
	if psErr == nil {
		return respData, nil
	}
	respData, curlErr := postAIWithCurl(url, apiKey, data)
	if curlErr == nil {
		return respData, nil
	}
	return nil, fmt.Errorf("PowerShell: %v；curl.exe: %v", psErr, curlErr)
}

func postAIWithPowerShell(url, apiKey string, data []byte) ([]byte, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("当前系统不支持 PowerShell fallback")
	}
	tmp, err := os.CreateTemp("", "meow-ai-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	_ = tmp.Close()
	script := `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; $OutputEncoding=[System.Text.Encoding]::UTF8; $ProgressPreference='SilentlyContinue'; $ErrorActionPreference='Stop'; $headers=@{Authorization='Bearer ' + $env:AI_API_KEY; 'Content-Type'='application/json'}; $body=Get-Content -LiteralPath $env:AI_BODY_FILE -Raw -Encoding UTF8; try { (Invoke-WebRequest -Uri $env:AI_POST_URL -Method POST -Headers $headers -Body $body -UseBasicParsing -TimeoutSec 180).Content } catch { $msg=$_.Exception.Message; if ($_.Exception.Response) { try { $reader=New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream(), [System.Text.Encoding]::UTF8); $bodyText=$reader.ReadToEnd(); if ($bodyText) { $msg=$bodyText } } catch {} }; [Console]::Error.WriteLine($msg); exit 1 }`
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Env = append(os.Environ(), "AI_API_KEY="+apiKey, "AI_POST_URL="+url, "AI_BODY_FILE="+tmp.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func postAIWithCurl(url, apiKey string, data []byte) ([]byte, error) {
	tmp, err := os.CreateTemp("", "meow-ai-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	_ = tmp.Close()
	cmd := exec.Command("curl.exe", "--silent", "--show-error", "--fail-with-body", "--connect-timeout", "30", "--max-time", "180", "-X", "POST", url, "-H", "Authorization: Bearer "+apiKey, "-H", "Content-Type: application/json", "--data-binary", "@"+tmp.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func parseAIResponseOrRepair(endpoint, apiKey string, requestData, respData []byte) ([]Question, error) {
	text := extractResponseText(respData)
	questions, parseErr := parseAIQuestions(text)
	if parseErr == nil && len(questions) > 0 {
		return questions, nil
	}
	repaired, repairErr := repairAIQuestionJSON(endpoint, apiKey, extractRequestModel(requestData), text)
	if repairErr != nil {
		return nil, fmt.Errorf("AI 返回题库格式无效: %v；自动修复也失败: %v", parseErr, repairErr)
	}
	questions, parseErr = parseAIQuestions(repaired)
	if parseErr != nil || len(questions) == 0 {
		return nil, fmt.Errorf("AI 返回题库格式无效，自动修复后仍不可用: %v", parseErr)
	}
	return questions, nil
}

func extractRequestModel(data []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err == nil {
		if model, ok := payload["model"].(string); ok && model != "" {
			return model
		}
	}
	return defaultAIModel
}

func repairAIQuestionJSON(endpoint, apiKey, model, brokenText string) (string, error) {
	repairPrompt := "你是 JSON 修复器。只输出修复后的 JSON 数组，不要解释，不要 Markdown。数组元素必须使用 question_number, question_type, question_text, options, correct_answer 字段。"
	var body map[string]any
	if strings.Contains(endpoint, "chat/completions") {
		body = map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": repairPrompt},
				{"role": "user", "content": brokenText},
			},
		}
	} else {
		body = map[string]any{"model": model, "input": repairPrompt + "\n\n待修复内容：\n" + brokenText}
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		respData, fallbackErr := postAIWithFallback(endpoint, apiKey, data)
		if fallbackErr != nil {
			return "", fmt.Errorf("Go 修复请求失败: %w；备用请求也失败: %v", err, fallbackErr)
		}
		return extractResponseText(respData), nil
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("修复请求返回 %d: %s", resp.StatusCode, string(respData))
	}
	return extractResponseText(respData), nil
}

func extractResponseText(data []byte) string {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return string(data)
	}
	if s, ok := raw["output_text"].(string); ok {
		return s
	}
	if choices, ok := raw["choices"].([]any); ok {
		for _, choice := range choices {
			choiceMap, ok := choice.(map[string]any)
			if !ok {
				continue
			}
			if message, ok := choiceMap["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
			if text, ok := choiceMap["text"].(string); ok {
				return text
			}
		}
	}
	var walk func(any) string
	walk = func(v any) string {
		switch x := v.(type) {
		case map[string]any:
			if t, ok := x["text"].(string); ok {
				return t
			}
			for _, vv := range x {
				if found := walk(vv); found != "" {
					return found
				}
			}
		case []any:
			for _, vv := range x {
				if found := walk(vv); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return walk(raw)
}

func parseAIQuestions(text string) ([]Question, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}
	questions, err := decodeAIQuestions([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("AI 返回内容不是合法题库 JSON: %w", err)
	}
	return normalizeQuestionList(questions)
}

func normalizeQuestionList(questions []Question) ([]Question, error) {
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
		if len(q.Options) < 2 || !answerMatchesOptions(q.CorrectAnswer, q.Options) {
			continue
		}
		if q.QuestionNumber == "" {
			q.QuestionNumber = fmt.Sprintf("AI-%03d", i+1)
		}
		q.GlobalCorrectCount = 0
		q.GlobalErrorCount = 0
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

func decodeAIQuestions(data []byte) ([]Question, error) {
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

func answerMatchesOptions(answer string, options map[string]string) bool {
	for _, r := range answer {
		if _, ok := options[string(r)]; !ok {
			return false
		}
	}
	return true
}
