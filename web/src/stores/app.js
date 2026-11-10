import { reactive, computed } from 'vue'
import { api } from '../api'

export const MODES = [
  { key: 'quickReview', name: '速刷模式', desc: '看题 + 答案，考前快速过一遍', icon: 'zap' },
  { key: 'quizMode', name: '答题模式', desc: '认真作答，记录统计与错题', icon: 'pen' },
  { key: 'incorrectReview', name: '错题回顾', desc: '只刷答错过的题，答对可自动移出', icon: 'rotate' },
]

const USER_KEY = 'quizAppUserId'
const autoDeleteKey = (id) => `quizAppAutoDeleteIncorrect:${id}`
const sessionKey = (id) => `quizAppSession:${id}`

export const state = reactive({
  userId: localStorage.getItem(USER_KEY) || '',
  view: 'home',
  stack: [],

  courses: [],
  selectedCourse: '',
  mode: '',
  selectedChapters: [],
  order: 'sequential',
  autoDeleteIncorrect: true,

  questions: [],
  index: 0,
  selections: [],
  revealed: false,
  lastCorrect: null,
  results: { answered: 0, correct: 0 },

  loading: false,
  loadingText: '',
  error: '',

  networkInfo: null,
  bankDraft: null,
  canResume: false,
})

export const currentCourse = computed(() => state.courses.find((c) => c.key === state.selectedCourse) || null)
export const currentQuestion = computed(() => state.questions[state.index] || null)
export const totalQuestions = computed(() => state.questions.length)
export const modeInfo = computed(() => MODES.find((m) => m.key === state.mode) || null)
export const isLastQuestion = computed(() => state.index >= state.questions.length - 1)
export const progressPercent = computed(() =>
  state.questions.length ? Math.round(((state.index + 1) / state.questions.length) * 100) : 0,
)
export const accuracy = computed(() =>
  state.results.answered ? Math.round((state.results.correct / state.results.answered) * 100) : 0,
)

export function chaptersOf(course) {
  if (!course) return []
  const max = course.max_chapter ?? 0
  const names = course.chapter_names || []
  const list = []
  for (let i = 0; i <= max; i++) {
    const custom = String(names[i] || '').trim()
    let name = custom
    if (!name) name = course.custom ? `章节 ${i + 1}` : max === 0 ? '期末复习' : `第 ${i + 1} 章`
    list.push({ key: String(i), name })
  }
  return list
}

export const chapters = computed(() => chaptersOf(currentCourse.value))

export function setView(view, { push = true } = {}) {
  if (push && state.view !== view) state.stack.push(state.view)
  state.view = view
}

export function goBack() {
  state.error = ''
  const prev = state.stack.pop()
  state.view = prev || 'home'
}

export function setError(message) {
  state.error = message || ''
}

function normalizeAnswer(value) {
  return String(value || '')
    .toUpperCase()
    .replace(/[\s,，]/g, '')
    .split('')
    .sort()
    .join('')
}

function loadAutoDelete() {
  try {
    const stored = localStorage.getItem(autoDeleteKey(state.userId))
    return stored === null ? true : stored === '1'
  } catch {
    return true
  }
}

export function toggleAutoDelete() {
  state.autoDeleteIncorrect = !state.autoDeleteIncorrect
  try {
    const key = autoDeleteKey(state.userId)
    if (state.autoDeleteIncorrect) localStorage.setItem(key, '1')
    else localStorage.removeItem(key)
  } catch {
    /* 忽略 */
  }
}

async function withLoading(text, fn) {
  state.loading = true
  state.loadingText = text
  state.error = ''
  try {
    return await fn()
  } finally {
    state.loading = false
    state.loadingText = ''
  }
}

export async function login(userId) {
  const id = String(userId || '').trim()
  if (!id) throw new Error('请输入用户 ID')
  if (!/^[A-Za-z0-9_-]{1,50}$/.test(id)) throw new Error('用户 ID 只能包含字母、数字、下划线和短横线')
  await withLoading('正在建立会话…', async () => {
    await api.initSession(id)
    state.userId = id
    localStorage.setItem(USER_KEY, id)
    state.autoDeleteIncorrect = loadAutoDelete()
    await refreshCourses()
  })
  setView('home')
}

export function logout() {
  state.userId = ''
  state.courses = []
  state.selectedCourse = ''
  state.questions = []
  state.stack = []
  localStorage.removeItem(USER_KEY)
  state.view = 'home'
}

export async function refreshCourses() {
  const data = await api.courses(state.userId)
  state.courses = data.courses || []
  if (!state.courses.some((c) => c.key === state.selectedCourse)) {
    // 默认优先毛概（与旧版一致），没有再退回第一个
    const preferred = state.courses.find((c) => c.key === 'maogai')
    state.selectedCourse = preferred?.key || state.courses[0]?.key || ''
  }
  state.canResume = Boolean(readSession())
}

