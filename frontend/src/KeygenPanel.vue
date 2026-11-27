<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type {
  GitKeyAlgorithm, GitKeyExpiry, GitKeyType, GitProfile,
  KeygenOptions, KeygenPlan, KeygenResult, LocalKeySummary, PublicKeyExport,
} from './types'

const props = defineProps<{
  open: boolean
  mode: 'generate' | 'select'
  profileId: string
  profileName: string
  profileEmail: string
  currentSigningKey?: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'applied', profile: GitProfile): void
}>()

const algoLabels: Record<GitKeyAlgorithm, string> = {
  ed25519: 'Ed25519',
  rsa4096: 'RSA 4096',
  rsa3072: 'RSA 3072',
  nistp256: 'NIST P-256',
  ecdsa256: 'ECDSA P-256',
}
const expiryLabels: Record<GitKeyExpiry, string> = {
  never: '永不过期',
  '1y': '1 年',
  '2y': '2 年',
  '5y': '5 年',
}

const loading = ref(false)
const busy = ref(false)
const errorText = ref('')
const options = ref<KeygenOptions>()

const keyType = ref<GitKeyType>('openpgp')
const algorithm = ref<GitKeyAlgorithm>('ed25519')
const expiry = ref<GitKeyExpiry>('never')
const passphrase = ref('')
const enableGpgSign = ref(true)
const outputPath = ref('')
const plan = ref<KeygenPlan>()
const result = ref<KeygenResult>()

const selectType = ref<GitKeyType>('openpgp')
const keys = ref<LocalKeySummary[]>([])
const listLoading = ref(false)

const exportOpen = ref(false)
const exportTarget = ref<{ keyType: GitKeyType; id: string }>()
const exportPath = ref('')
const exportBusy = ref(false)

let previewTimer: ReturnType<typeof setTimeout> | undefined

const title = computed(() => (props.mode === 'generate' ? '生成密钥' : '选用本机密钥'))

const toolFor = (type: GitKeyType) => options.value?.tools.find((item) => item.keyType === type)

const currentTool = computed(() => toolFor(props.mode === 'generate' ? keyType.value : selectType.value))

const algorithmOptions = computed(() => {
  const values = options.value?.algorithms.find((item) => item.keyType === keyType.value)?.values ?? []
  return values.map((value) => ({ value, label: algoLabels[value] ?? value }))
})

const expiryOptions = computed(() => {
  const values = options.value?.expiries.find((item) => item.keyType === keyType.value)?.values ?? ['never']
  return values.map((value) => ({ value, label: expiryLabels[value] ?? value }))
})

const sshExpiryLocked = computed(() => keyType.value === 'ssh')

const openpgpReady = computed(() => !!props.profileName.trim() && !!props.profileEmail.trim())

const canGenerate = computed(() => {
  if (!props.profileId) return false
  if (!currentTool.value?.available) return false
  if (keyType.value === 'openpgp' && !openpgpReady.value) return false
  if (plan.value?.wouldFail) return false
  return true
})

const sshPathPlaceholder = computed(() =>
  options.value?.sshDefaultPathHint
  || `~/.ssh/devtoolbox_${props.profileId || 'profile'}`,
)

function friendlyKeygenError(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error)
  if (raw.includes('GIT_PROFILE_NOT_FOUND')) return '找不到这个档案'
  if (raw.includes('GIT_PROFILE_INVALID')) return '档案需要姓名和邮箱才能生成 OpenPGP 密钥'
  if (raw.includes('GIT_KEYGEN_TOOL_MISSING')) {
    return keyType.value === 'ssh' || selectType.value === 'ssh'
      ? '未检测到 ssh-keygen，请安装后再试'
      : '未检测到 gpg，请安装后再试'
  }
  if (raw.includes('GIT_KEYGEN_TOOL_FAILED')) return '本机密钥工具执行失败，请查看详情后重试'
  if (raw.includes('GIT_KEYGEN_PARAMS_INVALID')) return '密钥参数不合法，请检查类型、算法和路径'
  if (raw.includes('GIT_KEYGEN_PASSPHRASE_REJECTED')) return '口令不符合工具要求，请换一组后再试'
  if (raw.includes('GIT_KEY_NOT_FOUND')) return '找不到这把密钥，请刷新列表'
  if (raw.includes('GIT_KEY_EXPORT_PATH_INVALID')) return '导出路径无效，或父目录不存在'
  if (raw.includes('GIT_KEY_EXPORT_EXISTS')) return '目标文件已存在'
  if (raw.includes('GIT_KEYGEN_PROFILE_WRITE_FAILED')) return '密钥已生成，但档案签名字段未能写入'
  if (raw.includes('CONFIRMATION_REQUIRED')) return '这个操作需要确认后才能执行'
  return raw
}

