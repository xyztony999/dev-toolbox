<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { invoke } from './bridge'
import { setTheme, themePref, type ThemePref } from './theme'

interface ClientInfo {
  version: string
  scanToolVersion: string
  configDir: string
  projectStore: string
  runtimeState: string
  gitIdentity: string
  os: string
  arch: string
}

const theme = themePref
const info = ref<ClientInfo>()
const errorText = ref('')

const options: Array<{ id: ThemePref; label: string }> = [
  { id: 'system', label: '跟随系统' },
  { id: 'light', label: '浅色' },
  { id: 'dark', label: '深色' },
]

onMounted(async () => {
  try {
    info.value = await invoke<ClientInfo>('app.info')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
})

function choose(pref: ThemePref) {
  setTheme(pref)
}

async function copyPath(path: string) {
  try {
    await navigator.clipboard.writeText(path)
    message.success('路径已复制')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '复制失败')
  }
}

async function openConfig() {
  if (!info.value) return
  try {
    await invoke('app.open', { kind: 'config', target: info.value.configDir })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />
  <div class="doc">
    <section class="doc-section">
      <h2>外观</h2>
      <p class="doc-note">跟随系统时，和 Windows 的浅色或深色一起变。</p>
      <div class="theme-pick" role="radiogroup" aria-label="外观">
        <button v-for="item in options" :key="item.id" type="button" role="radio" :aria-checked="theme === item.id" :class="{ on: theme === item.id }" @click="choose(item.id)">{{ item.label }}</button>
      </div>
    </section>
    <section class="doc-section">
      <h2>本机数据</h2>
      <p class="doc-note">登记、运行时和 Git 身份都写在这个目录，不进项目仓库。</p>
      <div v-if="info" class="kv">
        <div class="kv-row"><span class="kv-k">配置目录</span><span class="kv-v">{{ info.configDir }}</span><span class="kv-actions"><button type="button" @click="copyPath(info.configDir)">复制</button><button type="button" @click="openConfig">打开</button></span></div>
        <div class="kv-row"><span class="kv-k">项目登记</span><span class="kv-v">{{ info.projectStore }}</span><button type="button" @click="copyPath(info.projectStore)">复制</button></div>
        <div class="kv-row"><span class="kv-k">运行时状态</span><span class="kv-v">{{ info.runtimeState }}</span><button type="button" @click="copyPath(info.runtimeState)">复制</button></div>
        <div class="kv-row"><span class="kv-k">Git 身份</span><span class="kv-v">{{ info.gitIdentity }}</span><button type="button" @click="copyPath(info.gitIdentity)">复制</button></div>
      </div>
    </section>
  </div>
</template>
