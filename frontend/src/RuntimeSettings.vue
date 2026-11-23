<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import RuntimeInstall from './RuntimeInstall.vue'

interface GoEnvChange { key: string; from: string; to: string; action: 'set' | 'unset' }
interface ToolConfigItem { key: string; label: string; value: string; editable: boolean; source: string; presets?: Array<{ id: string; label: string; value: string }>; note?: string }
interface ToolConfigReport { tool: string; binary?: string; items: ToolConfigItem[]; note?: string }
interface ToolConfigSetResult { change: GoEnvChange; report: ToolConfigReport }
interface RemovePreview { id: string; runtime: string; version: string; source: string; localHome: string; ownedByProduct: boolean; willDeleteDir: boolean; isDefault: boolean }
interface RemoveResult { preview: RemovePreview; state: UserState; removed: boolean; dirDeleted: boolean; wasDefault: boolean }

interface Selection { installationId?: string; unresolvedHome?: string; status: 'active' | 'unresolved' | 'empty'; source: string }
interface Installation { id: string; runtime: string; version: string; major: string; localHome: string; source: string; label?: string; vendor?: string; available: boolean }
interface UserState { schema: number; revision: number; globalSelections: Record<string, Selection>; installations: Installation[]; legacyImports: Array<{ sourceDigest: string; importedAt: string; backupPath: string; created: number; linked: number; unresolved: number; conflicts: number }> }
interface LegacyInspection { found: boolean; sourcePath: string; readable: boolean; valid?: boolean; schema?: number; sourceDigest?: string }
interface LegacyPreview { sourceId: string; sourcePath: string; sourceSchema: number; sourceDigest: string; selections: Record<string, Selection>; installations: Installation[]; unresolved: string[]; unsupported: string[]; duplicateCount: number; backupDirectory: string }
interface RuntimeCandidate { candidateId: string; digest: string; tool: string; version: string; major: string; home: string; binary?: string; source: string; evidence: string[]; availability: string; confidence: string; vendor?: string }
interface DiscoveryResult { status: string; candidates: RuntimeCandidate[]; issues: Array<{path: string; code: string; message: string}> }
interface RegistrationResult { state: UserState; counts: {created: number; linked: number; skipped: number} }

const tab = ref('tools')
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

// ---- 运行时统一管理：默认版本 / 版本管理 / 环境配置 ----
const TOOLS = ['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python', 'rust']
const toolConfigs = reactive<Record<string, ToolConfigReport | undefined>>({})
const toolEdits = reactive<Record<string, Record<string, string>>>({})
const toolPreviews = reactive<Record<string, GoEnvChange | undefined>>({})
const activeTools = ref<string[]>(['golang'])

async function loadToolConfig(tool: string) {
  try {
    const report = await invoke<ToolConfigReport>('runtime.toolConfigRead', { tool })
    toolConfigs[tool] = report
    const edits: Record<string, string> = {}
    for (const item of report.items) if (item.editable) edits[item.key] = item.value
    toolEdits[tool] = edits
  } catch (error) {
    toolConfigs[tool] = { tool, items: [], note: error instanceof Error ? error.message : String(error) }
  }
}

function onCollapseChange(keys: unknown) {
  const list = Array.isArray(keys) ? (keys as string[]) : [keys as string]
  activeTools.value = list
  for (const tool of list) if (!toolConfigs[tool]) void loadToolConfig(tool)
}