function clearPassphrase() {
  passphrase.value = ''
}

function resetGenerateState() {
  keyType.value = 'openpgp'
  algorithm.value = 'ed25519'
  expiry.value = 'never'
  enableGpgSign.value = true
  outputPath.value = ''
  plan.value = undefined
  result.value = undefined
  clearPassphrase()
  errorText.value = ''
}

function applyDefaults(type: GitKeyType) {
  const defaults = options.value?.defaults[type]
  if (defaults) {
    algorithm.value = defaults.algorithm
    expiry.value = type === 'ssh' ? 'never' : defaults.expiry
  } else if (type === 'ssh') {
    expiry.value = 'never'
  }
  const allowed = options.value?.algorithms.find((item) => item.keyType === type)?.values ?? []
  if (allowed.length && !allowed.includes(algorithm.value)) {
    algorithm.value = allowed[0]
  }
}

async function loadOptions() {
  loading.value = true
  errorText.value = ''
  try {
    options.value = await invoke<KeygenOptions>('git.keygenOptions', {})
    applyDefaults(keyType.value)
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  } finally {
    loading.value = false
  }
}

async function loadKeys() {
  listLoading.value = true
  errorText.value = ''
  keys.value = []
  try {
    if (!options.value) options.value = await invoke<KeygenOptions>('git.keygenOptions', {})
    const tool = toolFor(selectType.value)
    if (tool && !tool.available) return
    keys.value = await invoke<LocalKeySummary[]>('git.keyList', { keyType: selectType.value })
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  } finally {
    listLoading.value = false
  }
}

watch(() => [props.open, props.mode] as const, async ([open, mode]) => {
  if (!open) {
    clearPassphrase()
    if (previewTimer) clearTimeout(previewTimer)
    return
  }
  resetGenerateState()
  selectType.value = 'openpgp'
  keys.value = []
  if (mode === 'generate') {
    await loadOptions()
  } else {
    await loadOptions()
    await loadKeys()
  }
})

watch(keyType, (type) => {
  applyDefaults(type)
  plan.value = undefined
  result.value = undefined
  schedulePreview()
})

watch([algorithm, expiry, enableGpgSign, outputPath], () => {
  if (props.open && props.mode === 'generate') schedulePreview()
})

watch(selectType, () => {
  if (props.open && props.mode === 'select') void loadKeys()
})

function schedulePreview() {
  if (!props.open || props.mode !== 'generate' || !props.profileId) return
  if (previewTimer) clearTimeout(previewTimer)
  previewTimer = setTimeout(() => { void runPreview() }, 400)
}

function buildCreatePayload(extra: Record<string, unknown> = {}) {
  const payload: Record<string, unknown> = {
    profileId: props.profileId,
    keyType: keyType.value,
    algorithm: algorithm.value,
    expiry: keyType.value === 'ssh' ? 'never' : expiry.value,
    enableGpgSign: enableGpgSign.value,
    ...extra,
  }
  if (passphrase.value) payload.passphrase = passphrase.value
  if (keyType.value === 'ssh' && outputPath.value.trim()) payload.outputPath = outputPath.value.trim()
  return payload
}

async function runPreview() {
  if (!props.profileId || props.mode !== 'generate') return
  if (keyType.value === 'openpgp' && !openpgpReady.value) {
    plan.value = undefined
    return
  }
  if (!currentTool.value?.available) {
    plan.value = undefined
    return
  }
  try {
    plan.value = await invoke<KeygenPlan>('git.keygenPreview', buildCreatePayload({ dryRun: true }))
    errorText.value = ''
  } catch (error) {
    plan.value = undefined
    errorText.value = friendlyKeygenError(error)
  }
}

