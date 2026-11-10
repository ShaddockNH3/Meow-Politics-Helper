<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { ChevronLeft, ChevronRight, Flag } from 'lucide-vue-next'
import {
  accuracy,
  currentQuestion,
  finish,
  isLastQuestion,
  nextQuestion,
  pickOption,
  prevQuestion,
  progressPercent,
  resetForHome,
  revealQuickAnswer,
  state,
  submitAnswer,
  totalQuestions,
} from '../stores/app'
import { loadShortcuts, normalizeKey } from '../composables/useShortcuts'

const message = useMessage()
const dialog = useDialog()

const question = computed(() => currentQuestion.value)
const isQuick = computed(() => state.mode === 'quickReview')
const isMulti = computed(() => question.value?.question_type === '多选题')
const showAnswer = computed(() => isQuick.value || state.revealed)
const correctKeys = computed(() =>
  String(question.value?.correct_answer || '')
    .toUpperCase()
    .split(''),
)
const shortcuts = ref(loadShortcuts(state.userId))
const jumpTo = ref(null)

function optionClass(key) {
  if (!question.value) return ''
  const selected = state.selections.includes(key)
  if (!showAnswer.value) return selected ? 'is-selected' : ''
  if (correctKeys.value.includes(key)) return 'is-correct'
  if (selected) return 'is-wrong'
  return ''
}

function onMultiChange(values) {
  if (state.revealed) return
  state.selections = [...values].sort()
}

function onSingleChange(value) {
  pickOption(value)
}

async function submit() {
  try {
    await submitAnswer()
  } catch (err) {
    message.warning(err.message)
  }
}

function primaryAction() {
  if (isQuick.value) {
    if (isLastQuestion.value) finish()
    else nextQuestion()
    return
  }
  if (!state.revealed) submit()
  else if (isLastQuestion.value) finish()
  else nextQuestion()
}

function goNext() {
  if (isLastQuestion.value) finish()
  else nextQuestion()
}

function goPrev() {
  prevQuestion()
}

function confirmExit() {
  dialog.warning({
    title: '结束本次练习',
    content: '当前进度会保留在本地（答题模式可继续），确定离开吗？',
    positiveText: '结束',
    negativeText: '继续做题',
    onPositiveClick: () => resetForHome(),
  })
}

function doJump() {
  if (jumpTo.value == null) return
  jumpToIndex(Math.min(Math.max(1, jumpTo.value), totalQuestions.value) - 1)
}

function jumpToIndex(index) {
  state.index = index
  state.selections = []
  state.revealed = false
  state.lastCorrect = null
}

