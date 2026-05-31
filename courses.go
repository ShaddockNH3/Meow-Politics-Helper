package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func customQuestionBankDir(userID string) string {
	return filepath.Join(userDataBaseDir, userID, aiQuestionBanksDir)
}

func sanitizeCourseKey(value string) string {
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
		key = fmt.Sprintf("ai_%d", time.Now().Unix())
	}
	if !strings.HasPrefix(key, "custom_") {
		key = "custom_" + key
	}
	return key
}

func countQuestionsByChapter(chapters map[string][]Question) int {
	total := 0
	for _, questions := range chapters {
		total += len(questions)
	}
	return total
}

func registerCourse(cfg *CourseConfig) {
	coursesMu.Lock()
	defer coursesMu.Unlock()
	for key := range questionMapByID {
		if strings.HasPrefix(key, cfg.Key+"_") {
			delete(questionMapByID, key)
		}
	}
	coursesByKey[cfg.Key] = cfg
	for chapterKey, questions := range cfg.Questions {
		for idx := range questions {
			questions[idx].OriginalChapterKey = chapterKey
			questions[idx].OriginalIndex = idx
			questionMapByID[fmt.Sprintf("%s_%s_%d", cfg.Key, chapterKey, idx)] = questions[idx]
		}
	}
}

func loadCustomCoursesForUser(userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	dir := customQuestionBankDir(userID)
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
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("读取自定义题库失败 %s: %v", path, err)
			continue
		}
		var payload struct {
			Meta     AIQuestionBankMeta    `json:"meta"`
			Chapters map[string][]Question `json:"chapters"`
		}
		if err := json.Unmarshal(data, &payload); err != nil || payload.Meta.CourseKey == "" {
			log.Printf("解析自定义题库失败 %s: %v", path, err)
			continue
		}
		maxChapter := 0
		for key := range payload.Chapters {
			if n := atoiOrZero(key); n > maxChapter {
				maxChapter = n
			}
		}
		registerCourse(&CourseConfig{
			Key:           payload.Meta.CourseKey,
			Name:          payload.Meta.CourseName,
			MaxChapter:    maxChapter,
			ChapterNames:  payload.Meta.ChapterNames,
			Questions:     payload.Chapters,
			IncorrectFile: payload.Meta.CourseKey + "_incorrect_questions.json",
		})
	}
	return nil
}

func atoiOrZero(value string) int {
	var n int
	fmt.Sscanf(value, "%d", &n)
	return n
}

func listCourseInfos() []CourseInfo {
	coursesMu.RLock()
	defer coursesMu.RUnlock()
	infos := make([]CourseInfo, 0, len(coursesByKey))
	for _, cfg := range coursesByKey {
		infos = append(infos, CourseInfo{
			Key:           cfg.Key,
			Name:          cfg.Name,
			MaxChapter:    cfg.MaxChapter,
			ChapterNames:  chapterNamesForInfo(cfg),
			Custom:        strings.HasPrefix(cfg.Key, "custom_"),
			QuestionCount: countQuestionsByChapter(cfg.Questions),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Custom != infos[j].Custom {
			return !infos[i].Custom
		}
		return infos[i].Name < infos[j].Name
	})
	return infos
}

func chapterNamesForInfo(cfg *CourseConfig) []string {
	if len(cfg.ChapterNames) == 0 {
		return nil
	}
	names := make([]string, cfg.MaxChapter+1)
	for i := 0; i <= cfg.MaxChapter; i++ {
		key := fmt.Sprintf("%d", i)
		if name := strings.TrimSpace(cfg.ChapterNames[key]); name != "" {
			names[i] = name
		}
	}
	return names
}

func CoursesListHandler(ctx context.Context, c *app.RequestContext) {
	userID := string(c.Query("user_id"))
	if userID != "" {
		_ = loadCustomCoursesForUser(userID)
	}
	c.JSON(consts.StatusOK, utils.H{"courses": listCourseInfos()})
}

func DeleteCustomCourseHandler(ctx context.Context, c *app.RequestContext) {
	var req DeleteCustomCourseRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	if err := validateUserID(req.UserID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	if !strings.HasPrefix(req.Course, "custom_") {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "内置题库不能删除"})
		return
	}
	path := filepath.Join(customQuestionBankDir(req.UserID), filepath.Base(req.Course)+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "删除题库文件失败"})
		return
	}
	coursesMu.Lock()
	delete(coursesByKey, req.Course)
	for key := range questionMapByID {
		if strings.HasPrefix(key, req.Course+"_") {
			delete(questionMapByID, key)
		}
	}
	coursesMu.Unlock()
	c.JSON(consts.StatusOK, utils.H{"message": "题库已删除"})
}