async function pickOutputDir() {
  try {
    const dir = await invoke<string>('app.pickFolder', {})
    if (!dir) return
    const fileName = `devtoolbox_${props.profileId || 'key'}`
    const sep = dir.includes('\\') ? '\\' : '/'
    outputPath.value = `${dir.replace(/[\\/]+$/, '')}${sep}${fileName}`
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  }
}

function confirmCreate() {
  if (!canGenerate.value) return
  Modal.confirm({
    title: '生成并写入档案',
    content: `类型 ${keyType.value === 'openpgp' ? 'OpenPGP' : 'SSH'}，算法 ${algoLabels[algorithm.value]}，档案 ${props.profileId}`,
    okText: '生成',
    cancelText: '取消',
    zIndex: 1200,
    onOk: () => doCreate(),
  })
}

async function doCreate() {
  busy.value = true
  errorText.value = ''
  try {
    const created = await invoke<KeygenResult>('git.keygenCreate', buildCreatePayload({ confirm: true }))
    result.value = created
    clearPassphrase()
    if (created.profileUpdated) {
      message.success('已更新档案的签名字段')
      emit('applied', created.profile)
    } else if (created.localKeyCreated) {
      message.warning(created.warning || '密钥已生成，但档案尚未写入')
    }
  } catch (error) {
    clearPassphrase()
    errorText.value = friendlyKeygenError(error)
  } finally {
    busy.value = false
  }
}

function confirmSelect(row: LocalKeySummary) {
  Modal.confirm({
    title: '选用并写入档案',
    content: `将把 ${row.fingerprint} 写入档案 ${props.profileId}`,
    okText: '选用',
    cancelText: '取消',
    zIndex: 1200,
    onOk: () => doSelect(row),
  })
}

async function doSelect(row: LocalKeySummary) {
  busy.value = true
  errorText.value = ''
  try {
    const selected = await invoke<KeygenResult>('git.keySelect', {
      profileId: props.profileId,
      keyType: row.keyType,
      id: row.id,
      enableGpgSign: true,
      confirm: true,
    })
    message.success('已更新档案的签名字段')
    emit('applied', selected.profile)
    emit('update:open', false)
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  } finally {
    busy.value = false
  }
}

async function copyPublicKey(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制公钥')
  } catch {
    message.error('复制失败')
  }
}

async function copyFromExport(keyTypeValue: GitKeyType, id: string) {
  try {
    const exported = await invoke<PublicKeyExport>('git.keyExport', { keyType: keyTypeValue, id })
    await copyPublicKey(exported.text)
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  }
}

function openExport(keyTypeValue: GitKeyType, id: string, suggestedName?: string) {
  exportTarget.value = { keyType: keyTypeValue, id }
  const name = suggestedName || `${id.replace(/[^A-Za-z0-9._-]+/g, '_').slice(0, 40)}.pub`
  exportPath.value = name
  exportOpen.value = true
}

async function pickExportDir() {
  try {
    const dir = await invoke<string>('app.pickFolder', {})
    if (!dir) return
    const base = exportPath.value.split(/[\\/]/).filter(Boolean).at(-1) || 'public.pub'
    const sep = dir.includes('\\') ? '\\' : '/'
    exportPath.value = `${dir.replace(/[\\/]+$/, '')}${sep}${base}`
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  }
}

async function runExport(overwrite = false) {
  if (!exportTarget.value || !exportPath.value.trim()) return
  exportBusy.value = true
  try {
    const exported = await invoke<PublicKeyExport>('git.keyExport', {
      keyType: exportTarget.value.keyType,
      id: exportTarget.value.id,
      writePath: exportPath.value.trim(),
      overwrite,
      confirm: true,
    })
    message.success(exported.writtenPath ? `已导出到 ${exported.writtenPath}` : '已导出')
    exportOpen.value = false
  } catch (error) {
    const raw = error instanceof Error ? error.message : String(error)
    if (raw.includes('GIT_KEY_EXPORT_EXISTS')) {
      Modal.confirm({
        title: '文件已存在，要覆盖吗？',
        content: exportPath.value,
        okText: '覆盖',
        cancelText: '取消',
        zIndex: 1300,
        onOk: () => runExport(true),
      })
    } else {
      errorText.value = friendlyKeygenError(error)
    }
  } finally {
    exportBusy.value = false
  }
}

