<script setup>
import { computed } from 'vue'
import { Cat, Github } from 'lucide-vue-next'
import { state, goBack, setError } from '../stores/app'
import LoginView from '../views/LoginView.vue'
import HomeView from '../views/HomeView.vue'
import SetupView from '../views/SetupView.vue'
import QuizView from '../views/QuizView.vue'
import ResultView from '../views/ResultView.vue'
import ImportView from '../views/ImportView.vue'
import BanksView from '../views/BanksView.vue'
import BankEditorView from '../views/BankEditorView.vue'
import NetworkView from '../views/NetworkView.vue'
import GuideView from '../views/GuideView.vue'
import ShortcutsView from '../views/ShortcutsView.vue'
import SettingsView from '../views/SettingsView.vue'

const PROJECT_URL = 'https://github.com/ShaddockNH3/Meow-Politics-Helper'

const views = {
  home: HomeView,
  setup: SetupView,
  quiz: QuizView,
  result: ResultView,
  import: ImportView,
  banks: BanksView,
  bankEditor: BankEditorView,
  network: NetworkView,
  guide: GuideView,
  shortcuts: ShortcutsView,
  settings: SettingsView,
}

const currentView = computed(() => (state.userId ? views[state.view] || HomeView : LoginView))
const narrow = computed(() => ['import', 'guide', 'shortcuts', 'network'].includes(state.view))
const showBack = computed(() => state.userId && state.view !== 'home' && state.view !== 'quiz')
</script>

<template>
  <div class="shell">
    <header v-if="state.userId" class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <Cat class="brand-icon" :size="22" :stroke-width="1.8" />
          <span>喵喵学习小助手</span>
        </div>
        <div class="topbar-right">
          <a
            class="gh-link"
            :href="PROJECT_URL"
            target="_blank"
            rel="noopener"
            title="项目源码（GitHub）"
            aria-label="项目源码（GitHub）"
          >
            <Github :size="17" :stroke-width="1.8" />
          </a>
          <n-tag size="small" :bordered="false" round>{{ state.userId }}</n-tag>
          <n-button v-if="showBack" size="small" quaternary @click="goBack">返回</n-button>
        </div>
      </div>
    </header>

    <main class="page" :class="{ 'page-narrow': narrow }">
      <n-alert
        v-if="state.error"
        type="error"
        closable
        style="margin-bottom: 14px"
        @close="setError('')"
      >
        {{ state.error }}
      </n-alert>
      <component :is="currentView" />
    </main>

    <div v-if="state.loading" class="loading-mask">
      <n-spin size="large" />
      <p>{{ state.loadingText || '处理中…' }}</p>
    </div>
  </div>
</template>

<style scoped>
.loading-mask {
  position: fixed;
  inset: 0;
  background: rgba(244, 246, 250, 0.72);
  backdrop-filter: blur(2px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  z-index: 60;
}

.loading-mask p {
  font-size: 13px;
  color: var(--muted);
  margin: 0;
}
</style>