func customCourseFilePath(userID, course string) string {
	return filepath.Join(customQuestionBankDir(userID), filepath.Base(course)+".json")
}

func loadCustomQuestionBankPayload(userID, course string) (AIQuestionBankMeta, map[string][]Question, error) {
	var payload struct {
		Meta     AIQuestionBankMeta    `json:"meta"`
		Chapters map[string][]Question `json:"chapters"`
	}
	data, err := os.ReadFile(customCourseFilePath(userID, course))
	if err != nil {
		return payload.Meta, nil, err
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload.Meta, nil, err
	}
	return payload.Meta, payload.Chapters, nil
}

func nextChapterKey(chapters map[string][]Question) string {
	max := -1
	for key := range chapters {
		if n := atoiOrZero(key); n > max {
			max = n
		}
	}
	return fmt.Sprintf("%d", max+1)
}

func maxChapterKey(chapters map[string][]Question) int {
	max := 0
	for key := range chapters {
		if n := atoiOrZero(key); n > max {
			max = n
		}
	}
	return max
}

func compactChapters(chapters map[string][]Question, chapterNames map[string]string) (map[string][]Question, map[string]string) {
	keys := make([]string, 0, len(chapters))
	for key := range chapters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return atoiOrZero(keys[i]) < atoiOrZero(keys[j]) })
	compacted := make(map[string][]Question, len(chapters))
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

func pruneIncorrectQuestionsForCourse(userID, course string, chapters map[string][]Question) error {
	incorrectFileName := getIncorrectQuestionsFileName(course)
	path := getUserDataPath(userID, incorrectFileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	userIncorrect := []UserIncorrectQuestion{}
	if err := loadUserJSONData(userID, incorrectFileName, &userIncorrect); err != nil {
		return err
	}
	if len(userIncorrect) == 0 {
		return nil
	}
	existing := make(map[string]bool)
	for chapterKey, questions := range chapters {
		for _, q := range questions {
			existing[chapterKey+"\x00"+q.QuestionNumber] = true
		}
	}
	filtered := make([]UserIncorrectQuestion, 0, len(userIncorrect))
	for _, iq := range userIncorrect {
		if existing[iq.OriginalChapter+"\x00"+iq.QuestionNumber] {
			filtered = append(filtered, iq)
		}
	}
	if len(filtered) == len(userIncorrect) {
		return nil
	}
	return saveUserJSONData(userID, incorrectFileName, filtered)
}

func normalizeChapterKey(value string, chapters map[string][]Question) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "auto") {
		return nextChapterKey(chapters)
	}
	return fmt.Sprintf("%d", atoiOrZero(value))
}

func saveQuestionsToCustomBank(userID, targetMode, targetCourse, courseKey, courseName, chapterKey, chapterName string, questions []Question) (AIQuestionBankMeta, map[string][]Question, string, error) {
	if err := validateUserID(userID); err != nil {
		return AIQuestionBankMeta{}, nil, "", err
	}
	questions, err := normalizeQuestionList(questions)
	if err != nil {
		return AIQuestionBankMeta{}, nil, "", err
	}
	var meta AIQuestionBankMeta
	chapters := map[string][]Question{}
	now := time.Now()
	if targetMode == "existing" {
		if !strings.HasPrefix(targetCourse, "custom_") {
			return meta, nil, "", fmt.Errorf("请选择一个已有自定义题库")
		}
		loadedMeta, loadedChapters, err := loadCustomQuestionBankPayload(userID, targetCourse)
		if err != nil {
			return meta, nil, "", fmt.Errorf("读取已有题库失败: %w", err)
		}
		meta = loadedMeta
		if meta.ChapterNames == nil {
			meta.ChapterNames = map[string]string{}
		}
		for key, value := range loadedChapters {
			chapters[key] = value
		}
		chapterKey = normalizeChapterKey(chapterKey, chapters)
		if _, exists := chapters[chapterKey]; exists {
			return meta, nil, "", fmt.Errorf("章节 %s 已存在，请换一个章节号或选择自动新章节", chapterKey)
		}
	} else {
		courseName = strings.TrimSpace(courseName)
		if courseName == "" {
			courseName = "自定义题库"
		}
		if strings.TrimSpace(courseKey) == "" {
			courseKey = sanitizeCourseKey(courseName)
		} else {
			courseKey = sanitizeCourseKey(courseKey)
		}
		meta = AIQuestionBankMeta{CourseKey: courseKey, CourseName: courseName, CreatedAt: now, ChapterNames: map[string]string{}}
		chapterKey = normalizeChapterKey(chapterKey, chapters)
	}
	if chapterName = strings.TrimSpace(chapterName); chapterName == "" {
		chapterName = "章节 " + chapterKey
	}
	chapters[chapterKey] = questions
	meta.ChapterNames[chapterKey] = chapterName
	meta.ChapterCount = len(chapters)
	meta.QuestionCount = countQuestionsByChapter(chapters)
	meta.UpdatedAt = now
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	if err := saveCustomQuestionBank(userID, meta, chapters); err != nil {
		return meta, nil, "", err
	}
	registerCourse(&CourseConfig{Key: meta.CourseKey, Name: meta.CourseName, MaxChapter: maxChapterKey(chapters), ChapterNames: meta.ChapterNames, Questions: chapters, IncorrectFile: meta.CourseKey + "_incorrect_questions.json"})
	return meta, chapters, chapterKey, nil
}

