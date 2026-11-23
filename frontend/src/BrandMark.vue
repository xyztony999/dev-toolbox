<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 16 })
const maskId = `dt-mark-${Math.random().toString(36).slice(2, 8)}`
</script>

<template>
  <svg class="brand-mark" :width="size * 2" :height="size" viewBox="0 0 116 58" aria-hidden="true">
    <mask :id="maskId">
      <rect width="116" height="58" fill="black" />
      <circle cx="28" cy="28" r="22" fill="white" />
      <rect x="42" y="21" width="62" height="14" rx="3" fill="white" />
      <rect x="82" y="30" width="8" height="16" rx="2" fill="white" />
      <rect x="96" y="30" width="8" height="20" rx="2" fill="white" />
      <circle cx="28" cy="28" r="10" fill="black" />
      <circle cx="28" cy="28" r="4" fill="white" />
    </mask>
    <rect width="116" height="58" fill="currentColor" :mask="`url(#${maskId})`" />
  </svg>
</template>

<style scoped>
.brand-mark { display: block; flex: none; }
</style>
