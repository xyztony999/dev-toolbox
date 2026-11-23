<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { invoke } from './bridge'
import BrandMark from './BrandMark.vue'

interface ClientInfo {
  version: string
  scanToolVersion: string
  configDir: string
  os: string
  arch: string
}

const info = ref<ClientInfo>()
const errorText = ref('')
const repo = 'https://github.com/xyztony999/dev-toolbox'

onMounted(async () => {
  try {
    info.value = await invoke<ClientInfo>('app.info')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
})

async function openRepo(target: string) {
  try {
    await invoke('app.open', { kind: 'repo', target })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />
  <div class="doc">
    <div class="about-head">
      <BrandMark class="about-mark" :size="36" />
      <div>
        <h2>DevToolbox <span v-if="info">{{ info.version }}</span></h2>
        <p class="doc-note">管理项目用哪套运行时、提交用哪个 Git 身份。</p>
      </div>
    </div>
    <section class="doc-section">
      <h2>这台机器</h2>
      <div v-if="info" class="kv">
        <div class="kv-row"><span class="kv-k">客户端</span><span class="kv-v">{{ info.version }}</span></div>
        <div class="kv-row"><span class="kv-k">扫描器</span><span class="kv-v">v{{ info.scanToolVersion }}</span></div>
        <div class="kv-row"><span class="kv-k">系统</span><span class="kv-v">{{ info.os }} / {{ info.arch }}</span></div>
        <div class="kv-row"><span class="kv-k">界面宿主</span><span class="kv-v">Go WebView</span></div>
        <div class="kv-row"><span class="kv-k">数据目录</span><span class="kv-v">{{ info.configDir }}</span></div>
      </div>
    </section>
    <section class="doc-section">
      <h2>来源</h2>
      <div class="kv">
        <div class="kv-row"><span class="kv-k">仓库</span><button type="button" class="linkish" @click="openRepo(repo)">github.com/xyztony999/dev-toolbox</button></div>
        <div class="kv-row"><span class="kv-k">发布</span><button type="button" class="linkish" @click="openRepo(repo + '/releases')">Releases</button></div>
        <div class="kv-row"><span class="kv-k">许可</span><span class="kv-v">MIT</span></div>
      </div>
    </section>
  </div>
</template>
