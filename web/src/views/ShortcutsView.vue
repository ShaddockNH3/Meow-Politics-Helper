<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  defaultShortcuts,
  loadShortcuts,
  normalizeKey,
  resetShortcuts,
  saveShortcuts,
} from '../composables/useShortcuts'
import { state } from '../stores/app'

const message = useMessage()
const shortcuts = ref(loadShortcuts(state.userId))
const capturing = ref(null)
const optionKeys = ['A', 'B', 'C', 'D', 'E']

const navItems = [
  { key: 'submit', label: '提交 / 下一题' },
  { key: 'previous', label: '上一题' },
  { key: 'next', label: '下一题' },
  { key: 'back', label: '结束 / 返回' },
]

function startCapture(kind, key) {
  capturing.value = { kind, key }
}

function isCapturing(kind, key) {
  return capturing.value?.kind === kind && capturing.value?.key === key
}

function onKeydown(event) {
  if (!capturing.value) return
  event.preventDefault()
  const key = normalizeKey(event.key)
  if (!key) return
  if (capturing.value.kind === 'option') {
    shortcuts.value.options[capturing.value.key] = key
  } else {
    shortcuts.value[capturing.value.kind] = key
  }
  capturing.value = null
}

function save() {
  saveShortcuts(state.userId, shortcuts.value)
  message.success('快捷键已保存')
}

function restore() {
  shortcuts.value = resetShortcuts(state.userId)
  message.success('已恢复默认')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div>
    <n-card size="small" style="margin-bottom: 12px">
      <div class="sc-title">选项键</div>
      <div class="sc-grid">
        <div
          v-for="key in optionKeys"
          :key="key"
          class="kbd-cell"
          :class="{ capturing: isCapturing('option', key) }"
          @click="startCapture('option', key)"
        >
          <span>{{ key }} 选项</span>
          <span class="kbd-value">{{ shortcuts.options[key] }}</span>
        </div>
      </div>
    </n-card>

    <n-card size="small" style="margin-bottom: 12px">
      <div class="sc-title">导航键</div>
      <div class="sc-grid">
        <div
          v-for="item in navItems"
          :key="item.key"
          class="kbd-cell"
          :class="{ capturing: isCapturing('action', item.key) }"
          @click="startCapture('action', item.key)"
        >
          <span>{{ item.label }}</span>
          <span class="kbd-value">{{ shortcuts[item.key] }}</span>
        </div>
      </div>
      <p class="empty-hint" style="margin: 10px 0 0">
        {{ capturing ? '正在等待按键…按下想绑定的键' : '点一个格子，然后按下想要的键。' }}
      </p>
    </n-card>

    <div style="display: grid; gap: 8px">
      <n-button type="primary" block @click="save">保存快捷键</n-button>
      <n-button block quaternary @click="restore">恢复默认（{{ defaultShortcuts.submit }} / {{ defaultShortcuts.back }} 等）</n-button>
    </div>
  </div>
</template>

<style scoped>
.sc-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  margin-bottom: 10px;
}

.sc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px;
}
</style>
