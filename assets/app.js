        const { createApp, ref, computed, onMounted, onBeforeUnmount, watch } = Vue;

        createApp({
            setup() {
                const isLoading = ref(false);
                const errorMessage = ref(''); 
                const userIdError = ref(''); 
                const currentView = ref('mainMenu'); 
                const viewTitle = ref('喵喵学习小助手');
                
                const userId = ref(localStorage.getItem('quizAppUserId') || '');
                const inputUserId = ref(''); 

                const API_BASE_URL = ''; 
                const networkInfo = ref(null);
                const defaultShortcuts = {
                    options: { A: 'A', B: 'B', C: 'C', D: 'D', E: 'E' },
                    submit: 'Enter',
                    previous: 'ArrowLeft',
                    next: 'ArrowRight',
                    back: 'Escape'
                };
                const normalizeShortcutKey = (value) => {
                    const text = String(value || '').trim();
                    if (!text) return '';
                    const aliases = {
                        esc: 'Escape', escape: 'Escape', enter: 'Enter', 回车: 'Enter',
                        left: 'ArrowLeft', arrowleft: 'ArrowLeft',  左: 'ArrowLeft',
                        right: 'ArrowRight', arrowright: 'ArrowRight', 右: 'ArrowRight'
                    };
                    const lower = text.toLowerCase();
                    if (aliases[lower]) return aliases[lower];
                    return text.length === 1 ? text.toUpperCase() : text;
                };
                const cloneDefaultShortcuts = () => JSON.parse(JSON.stringify(defaultShortcuts));
                const shortcutStorageKey = (id = userId.value) => id ? `quizAppShortcuts:${id}` : 'quizAppShortcuts:anonymous';
                const loadShortcutSettings = (id = userId.value) => {
                    try {
                        const key = shortcutStorageKey(id);
                        let saved = JSON.parse(localStorage.getItem(key) || 'null');
                        const legacySavedText = localStorage.getItem('quizAppShortcuts');
                        if (!saved && id && legacySavedText) {
                            saved = JSON.parse(legacySavedText || 'null');
                            if (saved) localStorage.setItem(key, JSON.stringify(saved));
                            localStorage.removeItem('quizAppShortcuts');
                        }
                        return {
                            options: { ...defaultShortcuts.options, ...(saved?.options || {}) },
                            submit: saved?.submit || defaultShortcuts.submit,
                            previous: saved?.previous || defaultShortcuts.previous,
                            next: saved?.next || defaultShortcuts.next,
                            back: saved?.back || defaultShortcuts.back
                        };
                    } catch (e) {
                        return cloneDefaultShortcuts();
                    }
                };
                const shortcutSettings = ref(loadShortcutSettings());
                const shortcutDraft = ref(JSON.parse(JSON.stringify(shortcutSettings.value)));
                const shortcutCapture = ref(null);
                let shortcutCaptureEscArmed = false;
                const refreshShortcutSettings = () => {
                    shortcutSettings.value = loadShortcutSettings();
                    shortcutDraft.value = JSON.parse(JSON.stringify(shortcutSettings.value));
                    shortcutCapture.value = null;
                    shortcutCaptureEscArmed = false;
                };
                const courses = ref([]);
                const selectedCourse = ref('maogai'); // 默认选择毛概
                const availableChapters = computed(() => {
                    const course = courses.value.find(item => item.key === selectedCourse.value);
                    if (course?.custom) {
                        const chapters = Array.from({ length: (course.max_chapter || 0) + 1 }, (_, i) => ({ value: i.toString(), text: course.chapter_names?.[i] || `章节 ${i}` }));
                        chapters.push({ value: 'all', text: '全部章节' });
                        return chapters;
                    }
                    // 支持三种 course key: 'maogai', 'xigai_li'(李老师 单章), 'xigai_yang'(杨老师 多章)
                    if (selectedCourse.value === 'xigai_li') {
                        return [{ value: '0', text: '期末复习' }];
                    } else if (selectedCourse.value === 'xigai_yang') {
                        // 杨老师题库：章节 0..17
                        const chapters = [...Array.from({ length: 18 }, (_, i) => ({ value: i.toString(), text: `章节 ${i}` }))];
                        chapters.push({ value: 'all', text: '全部章节' });
                        return chapters;
                    } else {
                        return [...Array.from({ length: 9 }, (_, i) => ({ value: i.toString(), text: `章节 ${i}` })), { value: 'all', text: '全部章节' }];
                    }
                });
                const selectedChapters = ref([]); 
                const selectedOrder = ref('sequential'); 
                const activeMode = ref(''); 
                const modeDisplayName = ref('');
                
                const allModeQuestions = ref([]); 
                const currentQuestion = ref(null); 
                const currentQuestionIndex = ref(-1); 
                const totalQuestions = ref(0);
                const originalTotalQuestions = ref(0);
                const isQuizCompleted = ref(false);

                const selectedAnswers = ref([]); 
                const quizModeState = ref('inProgress'); 
                const feedbackMessage = ref('');
                const isCurrentAnswerCorrect = ref(false);
                const quizResults = ref({ total_answered: 0, total_correct: 0 });
                const aiStatus = ref({ configured: false, base_url: 'https://api.openai.com/v1', api_path: '/chat/completions', model: 'gpt-4.1-mini' });
                const aiConfigForm = ref({ api_key: '', base_url: 'https://api.openai.com/v1', api_path: '/chat/completions', model: 'gpt-4.1-mini' });
                const aiModels = ref([{ id: 'gpt-4.1-mini' }]);
                const aiModelMessage = ref('');
                const defaultAISystemPrompt = `你是一个严谨的题库生成器。请从用户提供的材料中生成适合刷题的选择题。

重要边界：
- “参考资料”才是出题依据。
- 本提示词里的字段名、JSON 示例、输出要求都不是参考资料，绝对不能围绕 question_number、question_type、options、correct_answer、correct_count、error_count 等格式字段出题。
- 如果参考资料是一篇文章、课文、讲义或摘录，题目必须考察该资料的内容本身，而不是考察 JSON 格式。

输出要求：
1. 只输出 JSON，不要 Markdown，不要解释，不要代码块。
2. 顶层必须是一个 JSON 数组。
3. 每个题目对象必须严格包含 question_number、question_type、question_text、options、correct_answer、correct_count、error_count。
4. question_type 只能是 "单选题" 或 "多选题"。
5. options 是对象，键只能使用 A、B、C、D、E，值为非空选项文本。
6. correct_answer 是字符串：单选如 "A"，多选如 "AB"，必须只包含 options 中存在的键。
7. correct_count 和 error_count 固定为 0。
8. 不要编造材料中没有依据的事实。材料不足时少出题，不要硬凑。
9. 题目不要重复，选项不要重复。
10. 默认生成 20-30 题；如果用户指定题量，以用户要求为准。

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
    "correct_answer": "A",
    "correct_count": 0,
    "error_count": 0
  }
]`;
                const aiImportForm = ref({ course_key: '', course_name: 'AI 生成题库', chapter_name: 'AI 导入', system_prompt: defaultAISystemPrompt, source_text: '' });
                const importTargetDefaults = () => ({ target_mode: 'new', target_course: '', course_key: '', course_name: '自定义题库', chapter_key: '', chapter_name: '新章节' });
                const jsonPromptCopied = ref(false);
                const jsonImportForm = ref({ ...importTargetDefaults(), json_text: '' });
                aiImportForm.value = { ...aiImportForm.value, target_mode: 'new', target_course: '', chapter_key: '' };
                const aiImportMessage = ref('');
                const aiImportTask = ref(null);
                const jsonImportMessage = ref('');
                let savingAIConfig = false;
                const customCourses = computed(() => courses.value.filter(course => course.custom));
                const customEditor = ref({ course: '', meta: null, chapters: {}, selectedChapter: '', message: '' });
                const customEditorChapterKeys = computed(() => Object.keys(customEditor.value.chapters || {}).sort((a, b) => Number(a) - Number(b)));
                const customEditorQuestions = computed(() => customEditor.value.chapters?.[customEditor.value.selectedChapter] || []);
                
                const showJumpInput = ref(false);
                const jumpToQuestionNumberInput = ref(null);

                // New state for continuing last quiz
                const savedQuizState = ref(null);
                const canContinueQuiz = computed(() => savedQuizState.value && savedQuizState.value.questions && savedQuizState.value.questions.length > 0);

                const isInQuestionView = computed(() => {
                    return ['quickReview', 'quizMode', 'incorrectReview'].includes(currentView.value);
                });

                const showNextButton = computed(() => {
                    if (!isInQuestionView.value || isQuizCompleted.value) return false;
                    if (currentView.value === 'quickReview' && currentQuestionIndex.value < totalQuestions.value - 1) return true;
                    if ((currentView.value === 'quizMode' || currentView.value === 'incorrectReview') && quizModeState.value === 'showAnswer') return true;
                    return false;
                });

                const getSavedQuizKey = () => `quizAppLastQuiz_${userId.value}`;

                const loadSavedQuizState = () => {
                    if (!userId.value) return;
                    const savedData = localStorage.getItem(getSavedQuizKey());
                    if (savedData) {
                        savedQuizState.value = JSON.parse(savedData);
                    } else {
                        savedQuizState.value = null;
                    }
                };

                const saveCurrentQuizState = () => {
                    if (!userId.value || activeMode.value !== 'quizMode') return;
                    const stateToSave = {
                        questions: allModeQuestions.value,
                        currentIndex: currentQuestionIndex.value,
                        results: quizResults.value,
                        isCompleted: isQuizCompleted.value,
                        originalTotalQuestions: originalTotalQuestions.value
                    };
                    localStorage.setItem(getSavedQuizKey(), JSON.stringify(stateToSave));
                    savedQuizState.value = stateToSave;
                };

                const loadNetworkInfo = async () => {
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/network/info`);
                        if (response.ok) {
                            networkInfo.value = await response.json();
                        }
                    } catch (err) {
                        console.warn('网络地址检测失败:', err);
                    }
                };

                const loadCourses = async () => {
                    try {
                        const qs = userId.value ? `?user_id=${encodeURIComponent(userId.value)}` : '';
                        const response = await fetch(`${API_BASE_URL}/api/courses${qs}`);
                        if (response.ok) {
                            const data = await response.json();
                            courses.value = data.courses || [];
                            if (courses.value.length > 0 && !courses.value.some(item => item.key === selectedCourse.value)) {
                                selectedCourse.value = courses.value[0].key;
                            }
                        }
                    } catch (err) {
                        console.warn('课程列表加载失败:', err);
                    }
                };

                const loadAIStatus = async () => {
                    if (!userId.value) return;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/ai/config?user_id=${encodeURIComponent(userId.value)}`);
                        if (response.ok) {
                            aiStatus.value = await response.json();
                            aiConfigForm.value.base_url = aiStatus.value.base_url || aiConfigForm.value.base_url;
                            aiConfigForm.value.api_path = aiStatus.value.api_path || aiConfigForm.value.api_path;
                            aiConfigForm.value.model = aiStatus.value.model || aiConfigForm.value.model;
                            await loadAIModels();
                        }
                    } catch (err) {
                        console.warn('AI 配置状态加载失败:', err);
                    }
                };

                const loadAIModels = async () => {
                    if (!userId.value) return;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/ai/models?user_id=${encodeURIComponent(userId.value)}`);
                            const data = await response.json();
                            if (response.ok && Array.isArray(data.models) && data.models.length > 0) {
                                aiModels.value = data.models;
                                aiModelMessage.value = data.message || '';
                                if (!aiModels.value.some(item => item.id === aiConfigForm.value.model)) {
                                    aiConfigForm.value.model = aiModels.value[0].id;
                                    aiModelMessage.value = (aiModelMessage.value ? aiModelMessage.value + '；' : '') + `已自动切换到 ${aiConfigForm.value.model}`;
                                    await syncAIConfig({ quiet: true });
                                }
                            }
                    } catch (err) {
                        aiModelMessage.value = '模型列表获取失败，保留默认模型';
                    }
                };

                const saveAIConfig = async () => {
                    if (!userId.value) return;
                    savingAIConfig = true;
                    isLoading.value = true;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/ai/config`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: userId.value, ...aiConfigForm.value })
                        });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || 'AI 配置保存失败');
                        aiImportMessage.value = data.message || 'AI 配置已保存';
                        aiConfigForm.value.api_key = '';
                        await loadAIStatus();
                    } catch (err) {
                        errorMessage.value = err.message;
                    } finally {
                        isLoading.value = false;
                        savingAIConfig = false;
                    }
                };

                const syncAIConfig = async ({ quiet = false } = {}) => {
                    if (!userId.value || savingAIConfig) return false;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/ai/config`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: userId.value, ...aiConfigForm.value })
                        });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || 'AI 配置保存失败');
                        aiStatus.value = { ...aiStatus.value, base_url: aiConfigForm.value.base_url, api_path: aiConfigForm.value.api_path, model: aiConfigForm.value.model, configured: true, key_preview: data.key_preview || aiStatus.value.key_preview };
                        if (!quiet) aiImportMessage.value = data.message || 'AI 配置已保存';
                        aiConfigForm.value.api_key = '';
                        return true;
                    } catch (err) {
                        if (!quiet) errorMessage.value = err.message;
                        return false;
                    }
                };

                const appendImportTarget = (formData, form) => {
                    formData.append('target_mode', form.target_mode || 'new');
                    formData.append('target_course', form.target_course || '');
                    formData.append('course_key', form.course_key || form.course_name || '自定义题库');
                    formData.append('course_name', form.course_name || '自定义题库');
                    formData.append('chapter_key', form.chapter_key || '');
                    formData.append('chapter_name', form.chapter_name || '新章节');
                };

                const normalizeImportTarget = (form) => {
                    if (form.target_mode === 'existing' && !form.target_course) {
                        const first = customCourses.value[0];
                        if (first) form.target_course = first.key;
                    }
                };

                const importAIQuestionBank = async () => {
                    if (!userId.value) return;
                    aiImportMessage.value = '';
                    aiImportTask.value = null;
                    try {
                        const synced = await syncAIConfig({ quiet: true });
                        if (!synced) throw new Error('AI 配置同步失败，请先保存配置');
                        const formData = new FormData();
                        formData.append('user_id', userId.value);
                        normalizeImportTarget(aiImportForm.value);
                        appendImportTarget(formData, aiImportForm.value);
                        formData.append('system_prompt', aiImportForm.value.system_prompt || defaultAISystemPrompt);
                        formData.append('source_text', aiImportForm.value.source_text);
                        const response = await fetch(`${API_BASE_URL}/api/ai/import`, { method: 'POST', body: formData });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || 'AI 导入失败');
                        aiImportMessage.value = data.message || '题库生成已在后台开始';
                        aiImportTask.value = { id: data.task_id, status: 'running', message: aiImportMessage.value };
                        pollAIImportTask(data.task_id);
                    } catch (err) {
                        errorMessage.value = err.message === 'Failed to fetch'
                            ? '无法连接本地导入接口。请确认当前页面来自正在运行的新版 exe，并检查服务是否仍在运行。'
                            : err.message;
                    }
                };

                const importJSONQuestionBank = async () => {
                    if (!userId.value) return;
                    jsonImportMessage.value = '';
                    try {
                        normalizeImportTarget(jsonImportForm.value);
                        const response = await fetch(`${API_BASE_URL}/api/courses/import-json`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: userId.value, ...jsonImportForm.value })
                        });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || 'JSON 导入失败');
                        jsonImportMessage.value = `${data.message || '导入完成'}，章节 ${data.chapter_key}，共 ${data.questions || 0} 题`;
                        await loadCourses();
                        selectedCourse.value = data.course?.course_key || selectedCourse.value;
                    } catch (err) {
                        errorMessage.value = err.message;
                    }
                };

                const copyJSONPrompt = async () => {
                    await copyText(defaultAISystemPrompt);
                    jsonPromptCopied.value = true;
                    setTimeout(() => { jsonPromptCopied.value = false; }, 1800);
                };

                const pollAIImportTask = async (taskId) => {
                    if (!taskId) return;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/ai/import/status?user_id=${encodeURIComponent(userId.value)}&task_id=${encodeURIComponent(taskId)}`);
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || '任务状态获取失败');
                        aiImportTask.value = data;
                        aiImportMessage.value = data.message || aiImportMessage.value;
                        if (data.status === 'completed') {
                            await loadCourses();
                            selectedCourse.value = data.course?.course_key || selectedCourse.value;
                            return;
                        }
                        if (data.status === 'failed') {
                            errorMessage.value = data.error || '题库生成失败';
                            return;
                        }
                        setTimeout(() => pollAIImportTask(taskId), 2000);
                    } catch (err) {
                        errorMessage.value = err.message;
                    }
                };

                const deleteCustomCourse = async (course) => {
                    if (!course?.custom || !userId.value) return;
                    if (!confirm(`确定删除自定义题库「${course.name}」吗？内置题库不会受影响。`)) return;
                    isLoading.value = true;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/courses/delete`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: userId.value, course: course.key })
                        });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || '删除题库失败');
                        await loadCourses();
                        if (selectedCourse.value === course.key) selectedCourse.value = 'maogai';
                        aiImportMessage.value = data.message || '题库已删除';
                    } catch (err) {
                        errorMessage.value = err.message;
                    } finally {
                        isLoading.value = false;
                    }
                };

                const openCustomCourseEditor = async (course) => {
                    if (!course?.custom || !userId.value) return;
                    currentView.value = 'customCourseEditor';
                    customEditor.value = { course: course.key, meta: null, chapters: {}, selectedChapter: '', message: '' };
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/courses/custom/detail?user_id=${encodeURIComponent(userId.value)}&course=${encodeURIComponent(course.key)}`);
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || '读取题库失败');
                        data.meta.chapter_names = data.meta.chapter_names || {};
                        customEditor.value = { course: course.key, meta: data.meta, chapters: data.chapters || {}, selectedChapter: '', message: '' };
                        const first = customEditorChapterKeys.value[0];
                        customEditor.value.selectedChapter = first || '0';
                    } catch (err) {
                        errorMessage.value = err.message;
                    }
                };

                const makeBlankQuestion = () => ({
                    question_number: `编辑-${customEditorQuestions.value.length + 1}`,
                    question_type: '单选题',
                    question_text: '',
                    options: { A: '', B: '', C: '', D: '' },
                    correct_answer: 'A',
                    correct_count: 0,
                    error_count: 0
                });

                const addEditorQuestion = () => {
                    const chapter = customEditor.value.selectedChapter || '0';
                    if (!customEditor.value.chapters[chapter]) customEditor.value.chapters[chapter] = [];
                    customEditor.value.chapters[chapter].push(makeBlankQuestion());
                };

                const removeEditorQuestion = (index) => {
                    customEditor.value.chapters[customEditor.value.selectedChapter].splice(index, 1);
                };

                const addEditorChapter = () => {
                    const next = customEditorChapterKeys.value.reduce((max, key) => Math.max(max, Number(key) || 0), -1) + 1;
                    const key = String(next);
                    customEditor.value.chapters[key] = [makeBlankQuestion()];
                    customEditor.value.meta.chapter_names[key] = `章节 ${key}`;
                    customEditor.value.selectedChapter = key;
                };

                const deleteEditorChapter = () => {
                    const key = customEditor.value.selectedChapter;
                    if (customEditorChapterKeys.value.length <= 1) {
                        errorMessage.value = '题库至少需要保留一个章节';
                        return;
                    }
                    if (!confirm(`确定删除章节 ${key} 吗？`)) return;
                    delete customEditor.value.chapters[key];
                    delete customEditor.value.meta.chapter_names[key];
                    customEditor.value.selectedChapter = customEditorChapterKeys.value[0] || '';
                };

                const saveCustomCourse = async () => {
                    if (!customEditor.value.meta) return;
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/courses/custom/save`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                user_id: userId.value,
                                course: customEditor.value.course,
                                course_name: customEditor.value.meta.course_name,
                                chapter_names: customEditor.value.meta.chapter_names || {},
                                chapters: customEditor.value.chapters
                            })
                        });
                        const data = await response.json();
                        if (!response.ok) throw new Error(data.error || '保存题库失败');
                        customEditor.value.message = data.message || '题库已保存';
                        await loadCourses();
                    } catch (err) {
                        errorMessage.value = err.message;
                    }
                };

                const copyText = async (text) => {
                    if (!text) return;
                    try {
                        await navigator.clipboard.writeText(text);
                        alert('已复制：' + text);
                    } catch (err) {
                        prompt('复制失败，可以手动复制：', text);
                    }
                };

                const saveShortcutSettings = () => {
                    const normalized = {
                        options: {},
                        submit: normalizeShortcutKey(shortcutDraft.value.submit) || defaultShortcuts.submit,
                        previous: normalizeShortcutKey(shortcutDraft.value.previous) || defaultShortcuts.previous,
                        next: normalizeShortcutKey(shortcutDraft.value.next) || defaultShortcuts.next,
                        back: normalizeShortcutKey(shortcutDraft.value.back) || defaultShortcuts.back
                    };
                    for (const optionKey of ['A', 'B', 'C', 'D', 'E']) {
                        normalized.options[optionKey] = normalizeShortcutKey(shortcutDraft.value.options[optionKey]) || optionKey;
                    }
                    shortcutSettings.value = normalized;
                    shortcutDraft.value = JSON.parse(JSON.stringify(normalized));
                    localStorage.setItem(shortcutStorageKey(), JSON.stringify(normalized));
                    localStorage.removeItem('quizAppShortcuts');
                    alert('快捷键已保存。');
                };

                const resetShortcutSettings = () => {
                    shortcutSettings.value = cloneDefaultShortcuts();
                    shortcutDraft.value = cloneDefaultShortcuts();
                    shortcutCapture.value = null;
                    localStorage.removeItem(shortcutStorageKey());
                    localStorage.removeItem('quizAppShortcuts');
                };

                const startShortcutCapture = (type, key) => {
                    shortcutCapture.value = { type, key };
                    shortcutCaptureEscArmed = false;
                    setTimeout(() => { shortcutCaptureEscArmed = true; }, 0);
                };

                const isCapturingShortcut = (type, key) => {
                    return shortcutCapture.value?.type === type && shortcutCapture.value?.key === key;
                };

                const applyShortcutCapture = (key) => {
                    if (!shortcutCapture.value) return false;
                    const normalizedKey = normalizeShortcutKey(key);
                    if (!normalizedKey) return true;
                    if (shortcutCapture.value.type === 'option') {
                        shortcutDraft.value.options[shortcutCapture.value.key] = normalizedKey;
                    } else {
                        shortcutDraft.value[shortcutCapture.value.key] = normalizedKey;
                    }
                    shortcutCapture.value = null;
                    shortcutCaptureEscArmed = false;
                    saveShortcutSettings();
                    return true;
                };

                const initializeUserSession = async (idToInit) => {
                    if (!idToInit) {
                        userId.value = ''; 
                        return; 
                    }
                    isLoading.value = true;
                    errorMessage.value = ''; 
                    userIdError.value = ''; 
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/session/init`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: idToInit })
                        });
                        if (!response.ok) {
                            let errorDetailMessage = `HTTP 错误: ${response.status} ${response.statusText}. `;
                            try { const errorBodyText = await response.text(); errorDetailMessage += `服务器详情: "${errorBodyText || '(无响应体)'}"`; } catch (e) { errorDetailMessage += "无法读取服务器错误响应。"; }
                            throw new Error(`用户会话初始化失败。${errorDetailMessage}`);
                        }
                        const data = await response.json();
                        if (data.user_id) {
                            userId.value = data.user_id;
                            localStorage.setItem('quizAppUserId', userId.value);
                            refreshShortcutSettings();
                            loadCourses();
                            loadAIStatus();
                            loadSavedQuizState(); // Load quiz state for this user
                            inputUserId.value = ''; 
                            console.log(data.is_new_user ? `新用户 '${userId.value}' 会话已创建。` : `用户 '${userId.value}' 会话已建立。`);
                        } else {
                            throw new Error("服务器响应中未找到 user_id。");
                        }
                    } catch (err) {
                        if (!userId.value && inputUserId.value) { 
                           userIdError.value = err.message || "初始化用户会话时发生未知网络错误。请检查您的用户ID或网络连接。";
                        } else { 
                           errorMessage.value = err.message || "初始化用户会话时发生未知网络错误。";
                        }
                        userId.value = ''; 
                        localStorage.removeItem('quizAppUserId'); 
                        refreshShortcutSettings();
                    } finally {
                        isLoading.value = false;
                    }
                };
                
                const submitUserId = async () => {
                    userIdError.value = ''; 
                    if (!inputUserId.value.trim()) {
                        userIdError.value = "用户ID不能为空，请输入一个有效的ID。";
                        return;
                    }
                    if (inputUserId.value.trim().length < 3 || inputUserId.value.trim().length > 50) {
                        userIdError.value = "用户ID长度应在3到50个字符之间。";
                        return;
                    }
                    if (!/^[a-zA-Z0-9_-]+$/.test(inputUserId.value.trim())) {
                        userIdError.value = "用户ID只能包含字母、数字、下划线(_)和连字符(-)。";
                        return;
                    }
                    await initializeUserSession(inputUserId.value.trim());
                };

                const isTypingTarget = (target) => {
                    if (!target) return false;
                    const tagName = target.tagName ? target.tagName.toLowerCase() : '';
                    return tagName === 'input' || tagName === 'textarea' || tagName === 'select' || target.isContentEditable;
                };

                const toggleAnswerByKey = (optionKey) => {
                    if (!currentQuestion.value || quizModeState.value !== 'inProgress') return;
                    if (!currentQuestion.value.options || !Object.prototype.hasOwnProperty.call(currentQuestion.value.options, optionKey)) return;

                    if (currentQuestion.value.question_type === '多选题') {
                        const answers = Array.isArray(selectedAnswers.value) ? [...selectedAnswers.value] : [];
                        const index = answers.indexOf(optionKey);
                        if (index >= 0) {
                            answers.splice(index, 1);
                        } else {
                            answers.push(optionKey);
                        }
                        selectedAnswers.value = answers.sort();
                    } else {
                        selectedAnswers.value = optionKey;
                    }
                };

                const handleKeyboardShortcut = (event) => {
                    if (shortcutCapture.value) {
                        event.preventDefault();
                        if (event.key === 'Escape' && !shortcutCaptureEscArmed) {
                            shortcutCapture.value = null;
                            shortcutCaptureEscArmed = false;
                            return;
                        }
                        applyShortcutCapture(event.key);
                        return;
                    }
                    if (isTypingTarget(event.target) || event.ctrlKey || event.metaKey || event.altKey) return;

                    const key = event.key;
                    const shortcuts = shortcutSettings.value;
                    if (key === shortcuts.back) {
                        event.preventDefault();
                        goBackOneView();
                        return;
                    }
                    if (!isInQuestionView.value) return;

                    const optionEntry = Object.entries(shortcuts.options).find(([, shortcutKey]) => key.toUpperCase() === String(shortcutKey).toUpperCase());
                    if (optionEntry) {
                        event.preventDefault();
                        if (currentView.value === 'quizMode' || currentView.value === 'incorrectReview') {
                            toggleAnswerByKey(optionEntry[0]);
                        }
                        return;
                    }

                    if (key === shortcuts.submit) {
                        event.preventDefault();
                        if (currentView.value === 'quickReview' || quizModeState.value === 'showAnswer') {
                            fetchNextQuestion();
                        } else if (currentView.value === 'quizMode' || currentView.value === 'incorrectReview') {
                            submitAnswerForMode();
                        }
                        return;
                    }

                    if (key === shortcuts.previous) {
                        event.preventDefault();
                        previousQuestion();
                        return;
                    }

                    if (key === shortcuts.next) {
                        event.preventDefault();
                        fetchNextQuestion();
                    }
                };

                onMounted(() => {
                    loadNetworkInfo();
                    loadCourses();
                    const storedUserId = localStorage.getItem('quizAppUserId');
                    if (storedUserId) {
                        userId.value = storedUserId; 
                        initializeUserSession(storedUserId); 
                    }
                    window.addEventListener('keydown', handleKeyboardShortcut);
                });
                const courseDisplayName = computed(() => {
                    const course = courses.value.find(item => item.key === selectedCourse.value);
                    if (course) return course.name;
                    if (selectedCourse.value === 'xigai_li') return '2024下习概李老师';
                    if (selectedCourse.value === 'xigai_yang') return '2025下习概杨老师';
                    return '2025上毛概康老师';
                });

                onBeforeUnmount(() => {
                    window.removeEventListener('keydown', handleKeyboardShortcut);
                });

                watch(currentView, (newView) => { 
                    switch(newView) {
                        case 'mainMenu': viewTitle.value = '喵喵学习小助手'; break;
                        case 'modeSelection': viewTitle.value = '选择学习模式'; break;
                        case 'chapterOrderSelection': viewTitle.value = `模式设置 - ${modeDisplayName.value}`; break;
                        case 'quickReview': viewTitle.value = '速刷模式'; break;
                        case 'quizMode': viewTitle.value = '答题模式'; break;
                        case 'incorrectReview': viewTitle.value = '错题回顾'; break;
                        case 'resultsView': viewTitle.value = '本轮总结'; break;
                        case 'controlMode': viewTitle.value = '控制模式'; break;
                        case 'guideView': viewTitle.value = '使用指南'; break;
                        case 'shortcutSettings': viewTitle.value = '自定义快捷键'; break;
                        case 'mobileAccess': viewTitle.value = '手机访问'; break;
                        case 'jsonImport': viewTitle.value = '导入题库'; break;
                        case 'aiImport': viewTitle.value = 'AI API 导入题库'; break;
                        case 'customCourseManager': viewTitle.value = '自定义题库管理'; break;
                        case 'customCourseEditor': viewTitle.value = '自定义题库编辑'; break;
                        default: viewTitle.value = '喵喵学习小助手';
                    }
                    if (errorMessage.value && (userId.value || newView !== 'mainMenu')) {
                         errorMessage.value = '';
                    }
                    if (!isInQuestionView.value) { 
                        showJumpInput.value = false;
                    }
                });

                watch(() => aiConfigForm.value.model, async (newModel, oldModel) => {
                    if (!userId.value || !newModel || newModel === oldModel) return;
                    if (!aiStatus.value.configured) return;
                    await syncAIConfig({ quiet: true });
                    aiModelMessage.value = `已切换到 ${newModel}`;
                });
                
                const navigateTo = (view) => {
                    currentView.value = view;
                };

                const goBackOneView = () => {
                    if (isInQuestionView.value || currentView.value === 'chapterOrderSelection' || currentView.value === 'modeSelection' || currentView.value === 'guideView' || currentView.value === 'shortcutSettings' || currentView.value === 'mobileAccess' || currentView.value === 'jsonImport' || currentView.value === 'aiImport' || currentView.value === 'customCourseManager' || currentView.value === 'customCourseEditor' || currentView.value === 'controlMode' || currentView.value === 'resultsView') {
                        goBackToMenu();
                    }
                };

                const goBackToMenu = () => {
                    resetModeState(); 
                    if (userId.value) { 
                        navigateTo('mainMenu');
                    } else { 
                        inputUserId.value = ''; 
                        userIdError.value = ''; 
                    }
                };

                const resetModeState = () => {
                    allModeQuestions.value = [];
                    currentQuestion.value = null;
                    currentQuestionIndex.value = -1;
                    totalQuestions.value = 0;
                    originalTotalQuestions.value = 0;
                    isQuizCompleted.value = false;
                    selectedAnswers.value = [];
                    feedbackMessage.value = '';
                    quizModeState.value = 'inProgress';
                    quizResults.value = { total_answered: 0, total_correct: 0 };
                    showJumpInput.value = false;
                    jumpToQuestionNumberInput.value = null;
                };
                
                const selectCourse = (course) => {
                    selectedCourse.value = course;
                    selectedChapters.value = []; // 清空之前的章节选择
                    
                    // 如果是习概（任意 xigai 变体），自动选择第0章
                    if (course === 'xigai_li' || course === 'xigai_yang') {
                        selectedChapters.value = ['0'];
                    }
                };
                
                const selectMode = (mode) => { 
                    if (!userId.value) {
                        errorMessage.value = "用户ID丢失，请重新输入用户ID或刷新页面。";
                        return;
                    }
                    
                    // 如果是习概（任意 xigai 变体），自动选择第0章
                    if ((selectedCourse.value === 'xigai_li' || selectedCourse.value === 'xigai_yang') && !selectedChapters.value.includes('0')) {
                        selectedChapters.value = ['0'];
                    }
                    
                    activeMode.value = mode; 
                    
                    if (mode === 'quickReview') modeDisplayName.value = '速刷模式';
                    else if (mode === 'quizMode') modeDisplayName.value = '答题模式';
                    else if (mode === 'incorrectReview') modeDisplayName.value = '错题回顾';

                    navigateTo('chapterOrderSelection');
                };

                const toggleChapterSelection = (chapterValue) => {
                    if (chapterValue === 'all') {
                        if (selectedChapters.value.includes('all')) { 
                            selectedChapters.value = [];
                        } else { 
                            selectedChapters.value = availableChapters.value.map(c => c.value);
                        }
                    } else {
                        const index = selectedChapters.value.indexOf(chapterValue);
                        if (index > -1) {
                            selectedChapters.value.splice(index, 1);
                        } else {
                            selectedChapters.value.push(chapterValue);
                        }
                        const allIndividualChapters = availableChapters.value.filter(c => c.value !== 'all');
                        const allIndividualSelected = allIndividualChapters.every(c => selectedChapters.value.includes(c.value));
                        
                        const allIndex = selectedChapters.value.indexOf('all');
                        if (allIndividualSelected && allIndex === -1) {
                            selectedChapters.value.push('all');
                        } else if (!allIndividualSelected && allIndex > -1) {
                            selectedChapters.value.splice(allIndex, 1);
                        }
                    }
                };
                
                const startSelectedMode = async () => { 
                    if (!userId.value) {
                        errorMessage.value = "用户未初始化，请返回并输入用户ID。";
                        return; 
                    }
                    if (activeMode.value !== 'incorrectReview' && selectedChapters.value.length === 0) {
                        errorMessage.value = "请至少选择一个章节。";
                        return;
                    }
                    isLoading.value = true;
                    errorMessage.value = ''; 
                    resetModeState(); 

                    let url = '';
                    let requestBody = { 
                        user_id: userId.value,
                        course: selectedCourse.value
                    };

                    if (activeMode.value === 'quickReview' || activeMode.value === 'quizMode') {
                        url = activeMode.value === 'quickReview' ? `${API_BASE_URL}/api/review/start` : `${API_BASE_URL}/api/quiz/start`;
                        requestBody.chapter_choice = selectedChapters.value.includes('all') ? ['all'] : selectedChapters.value.filter(c => c !== 'all' && c !== undefined && c !== null);
                        requestBody.order_choice = selectedOrder.value;
                    } else if (activeMode.value === 'incorrectReview') {
                        startIncorrectReview();
                        return;
                    } else {
                        errorMessage.value = "未知的模式: " + activeMode.value;
                        isLoading.value = false;
                        return;
                    }
                    
                    try {
                        const response = await fetch(url, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(requestBody)
                        });
                        if (!response.ok) {
                            const errBody = await response.text();
                            throw new Error(`开始${modeDisplayName.value}失败: ${response.statusText} (${response.status}) - ${errBody || '(无响应体)'}`);
                        }
                        const data = await response.json();
                        if (data && data.questions && Array.isArray(data.questions)) { 
                            allModeQuestions.value = data.questions; 
                            totalQuestions.value = data.questions.length;
                            originalTotalQuestions.value = data.questions.length;
                            if (totalQuestions.value > 0) {
                                currentQuestionIndex.value = 0;
                                setCurrentQuestionFromIndex(); 
                                isQuizCompleted.value = false;
                                if (activeMode.value === 'quizMode') saveCurrentQuizState(); // Save new quiz state
                                navigateTo(activeMode.value); 
                            } else {
                                errorMessage.value = data.message || `当前选择的「${courseDisplayName.value}」题库没有可用题目，请换一个题库或章节。`;
                                alert(errorMessage.value);
                                navigateTo('chapterOrderSelection'); 
                            }
                        } else {
                             errorMessage.value = data.message || "没有获取到题目或响应格式不正确 (期望 'questions' 数组)。";
                             alert(errorMessage.value);
                             navigateTo('chapterOrderSelection');
                        }
                    } catch (err) {
                        errorMessage.value = err.message;
                    } finally {
                        isLoading.value = false;
                    }
                };

                const continueLastQuiz = () => {
                    if (!canContinueQuiz.value) {
                        errorMessage.value = "没有可以继续的答题记录。";
                        return;
                    }
                    resetModeState();
                    activeMode.value = 'quizMode';
                    modeDisplayName.value = '答题模式';
                    
                    const state = savedQuizState.value;
                    allModeQuestions.value = state.questions;
                    totalQuestions.value = state.questions.length;
                    originalTotalQuestions.value = state.originalTotalQuestions;

                    if (state.isCompleted) { // If last quiz was finished, restart it
                        currentQuestionIndex.value = 0;
                        quizResults.value = { total_answered: 0, total_correct: 0 };
                        isQuizCompleted.value = false;
                    } else { // Otherwise, continue from where left off
                        currentQuestionIndex.value = state.currentIndex;
                        quizResults.value = state.results;
                        isQuizCompleted.value = state.isCompleted;
                    }

                    setCurrentQuestionFromIndex();
                    saveCurrentQuizState(); // Re-save state to update isCompleted flag if it was restarted
                    navigateTo('quizMode');
                };

                const setCurrentQuestionFromIndex = () => {
                    if (currentQuestionIndex.value >= 0 && currentQuestionIndex.value < allModeQuestions.value.length) {
                        currentQuestion.value = allModeQuestions.value[currentQuestionIndex.value];
                        resetQuizStateForNewQuestion(); 
                        quizModeState.value = 'inProgress'; 
                        isQuizCompleted.value = false;
                    } else if (currentQuestionIndex.value >= allModeQuestions.value.length && allModeQuestions.value.length > 0) {
                        isQuizCompleted.value = true;
                        currentQuestion.value = null; 
                        if (activeMode.value === 'quizMode') saveCurrentQuizState();
                        navigateTo('resultsView');
                    } else { 
                        isQuizCompleted.value = true;
                        currentQuestion.value = null;
                        navigateTo('resultsView');
                    }
                };

                const previousQuestion = () => {
                    if (currentQuestionIndex.value > 0) {
                        quizModeState.value = 'inProgress'; 
                        currentQuestionIndex.value--;
                        setCurrentQuestionFromIndex();
                        if (activeMode.value === 'quizMode') saveCurrentQuizState();
                    }
                };

                const toggleJumpInput = () => {
                    showJumpInput.value = !showJumpInput.value;
                    if (!showJumpInput.value) jumpToQuestionNumberInput.value = null;
                };

                const jumpToQuestion = () => {
                    const targetNum = parseInt(jumpToQuestionNumberInput.value, 10);
                    if (targetNum && targetNum >= 1 && targetNum <= totalQuestions.value) { 
                        quizModeState.value = 'inProgress'; 
                        currentQuestionIndex.value = targetNum - 1; 
                        setCurrentQuestionFromIndex();
                        if (activeMode.value === 'quizMode') saveCurrentQuizState();
                        showJumpInput.value = false; 
                        jumpToQuestionNumberInput.value = null;
                    } else {
                        alert(`请输入有效的题号 (1-${totalQuestions.value})`);
                    }
                };
                
                const fetchNextQuestion = async () => { 
                    if (isQuizCompleted.value || currentQuestionIndex.value >= totalQuestions.value -1) {
                        isQuizCompleted.value = true;
                        if(activeMode.value === 'quizMode') saveCurrentQuizState();
                        navigateTo('resultsView');
                        return;
                    }
                    if (currentQuestionIndex.value < totalQuestions.value - 1) {
                        currentQuestionIndex.value++;
                        setCurrentQuestionFromIndex();
                        if(activeMode.value === 'quizMode') saveCurrentQuizState();
                    }
                };

                const submitAnswerForMode = async () => {
                    let userAnswerString = '';
                    if (currentQuestion.value?.question_type === '多选题') {
                        userAnswerString = Array.isArray(selectedAnswers.value) ? [...selectedAnswers.value].sort().join('') : '';
                    } else { 
                        // 单选题：确保返回字符串而不是数组
                        userAnswerString = Array.isArray(selectedAnswers.value) 
                            ? (selectedAnswers.value.length > 0 ? selectedAnswers.value[0] : '') 
                            : (selectedAnswers.value || '');
                    }

                    if ((currentQuestion.value?.question_type === '单选题' && !userAnswerString) ||
                        (currentQuestion.value?.question_type === '多选题' && userAnswerString.length === 0)) {
                        errorMessage.value = '请选择答案后再提交！'; return;
                    }
                    errorMessage.value = ''; 

                    const correctAnswerProcessed = (currentQuestion.value && typeof currentQuestion.value.correct_answer === 'string') 
                                                ? currentQuestion.value.correct_answer.split('').sort().join('') : '';
                    isCurrentAnswerCorrect.value = userAnswerString === correctAnswerProcessed;
                    feedbackMessage.value = isCurrentAnswerCorrect.value ? '回答正确！👍' : `回答错误！正确答案是：<strong>${currentQuestion.value.correct_answer}</strong>`;
                    quizModeState.value = 'showAnswer'; 

                    if (activeMode.value === 'quizMode' || activeMode.value === 'incorrectReview') {
                        quizResults.value.total_answered++;
                        if(isCurrentAnswerCorrect.value) quizResults.value.total_correct++;
                    }

                    if (currentQuestionIndex.value >= totalQuestions.value - 1) {
                        isQuizCompleted.value = true;
                    } else {
                        isQuizCompleted.value = false; 
                    }
                    if (activeMode.value === 'quizMode') saveCurrentQuizState();
                    
                    let url = '';
                    if (activeMode.value === 'quizMode') url = `${API_BASE_URL}/api/quiz/submit_answer`;
                    else if (activeMode.value === 'incorrectReview') url = `${API_BASE_URL}/api/incorrect_questions/review/submit_answer`;
                    else { return; }

                    try {
                        const requestBody = {
                            user_id: userId.value,
                            quiz_question_id: currentQuestion.value.quiz_question_id,
                            user_answer: userAnswerString,
                            was_correct: isCurrentAnswerCorrect.value 
                        };
                        console.log('[DEBUG] 提交答案:', JSON.stringify(requestBody, null, 2));
                        fetch(url, { 
                            method: 'POST', 
                            headers: { 'Content-Type': 'application/json' }, 
                            body: JSON.stringify(requestBody) 
                        })
                        .then(response => { 
                            if (!response.ok) { console.error(`服务器记录答案失败 (后台)`); } 
                            else { console.log("答案已成功同步到服务器 (后台)。"); }
                        })
                        .catch(err => { console.error(`提交答案到服务器时发生网络错误 (后台): ${err.message}`); });
                    } catch (err) { console.error(`准备提交答案到服务器时出错 (后台): ${err.message}`); } 
                };
                                
                const startIncorrectReview = async () => { 
                    if (!userId.value) { errorMessage.value = "用户未初始化，请返回并输入用户ID。"; return; }
                    isLoading.value = true;
                    errorMessage.value = '';
                    resetModeState(); 
                    activeMode.value = 'incorrectReview'; 
                    modeDisplayName.value = '错题回顾';

                    try {
                        const response = await fetch(`${API_BASE_URL}/api/incorrect_questions/review/start`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ 
                                user_id: userId.value,
                                course: selectedCourse.value
                            })
                        });
                        if (!response.ok) { 
                            const errBody = await response.text();
                            throw new Error(`开始错题回顾失败: ${response.statusText} (${response.status}) - ${errBody || '(无响应体)'}`);
                         }
                        const data = await response.json();
                        if (data && data.questions && Array.isArray(data.questions)) { 
                            allModeQuestions.value = data.questions; 
                            totalQuestions.value = data.questions.length;
                            originalTotalQuestions.value = data.questions.length;
                            if (totalQuestions.value > 0) {
                                currentQuestionIndex.value = 0;
                                setCurrentQuestionFromIndex();
                                isQuizCompleted.value = false;
                                navigateTo('incorrectReview'); 
                            } else {
                                errorMessage.value = data.message || `「${courseDisplayName.value}」错题题库为空。`;
                                navigateTo('chapterOrderSelection');
                            }
                        } else { 
                            errorMessage.value = data.message || "错题簿是空的或无法加载。";
                            navigateTo('chapterOrderSelection');
                        }
                    } catch (err) { 
                        errorMessage.value = err.message;
                    } 
                    finally { isLoading.value = false; }
                };

                const deleteCurrentIncorrectQuestion = async () => {
                    if (!currentQuestion.value || !userId.value) {
                        errorMessage.value = "无法删除：当前问题或用户ID无效。";
                        return;
                    }

                    const questionToDelete = currentQuestion.value;
                    const indexToDelete = currentQuestionIndex.value;

                    allModeQuestions.value.splice(indexToDelete, 1);
                    totalQuestions.value = allModeQuestions.value.length;
                    
                    setCurrentQuestionFromIndex();

                    try {
                        const requestBody = {
                            user_id: userId.value,
                            course: questionToDelete.course || selectedCourse.value,
                            original_chapter: questionToDelete.original_chapter,
                            original_question_number: questionToDelete.original_question_number
                        };
                        
                        fetch(`${API_BASE_URL}/api/incorrect_questions/delete`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(requestBody)
                        }).then(response => {
                            if (!response.ok) {
                                console.error(`后台删除错题失败 (用户: ${userId.value}, 题号: ${questionToDelete.original_question_number})`);
                            } else {
                                console.log(`错题 (用户: ${userId.value}, 题号: ${questionToDelete.original_question_number}) 已在后台请求删除。`);
                            }
                        }).catch(err => { console.error(`发送删除错题请求时发生网络错误: ${err.message}`); });
                    } catch (err) { console.error(`准备删除错题请求时出错: ${err.message}`); }
                };

                const confirmClearUserData = () => {
                    if (confirm(`喵呜！警告：此操作将重置用户 '${userId.value}' 的所有对错统计并删除错题簿，数据无法恢复！\n确定要清理吗？`)) {
                        clearUserData();
                    }
                };

                const clearUserData = async () => { 
                    if (!userId.value) { errorMessage.value = "用户未初始化，无法清理数据。"; return; }
                    isLoading.value = true;
                    errorMessage.value = '';
                    try {
                        const response = await fetch(`${API_BASE_URL}/api/user/data/clear`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ user_id: userId.value })
                        });
                        if (!response.ok) {
                             const errBody = await response.text();
                             throw new Error(`清理用户数据失败: ${response.statusText} (${response.status}) - ${errBody || '(无响应体)'}`);
                        }
                        localStorage.removeItem(getSavedQuizKey()); // Clear saved quiz state
                        savedQuizState.value = null;
                        const data = await response.json();
                        alert(data.message || "用户数据已清理。");
                        logoutCurrentUser(false); 
                    } catch (err) { 
                        errorMessage.value = err.message;
                    } 
                    finally { isLoading.value = false; }
                };

                const logoutCurrentUser = (confirmLogout = true) => {
                    if (confirmLogout && !confirm("确定要退出当前用户并返回登录界面吗？")) {
                        return;
                    }
                    localStorage.removeItem('quizAppUserId');
                    localStorage.removeItem(getSavedQuizKey()); // Clear saved quiz state on logout
                    savedQuizState.value = null;
                    userId.value = '';
                    refreshShortcutSettings();
                    inputUserId.value = '';
                    userIdError.value = '';
                    errorMessage.value = ''; 
                    resetModeState(); 
                    currentView.value = 'mainMenu'; 
                };


                const sortedOptions = computed(() => {
                    if (currentQuestion.value && typeof currentQuestion.value.options === 'object' && currentQuestion.value.options !== null) {
                        try {
                            return Object.entries(currentQuestion.value.options)
                                .sort(([keyA], [keyB]) => keyA.localeCompare(keyB))
                                .reduce((obj, [key, value]) => { obj[key] = value; return obj; }, {});
                        } catch (e) { console.error("Error sorting options:", e); return {}; }
                    }
                    return {};
                });

                const formatQuestionText = (text) => text ? text.replace(/\n/g, '<br>') : '';

                const getOptionLabelClass = (optionKey) => {
                    const classes = [];
                    const userAnswersArray = Array.isArray(selectedAnswers.value) 
                                            ? selectedAnswers.value 
                                            : (selectedAnswers.value !== null && selectedAnswers.value !== undefined ? [String(selectedAnswers.value)] : []);
                                            
                    const correctAnswerString = (currentQuestion.value && typeof currentQuestion.value.correct_answer === 'string') 
                                            ? currentQuestion.value.correct_answer
                                            : '';
                    const correctAnswerKeys = correctAnswerString.split('');

                    if ((activeMode.value === 'quizMode' || activeMode.value === 'incorrectReview') && quizModeState.value === 'showAnswer') {
                        const isThisOptionActuallyCorrect = correctAnswerKeys.includes(optionKey);
                        const wasThisOptionSelectedByUser = userAnswersArray.includes(optionKey);

                        if (isThisOptionActuallyCorrect) { 
                            classes.push('correct'); 
                        } else if (wasThisOptionSelectedByUser) { 
                           classes.push('incorrect');
                        }
                    } else if ((activeMode.value === 'quizMode' || activeMode.value === 'incorrectReview') && quizModeState.value === 'inProgress') { 
                         if (userAnswersArray.includes(optionKey)) {
                            classes.push('selected');
                        }
                    }
                    return classes.join(' ');
                };

                const resetQuizStateForNewQuestion = () => {
                    if (currentQuestion.value) { 
                        selectedAnswers.value = currentQuestion.value.question_type === '单选题' ? null : [];
                    } else {
                        selectedAnswers.value = null; 
                    }
                    feedbackMessage.value = '';
                    isCurrentAnswerCorrect.value = false; 
                };

                return {
                    isLoading, errorMessage, userIdError, currentView, viewTitle, userId, inputUserId,
                    networkInfo, loadNetworkInfo, copyText, shortcutDraft, shortcutCapture, startShortcutCapture, isCapturingShortcut, saveShortcutSettings, resetShortcutSettings,
                    courses, availableChapters, courseDisplayName, selectedChapters, selectedOrder, selectedCourse,
                    aiStatus, aiConfigForm, aiModels, aiModelMessage, loadAIModels, aiImportForm, aiImportMessage, aiImportTask, saveAIConfig, importAIQuestionBank,
                    defaultAISystemPrompt, jsonPromptCopied, jsonImportForm, jsonImportMessage, importJSONQuestionBank, copyJSONPrompt, customCourses, customEditor, customEditorChapterKeys, customEditorQuestions, openCustomCourseEditor, addEditorQuestion, removeEditorQuestion, addEditorChapter, deleteEditorChapter, saveCustomCourse, deleteCustomCourse,
                    activeMode, modeDisplayName,
                    allModeQuestions, currentQuestion, totalQuestions, originalTotalQuestions, isQuizCompleted, currentQuestionIndex,
                    selectedAnswers, quizModeState, feedbackMessage, isCurrentAnswerCorrect, quizResults,
                    isInQuestionView, showNextButton,
                    showJumpInput, jumpToQuestionNumberInput,
                    navigateTo, goBackToMenu, selectCourse, selectMode, toggleChapterSelection, startSelectedMode,
                    fetchNextQuestion, submitAnswerForMode, 
                    startIncorrectReview,
                    deleteCurrentIncorrectQuestion,
                    confirmClearUserData, clearUserData, logoutCurrentUser, 
                    sortedOptions, formatQuestionText, getOptionLabelClass,
                    previousQuestion, toggleJumpInput, jumpToQuestion,
                    submitUserId,
                    canContinueQuiz,
                    continueLastQuiz,
                    savedQuizState
                };
            }
        }).mount('#app');
