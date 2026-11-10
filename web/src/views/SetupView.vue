<script setup>
import { computed } from 'vue'
import { useMessage } from 'naive-ui'
import {
  chapters,
  currentCourse,
  modeInfo,
  selectAllChapters,
  startMode,
  state,
  toggleAutoDelete,
  toggleChapter,
} from '../stores/app'

const message = useMessage()
const course = computed(() => currentCourse.value)
const allSelected = computed(
  () => chapters.value.length > 0 && state.selectedChapters.length === chapters.value.length,
)
const canStart = computed(
  () => state.mode === 'incorrectReview' || state.selectedChapters.length > 0,
)

async function start() {
  try {
    await startMode()
  } catch (err) {
    message.error(err.message)
  }
}
</script>

<template>
  <div>
    <div class="setup-head">
      <div>
        <h2 class="setup-title">{{ modeInfo?.name }}</h2>
        <p class="setup-sub">
          {{ course?.name || '未选择题库' }}
          <span v-if="course"> · {{ course.question_count }} 题</span>
        </p>
      </div>
      <n-button quaternary size="small" @click="state.view = 'home'">换课程</n-button>
    </div>

    <n-card size="small" style="margin-bottom: 14px">
      <div v-if="state.mode !== 'incorrectReview'">
        <div class="setup-row-head">
          <span class="section-title" style="margin: 0">选择章节</span>
          <n-button text size="tiny" type="primary" @click="selectAllChapters">
            {{ allSelected ? '取消全选' : '全选' }}
          </n-button>
        </div>
        <div class="chapter-grid">
          <button
            v-for="chapter in chapters"
            :key="chapter.key"
            class="chapter-chip"
            :class="{ active: state.selectedChapters.includes(chapter.key) }"
            @click="toggleChapter(chapter.key)"
          >
            {{ chapter.name }}
          </button>
        </div>
      </div>

      <div v-else>
        <p class="section-title" style="margin: 0 0 8px">错题范围</p>
        <p class="empty-hint" style="margin: 0">
          错题本按课程单独保存，本次回顾「{{ course?.name }}」的全部错题，顺序随机。
        </p>
      </div>
    </n-card>

    <n-card v-if="state.mode !== 'incorrectReview'" size="small" style="margin-bottom: 14px">
      <p class="section-title" style="margin: 0 0 8px">题目顺序</p>
      <n-radio-group v-model:value="state.order" size="small">
        <n-radio-button value="sequential">正序</n-radio-button>
        <n-radio-button value="random">随机</n-radio-button>
      </n-radio-group>
    </n-card>

    <n-card v-if="state.mode === 'incorrectReview'" size="small" style="margin-bottom: 14px">
      <div class="switch-row">
        <div>
          <div class="switch-title">答对后自动移出错题本</div>
          <div class="mode-desc">开启后，回顾中答对的题会从错题本里移除，只影响错题回顾。</div>
        </div>
        <n-switch :value="state.autoDeleteIncorrect" @update:value="toggleAutoDelete" />
      </div>
    </n-card>

    <div class="mobile-actions">
      <n-button type="primary" size="large" block :disabled="!canStart" @click="start">
        开始{{ modeInfo?.name }}
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.setup-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 16px;
}

.setup-title {
  font-size: 16px;
  font-weight: 500;
  margin: 0;
  color: var(--text);
}

.setup-sub {
  font-size: 12px;
  color: var(--faint);
  margin: 4px 0 0;
}

.setup-row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.switch-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.switch-title {
  font-size: 13px;
  color: var(--text);
}
</style>