function close() {
  clearPassphrase()
  emit('update:open', false)
}

async function writePartialViaSelect() {
  if (!result.value?.signingKey) return
  busy.value = true
  try {
    const selected = await invoke<KeygenResult>('git.keySelect', {
      profileId: props.profileId,
      keyType: result.value.keyType,
      id: result.value.fingerprint || result.value.signingKey,
      enableGpgSign: enableGpgSign.value,
      confirm: true,
    })
    message.success('已更新档案的签名字段')
    emit('applied', selected.profile)
    result.value = selected
  } catch (error) {
    errorText.value = friendlyKeygenError(error)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <a-drawer :open="open" width="min(520px, 100vw)" :title="title" :z-index="1100"
    :mask-closable="false" @close="close">
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="panel-alert"
      @close="errorText = ''" />

    <a-skeleton v-if="loading" active :paragraph="{ rows: 4 }" />

    <template v-else-if="mode === 'generate'">
      <a-alert v-if="currentTool && !currentTool.available" type="warning" show-icon class="panel-alert"
        :message="keyType === 'ssh' ? '未检测到 ssh-keygen，请安装后再试' : '未检测到 gpg，请安装后再试'" />
      <a-alert v-if="keyType === 'openpgp' && !openpgpReady" type="info" show-icon class="panel-alert"
        message="档案需要姓名和邮箱" />

      <a-form layout="vertical" class="keygen-form">
        <a-form-item label="密钥类型">
          <a-radio-group v-model:value="keyType" button-style="solid">
            <a-radio-button value="openpgp">OpenPGP</a-radio-button>
            <a-radio-button value="ssh">SSH 签名</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="算法">
          <a-select v-model:value="algorithm" :options="algorithmOptions" />
        </a-form-item>
        <a-form-item label="有效期">
          <a-select v-if="!sshExpiryLocked" v-model:value="expiry" :options="expiryOptions" />
          <a-input v-else value="SSH 密钥无到期" disabled />
        </a-form-item>
        <a-form-item label="口令" extra="可留空；不会写入本地存储">
          <a-input-password v-model:value="passphrase" placeholder="可留空" autocomplete="new-password" />
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="enableGpgSign">打开提交签名</a-checkbox>
        </a-form-item>
        <a-form-item v-if="keyType === 'ssh'" label="SSH 输出路径">
          <div class="path-row">
            <a-input v-model:value="outputPath" :placeholder="sshPathPlaceholder" autocomplete="off" />
            <a-button @click="pickOutputDir">浏览…</a-button>
          </div>
        </a-form-item>
      </a-form>

      <div v-if="plan" class="preview-box">
        <h4>预览</h4>
        <p class="preview-line">工具：{{ plan.toolPath || '（未探测到路径）' }}</p>
        <ul class="preview-steps">
          <li v-for="(step, index) in plan.commandSummary" :key="index">{{ step }}</li>
        </ul>
        <p class="preview-line">
          口令：{{ passphrase ? '（已设置，不展示）' : '（未设置）' }}
        </p>
        <p class="preview-line">
          将写入 gpg.format={{ plan.profilePatch.gpgFormat }}，
          gpgSign={{ plan.profilePatch.gpgSign ? 'true' : 'false' }}
        </p>
        <a-alert v-for="(warn, index) in plan.warnings" :key="'w' + index" type="warning" show-icon
          :message="warn" class="panel-alert" />
        <a-alert v-if="plan.wouldFail" type="error" show-icon :message="plan.wouldFail" class="panel-alert" />
      </div>

      <div class="action-row">
        <a-button :disabled="!profileId || !currentTool?.available" @click="runPreview">预览</a-button>
        <a-button type="primary" :loading="busy" :disabled="!canGenerate" @click="confirmCreate">生成</a-button>
      </div>

      <div v-if="result" class="result-box">
        <a-alert v-if="result.warning || (result.localKeyCreated && !result.profileUpdated)" type="warning"
          show-icon class="panel-alert"
          :message="result.warning || '密钥已生成，档案签名字段尚未写入'" />
        <p v-if="result.fingerprint" class="result-line">指纹：<code>{{ result.fingerprint }}</code></p>
        <p v-if="result.publicKeyPath" class="result-line">公钥路径：<code>{{ result.publicKeyPath }}</code></p>
        <p v-if="result.signingKey" class="result-line">signingKey：<code>{{ result.signingKey }}</code></p>
        <div class="action-row">
          <a-button size="small"
            @click="copyFromExport(result.keyType, result.fingerprint || result.signingKey)">
            复制公钥
          </a-button>
          <a-button size="small"
            @click="openExport(result.keyType, result.fingerprint || result.signingKey, result.publicKeyPath?.split(/[\\/]/).at(-1))">
            导出到文件
          </a-button>
          <a-button v-if="result.localKeyCreated && !result.profileUpdated" size="small" type="primary"
            :loading="busy" @click="writePartialViaSelect">
            写入档案
          </a-button>
        </div>
      </div>
    </template>

    <template v-else>
      <a-tabs v-model:activeKey="selectType" size="small">
        <a-tab-pane key="openpgp" tab="OpenPGP" />
        <a-tab-pane key="ssh" tab="SSH" />
      </a-tabs>

      <a-alert v-if="currentTool && !currentTool.available" type="warning" show-icon class="panel-alert"
        :message="selectType === 'ssh' ? '未检测到 ssh-keygen，请安装后再试' : '未检测到 gpg，请安装后再试'" />

      <a-skeleton v-if="listLoading" active :paragraph="{ rows: 3 }" />
      <a-empty v-else-if="currentTool?.available !== false && !keys.length"
        description="未找到可用密钥。可先生成，或检查本机是否已安装 gpg / 是否已有 ~/.ssh 下的公钥。" />
      <div v-else-if="keys.length" class="key-list">
        <div v-for="row in keys" :key="row.id" class="key-row">
          <div class="key-main">
            <code class="key-id">{{ row.fingerprint }}</code>
            <span class="key-uid">{{ row.uidOrComment || '（无注释）' }}</span>
            <span v-if="row.createdAt" class="key-meta">{{ row.createdAt }}</span>
          </div>
          <div class="key-actions">
            <a-button size="small" type="primary" :loading="busy" :disabled="!profileId"
              @click="confirmSelect(row)">选用</a-button>
            <a-button size="small" @click="copyFromExport(row.keyType, row.id)">复制</a-button>
            <a-button size="small"
              @click="openExport(row.keyType, row.id, row.publicKeyPath?.split(/[\\/]/).at(-1))">导出</a-button>
          </div>
        </div>
      </div>
    </template>
  </a-drawer>

  <a-modal :open="exportOpen" title="导出公钥到文件" :z-index="1200" :confirm-loading="exportBusy"
    ok-text="导出" cancel-text="取消" @ok="runExport(false)" @cancel="exportOpen = false">
    <a-form layout="vertical">
      <a-form-item label="文件路径">
        <div class="path-row">
          <a-input v-model:value="exportPath" placeholder="绝对路径" autocomplete="off" />
          <a-button @click="pickExportDir">浏览…</a-button>
        </div>
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped>
.panel-alert{margin-bottom:12px}
.keygen-form{margin-bottom:8px}
.path-row{display:flex;gap:8px}
.path-row .ant-input{flex:1;min-width:0}
.preview-box,.result-box{margin:12px 0;padding:12px;border:1px solid var(--border-soft);border-radius:8px;background:var(--bg-chip)}
.preview-box h4{margin:0 0 8px;font-size:13px}
.preview-line,.result-line{margin:4px 0;font-size:12.5px;line-height:1.5;overflow-wrap:anywhere}
.preview-steps{margin:6px 0;padding-left:18px;font-size:12.5px;color:var(--text-muted)}
.action-row{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}
.key-list{display:flex;flex-direction:column;gap:10px}
.key-row{padding:10px 0;border-top:1px solid var(--border-soft)}
.key-row:first-child{border-top:0;padding-top:0}
.key-main{display:flex;flex-direction:column;gap:4px;margin-bottom:8px}
.key-id{font-size:12.5px;overflow-wrap:anywhere}
.key-uid,.key-meta{color:var(--text-muted);font-size:12px}
.key-actions{display:flex;flex-wrap:wrap;gap:6px}
</style>
