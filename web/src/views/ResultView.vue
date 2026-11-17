<script setup>
import { computed } from 'vue'
import { CheckCircle2 } from 'lucide-vue-next'
import { accuracy, modeInfo, resetForHome, state, totalQuestions } from '../stores/app'

const wrong = computed(() => state.results.answered - state.results.correct)
const isQuizLike = computed(() => state.mode !== 'quickReview')
const gridBreakpoints = { xs: 0, s: 481, m: 769, l: 1025, xl: 1281, '2xl': 1537 }
</script>

<template>
  <div class="result-wrap">
    <n-card>
      <n-space align="center" :size="12" style="margin-bottom: 20px">
        <n-icon :size="24" :component="CheckCircle2" color="#3f7a52" />
        <div>
          <n-text strong style="font-size: 16px">这轮{{ modeInfo?.name }}结束啦</n-text>
          <div class="mode-desc">本轮共 {{ totalQuestions }} 道题</div>
        </div>
      </n-space>

      <n-grid
        v-if="isQuizLike"
        :cols="'2 s:4'"
        :breakpoints="gridBreakpoints"
        :x-gap="12"
        :y-gap="12"
        responsive="screen"
        item-responsive
      >
        <n-gi>
          <n-card size="small" embedded>
            <n-statistic label="已作答" :value="state.results.answered" />
          </n-card>
        </n-gi>
        <n-gi>
          <n-card size="small" embedded>
            <n-statistic label="答对" :value="state.results.correct" />
          </n-card>
        </n-gi>
        <n-gi>
          <n-card size="small" embedded>
            <n-statistic label="答错" :value="wrong" />
          </n-card>
        </n-gi>
        <n-gi>
          <n-card size="small" embedded>
            <n-statistic label="正确率" :value="state.results.answered ? accuracy + '%' : '—'" />
          </n-card>
        </n-gi>
      </n-grid>

      <n-button type="primary" size="large" block style="margin-top: 20px" @click="resetForHome">
        返回主菜单
      </n-button>
    </n-card>
  </div>
</template>

<style scoped>
.result-wrap {
  max-width: 660px;
  margin: 0 auto;
}
</style>
