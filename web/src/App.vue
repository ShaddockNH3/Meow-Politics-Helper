<script setup>
import { zhCN, dateZhCN } from 'naive-ui'
import { theme } from './theme'
import AppShell from './components/AppShell.vue'
</script>

<template>
  <n-config-provider :theme-overrides="theme" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <AppShell />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
