import { reactive } from 'vue'

const DEFAULT = {
  options: { A: 'A', B: 'B', C: 'C', D: 'D', E: 'E' },
  submit: 'Enter',
  previous: 'ArrowLeft',
  next: 'ArrowRight',
  back: 'Escape',
}

const ALIASES = {
  esc: 'Escape',
  escape: 'Escape',
  enter: 'Enter',
  回车: 'Enter',
  left: 'ArrowLeft',
  arrowleft: 'ArrowLeft',
  左: 'ArrowLeft',
  right: 'ArrowRight',
  arrowright: 'ArrowRight',
  右: 'ArrowRight',
}

export function normalizeKey(value) {
  const text = String(value || '').trim()
  if (!text) return ''
  const lower = text.toLowerCase()
  if (ALIASES[lower]) return ALIASES[lower]
  return text.length === 1 ? text.toUpperCase() : text
}

const key = (userId) => `quizAppShortcuts:${userId}`

export function loadShortcuts(userId) {
  if (!userId) return JSON.parse(JSON.stringify(DEFAULT))
  try {
    const raw = localStorage.getItem(key(userId))
    if (!raw) return JSON.parse(JSON.stringify(DEFAULT))
    const saved = JSON.parse(raw)
    return {
      options: { ...DEFAULT.options, ...(saved.options || {}) },
      submit: saved.submit || DEFAULT.submit,
      previous: saved.previous || DEFAULT.previous,
      next: saved.next || DEFAULT.next,
      back: saved.back || DEFAULT.back,
    }
  } catch {
    return JSON.parse(JSON.stringify(DEFAULT))
  }
}

export function saveShortcuts(userId, shortcuts) {
  if (!userId) return
  localStorage.setItem(key(userId), JSON.stringify(shortcuts))
}

export function resetShortcuts(userId) {
  localStorage.removeItem(key(userId))
  return JSON.parse(JSON.stringify(DEFAULT))
}

export const defaultShortcuts = DEFAULT
