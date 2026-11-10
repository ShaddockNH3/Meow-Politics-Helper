<script setup>
import { computed, reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { Copy, CopyCheck } from 'lucide-vue-next'
import { api } from '../api'
import { customCourseList, refreshCourses, state } from '../stores/app'
import { DEFAULT_PROMPT } from '../prompt'

const message = useMessage()
const copied = ref(false)
const showPrompt = ref(false)
const submitting = ref(false)

const custom = computed(() => customCourseList())

const form = reactive({
  target_mode: 'new',
  target_course: '',
  course_name: '',
  course_key: '',
  chapter_key: '',
  chapter_name: '新章节',
  json_text: '',
})

async function copyPrompt() {
  try {
    await navigator.clipboard.writeText(DEFAULT_PROMPT)
    copied.value = true
    message.success('提示词已复制')
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    message.warning('复制失败，请手动选中文本')
    showPrompt.value = true
  }
}

async function submit() {
  if (!form.json_text.trim()) {
    message.warning('先粘贴题库 JSON')
    return
  }
  submitting.value = true
  try {
    const data = await api.importJSON({
      user_id: state.userId,
      target_mode: form.target_mode,
      target_course: form.target_course,
      course_key: form.course_key,
      course_name: form.course_name || '自定义题库',
      chapter_key: form.chapter_key,
      chapter_name: form.chapter_name,
      json_text: form.json_text,
    })
    message.success(`已导入 ${data.questions || 0} 题`)
    form.json_text = ''
    await refreshCourses()
    state.selectedCourse = data.course?.course_key || state.selectedCourse
  } catch (err) {
    message.error(err.message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div>
    <n-card size="small" style="margin-bottom: 14px">
      <div class="prompt-head">
        <div>
          <div class="block-title">用 AI 生成题库</div>
          <div class="block-sub">
            复制提示词，连同学习材料发给任意 AI，把返回的 JSON 粘贴到下面导入。
          </div>
        </div>
        <n-button size="small" :type="copied ? 'success' : 'primary'" ghost @click="copyPrompt">
          <template #icon>
            <CopyCheck v-if="copied" :size="14" :stroke-width="1.8" />
            <Copy v-else :size="14" :stroke-width="1.8" />
          </template>
          {{ copied ? '已复制' : '复制提示词' }}
        </n-button>
      </div>
      <n-button text size="tiny" type="primary" style="margin-top: 8px" @click="showPrompt = !showPrompt">
        {{ showPrompt ? '收起提示词' : '查看提示词全文' }}
      </n-button>
      <pre v-if="showPrompt" class="prompt-box">{{ DEFAULT_PROMPT }}</pre>
    </n-card>

    <n-card size="small">
      <div class="block-title">粘贴 JSON 导入题库</div>
      <div class="block-sub">字段：question_number、question_type、question_text、options、correct_answer。</div>

      <n-radio-group v-model:value="form.target_mode" size="small" style="margin: 12px 0">
        <n-radio-button value="new">新建题库</n-radio-button>
        <n-radio-button value="existing" :disabled="!custom.length">导入到已有题库</n-radio-button>
      </n-radio-group>

      <template v-if="form.target_mode === 'existing'">
        <n-select
          v-model:value="form.target_course"
          :options="custom.map((c) => ({ label: c.name, value: c.key }))"
          placeholder="选择自定义题库"
          size="small"
        />
      </template>
      <template v-else>
        <div class="form-grid">
          <n-input v-model:value="form.course_name" placeholder="题库名称，例如 马原期末" size="small" />
          <n-input v-model:value="form.course_key" placeholder="题库标识（可留空）" size="small" />
        </div>
      </template>

      <div class="form-grid" style="margin-top: 8px">
        <n-input v-model:value="form.chapter_key" placeholder="章节号（可留空自动新章节）" size="small" />
        <n-input v-model:value="form.chapter_name" placeholder="章节名，例如 新章节" size="small" />
      </div>

      <n-input
        v-model:value="form.json_text"
        type="textarea"
        :rows="10"
        placeholder='[
  {
    "question_number": "AI-001",
    "question_type": "单选题",
    "question_text": "题干",
    "options": {"A": "选项A", "B": "选项B"},
    "correct_answer": "A"
  }
]'
        class="json-input"
        style="margin-top: 10px"
      />

      <n-button type="primary" block size="large" style="margin-top: 12px" :loading="submitting" @click="submit">
        导入题库
      </n-button>
    </n-card>
  </div>
</template>

<style scoped>
.block-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}

.block-sub {
  font-size: 12px;
  color: var(--faint);
  line-height: 1.6;
  margin-top: 3px;
}

.prompt-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.json-input :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 480px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
