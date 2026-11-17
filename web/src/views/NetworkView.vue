<script setup>
import { onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { Copy, RefreshCw } from 'lucide-vue-next'
import { loadNetworkInfo, state } from '../stores/app'

const message = useMessage()
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    await loadNetworkInfo()
  } catch (err) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

async function copy(text) {
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  } catch {
    message.warning('复制失败，请手动选中')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="net-head">
      <div>
        <h2 class="net-title">手机访问</h2>
        <p class="net-sub">手机和电脑连同一个 Wi-Fi 或热点，在手机浏览器打开下面任一地址。</p>
      </div>
      <n-button size="small" quaternary :loading="loading" @click="load">
        <template #icon><RefreshCw :size="14" :stroke-width="1.8" /></template>
        重新检测
      </n-button>
    </div>

    <n-card v-if="state.networkInfo" size="small">
      <n-list hoverable>
        <n-list-item>
          <n-thing title="本机地址">
            <template #description>
              <n-text code>{{ state.networkInfo.local_url }}</n-text>
            </template>
          </n-thing>
          <template #suffix>
            <n-button size="tiny" quaternary @click="copy(state.networkInfo.local_url)">
              <template #icon><Copy :size="13" :stroke-width="1.8" /></template>
            </n-button>
          </template>
        </n-list-item>
        <n-list-item v-for="(item, index) in state.networkInfo.lan_urls" :key="item.url">
          <n-thing :title="`网卡 ${index + 1}`">
            <template #description>
              <n-text code>{{ item.url }}</n-text>
              <n-text depth="3" style="margin-left: 8px; font-size: 12px">{{ item.name }}</n-text>
            </template>
          </n-thing>
          <template #suffix>
            <n-button size="tiny" quaternary @click="copy(item.url)">
              <template #icon><Copy :size="13" :stroke-width="1.8" /></template>
            </n-button>
          </template>
        </n-list-item>
      </n-list>
      <n-alert v-if="!state.networkInfo.lan_urls?.length" type="warning" :bordered="false" style="margin-top: 10px">
        暂时没检测到可用的局域网 IPv4 地址，确认电脑已连 Wi-Fi、有线局域网或热点。
      </n-alert>
      <p class="empty-hint" style="margin-top: 12px">{{ state.networkInfo.hint }}</p>
    </n-card>
    <n-card v-else size="small">
      <n-spin size="small" />
      <span class="empty-hint" style="margin-left: 8px">正在检测网络…</span>
    </n-card>
  </div>
</template>

<style scoped>
.net-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.net-title {
  font-size: 16px;
  font-weight: 500;
  margin: 0;
  color: var(--text);
}

.net-sub {
  font-size: 12px;
  color: var(--faint);
  margin: 4px 0 0;
  line-height: 1.6;
}

.url-label {
  flex: 0 0 68px;
  font-size: 12px;
  color: var(--muted);
}

.url-text {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--primary);
  word-break: break-all;
}

.url-text small {
  display: block;
  color: var(--faint);
  margin-top: 2px;
}
</style>