func CustomCourseDetailHandler(ctx context.Context, c *app.RequestContext) {
	userID := string(c.Query("user_id"))
	course := string(c.Query("course"))
	if err := validateUserID(userID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	if !strings.HasPrefix(course, "custom_") {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "只能编辑自定义题库"})
		return
	}
	meta, chapters, err := loadCustomQuestionBankPayload(userID, course)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{"error": "题库不存在或无法读取"})
		return
	}
	c.JSON(consts.StatusOK, utils.H{"meta": meta, "chapters": chapters})
}

func SaveCustomCourseHandler(ctx context.Context, c *app.RequestContext) {
	var req SaveCustomCourseRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	if err := validateUserID(req.UserID); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	if !strings.HasPrefix(req.Course, "custom_") {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "内置题库不能编辑"})
		return
	}
	chapters := req.Chapters
	if len(chapters) == 0 && len(req.Questions) > 0 {
		chapters = map[string][]Question{"0": req.Questions}
	}
	if len(chapters) == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "题库至少需要保留一个章节"})
		return
	}
	cleanedChapters := make(map[string][]Question, len(chapters))
	for key, questions := range chapters {
		cleaned, err := normalizeQuestionList(questions)
		if err != nil {
			c.JSON(consts.StatusBadRequest, utils.H{"error": fmt.Sprintf("章节 %s 不可用: %v", key, err)})
			return
		}
		cleanedChapters[fmt.Sprintf("%d", atoiOrZero(key))] = cleaned
	}
	meta, _, err := loadCustomQuestionBankPayload(req.UserID, req.Course)
	if err != nil {
		meta = AIQuestionBankMeta{CourseKey: req.Course, CreatedAt: time.Now()}
	}
	meta.CourseName = req.CourseName
	meta.ChapterNames = req.ChapterNames
	if meta.ChapterNames == nil {
		meta.ChapterNames = map[string]string{}
	}
	cleanedChapters, meta.ChapterNames = compactChapters(cleanedChapters, meta.ChapterNames)
	for key := range cleanedChapters {
		if strings.TrimSpace(meta.ChapterNames[key]) == "" {
			meta.ChapterNames[key] = "章节 " + key
		}
	}
	meta.ChapterCount = len(cleanedChapters)
	meta.QuestionCount = countQuestionsByChapter(cleanedChapters)
	meta.UpdatedAt = time.Now()
	if err := saveCustomQuestionBank(req.UserID, meta, cleanedChapters); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存题库失败"})
		return
	}
	if err := pruneIncorrectQuestionsForCourse(req.UserID, req.Course, cleanedChapters); err != nil {
		log.Printf("警告: 保存自定义题库后同步清理错题本失败，用户=%s 题库=%s: %v", req.UserID, req.Course, err)
	}
	registerCourse(&CourseConfig{Key: req.Course, Name: req.CourseName, MaxChapter: maxChapterKey(cleanedChapters), ChapterNames: meta.ChapterNames, Questions: cleanedChapters, IncorrectFile: req.Course + "_incorrect_questions.json"})
	c.JSON(consts.StatusOK, utils.H{"message": "题库已保存", "questions": meta.QuestionCount})
}