function readSession() {
  if (!state.userId) return null
  const raw = localStorage.getItem(sessionKey(state.userId))
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

export function detectResume() {
  const saved = readSession()
  state.canResume = Boolean(saved && saved.questions?.length)
  return saved
}

export function resumeQuiz() {
  const saved = readSession()
  if (!saved || !saved.questions?.length) {
    state.canResume = false
    return false
  }
  state.mode = 'quizMode'
  state.selectedCourse = saved.course || state.selectedCourse
  state.questions = saved.questions
  state.index = Math.min(saved.index || 0, saved.questions.length - 1)
  state.results = saved.results || { answered: 0, correct: 0 }
  state.selections = []
  state.revealed = false
  state.lastCorrect = null
  setView('quiz')
  return true
}

function saveSession() {
  if (state.mode !== 'quizMode' || !state.questions.length) return
  try {
    localStorage.setItem(
      sessionKey(state.userId),
      JSON.stringify({
        course: state.selectedCourse,
        questions: state.questions,
        index: state.index,
        results: state.results,
        savedAt: Date.now(),
      }),
    )
    state.canResume = true
  } catch {
    /* 存储满了就算了 */
  }
}

export function clearSession() {
  localStorage.removeItem(sessionKey(state.userId))
  state.canResume = false
}

export function selectCourse(key) {
  state.selectedCourse = key
  state.selectedChapters = []
}

export function chooseMode(mode) {
  state.mode = mode
  state.selectedChapters = []
  state.order = 'sequential'
  state.results = { answered: 0, correct: 0 }
  if (mode === 'incorrectReview') {
    state.autoDeleteIncorrect = loadAutoDelete()
  }
  setView('setup')
}

export function toggleChapter(key) {
  const i = state.selectedChapters.indexOf(key)
  if (i >= 0) state.selectedChapters.splice(i, 1)
  else state.selectedChapters.push(key)
}

export function selectAllChapters() {
  const all = chapters.value.map((c) => c.key)
  state.selectedChapters = state.selectedChapters.length === all.length ? [] : all
}

export async function startMode() {
  if (!state.selectedCourse) throw new Error('请先选择题库')
  if (state.mode !== 'incorrectReview' && state.selectedChapters.length === 0) {
    throw new Error('请至少选择一个章节')
  }
  const payload = {
    user_id: state.userId,
    course: state.selectedCourse,
    chapter_choice: state.selectedChapters.length ? state.selectedChapters : ['all'],
    order_choice: state.order,
  }

  await withLoading('正在准备题目…', async () => {
    let data
    if (state.mode === 'incorrectReview') {
      data = await api.incorrectStart(state.userId, state.selectedCourse)
    } else if (state.mode === 'quickReview') {
      data = await api.reviewStart(payload)
    } else {
      data = await api.quizStart(payload)
    }
    const questions = data.questions || []
    if (!questions.length) {
      throw new Error(data.message || '所选范围没有题目')
    }
    state.questions = questions
    state.index = 0
    state.selections = []
    state.revealed = false
    state.lastCorrect = null
    state.results = { answered: 0, correct: 0 }
    if (state.mode === 'quizMode') saveSession()
    setView('quiz')
  })
}

export function pickOption(key) {
  if (state.revealed || state.mode === 'quickReview') return
  const question = currentQuestion.value
  if (!question) return
  if (question.question_type === '多选题') {
    const i = state.selections.indexOf(key)
    if (i >= 0) state.selections.splice(i, 1)
    else state.selections.push(key)
    state.selections.sort()
  } else {
    state.selections = [key]
  }
}

export function clearSelection() {
  state.selections = []
}

export async function submitAnswer() {
  const question = currentQuestion.value
  if (!question) return
  if (!state.selections.length) throw new Error('请先选择答案')

  const userAnswer = [...state.selections].sort().join('')
  const correct = normalizeAnswer(userAnswer) === normalizeAnswer(question.correct_answer)

  state.revealed = true
  state.lastCorrect = correct
  state.results.answered += 1
  if (correct) state.results.correct += 1

  if (state.mode === 'quizMode') {
    saveSession()
    api
      .submitAnswer({
        user_id: state.userId,
        quiz_question_id: question.quiz_question_id,
        user_answer: userAnswer,
      })
      .catch((err) => console.warn('答案同步失败:', err.message))
  } else if (state.mode === 'incorrectReview') {
    api
      .incorrectSubmit({
        user_id: state.userId,
        quiz_question_id: question.quiz_question_id,
        user_answer: userAnswer,
      })
      .catch(() => {})

    if (correct && state.autoDeleteIncorrect) {
      api
        .deleteIncorrect({
          user_id: state.userId,
          course: question.course || state.selectedCourse,
          original_chapter: question.original_chapter,
          original_question_number: question.original_question_number,
        })
        .catch((err) => console.warn('自动移出错题本失败:', err.message))
    }
  }
}

export function revealQuickAnswer() {
  state.revealed = true
}

export function nextQuestion() {
  if (state.index < state.questions.length - 1) {
    state.index += 1
    state.selections = []
    state.revealed = false
    state.lastCorrect = null
    if (state.mode === 'quizMode') saveSession()
  } else {
    finish()
  }
}

export function prevQuestion() {
  if (state.index > 0) {
    state.index -= 1
    state.selections = []
    state.revealed = false
    state.lastCorrect = null
    if (state.mode === 'quizMode') saveSession()
  }
}

export function jumpTo(index) {
  if (index >= 0 && index < state.questions.length) {
    state.index = index
    state.selections = []
    state.revealed = false
    state.lastCorrect = null
  }
}

export function finish() {
  if (state.mode === 'quizMode') clearSession()
  setView('result')
}

export async function loadNetworkInfo() {
  state.networkInfo = await api.networkInfo()
}

export async function clearUserData() {
  await api.clearData(state.userId)
  clearSession()
  state.results = { answered: 0, correct: 0 }
}

export function resetForHome() {
  state.stack = []
  state.view = 'home'
  state.questions = []
  state.index = 0
  state.selections = []
  state.revealed = false
  state.lastCorrect = null
}

export function courseList() {
  return state.courses.filter((c) => !c.custom)
}

export function customCourseList() {
  return state.courses.filter((c) => c.custom)
}