async function previewToolSet(tool: string, key: string) {
  busy.value = true
  errorText.value = ''
  try {
    const res = await invoke<ToolConfigSetResult>('runtime.toolConfigWrite', { tool, key, value: toolEdits[tool]?.[key] ?? '', dryRun: true })
    toolPreviews[`${tool}/${key}`] = res.change
  } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

function applyToolSet(tool: string, key: string, unset: boolean) {
  const target = toolConfigs[tool]?.items.find((item) => item.key === key)
  Modal.confirm({
    title: unset ? `恢复 ${key} 为工具默认？` : `写入 ${key}？`,
    content: unset
      ? `清除该用户级配置，恢复 ${tool} 自身默认。不改系统环境。`
      : `写入 ${tool} 的用户级配置（${target?.source ?? ''}）：${key}=${toolEdits[tool]?.[key] ?? ''}。不改系统 PATH/注册表。`,
    okText: unset ? '恢复默认' : '应用', cancelText: '取消',
    onOk: async () => {
      busy.value = true
      try {
        await invoke<ToolConfigSetResult>('runtime.toolConfigWrite', { tool, key, value: toolEdits[tool]?.[key] ?? '', unset, confirm: true })
        toolPreviews[`${tool}/${key}`] = undefined
        message.success(unset ? `${key} 已恢复默认` : `${key} 已应用`)
        await loadToolConfig(tool)
      } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
      finally { busy.value = false }
    },
  })
}

function applyPreset(tool: string, key: string, value: string) {
  if (!toolEdits[tool]) toolEdits[tool] = {}
  toolEdits[tool][key] = value
  toolPreviews[`${tool}/${key}`] = undefined
}

function installationsOf(tool: string): Installation[] {
  return state.value?.installations.filter((item) => item.runtime === tool) ?? []
}

function isDefault(tool: string, inst: Installation): boolean {
  return state.value?.globalSelections[tool]?.installationId === inst.id
}

async function removeInstallation(inst: Installation) {
  if (!state.value) return
  busy.value = true
  errorText.value = ''
  try {
    const dry = await invoke<RemoveResult>('runtime.removeInstallation', {
      id: inst.id, expectedRevision: state.value.revision, deleteFiles: inst.source === 'downloaded', dryRun: true,
    })
    const p = dry.preview
    Modal.confirm({
      title: p.willDeleteDir ? `卸载 ${inst.runtime} ${inst.version}？` : `注销 ${inst.runtime} ${inst.version} 的登记？`,
      content: (p.willDeleteDir
        ? `将删除产品自有目录 ${p.localHome} 并移除记录。`
        : `仅移除登记记录，不删除外部目录 ${p.localHome}。`)
        + (p.isDefault ? ' 它当前是全局默认，将一并清除。' : ''),
      okText: p.willDeleteDir ? '卸载' : '注销', okButtonProps: { danger: true }, cancelText: '取消',
      onOk: async () => {
        busy.value = true
        try {
          const res = await invoke<RemoveResult>('runtime.removeInstallation', {
            id: inst.id, expectedRevision: state.value!.revision, deleteFiles: inst.source === 'downloaded', confirm: true,
          })
          state.value = res.state
          message.success(res.dirDeleted ? '已卸载并删除目录' : '已注销登记')
        } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
        finally { busy.value = false }
      },
    })
  } catch (error) { errorText.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false }
}

onMounted(() => { void loadToolConfig('golang') })

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

    <a-tab-pane key="tools" tab="运行时">
      <div class="group-actions"><a-button :loading="busy" @click="reload">刷新</a-button></div>
      <a-collapse v-model:activeKey="activeTools" class="tool-collapse" @change="onCollapseChange">
        <a-collapse-panel v-for="tool in TOOLS" :key="tool" :header="tool">
          <template #extra><span class="collapse-extra">{{ selectedLabel(tool) }}</span></template>

          <div class="group-label">版本管理</div>
          <a-empty v-if="!installationsOf(tool).length" :description="`尚无 ${tool} 安装记录；可在「获取运行时」下载或「发现与登记」登记。`" />
          <div v-for="inst in installationsOf(tool)" :key="inst.id" class="runtime-pref-row">
            <div><strong>{{ tool }} {{ inst.version || inst.label || '版本未知' }}</strong><small>{{ inst.localHome }}</small></div>
            <a-space wrap>
              <a-tag :color="inst.source === 'downloaded' ? 'blue' : 'default'">{{ inst.source === 'downloaded' ? '已下载' : '已登记' }}</a-tag>
              <a-tag v-if="isDefault(tool, inst)" color="green">全局默认</a-tag>
              <a-button size="small" :disabled="isDefault(tool, inst) || !inst.available" :loading="busy" @click="setDefault(tool, inst.id)">设为默认</a-button>
              <a-button size="small" danger :loading="busy" @click="removeInstallation(inst)">{{ inst.source === 'downloaded' ? '卸载' : '注销' }}</a-button>
            </a-space>
          </div>

          <div class="group-label">环境配置</div>
          <template v-if="toolConfigs[tool]">
            <a-alert v-if="toolConfigs[tool]?.note" type="info" show-icon :message="toolConfigs[tool]?.note" class="page-alert" />
            <a-empty v-if="!toolConfigs[tool]?.items.length && !toolConfigs[tool]?.note" description="该工具暂无可管理的环境配置" />
            <div v-for="item in toolConfigs[tool]?.items ?? []" :key="item.key" class="go-edit-row">
              <template v-if="item.editable">
                <div class="go-edit-head"><strong>{{ item.label }}</strong><small>当前 {{ item.value || '未设置' }} · {{ item.source }}</small></div>
                <div class="toolbar">
                  <a-input v-model:value="toolEdits[tool][item.key]" class="grow" :placeholder="item.label" @change="toolPreviews[tool + '/' + item.key] = undefined" />
                  <a-select v-if="item.presets?.length" style="width: 190px" placeholder="预设" :options="item.presets.map((p) => ({ label: p.label, value: p.value }))" @change="(value: unknown) => applyPreset(tool, item.key, String(value))" />
                  <a-button :loading="busy" @click="previewToolSet(tool, item.key)">预览</a-button>
                  <a-button type="primary" :loading="busy" @click="applyToolSet(tool, item.key, false)">应用</a-button>
                  <a-button :loading="busy" :disabled="!item.value" @click="applyToolSet(tool, item.key, true)">恢复默认</a-button>
                </div>
                <div v-if="toolPreviews[tool + '/' + item.key]" class="go-preview"><code>{{ toolPreviews[tool + '/' + item.key]?.from || '未设置' }}</code><span>→</span><code>{{ toolPreviews[tool + '/' + item.key]?.to || '（清除）' }}</code></div>
                <p v-if="item.note" class="muted">{{ item.note }}</p>
              </template>
              <template v-else>
                <div class="go-readonly"><span class="go-chip"><em>{{ item.label }}</em>{{ item.value || '—' }}</span></div>
                <p v-if="item.note" class="muted">{{ item.note }}</p>
              </template>
            </div>
          </template>
          <a-skeleton v-else active :paragraph="{ rows: 2 }" />
        </a-collapse-panel>
      </a-collapse>
    </a-tab-pane>

    <a-tab-pane key="install" tab="获取运行时">
      <RuntimeInstall />
    </a-tab-pane>

    <a-tab-pane key="discover" tab="发现与登记">
      <div class="toolbar">
        <a-input v-model:value="discoveryRoot" class="grow" placeholder="可选：指定目录（只检查该目录及直接子目录）" />
        <a-select v-model:value="inspectTool" style="width: 132px" :options="['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python', 'rust'].map((value) => ({ label: value, value }))" />
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
.runtime-pref-row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 0;border-top:1px solid var(--border-soft)}.runtime-pref-row strong,.runtime-pref-row small{display:block;overflow-wrap:anywhere}.runtime-pref-row small{margin-top:5px;color:var(--text-muted)}.source-box{display:flex;align-items:center;gap:14px;flex-wrap:wrap;margin-bottom:18px;padding:12px 14px;border-radius:8px;background:var(--bg-soft)}.source-box code{overflow-wrap:anywhere;color:var(--text-code)}.candidate-list{display:block}.candidate-row{display:flex;align-items:center;gap:12px;padding:11px 0;border-top:1px solid var(--border-soft)}.candidate-row>div{flex:1;min-width:0}.candidate-row strong,.candidate-row small{display:block;overflow-wrap:anywhere}.candidate-row small{margin-top:4px;color:var(--text-muted)}.settings-tabs :deep(.ant-tabs-tab){padding:8px 4px}
.go-readonly{display:flex;flex-wrap:wrap;gap:8px;margin:0 0 16px}
.go-chip{display:inline-flex;align-items:center;gap:8px;padding:4px 10px;border-radius:6px;background:var(--bg-chip);font-size:12px;color:var(--text-code);overflow-wrap:anywhere}
.go-chip em{font-style:normal;color:var(--text-muted)}
.go-edit-row{padding:12px 0;border-top:1px solid var(--border-soft)}
.go-edit-head{display:flex;align-items:baseline;gap:12px;margin-bottom:8px}
.go-edit-head strong{color:var(--text-title);font-size:13.5px}
.go-edit-head small{color:var(--text-muted);overflow-wrap:anywhere}
.go-preview{display:flex;align-items:center;gap:8px;margin-top:8px;color:var(--text-muted);font-size:12.5px}
.go-preview code{color:var(--text-code);overflow-wrap:anywhere}
.go-preview span{color:var(--text-muted)}
.tool-collapse{background:transparent;border:0}
.tool-collapse :deep(.ant-collapse-item){border-bottom:1px solid var(--border-soft)}
.tool-collapse :deep(.ant-collapse-header){align-items:center}
.collapse-extra{color:var(--text-muted);font-size:12px}
.tool-collapse .group-label{margin-top:4px}@media(max-width:640px){.runtime-pref-row{align-items:flex-start;flex-direction:column}.candidate-row{align-items:flex-start}}
</style>
