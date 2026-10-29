<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import RuntimeInstall from './RuntimeInstall.vue'

interface Selection { installationId?: string; unresolvedHome?: string; status: 'active' | 'unresolved' | 'empty'; source: string }
interface Installation { id: string; runtime: string; version: string; major: string; localHome: string; source: string; label?: string; vendor?: string; available: boolean }
interface UserState { schema: number; revision: number; globalSelections: Record<string, Selection>; installations: Installation[]; legacyImports: Array<{ sourceDigest: string; importedAt: string; backupPath: string; created: number; linked: number; unresolved: number; conflicts: number }> }
interface LegacyInspection { found: boolean; sourcePath: string; readable: boolean; valid?: boolean; schema?: number; sourceDigest?: string }
interface LegacyPreview { sourceId: string; sourcePath: string; sourceSchema: number; sourceDigest: string; selections: Record<string, Selection>; installations: Installation[]; unresolved: string[]; unsupported: string[]; duplicateCount: number; backupDirectory: string }
interface RuntimeCandidate { candidateId: string; digest: string; tool: string; version: string; major: string; home: string; binary?: string; source: string; evidence: string[]; availability: string; confidence: string; vendor?: string }
interface DiscoveryResult { status: string; candidates: RuntimeCandidate[]; issues: Array<{path: string; code: string; message: string}> }
interface RegistrationResult { state: UserState; counts: {created: number; linked: number; skipped: number} }

const tab = ref('defaults')
const state = ref<UserState>()
const inspection = ref<LegacyInspection>()
const preview = ref<LegacyPreview>()
const busy = ref(false)
const errorText = ref('')
const discoveryRoot = ref('')
const candidates = ref<RuntimeCandidate[]>([])
const selectedCandidates = ref<string[]>([])
const discoveryStatus = ref('')
const inspectTool = ref('node')
const scanStatusLabel: Record<string, string> = {
  complete: '扫描完成',
  partial: '部分完成：部分目录未能读取',
  cancelled: '已取消',
}

