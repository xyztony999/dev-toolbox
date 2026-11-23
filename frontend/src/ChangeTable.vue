<script setup lang="ts">
import type { GitConfigChange } from './types'

// 变更预览表：逐键列出「当前值 → 应用后」，数据原样来自服务端的
// dry-run，前端不推断覆盖范围。
defineProps<{ changes: GitConfigChange[] }>()

function text(value?: string | null): string {
  return (value ?? '').trim()
}
</script>

<template>
  <div class="change-table-scroll">
    <table class="change-table">
      <thead>
        <tr><th scope="col">配置项</th><th scope="col">当前值</th><th scope="col"></th><th scope="col">应用后</th></tr>
      </thead>
      <tbody>
        <tr v-for="(change, index) in changes" :key="`${change.key}-${index}`">
          <td><code>{{ change.key }}</code></td>
          <td :class="{ unset: !text(change.from) }">{{ text(change.from) || '未设置' }}</td>
          <td class="arrow" aria-hidden="true">→</td>
          <td :class="{ unset: change.action === 'unset' || !text(change.to) }">
            {{ change.action === 'unset' ? '清除' : text(change.to) || '空' }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.change-table-scroll{width:100%;overflow-x:auto}
.change-table{width:100%;min-width:340px;border-collapse:collapse;font-size:12.5px;color:var(--text-strong)}
.change-table th{padding:6px 10px;text-align:left;font-weight:500;font-size:12px;color:var(--text-muted);border-bottom:1px solid var(--change-rule,var(--border-soft));white-space:nowrap}
.change-table td{padding:7px 10px;vertical-align:top;border-bottom:1px solid var(--change-rule,var(--border-soft));overflow-wrap:anywhere}
.change-table tbody tr:last-child td{border-bottom:0}
.change-table td:first-child{white-space:nowrap}
.change-table code{color:var(--text-code);font:12px ui-monospace,Consolas,'Cascadia Mono',monospace}
.change-table .arrow{width:20px;padding:0;text-align:center;color:var(--text-muted)}
.change-table .unset{color:var(--text-muted)}
</style>
