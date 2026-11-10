<script setup>
import { ref } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { LogOut, Trash2 } from 'lucide-vue-next'
import { clearUserData, logout, state } from '../stores/app'

const dialog = useDialog()
const message = useMessage()
const busy = ref(false)

function confirmClear() {
  dialog.warning({
    title: '清理当前用户数据',
    content: '会清空这个账号的错题本和答题统计（文件会备份成 .bak）。自定义题库会保留。',
    positiveText: '确定清理',
    negativeText: '取消',
    onPositiveClick: async () => {
      busy.value = true
      try {
        await clearUserData()
        message.success('已清理')
      } catch (err) {
        message.error(err.message)
      } finally {
        busy.value = false
      }
    },
  })
}

function confirmLogout() {
  dialog.info({
    title: '退出登录',
    content: '只是切回登录页，本地数据不会删除。',
    positiveText: '退出',
    negativeText: '取消',
    onPositiveClick: () => logout(),
  })
}
</script>

<template>
  <div>
    <n-card size="small" style="margin-bottom: 12px">
      <div class="row-between">
        <div>
          <div class="set-title">当前用户</div>
          <div class="set-sub">数据保存在本机 user_data/{{ state.userId }}</div>
        </div>
        <n-tag :bordered="false">{{ state.userId }}</n-tag>
      </div>
    </n-card>

    <n-card size="small" style="margin-bottom: 12px">
      <div class="set-title">数据管理</div>
      <div class="set-sub">
        清理错题本与答题统计，清理前会自动备份为 .bak 文件；自定义题库不受影响。
      </div>
      <n-button type="error" ghost block style="margin-top: 12px" :loading="busy" @click="confirmClear">
        <template #icon><Trash2 :size="14" :stroke-width="1.8" /></template>
        清理当前用户数据
      </n-button>
    </n-card>

    <n-button block quaternary @click="confirmLogout">
      <template #icon><LogOut :size="14" :stroke-width="1.8" /></template>
      退出登录 / 切换用户
    </n-button>
  </div>
</template>

<style scoped>
.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.set-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}

.set-sub {
  font-size: 12px;
  color: var(--faint);
  line-height: 1.6;
  margin-top: 3px;
  word-break: break-all;
}
</style>
