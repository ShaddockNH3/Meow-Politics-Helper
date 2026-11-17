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

const gridBreakpoints = { xs: 0, s: 481, m: 769, l: 1025, xl: 1281, '2xl': 1537 }

const entries = [
  { key: 'import', label: '导入题库', icon: Upload, action: () => setView('import') },
  { key: 'network', label: '手机访问', icon: Smartphone, action: () => setView('network') },
  { key: 'guide', label: '使用指南', icon: BookOpen, action: () => setView('guide') },
  { key: 'banks', label: '题库管理', icon: Library, action: () => setView('banks') },
  { key: 'shortcuts', label: '快捷键', icon: Keyboard, action: () => setView('shortcuts') },
  { key: 'settings', label: '设置', icon: Settings, action: () => setView('settings') },
]

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

function continueLast() {
  if (!resumeQuiz()) message.info('没有可继续的答题记录')
}
</script>

<template>
  <div class="home-grid">
    <section>
      <p class="section-title">选择课程</p>
      <n-grid
        :cols="'1 s:2'"
        :breakpoints="gridBreakpoints"
        :x-gap="10"
        :y-gap="10"
        responsive="screen"
        item-responsive
      >
        <n-gi v-for="course in builtin" :key="course.key">
          <n-card
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
        </n-gi>
        <n-gi v-for="course in custom" :key="course.key">
          <n-card
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
        </n-gi>
        <n-gi :span="'1 s:2'">
          <n-card size="small" hoverable class="card-click course-card course-add" @click="setView('import')">
            <div class="add-inner">+ 新建自定义题库</div>
          </n-card>
        </n-gi>
      </n-grid>
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

      <n-grid :cols="1" :y-gap="10">
        <n-gi v-for="mode in MODES" :key="mode.key">
          <n-card size="small" hoverable class="card-click" @click="chooseMode(mode.key)">
            <n-space align="center" :size="12">
              <n-icon :size="18" :component="modeIcons[mode.icon]" color="#1f3f66" />
              <div>
                <n-text strong>{{ mode.name }}</n-text>
                <div class="mode-desc">{{ mode.desc }}</div>
              </div>
            </n-space>
          </n-card>
        </n-gi>
      </n-grid>

      <n-grid
        :cols="'2 s:2 m:3'"
        :breakpoints="gridBreakpoints"
        :x-gap="10"
        :y-gap="10"
        responsive="screen"
        item-responsive
        style="margin-top: 16px"
      >
        <n-gi v-for="entry in entries" :key="entry.key">
          <n-card size="small" hoverable class="card-click" @click="entry.action">
            <n-space align="center" :size="8">
              <n-icon :size="15" :component="entry.icon" color="#8a93a5" />
              <n-text>{{ entry.label }}</n-text>
            </n-space>
          </n-card>
        </n-gi>
      </n-grid>
    </section>
  </div>
</template>
