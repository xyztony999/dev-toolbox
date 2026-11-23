<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import RuntimeMark from './RuntimeMark.vue'

interface ToolState { tool: string; status: string; version?: string; reason?: string; binDir?: string }
interface TerminalInfo { id: string; label: string }
interface Preview {
  cwd: string
  tools: ToolState[]
  javaHome?: string
  pathPrepends: string[]
  gitIdentity: { resolved: string; profileId?: string }
  terminals: TerminalInfo[]
}

const props = defineProps<{ open: boolean; repoPaths: string[]; defaultPath: string }>()
const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()

const preview = ref<Preview>()
const loading = ref(false)
const errorText = ref('')
const cwd = ref('')
const terminalId = ref('')

const close = () => emit('update:open', false)

const injectedCount = computed(() => preview.value?.tools.filter((item) => item.status === 'injected').length ?? 0)
const skippedCount = computed(() => preview.value?.tools.filter((item) => item.status !== 'injected').length ?? 0)
const hasTerminal = computed(() => (preview.value?.terminals.length ?? 0) > 0)

function toolStateText(item: ToolState): string {
  if (item.status === 'injected') return `注入 ${item.version ?? ''}`.trim()
  return item.reason || '未注入'
}

function identityText(): string {
  const identity = preview.value?.gitIdentity
  if (!identity) return ''
  if (identity.resolved === 'none') return '未解析到 Git 身份（提交前请确认）'
  if (identity.resolved === 'manual') return '手工配置（全局或仓库级）'
  return identity.profileId ? `档案 ${identity.profileId}` : identity.resolved
}

async function load() {
  loading.value = true
  errorText.value = ''
  try {
    preview.value = await invoke<Preview>('session.preview', { path: cwd.value || props.defaultPath })
    if (!terminalId.value && preview.value.terminals.length) {
      terminalId.value = preview.value.terminals[0].id
    }
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}

watch(() => [props.open, cwd.value], () => { if (props.open) load() })
watch(() => props.open, (open) => {
  if (open) {
    cwd.value = props.defaultPath
    terminalId.value = ''
    preview.value = undefined
  }
})

function launch() {
  if (!hasTerminal.value) return
  Modal.confirm({
    title: '打开上下文终端？',
    content: '将按上面的注入预览打开一个新终端窗口：PATH/JAVA_HOME 仅对本会话生效，不修改全局默认，也不影响已经打开的终端。',
    okText: '打开终端',
    cancelText: '取消',
    zIndex: 1100,
    onOk: async () => {
      try {
        const result = await invoke<{ terminalId: string; pid: number }>('session.open', {
          path: cwd.value || props.defaultPath, terminalId: terminalId.value, confirm: true,
        })
        message.success(`终端已启动（${result.terminalId}，pid ${result.pid}）`)
        close()
      } catch (error) {
        errorText.value = error instanceof Error ? error.message : String(error)
      }
    },
  })
}
</script>

<template>
  <a-drawer :open="open" width="560" title="在此打开终端" @close="close">
    <a-alert
      type="info"
      show-icon
      class="session-intent"
      message="在所选目录打开新终端"
      description="启动时会按「用户级运行时设置」里的全局选择，把 PATH/JAVA_HOME 注入到这个新终端。注入只影响这一个窗口，不修改全局默认，也不改变任何已打开的终端。"
    />
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />
    <a-skeleton v-if="loading" active :paragraph="{ rows: 5 }" />
    <template v-if="preview">
      <section class="session-section">
        <div class="session-section-head">
          <strong>运行时注入预览</strong>
          <span class="session-count">{{ injectedCount }} 项注入 · {{ skippedCount }} 项跳过</span>
        </div>
        <div class="tool-list">
          <div v-for="item in preview.tools" :key="item.tool" class="tool-row">
            <strong class="tool-name"><RuntimeMark :name="item.tool" :size="16" />{{ item.tool }}</strong>
            <a-tag :color="item.status === 'injected' ? 'green' : 'default'">
              {{ item.status === 'injected' ? '注入' : '跳过' }}
            </a-tag>
            <span class="tool-state">{{ toolStateText(item) }}</span>
            <code v-if="item.binDir" class="tool-bin">{{ item.binDir }}</code>
          </div>
        </div>
        <div class="session-meta">
          <span>JAVA_HOME：<code>{{ preview.javaHome || '继承父环境' }}</code></span>
          <span>PATH 前置 {{ preview.pathPrepends.length }} 项</span>
        </div>
        <div class="session-meta">
          <span>Git 身份：<strong>{{ identityText() }}</strong></span>
        </div>
      </section>

      <section class="session-section">
        <div class="session-section-head"><strong>启动位置与终端</strong></div>
        <div class="session-field">
          <span class="session-label">工作目录</span>
          <a-select v-model:value="cwd" style="width: 100%" :options="repoPaths.map((p) => ({ label: p, value: p }))" />
        </div>
        <div class="session-field">
          <span class="session-label">终端类型</span>
          <a-select v-model:value="terminalId" style="width: 100%" :options="preview.terminals.map((t) => ({ label: t.label, value: t.id }))" />
        </div>
        <a-alert v-if="!hasTerminal" type="warning" show-icon message="未探测到可用终端" description="系统需至少存在 PowerShell 或 Git Bash；请先安装后再试。" />
        <a-button type="primary" block size="large" :disabled="!hasTerminal" class="session-launch" @click="launch">打开终端</a-button>
        <p class="session-hint">新窗口由 DevToolbox 直接启动，注入环境在服务端重新求值，不信任界面传入的值。</p>
      </section>
    </template>
  </a-drawer>
</template>

<style scoped>
.session-intent { margin-bottom: 16px; }
.session-section { padding: 16px 0 6px; border-top: 1px solid var(--border-soft); }
.session-section:first-of-type { border-top: 0; padding-top: 4px; }
.session-section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.session-count { color: var(--text-muted); font-size: 12px; }
.tool-list { display: flex; flex-direction: column; }
.tool-row { display: flex; align-items: center; gap: 10px; padding: 7px 0; border-top: 1px solid var(--border-soft); }
.tool-row:first-child { border-top: 0; }
.tool-name { min-width: 108px; display: inline-flex; align-items: center; gap: 8px; }
.tool-state { color: var(--text-strong); font-size: 12.5px; overflow-wrap: anywhere; }
.tool-bin { margin-left: auto; color: var(--text-muted); font-size: 11.5px; overflow-wrap: anywhere; }
.session-meta { display: flex; gap: 18px; flex-wrap: wrap; margin-top: 10px; color: var(--text-muted); font-size: 12.5px; }
.session-meta code { color: var(--text-code); overflow-wrap: anywhere; }
.session-field { margin-bottom: 12px; }
.session-label { display: block; margin-bottom: 6px; color: var(--text-muted); font-size: 12.5px; }
.session-launch { margin-top: 6px; }
.session-hint { margin: 10px 0 0; color: var(--text-muted); font-size: 12px; }
@media (max-width: 640px) {
  .tool-bin { margin-left: 0; width: 100%; }
  .tool-row { flex-wrap: wrap; }
}
</style>