function onKeydown(event) {
  const tag = (document.activeElement?.tagName || '').toLowerCase()
  if (tag === 'input' || tag === 'textarea' || tag === 'select') return
  const key = normalizeKey(event.key)

  const optionEntry = Object.entries(shortcuts.value.options).find(
    ([, bound]) => normalizeKey(bound) === key,
  )
  if (optionEntry && !showAnswer.value) {
    pickOption(optionEntry[0])
    event.preventDefault()
    return
  }

  if (key === normalizeKey(shortcuts.value.submit)) {
    primaryAction()
    event.preventDefault()
    return
  }
  if (key === normalizeKey(shortcuts.value.next)) {
    goNext()
    event.preventDefault()
    return
  }
  if (key === normalizeKey(shortcuts.value.previous)) {
    goPrev()
    event.preventDefault()
    return
  }
  if (key === normalizeKey(shortcuts.value.back)) {
    confirmExit()
    event.preventDefault()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  if (isQuick.value) revealQuickAnswer()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div v-if="question">
    <div class="quiz-top">
      <div class="quiz-crumb">
        <span>{{ state.mode === 'quickReview' ? '速刷' : state.mode === 'quizMode' ? '答题' : '错题回顾' }}</span>
        <span class="crumb-sep">·</span>
        <span>第 {{ state.index + 1 }} / {{ totalQuestions }} 题</span>
        <span v-if="question.original_question_number" class="crumb-faint">
          原 {{ question.original_question_number }}
        </span>
      </div>
      <n-button size="tiny" quaternary @click="confirmExit">
        <template #icon><Flag :size="14" :stroke-width="1.7" /></template>
        结束
      </n-button>
    </div>

    <div class="quiz-layout">
      <div>
        <n-card size="small">
          <div class="question-meta">
            <n-tag size="small" :bordered="false" type="info">{{ question.question_type }}</n-tag>
            <n-tag v-if="state.mode === 'incorrectReview'" size="small" :bordered="false">
              第 {{ question.original_chapter }} 章
            </n-tag>
            <n-tag v-if="showAnswer && !isQuick" size="small" :bordered="false" :type="state.lastCorrect ? 'success' : 'error'">
              {{ state.lastCorrect ? '答对' : '答错' }}
            </n-tag>
          </div>

          <p class="question-text">{{ question.question_text }}</p>

          <n-radio-group
            v-if="!isMulti"
            :value="state.selections[0]"
            class="option-list"
            :class="{ 'no-dot': isQuick }"
            @update:value="onSingleChange"
          >
            <n-radio
              v-for="(text, key) in question.options"
              :key="key"
              :value="key"
              :disabled="showAnswer"
              class="option-item"
              :class="optionClass(key)"
            >
              <span class="opt-key">{{ key }}</span>
              <span class="opt-text">{{ text }}</span>
            </n-radio>
          </n-radio-group>

          <n-checkbox-group
            v-else
            :value="state.selections"
            class="option-list"
            :class="{ 'no-dot': isQuick }"
            @update:value="onMultiChange"
          >
            <n-checkbox
              v-for="(text, key) in question.options"
              :key="key"
              :value="key"
              :disabled="showAnswer"
              class="option-item"
              :class="optionClass(key)"
            >
              <span class="opt-key">{{ key }}</span>
              <span class="opt-text">{{ text }}</span>
            </n-checkbox>
          </n-checkbox-group>

          <n-alert
            v-if="showAnswer && question.correct_answer"
            :type="isQuick ? 'default' : state.lastCorrect ? 'success' : 'error'"
            :bordered="false"
            style="margin-top: 14px"
          >
            <template v-if="isQuick">
              正确答案：<strong>{{ question.correct_answer }}</strong>
            </template>
            <template v-else>
              {{ state.lastCorrect ? '回答正确' : '回答错误' }}，正确答案是
              <strong>{{ question.correct_answer }}</strong>
            </template>
          </n-alert>
        </n-card>

        <div class="quiz-actions mobile-actions">
          <n-button
            v-if="!isQuick && !state.revealed"
            type="primary"
            size="large"
            block
            @click="submit"
          >
            提交答案
          </n-button>
          <n-button v-else type="primary" size="large" block @click="primaryAction">
            {{ isLastQuestion ? '查看本次总结' : '下一题' }}
          </n-button>
          <div class="nav-buttons">
            <n-button size="small" quaternary :disabled="state.index === 0" @click="goPrev">
              <template #icon><ChevronLeft :size="15" :stroke-width="1.7" /></template>
              上一题
            </n-button>
            <n-button size="small" quaternary @click="goNext">
              下一题
              <template #icon><ChevronRight :size="15" :stroke-width="1.7" /></template>
            </n-button>
          </div>
        </div>
      </div>

      <aside class="quiz-side">
        <n-card size="small" style="margin-bottom: 12px">
          <div class="side-stat"><span>本次进度</span><span>{{ progressPercent }}%</span></div>
          <n-progress
            type="line"
            :percentage="progressPercent"
            :show-indicator="false"
            :height="6"
            color="#1f3f66"
            rail-color="#e9edf4"
          />
          <div class="side-value">{{ state.results.answered }}<span class="side-unit"> 题已答</span></div>
          <div class="side-stat">
            <span>正确率</span>
            <span>{{ state.results.answered ? accuracy + '%' : '—' }}</span>
          </div>
        </n-card>

        <n-card size="small">
          <p class="section-title" style="margin: 0 0 8px">跳到</p>
          <n-input-group>
            <n-input-number
              v-model:value="jumpTo"
              size="small"
              :min="1"
              :max="totalQuestions"
              placeholder="题号"
              style="width: 100%"
            />
            <n-button size="small" type="primary" ghost @click="doJump">去</n-button>
          </n-input-group>
          <div class="jump-grid">
            <button
              v-for="(item, index) in state.questions"
              :key="item.quiz_question_id"
              class="jump-dot"
              :class="{ current: index === state.index, done: index < state.index }"
              :title="`第 ${index + 1} 题`"
              @click="jumpToIndex(index)"
            >
              {{ index + 1 }}
            </button>
          </div>
        </n-card>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.quiz-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.quiz-crumb {
  font-size: 12px;
  color: var(--muted);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.crumb-sep {
  color: #c7cfdc;
}

.crumb-faint {
  color: var(--faint);
}

.nav-buttons {
  display: flex;
  justify-content: space-between;
  margin-top: 6px;
}

.side-unit {
  font-size: 12px;
  color: var(--faint);
  font-weight: 400;
}

.jump-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(22px, 1fr));
  gap: 3px;
  margin-top: 10px;
  max-height: 224px;
  overflow: auto;
}

.jump-dot {
  border: 1px solid var(--border);
  background: #fff;
  border-radius: 4px;
  font-size: 11px;
  color: var(--faint);
  padding: 3px 0;
  cursor: pointer;
}

.jump-dot.done {
  background: var(--primary-soft);
  border-color: #c8d5e6;
  color: var(--primary);
}

.jump-dot.current {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

@media (max-width: 768px) {
  .quiz-side {
    margin-top: 12px;
  }
}
</style>
