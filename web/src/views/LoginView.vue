<script setup>
import { ref } from 'vue'
import { login } from '../stores/app'

const id = ref('')
const busy = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await login(id.value)
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <n-card class="login-card" :bordered="true">
      <div class="login-head">
        <div class="login-title">喵喵学习小助手</div>
        <p class="login-sub">输入一个用户 ID 就能开始，练习记录只保存在这台电脑上。</p>
      </div>
      <n-input
        v-model:value="id"
        placeholder="例如 catLover123"
        size="large"
        :maxlength="50"
        @keyup.enter="submit"
      />
      <n-alert v-if="error" type="error" :bordered="false" style="margin-top: 12px">
        {{ error }}
      </n-alert>
      <n-button type="primary" size="large" block style="margin-top: 14px" :loading="busy" @click="submit">
        进入
      </n-button>
      <p class="login-foot">字母、数字、下划线、短横线，1–50 位</p>
    </n-card>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 70vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px 0;
}

.login-card {
  width: 100%;
  max-width: 380px;
}

.login-head {
  margin-bottom: 18px;
}

.login-title {
  font-size: 17px;
  font-weight: 500;
  color: var(--text);
}

.login-sub {
  font-size: 12px;
  color: var(--faint);
  line-height: 1.6;
  margin: 6px 0 0;
}

.login-foot {
  font-size: 11px;
  color: var(--faint);
  text-align: center;
  margin: 12px 0 0;
}
</style>
