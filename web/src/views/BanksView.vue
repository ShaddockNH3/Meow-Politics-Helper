<script setup>
import { computed, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { Pencil, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import { customCourseList, refreshCourses, setView, state } from '../stores/app'

const message = useMessage()
const busy = ref('')

const banks = computed(() => customCourseList())

onMounted(async () => {
  try {
    await refreshCourses()
  } catch (err) {
    message.error(err.message)
  }
})

function edit(course) {
  state.bankDraft = course.key
  setView('bankEditor')
}

async function remove(course) {
  if (!window.confirm(`删除题库「${course.name}」？该题库的错题记录会一起清理。`)) return
  busy.value = course.key
  try {
    await api.deleteCourse({ user_id: state.userId, course: course.key })
    message.success('已删除')
    await refreshCourses()
  } catch (err) {
    message.error(err.message)
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <div>
    <div class="banks-head">
      <div>
        <h2 class="banks-title">自定义题库</h2>
        <p class="banks-sub">共 {{ banks.length }} 个，只能编辑自己导入的题库。</p>
      </div>
      <n-button size="small" type="primary" @click="setView('import')">
        <template #icon><Plus :size="14" :stroke-width="1.8" /></template>
        新建 / 导入
      </n-button>
    </div>

    <n-card size="small">
      <div v-if="!banks.length" class="empty-hint">
        还没有自定义题库。可以在「导入题库」里粘贴 JSON，或者复制提示词让 AI 生成后再导入。
      </div>
      <div v-for="course in banks" :key="course.key" class="bank-row">
        <div>
          <div class="course-name">{{ course.name }}</div>
          <div class="mode-desc">
            {{ course.max_chapter + 1 }} 章 · {{ course.question_count }} 题
          </div>
        </div>
        <n-button size="tiny" quaternary :disabled="busy === course.key" @click="edit(course)">
          <template #icon><Pencil :size="13" :stroke-width="1.8" /></template>
          编辑
        </n-button>
        <n-button size="tiny" quaternary type="error" :loading="busy === course.key" @click="remove(course)">
          <template #icon><Trash2 :size="13" :stroke-width="1.8" /></template>
          删除
        </n-button>
      </div>
    </n-card>
  </div>
</template>

<style scoped>
.banks-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.banks-title {
  font-size: 16px;
  font-weight: 500;
  margin: 0;
  color: var(--text);
}

.banks-sub {
  font-size: 12px;
  color: var(--faint);
  margin: 4px 0 0;
}
</style>
