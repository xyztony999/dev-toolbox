<script setup lang="ts">
import { ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'

interface ToolState { tool: string; status: string; version?: string; reason?: string }
interface TerminalInfo { id: string; label: string }
interface Preview { cwd: string; tools: ToolState[]; javaHome?: string; pathPrepends: string[]; gitIdentity: { resolved: string; profileId?: string }, terminals: TerminalInfo[] }

const props = defineProps<{ open: boolean; repoPaths: string[]; defaultPath: string }>()
const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()

const preview = ref<Preview>()
const loading = ref(false)
const errorText = ref('')
const cwd = ref('')
const terminalId = ref('')

const close = () => emit('update:open', false)

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
  } finally { loading.value = false }
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
  Modal.confirm({
    title: '打开上下文终端？',
    content: '新终端仅本会话生效：注入所选运行时的 PATH/JAVA_HOME，不修改全局默认与已打开的终端。',
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
  <a-drawer :open="open" width="520" title="在此打开终端" @close="close">
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
    <a-skeleton v-if="loading" active :paragraph="{ rows: 4 }" />
    <template v-if="preview">
      <div v-for="item in preview.tools" :key="item.tool">
        <span>{{ item.tool }}：{{ item.status === 'injected' ? item.version : item.reason }}</span>
      </div>
      <a-select v-model:value="cwd" style="width: 100%" :options="repoPaths.map((p) => ({ label: p, value: p }))" />
      <a-select v-model:value="terminalId" style="width: 100%" :options="preview.terminals.map((t) => ({ label: t.label, value: t.id }))" />
      <a-button type="primary" block :disabled="!terminalId" @click="launch">打开终端</a-button>
    </template>
  </a-drawer>
</template>
