<script setup lang="ts">
import { computed } from 'vue'
import { runtimeLogos } from './runtime-logos'

const props = withDefaults(defineProps<{ name: string; size?: number }>(), { size: 18 })

const key = computed(() => {
  const name = props.name.trim().toLowerCase()
  if (name === 'golang' || name === 'go') return 'go'
  if (name === 'androidsdk' || name === 'android') return 'android'
  if (name === 'nodejs' || name === 'nodedotjs') return 'node'
  if (name === 'openjdk' || name === 'jdk') return 'java'
  return name
})

const logo = computed(() => runtimeLogos[key.value])
const fill = computed(() => {
  if (key.value === 'gradle') return '#7dcec4'
  if (key.value === 'rust') return '#ce412b'
  return logo.value?.color ?? 'currentColor'
})
</script>

<template>
  <span v-if="key === 'java'" class="runtime-mark" :style="{ width: `${size}px`, height: `${size}px` }" aria-hidden="true">
    <svg viewBox="0 0 24 24">
      <path d="M7.4 2.4c1 .9.2 1.7-.3 2.4M11.2 1.5c1.1 1 .2 1.9-.4 2.7M14.8 2.4c1 .9.2 1.7-.3 2.4" fill="none" stroke="#f89820" stroke-width="1.5" stroke-linecap="round"/>
      <path fill="#5382a1" d="M5.4 8.4h10.1c-.2 4.2-2.1 6.8-5.05 6.8S5.6 12.6 5.4 8.4z"/>
      <path d="M15.4 9.1h.7a1.9 1.9 0 0 1 0 3.8h-.7" fill="none" stroke="#5382a1" stroke-width="1.6" stroke-linecap="round"/>
      <path fill="#f89820" d="M7.6 10.2h5.6c-.1 1.5-.8 2.5-2.8 2.5s-2.7-1-2.8-2.5z"/>
      <path fill="#5382a1" d="M4.3 16.2h12.4c.2.8-1.5 1.8-6.2 1.8s-6.4-1-6.2-1.8z"/>
    </svg>
  </span>
  <span v-else-if="logo" class="runtime-mark" :style="{ width: `${size}px`, height: `${size}px` }" aria-hidden="true">
    <svg viewBox="0 0 24 24">
      <path :d="logo.path" :fill="fill" />
    </svg>
  </span>
  <span v-else class="runtime-mark fallback" :style="{ width: `${size}px`, height: `${size}px` }" aria-hidden="true">{{ name.slice(0, 1).toUpperCase() }}</span>
</template>

<style scoped>
.runtime-mark { display: inline-flex; align-items: center; justify-content: center; flex: none; background: none; }
.runtime-mark svg { width: 100%; height: 100%; display: block; }
.fallback { border-radius: 3px; background: #6f675c; color: #fff; font-size: 10px; font-weight: 600; }
</style>
