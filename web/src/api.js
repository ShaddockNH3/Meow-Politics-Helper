const BASE = ''

async function request(path, { method = 'GET', body } = {}) {
  let res
  try {
    res = await fetch(BASE + path, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    })
  } catch (err) {
    throw new Error('无法连接本地服务，请确认程序仍在运行')
  }
  const text = await res.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = { raw: text }
    }
  }
  if (!res.ok) {
    const message = (data && (data.error || data.message)) || `请求失败（${res.status}）`
    throw new Error(message)
  }
  return data
}

const post = (path, body) => request(path, { method: 'POST', body })

export const api = {
  initSession: (userId) => post('/api/session/init', { user_id: userId }),
  courses: (userId) => request(`/api/courses?user_id=${encodeURIComponent(userId)}`),

  reviewStart: (payload) => post('/api/review/start', payload),
  reviewNext: (userId) => post('/api/review/next', { user_id: userId }),
  quizStart: (payload) => post('/api/quiz/start', payload),
  submitAnswer: (payload) => post('/api/quiz/submit_answer', payload),

  incorrectStart: (userId, course) =>
    post('/api/incorrect_questions/review/start', { user_id: userId, course }),
  incorrectSubmit: (payload) => post('/api/incorrect_questions/review/submit_answer', payload),
  deleteIncorrect: (payload) => post('/api/incorrect_questions/delete', payload),

  clearData: (userId) => post('/api/user/data/clear', { user_id: userId }),

  importJSON: (payload) => post('/api/courses/import-json', payload),
  courseDetail: (userId, course) =>
    request(`/api/courses/custom/detail?user_id=${encodeURIComponent(userId)}&course=${encodeURIComponent(course)}`),
  saveCourse: (payload) => post('/api/courses/custom/save', payload),
  deleteCourse: (payload) => post('/api/courses/delete', payload),

  networkInfo: () => request('/api/network/info'),
}
