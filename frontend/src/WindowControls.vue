<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

const maximized = ref(false)

async function refresh() {
  const result = await window.devToolbox?.window?.isMaximized()
  maximized.value = !!result?.maximized
}

function onResize() { void refresh() }

onMounted(() => {
  void refresh()
  window.addEventListener('resize', onResize)
})
onUnmounted(() => window.removeEventListener('resize', onResize))

function minimize() { void window.devToolbox?.window?.minimize() }
async function toggle() {
  await window.devToolbox?.window?.toggleMaximize()
  await refresh()
}
function close() { void window.devToolbox?.window?.close() }
</script>

<template>
  <div class="win-controls" @mousedown.stop @dblclick.stop>
    <button type="button" class="wc-btn" aria-label="最小化" title="最小化" @click="minimize">
      <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><path d="M0 5h10" stroke="currentColor" stroke-width="1"/></svg>
    </button>
    <button type="button" class="wc-btn" :aria-label="maximized ? '还原' : '最大化'" :title="maximized ? '还原' : '最大化'" @click="toggle">
      <svg v-if="!maximized" width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><rect x="0.5" y="0.5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1"/></svg>
      <svg v-else width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><path d="M2.5 2.5v-2h7v7h-2" fill="none" stroke="currentColor" stroke-width="1"/><rect x="0.5" y="2.5" width="7" height="7" fill="none" stroke="currentColor" stroke-width="1"/></svg>
    </button>
    <button type="button" class="wc-btn wc-close" aria-label="关闭" title="关闭" @click="close">
      <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1"/></svg>
    </button>
  </div>
</template>
