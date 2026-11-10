<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { ArrowLeft, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import { refreshCourses, setView, state } from '../stores/app'

const message = useMessage()
const loading = ref(true)
const saving = ref(false)
const draft = reactive({ courseName: '', chapters: {}, chapterNames: {} })
const activeChapter = ref('')

const chapterKeys = computed(() =>
  Object.keys(draft.chapters).sort((a, b) => Number(a) - Number(b)),
)
const currentList = computed(() => draft.chapters[activeChapter.value] || [])
const optionKeys = ['A', 'B', 'C', 'D', 'E']

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await api.courseDetail(state.userId, state.bankDraft)
    draft.courseName = data.meta?.course_name || '自定义题库'
    draft.chapters = normalizeChapters(data.chapters || {})
    draft.chapterNames = { ...(data.meta?.chapter_names || {}) }
    activeChapter.value = Object.keys(draft.chapters).sort((a, b) => Number(a) - Number(b))[0] || ''
  } catch (err) {
    message.error(err.message)
    setView('banks')
  } finally {
    loading.value = false
  }
}

function normalizeChapters(chapters) {
  const result = {}
  Object.keys(chapters).forEach((key) => {
    result[key] = (chapters[key] || []).map((q) => ({
      question_number: q.question_number || '',
      question_type: q.question_type || '单选题',
      question_text: q.question_text || '',
      options: { A: '', B: '', C: '', D: '', E: '', ...(q.options || {}) },
      correct_answer: q.correct_answer || '',
    }))
  })
  return result
}

function chapterName(key) {
  return draft.chapterNames[key] || `章节 ${Number(key) + 1}`
}

function setChapterName(key, value) {
  draft.chapterNames[key] = value
}

function addChapter() {
  const next = chapterKeys.value.length ? Number(chapterKeys.value[chapterKeys.value.length - 1]) + 1 : 0
  const key = String(next)
  draft.chapters[key] = []
  draft.chapterNames[key] = `章节 ${next + 1}`
  activeChapter.value = key
}

function removeChapter() {
  if (chapterKeys.value.length <= 1) {
    message.warning('至少保留一个章节')
    return
  }
  if (!window.confirm(`删除「${chapterName(activeChapter.value)}」及其中的题目？`)) return
  delete draft.chapters[activeChapter.value]
  delete draft.chapterNames[activeChapter.value]
  activeChapter.value = chapterKeys.value[0]
}

function addQuestion() {
  const list = draft.chapters[activeChapter.value]
  if (!list) return
  list.push({
    question_number: String(list.length + 1),
    question_type: '单选题',
    question_text: '',
    options: { A: '', B: '', C: '', D: '', E: '' },
    correct_answer: '',
  })
}

function removeQuestion(index) {
  draft.chapters[activeChapter.value].splice(index, 1)
}

async function save() {
  if (!draft.courseName.trim()) {
    message.warning('题库名不能为空')
    return
  }
  saving.value = true
  try {
    const data = await api.saveCourse({
      user_id: state.userId,
      course: state.bankDraft,
      course_name: draft.courseName.trim(),
      chapter_names: draft.chapterNames,
      chapters: draft.chapters,
    })
    message.success(`已保存，共 ${data.questions ?? 0} 题`)
    await refreshCourses()
    await load()
  } catch (err) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="editor-head">
      <n-button quaternary size="small" @click="setView('banks')">
        <template #icon><ArrowLeft :size="14" :stroke-width="1.8" /></template>
        题库列表
      </n-button>
      <n-button type="primary" size="small" :loading="saving" @click="save">保存</n-button>
    </div>

    <n-card v-if="!loading" size="small">
      <p class="section-title" style="margin: 0 0 6px">题库名称</p>
      <n-input v-model:value="draft.courseName" size="small" placeholder="题库名称" />

      <div class="chapter-bar">
        <n-select
          v-model:value="activeChapter"
          :options="chapterKeys.map((k) => ({ label: chapterName(k), value: k }))"
          size="small"
          style="flex: 1; min-width: 160px"
        />
        <n-button size="small" quaternary @click="addChapter">
          <template #icon><Plus :size="13" :stroke-width="1.8" /></template>
          章节
        </n-button>
        <n-button size="small" quaternary type="error" @click="removeChapter">
          <template #icon><Trash2 :size="13" :stroke-width="1.8" /></template>
          删章节
        </n-button>
      </div>

      <n-input
        :value="draft.chapterNames[activeChapter] || ''"
        size="small"
        placeholder="章节名称，例如 第一章 马克思主义"
        style="margin-top: 8px"
        @update:value="(v) => setChapterName(activeChapter, v)"
      />

      <div v-if="currentList.length" class="question-list">
        <div v-for="(question, index) in currentList" :key="index" class="question-editor">
          <div class="q-head">
            <n-input v-model:value="question.question_number" size="tiny" placeholder="题号" style="width: 80px" />
            <n-select
              v-model:value="question.question_type"
              size="tiny"
              :options="[
                { label: '单选题', value: '单选题' },
                { label: '多选题', value: '多选题' },
              ]"
              style="width: 96px"
            />
            <div style="flex: 1" />
            <n-button size="tiny" quaternary type="error" @click="removeQuestion(index)">
              <template #icon><Trash2 :size="12" :stroke-width="1.8" /></template>
              删除
            </n-button>
          </div>

          <n-input
            v-model:value="question.question_text"
            type="textarea"
            :rows="2"
            size="small"
            placeholder="题干"
            style="margin-bottom: 8px"
          />

          <div class="option-edit-grid">
            <n-input
              v-for="key in optionKeys"
              :key="key"
              v-model:value="question.options[key]"
              size="small"
              placeholder="选项内容"
            >
              <template #prefix>
                <span class="opt-prefix">{{ key }}</span>
              </template>
            </n-input>
          </div>

          <div class="answer-row">
            <span class="answer-label">正确答案</span>
            <n-input
              v-model:value="question.correct_answer"
              size="small"
              placeholder="如 A 或 AB"
              style="width: 110px"
            />
          </div>
        </div>
      </div>
      <div v-else class="empty-hint" style="padding: 16px 0">这个章节还没有题目。</div>

      <n-button size="small" dashed block style="margin-top: 10px" @click="addQuestion">
        <template #icon><Plus :size="13" :stroke-width="1.8" /></template>
        添加题目
      </n-button>
    </n-card>

    <n-spin v-else size="small" style="display: block; margin: 40px auto" />
  </div>
</template>

<style scoped>
.editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.chapter-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}

.question-list {
  margin-top: 14px;
}

.q-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.opt-prefix {
  font-size: 11px;
  color: var(--faint);
}

.answer-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.answer-label {
  font-size: 12px;
  color: var(--muted);
}
</style>
