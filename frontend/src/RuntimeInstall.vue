<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type { InstallEvent, InstallOption, InstallPlan, InstallTask } from './types'

const tools = ['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python']
const phaseLabels: Record<string, string> = {
  confirmed: '等待开始',
  downloading: '下载中',
  verifying: '校验中',
  extracting: '解包中',
  registering: '登记清单',
  complete: '完成',
  failed: '失败',
  cancelled: '已取消',
  partial: '部分成功',
}
const adviceByCode: Record<string, string> = {
  RUNTIME_PROVIDER_UNSUPPORTED: '该工具/镜像在本机平台没有已验证的发行包，请更换工具或镜像。',
  RUNTIME_PROVIDER_UNAVAILABLE: '无法读取 provider 元数据，请检查网络后重试。',
  RUNTIME_VERSION_UNAVAILABLE: '所选版本系列在来源上不可用，请尝试其他系列或镜像。',
  RUNTIME_PLAN_EXPIRED: '计划已过期，请重新预览后再安装。',
  RUNTIME_PLAN_STALE: '计划与预览结果不一致，请重新预览。',
  RUNTIME_REVISION_CONFLICT: '运行时清单已被其他操作修改，请刷新后重试。',
  RUNTIME_TARGET_CONFLICT: '目标目录已存在；本产品不会覆盖未知目录，请先处理或更换版本。',
  RUNTIME_CHECKSUM_MISSING: '来源未提供可验证的校验值，按安全规则拒绝安装。',
  RUNTIME_CHECKSUM_MISMATCH: '下载内容与校验值不一致，已丢弃；可重试或更换镜像。',
  RUNTIME_ARCHIVE_UNSAFE: '压缩包未通过安全检查（路径/链接/体积），已中止。',
  RUNTIME_DOWNLOAD_FAILED: '下载失败，请检查网络后重试。',
  RUNTIME_DISK_SPACE_INSUFFICIENT: '目标磁盘空间不足，请先清理空间。',
  RUNTIME_INSTALL_CANCELLED: '已取消；临时文件已清理，已有安装与默认选择不受影响。',
}

const options = ref<InstallOption[]>([])
const tool = ref('node')
const selector = ref('latest')
const mirror = ref('official')
const plan = ref<InstallPlan>()
const task = ref<InstallTask>()
const busy = ref(false)
const errorText = ref('')
let pollTimer: number | undefined

const currentOption = computed(() => options.value.find((item) => item.tool === tool.value))
const mirrors = computed(() => currentOption.value?.mirrors ?? [])
const unsupportedReason = computed(() => currentOption.value?.unsupported ?? '')
const activeStates = ['confirmed', 'downloading', 'verifying', 'extracting', 'registering']
const taskActive = computed(() => !!task.value && activeStates.includes(task.value.state))
const progressPercent = computed(() => {
  const progress = task.value?.progress
  if (!progress || progress.phase !== 'downloading' || !progress.totalBytes) return undefined
  return Math.min(100, Math.round((progress.receivedBytes ?? 0) * 100 / progress.totalBytes))
})

