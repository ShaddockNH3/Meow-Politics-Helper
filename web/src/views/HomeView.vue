<script setup>
import { computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  BookOpen,
  Keyboard,
  Library,
  PenLine,
  RotateCcw,
  Settings,
  Smartphone,
  Upload,
  Zap,
} from 'lucide-vue-next'
import {
  MODES,
  chaptersOf,
  chooseMode,
  customCourseList,
  detectResume,
  courseList,
  refreshCourses,
  resumeQuiz,
  selectCourse,
  setView,
  state,
} from '../stores/app'

const message = useMessage()

const builtin = computed(() => courseList())
const custom = computed(() => customCourseList())

const modeIcons = { zap: Zap, pen: PenLine, rotate: RotateCcw }

onMounted(async () => {
  try {
    await refreshCourses()
    detectResume()
  } catch (err) {
    message.error(err.message)
  }
})

function chapterSummary(course) {
  const list = chaptersOf(course)
  return `${list.length} 章 · ${course.question_count} 题`
}

function openBanks() {
  setView('banks')
}

function continueLast() {
  if (!resumeQuiz()) message.info('没有可继续的答题记录')
}
</script>

<template>
  <div class="home-grid">
    <section>
      <p class="section-title">选择课程</p>
      <div class="course-grid">
        <n-card
          v-for="course in builtin"
          :key="course.key"
          size="small"
          hoverable
          class="card-click course-card"
          :class="{ 'is-active': state.selectedCourse === course.key }"
          @click="selectCourse(course.key)"
        >
          <div class="course-name">{{ course.name }}</div>
          <div class="course-meta">
            <n-tag size="small" :bordered="false" :type="state.selectedCourse === course.key ? 'info' : 'default'">
              {{ chapterSummary(course) }}
            </n-tag>
          </div>
        </n-card>
        <n-card
          v-for="course in custom"
          :key="course.key"
          size="small"
          hoverable
          class="card-click course-card"
          :class="{ 'is-active': state.selectedCourse === course.key }"
          @click="selectCourse(course.key)"
        >
          <div class="course-name">{{ course.name }}</div>
          <div class="course-meta">
            <n-tag size="small" :bordered="false" type="warning">自定义</n-tag>
            <n-tag size="small" :bordered="false">{{ course.question_count }} 题</n-tag>
          </div>
        </n-card>
        <n-card size="small" hoverable class="card-click course-card course-add" @click="setView('import')">
          <div class="add-inner">+ 新建自定义题库</div>
        </n-card>
      </div>
      <div v-if="!builtin.length && !custom.length" class="empty-hint" style="margin-top: 10px">
        暂时没有可用题库，稍等一下或刷新页面。
      </div>
    </section>

    <section>
      <p class="section-title">选择学习模式</p>

      <n-alert
        v-if="state.canResume"
        type="info"
        :bordered="false"
        style="margin-bottom: 10px"
        closable
      >
        <div style="display: flex; align-items: center; justify-content: space-between; gap: 10px">
          <span>上次的答题还没做完</span>
          <n-button size="tiny" type="primary" @click="continueLast">继续</n-button>
        </div>
      </n-alert>

      <div class="mode-stack">
        <n-card
          v-for="mode in MODES"
          :key="mode.key"
          size="small"
          hoverable
          class="card-click"
          @click="chooseMode(mode.key)"
        >
          <div class="mode-card">
            <span class="mode-icon">
              <component :is="modeIcons[mode.icon]" :size="17" :stroke-width="1.7" />
            </span>
            <div>
              <div class="mode-title">{{ mode.name }}</div>
              <div class="mode-desc">{{ mode.desc }}</div>
            </div>
          </div>
        </n-card>
      </div>

      <div class="entry-grid">
        <div class="entry-tile" @click="setView('import')">
          <Upload :size="15" :stroke-width="1.7" />导入题库
        </div>
        <div class="entry-tile" @click="setView('network')">
          <Smartphone :size="15" :stroke-width="1.7" />手机访问
        </div>
        <div class="entry-tile" @click="setView('guide')">
          <BookOpen :size="15" :stroke-width="1.7" />使用指南
        </div>
        <div class="entry-tile" @click="openBanks">
          <Library :size="15" :stroke-width="1.7" />题库管理
        </div>
        <div class="entry-tile" @click="setView('shortcuts')">
          <Keyboard :size="15" :stroke-width="1.7" />快捷键
        </div>
        <div class="entry-tile" @click="setView('settings')">
          <Settings :size="15" :stroke-width="1.7" />设置
        </div>
      </div>
    </section>
  </div>
</template>
