<script setup>
import { computed } from 'vue'
import { CheckCircle2 } from 'lucide-vue-next'
import { accuracy, modeInfo, resetForHome, state, totalQuestions } from '../stores/app'

const wrong = computed(() => state.results.answered - state.results.correct)
const isQuizLike = computed(() => state.mode !== 'quickReview')
</script>

<template>
  <div class="result-wrap">
    <n-card>
      <div class="result-head">
        <CheckCircle2 :size="22" :stroke-width="1.6" style="color: var(--ok)" />
        <div>
          <div class="result-title">这轮{{ modeInfo?.name }}结束啦</div>
          <div class="result-sub">本轮共 {{ totalQuestions }} 道题</div>
        </div>
      </div>

      <div v-if="isQuizLike" class="result-grid">
        <div class="result-item">
          <div class="result-label">已作答</div>
          <div class="result-value">{{ state.results.answered }}</div>
        </div>
        <div class="result-item">
          <div class="result-label">答对</div>
          <div class="result-value" style="color: var(--ok)">{{ state.results.correct }}</div>
        </div>
        <div class="result-item">
          <div class="result-label">答错</div>
          <div class="result-value" style="color: var(--bad)">{{ wrong }}</div>
        </div>
        <div class="result-item">
          <div class="result-label">正确率</div>
          <div class="result-value">{{ state.results.answered ? accuracy + '%' : '—' }}</div>
        </div>
      </div>

      <n-button type="primary" size="large" block style="margin-top: 18px" @click="resetForHome">
        返回主菜单
      </n-button>
    </n-card>
  </div>
</template>

<style scoped>
.result-wrap {
  max-width: 620px;
  margin: 0 auto;
}

.result-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
}

.result-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--text);
}

.result-sub {
  font-size: 12px;
  color: var(--faint);
  margin-top: 3px;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.result-item {
  background: #f6f8fb;
  border-radius: 8px;
  padding: 12px;
}

.result-label {
  font-size: 12px;
  color: var(--faint);
}

.result-value {
  font-size: 22px;
  font-weight: 500;
  color: var(--text);
  margin-top: 4px;
}

@media (max-width: 480px) {
  .result-grid {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