async function loadOptions() {
  errorText.value = ''
  try {
    options.value = await invoke<InstallOption[]>('runtime.install.options', {})
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}

function optionFor(toolName: string): InstallOption | undefined {
  return options.value.find((item) => item.tool === toolName)
}

function onToolChange() {
  const option = optionFor(tool.value)
  if (option && !option.mirrors.some((item) => item.id === mirror.value)) {
    mirror.value = option.mirrors[0]?.id ?? 'official'
  }
  plan.value = undefined
}

async function previewPlan() {
  busy.value = true
  errorText.value = ''
  plan.value = undefined
  try {
    plan.value = await invoke<InstallPlan>('runtime.install.preview', {
      tool: tool.value, selector: selector.value.trim() || 'latest', mirror: mirror.value, expectedRevision: 0,
    })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { busy.value = false }
}

function confirmInstall() {
  const snapshot = plan.value
  if (!snapshot) return
  Modal.confirm({
    title: `安装 ${snapshot.tool} ${snapshot.resolvedVersion}？`,
    content: `将下载并解压到 ${snapshot.targetHome}。安装只登记到本机清单，不会切换全局默认，也不修改 PATH、注册表或已打开的终端。`,
    okText: '确认安装', cancelText: '取消',
    onOk: async () => {
      busy.value = true
      errorText.value = ''
      try {
        task.value = await invoke<InstallTask>('runtime.install.start', {
          planId: snapshot.planId, planDigest: snapshot.planDigest, expectedRevision: 0, confirm: true,
        })
        startPolling()
        plan.value = undefined
      } catch (error) {
        errorText.value = error instanceof Error ? error.message : String(error)
      } finally { busy.value = false }
    },
  })
}

function startPolling() {
  stopPolling()
  pollTimer = window.setInterval(async () => {
    if (!task.value) return stopPolling()
    try {
      const latest = await invoke<InstallTask>('runtime.install.status', { taskId: task.value.taskId })
      task.value = latest
      if (!activeStates.includes(latest.state)) {
        stopPolling()
        if (latest.state === 'complete') message.success(`${latest.packages[0]?.name ?? '运行时'} 安装完成，可在下方清单中设为默认`)
        await invoke('runtime.getUserState', {}).catch(() => undefined)
      }
    } catch { /* transient poll errors keep the last snapshot */ }
  }, 600)
}

function stopPolling() {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

async function cancelInstall() {
  if (!task.value) return
  try {
    task.value = await invoke<InstallTask>('runtime.install.cancel', { taskId: task.value.taskId })
    message.info('已请求取消；归位提交开始后无法中断，请以最终状态为准。')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}

function onHostEvent(raw: unknown) {
  const event = raw as InstallEvent
  if (!event?.taskId || event.taskId !== task.value?.taskId) return
  if (event.sequence <= lastSequence) return
  lastSequence = event.sequence
  task.value = event.task
}
let lastSequence = 0

onMounted(async () => {
  await loadOptions()
  window.__devToolboxEvent = onHostEvent
})
onUnmounted(() => {
  stopPolling()
  window.__devToolboxEvent = undefined
})
</script>

<template>
  <section class="section">
    <div class="section-title"><div><h2>获取运行时</h2></div></div>
    <div class="toolbar">
      <a-select v-model:value="tool" :options="tools.map((value) => ({ label: value, value }))" :disabled="taskActive" @change="onToolChange" />
      <a-select v-model:value="mirror" :options="mirrors.map((item) => ({ label: item.label, value: item.id }))" :disabled="taskActive || !mirrors.length" />
      <a-input v-model:value="selector" :placeholder="optionFor(tool)?.versionSelectors.join(' / ') || 'latest'" :disabled="taskActive" class="selector-input" aria-label="版本通道" />
      <a-button :loading="busy" :disabled="taskActive || !!unsupportedReason" @click="previewPlan">查询并预览</a-button>
    </div>
    <a-alert v-if="unsupportedReason" type="warning" show-icon :message="`当前平台不支持 ${tool}`" :description="unsupportedReason" class="page-alert" />
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />

    <template v-if="plan">
      <a-divider />
      <div class="plan-card">
        <div class="plan-head">
          <div><strong>{{ plan.tool }} {{ plan.resolvedVersion }}</strong><small>provider {{ plan.providerId }}，镜像 {{ plan.mirror }}，{{ plan.os }}/{{ plan.arch }}</small></div>
          <a-tag v-if="plan.targetExists" color="red">目标已存在</a-tag>
        </div>
        <div v-for="pkg in plan.packages" :key="pkg.name" class="plan-package">
          <span>{{ pkg.name }}</span><code>{{ pkg.fileName }}</code>
          <small>{{ pkg.checksumAlgorithm }} 校验<template v-if="pkg.estimatedBytes"> · 约 {{ (pkg.estimatedBytes / 1048576).toFixed(1) }} MB</template></small>
        </div>
        <p class="muted">目标目录：{{ plan.targetHome }}</p>
        <div class="form-actions">
          <a-button @click="plan = undefined">放弃</a-button>
          <a-button type="primary" :disabled="plan.targetExists" @click="confirmInstall">确认安装</a-button>
        </div>
      </div>
    </template>

    <template v-if="task">
      <a-divider />
      <div class="task-card">
        <div class="plan-head">
          <div><strong>安装任务 {{ task.taskId }}</strong><small>{{ phaseLabels[task.state] ?? task.state }}<template v-if="task.progress?.package"> · {{ task.progress.package }}</template></small></div>
          <a-space>
            <a-button v-if="taskActive" size="small" @click="cancelInstall">取消安装</a-button>
            <a-button v-if="!taskActive" size="small" @click="previewPlan">重新预览</a-button>
          </a-space>
        </div>
        <a-progress v-if="progressPercent !== undefined" :percent="progressPercent" size="small" />
        <div v-for="pkg in task.packages" :key="pkg.name" class="plan-package">
          <span>{{ pkg.name }}</span>
          <a-tag :color="pkg.state === 'done' ? 'green' : pkg.state === 'reused' ? 'blue' : pkg.state === 'failed' ? 'red' : 'default'">
            {{ pkg.state === 'done' ? '完成' : pkg.state === 'reused' ? '已存在，复用' : pkg.state === 'failed' ? '失败' : '等待' }}
          </a-tag>
          <small v-if="pkg.error">{{ pkg.error.code }}</small>
        </div>
        <a-alert v-if="task.error" type="error" show-icon :message="task.error.code" :description="adviceByCode[task.error.code] ?? task.error.message" class="page-alert" />
        <a-alert v-if="task.state === 'partial'" type="warning" show-icon message="部分包已安装；处理后可重新预览重试，已完成的包会被复用，不会重复下载。" class="page-alert" />
      </div>
    </template>
  </section>
</template>

<style scoped>
.selector-input{max-width:220px}.toolbar :deep(.ant-select){min-width:150px}
.plan-card,.task-card{border:1px solid var(--border);border-radius:10px;padding:16px}
.plan-head{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;flex-wrap:wrap}
.plan-head strong{display:block}
.plan-head small{display:block;margin-top:4px;color:var(--text-muted);overflow-wrap:anywhere}
.plan-package{display:flex;align-items:center;gap:10px;padding:8px 0;border-top:1px solid var(--border-soft);flex-wrap:wrap}
.plan-package code{overflow-wrap:anywhere;color:var(--text-code)}
.plan-package small{color:var(--text-muted)}
</style>
