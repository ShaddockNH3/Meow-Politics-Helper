<template>
  <div>
    <n-card size="small" style="margin-bottom: 12px">
      <div class="guide-title">快速开始</div>
      <ul class="guide-list">
        <li>打开程序会自动拉起浏览器；不需要装任何环境，题库已经打包在程序里。</li>
        <li>第一次输入一个用户 ID 就建好本地账号，之后自动记住。</li>
        <li>选课程 → 选模式 → 选章节 → 开始。</li>
      </ul>
    </n-card>

    <n-card size="small" style="margin-bottom: 12px">
      <div class="guide-title">三种模式</div>
      <ul class="guide-list">
        <li><strong>速刷模式</strong>：直接看题和答案，适合考前快速过一遍。</li>
        <li><strong>答题模式</strong>：认真作答，统计和错题自动记录，答错的题进对应课程的错题本。</li>
        <li><strong>错题回顾</strong>：只刷错题；可以开启「答对后自动移出错题本」。</li>
      </ul>
    </n-card>

    <n-card size="small" style="margin-bottom: 12px">
      <div class="guide-title">键盘操作</div>
      <ul class="guide-list">
        <li><strong>A</strong> / <strong>B</strong> / <strong>C</strong> / <strong>D</strong> / <strong>E</strong> 选择或取消选项。</li>
        <li><strong>Enter</strong> 提交当前题；答案出来后按一次进入下一题。</li>
        <li><strong>←</strong> 上一题，<strong>→</strong> 下一题，<strong>Esc</strong> 结束本轮。</li>
        <li>正在输入框里打字时快捷键不生效；按键可以在「快捷键」页面改。</li>
      </ul>
    </n-card>

    <n-card size="small">
      <div class="guide-title">题库与数据</div>
      <ul class="guide-list">
        <li>内置题库随程序发布；自定义题库可以手动编辑，或粘贴 JSON 导入。</li>
        <li>想用 AI 出题：在「导入题库」页复制提示词，发给任意 AI，再把返回的 JSON 粘回来。</li>
        <li>所有数据保存在程序目录的 <code>user_data/</code> 下，换电脑时整个目录拷过去即可。</li>
        <li>分享程序给别人时，不要把自己的 <code>user_data/</code> 一起打包。</li>
      </ul>
    </n-card>
  </div>
</template>

<style scoped>
.guide-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  margin-bottom: 8px;
}

.guide-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.85;
  color: #374151;
}

.guide-list code {
  background: #f1f4f9;
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 12px;
}
</style>