async function reload() {
  busy.value = true
  errorText.value = ''
  try {
    ;[state.value, inspection.value] = await Promise.all([
      invoke<UserState>('runtime.getUserState'),
      invoke<LegacyInspection>('legacy.inspectDevSwitch'),
    ])
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { busy.value = false }
}

async function previewImport() {
  busy.value = true
  errorText.value = ''
  try { preview.value = await invoke<LegacyPreview>('legacy.previewDevSwitchImport') }
  catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

function importLegacy() {
  if (!preview.value) return
  const snapshot = preview.value
  Modal.confirm({
    title: '备份并导入 DevSwitch 用户设置？',
    content: `导入前会将 state.json 备份到 ${snapshot.backupDirectory}。旧文件和运行时目录不会被修改。`,
    okText: '备份并导入', cancelText: '取消',
    onOk: async () => {
      busy.value = true
      try {
        await invoke('legacy.importDevSwitch', { sourceId: snapshot.sourceId, expectedSourceDigest: snapshot.sourceDigest, confirmBackup: true, confirmImport: true })
        preview.value = undefined
        message.success('用户级运行时设置已导入')
        await reload()
      } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
      finally { busy.value = false }
    },
  })
}

async function setDefault(runtime: string, installationId: string) {
  if (!state.value) return
  busy.value = true
  try {
    state.value = await invoke<UserState>('runtime.setGlobalDefault', { runtime, installationId, expectedRevision: state.value.revision, confirm: true })
    message.success(`${runtime} 全局默认已更新`)
  } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

async function discoverRuntimes() {
  busy.value = true
  errorText.value = ''
  selectedCandidates.value = []
  try {
    const roots = discoveryRoot.value.trim() ? [discoveryRoot.value.trim()] : []
    const result = await invoke<DiscoveryResult>('runtime.discover', { roots })
    candidates.value = result.candidates
    discoveryStatus.value = result.status
    if (result.issues.length) errorText.value = result.issues.map((issue) => `${issue.path}: ${issue.message}`).join('\n')
  } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

async function inspectRuntimePath() {
  if (!discoveryRoot.value.trim()) return
  busy.value = true
  errorText.value = ''
  try {
    const candidate = await invoke<RuntimeCandidate>('runtime.inspectPath', { tool: inspectTool.value, home: discoveryRoot.value.trim() })
    if (!candidates.value.some((item) => item.candidateId === candidate.candidateId)) candidates.value = [...candidates.value, candidate]
    discoveryStatus.value = 'complete'
  } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

function registerCandidates() {
  if (!state.value || selectedCandidates.value.length === 0) return
  const selected = candidates.value.filter((item) => selectedCandidates.value.includes(item.candidateId))
  Modal.confirm({
    title: `登记 ${selected.length} 个本机运行时？`,
    content: '只写入 DevToolbox 用户级清单，不会执行运行时程序、切换全局默认或修改 PATH。',
    okText: '确认登记', cancelText: '取消',
    onOk: async () => {
      busy.value = true
      try {
        const result = await invoke<RegistrationResult>('runtime.register', {
          candidates: selected.map(({ candidateId, digest, tool, home, source }) => ({ candidateId, digest, tool, home, source })),
          expectedRevision: state.value!.revision,
          confirm: true,
        })
        state.value = result.state
        candidates.value = []
        selectedCandidates.value = []
        message.success(`已登记 ${result.counts.created} 个，已存在 ${result.counts.linked} 个，跳过 ${result.counts.skipped} 个`)
      } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
      finally { busy.value = false }
    },
  })
}

function selectedLabel(runtime: string): string {
  const selection = state.value?.globalSelections[runtime]
  if (!selection || selection.status === 'empty') return '未选择'
  if (selection.status === 'unresolved') return `待重新关联 · ${selection.unresolvedHome ?? ''}`
  const item = state.value?.installations.find((entry) => entry.id === selection.installationId)
  return item ? `${item.version || '版本未知'} · ${item.available ? '可用' : '路径不可用'}` : '待重新关联'
}

onMounted(reload)
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
  <a-tabs v-model:activeKey="tab" class="settings-tabs">

    <a-tab-pane key="defaults" tab="全局默认">
      <div class="group-actions"><a-button :loading="busy" @click="reload">刷新</a-button></div>
      <a-empty v-if="!state?.installations.length" description="尚无已登记运行时；可在「获取运行时」下载，或「导入」迁移 DevSwitch 用户状态。" />
      <div v-for="tool in ['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python']" :key="tool" class="runtime-pref-row">
        <div><strong>{{ tool }}</strong><small>{{ selectedLabel(tool) }}</small></div>
        <a-space wrap>
          <a-button v-for="item in state?.installations.filter((entry) => entry.runtime === tool) ?? []" :key="item.id" size="small" :disabled="!item.available || state?.globalSelections[tool]?.installationId === item.id" @click="setDefault(tool, item.id)">
            设为默认 {{ item.version || item.label || item.localHome }}
          </a-button>
        </a-space>
      </div>
    </a-tab-pane>

    <a-tab-pane key="install" tab="获取运行时">
      <RuntimeInstall />
    </a-tab-pane>

    <a-tab-pane key="discover" tab="发现与登记">
      <div class="toolbar">
        <a-input v-model:value="discoveryRoot" class="grow" placeholder="可选：指定目录（只检查该目录及直接子目录）" />
        <a-select v-model:value="inspectTool" style="width: 132px" :options="['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python'].map((value) => ({ label: value, value }))" />
        <a-button :loading="busy" @click="discoverRuntimes">扫描本机</a-button>
        <a-button :loading="busy" :disabled="!discoveryRoot.trim()" @click="inspectRuntimePath">检查指定目录</a-button>
        <a-button type="primary" :disabled="!selectedCandidates.length" :loading="busy" @click="registerCandidates">登记所选</a-button>
      </div>
      <a-alert v-if="discoveryStatus" :type="discoveryStatus === 'partial' ? 'warning' : 'info'" :message="scanStatusLabel[discoveryStatus] ?? discoveryStatus" show-icon class="page-alert" />
      <a-empty v-if="!candidates.length" description="尚无待登记候选" />
      <a-checkbox-group v-else v-model:value="selectedCandidates" class="candidate-list">
        <div v-for="item in candidates" :key="item.candidateId" class="candidate-row">
          <a-checkbox :value="item.candidateId" />
          <div><strong>{{ item.tool }}，{{ item.version }}</strong><small>{{ item.home }}</small><small>来源 {{ item.source }}，证据 {{ item.confidence }}，{{ item.evidence.join('、') }}</small></div>
          <a-tag :color="item.availability === 'available' ? 'green' : 'orange'">{{ item.availability }}</a-tag>
        </div>
      </a-checkbox-group>
    </a-tab-pane>

    <a-tab-pane key="import" tab="导入">
      <div class="group-label">从 DevSwitch 迁移用户设置</div>
      <template v-if="inspection?.found">
        <div class="source-box"><span>来源</span><code>{{ inspection.sourcePath }}</code><a-tag>{{ inspection.valid ? `schema ${inspection.schema}` : '格式无效或不支持' }}</a-tag></div>
        <a-alert v-if="!inspection.valid" type="warning" show-icon message="当前来源不能导入，请保留旧状态文件并先处理格式问题。" />
        <a-button v-else type="primary" :loading="busy" @click="previewImport">预览导入</a-button>
      </template>
      <a-empty v-else description="未发现 DevSwitch 用户状态文件" />
      <template v-if="preview">
        <a-divider />
        <div class="preview-grid">
          <div class="preview-stat"><strong>{{ preview.installations.length }}</strong><span>运行时条目</span></div>
          <div class="preview-stat"><strong>{{ preview.unresolved.length }}</strong><span>待重新关联</span></div>
          <div class="preview-stat"><strong>{{ preview.duplicateCount }}</strong><span>重复条目</span></div>
          <div class="preview-stat"><strong>{{ preview.unsupported.length }}</strong><span>未支持条目</span></div>
        </div>
        <p class="muted">导入前备份目录：{{ preview.backupDirectory }}</p>
        <div v-for="item in preview.installations" :key="item.id" class="runtime-pref-row">
          <div><strong>{{ item.runtime }} {{ item.version || '版本未知' }}</strong><small>{{ item.localHome }}</small></div>
          <a-tag :color="item.available ? 'green' : 'orange'">{{ item.available ? '路径可用' : '路径不可用' }}</a-tag>
        </div>
        <div class="form-actions"><a-button @click="preview = undefined">取消</a-button><a-button type="primary" :loading="busy" @click="importLegacy">备份并导入</a-button></div>
      </template>
      <template v-if="state?.legacyImports.length">
        <a-divider />
        <div class="group-label">导入记录</div>
        <div v-for="record in state.legacyImports" :key="record.sourceDigest" class="repo-row"><div><strong>{{ new Date(record.importedAt).toLocaleString() }}</strong><small>新建 {{ record.created }}，关联 {{ record.linked }}，待处理 {{ record.unresolved }}，冲突 {{ record.conflicts }}</small></div><a-tag>{{ record.sourceDigest.slice(0, 12) }}</a-tag></div>
      </template>
    </a-tab-pane>
  </a-tabs>
</template>

<style scoped>
.group-label{font-weight:600;font-size:13.5px;color:var(--text-title);margin:0 0 10px}
.group-actions{display:flex;justify-content:flex-end;gap:9px;margin-bottom:14px}
.runtime-pref-row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 0;border-top:1px solid var(--border-soft)}.runtime-pref-row strong,.runtime-pref-row small{display:block;overflow-wrap:anywhere}.runtime-pref-row small{margin-top:5px;color:var(--text-muted)}.source-box{display:flex;align-items:center;gap:14px;flex-wrap:wrap;margin-bottom:18px;padding:12px 14px;border-radius:8px;background:var(--bg-soft)}.source-box code{overflow-wrap:anywhere;color:var(--text-code)}.candidate-list{display:block}.candidate-row{display:flex;align-items:center;gap:12px;padding:11px 0;border-top:1px solid var(--border-soft)}.candidate-row>div{flex:1;min-width:0}.candidate-row strong,.candidate-row small{display:block;overflow-wrap:anywhere}.candidate-row small{margin-top:4px;color:var(--text-muted)}.settings-tabs :deep(.ant-tabs-tab){padding:8px 4px}@media(max-width:640px){.runtime-pref-row{align-items:flex-start;flex-direction:column}.candidate-row{align-items:flex-start}}
</style>
