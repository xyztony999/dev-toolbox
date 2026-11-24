<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import { customAccent, normalizeAccent, setTheme, themePref, validAccent, type ThemePref } from './theme'

interface ClientInfo {
  version: string
  scanToolVersion: string
  configDir: string
  projectStore: string
  runtimeState: string
  gitIdentity: string
  os: string
  arch: string
}

interface Settings {
  schema: number
  revision: number
  theme: ThemePref
  customAccent?: string
  sourceProjects: SourceProject[]
  snapshots: SnapshotRecord[]
}

interface SourceProject {
  path: string
  label?: string
  binding: 'local' | 'shared'
  note?: string
}

interface SnapshotRecord {
  sourceDigest: string
  label?: string
  backupPath?: string
  importedAt: string
  applied: number
  skipped: number
}

interface ToolStatus {
  tool: string
  available: boolean
  version?: string
  home?: string
  source: string
  configurable: boolean
  configNote?: string
}

interface SettingsResult {
  settings: Settings
  removed?: boolean
  record?: SnapshotRecord
  backedUp?: boolean
}

interface SnapshotPayload {
  snapshot: { digest: string; label?: string }
  payload: string
}

const themeOptions: Array<{ id: ThemePref; label: string }> = [
  { id: 'system', label: '跟随系统' },
  { id: 'light', label: '浅色' },
  { id: 'dark', label: '深色' },
  { id: 'custom', label: '自定义' },
]

const info = ref<ClientInfo>()
const settings = ref<Settings>()
const tools = ref<ToolStatus[]>([])
const errorText = ref('')
const accentDraft = ref('')

const projectForm = reactive({ path: '', label: '', note: '', binding: 'local' as 'local' | 'shared' })
const savingProject = ref(false)

const theme = computed(() => themePref.value)
const sourceProjects = computed(() => settings.value?.sourceProjects ?? [])
const localCount = computed(() => sourceProjects.value.filter((item) => item.binding === 'local').length)
const sharedCount = computed(() => settings.value?.sourceProjects.filter((item) => item.binding === 'shared').length ?? 0)
const snapshots = computed(() => settings.value?.snapshots ?? [])
const accentValid = computed(() => accentDraft.value.trim() === '' || validAccent(accentDraft.value))
const accentDisplay = computed(() => normalizeAccent(accentDraft.value) || '#a34b2a')

onMounted(load)

async function load() {
  errorText.value = ''
  try {
    const [nextInfo, nextSettings, nextTools] = await Promise.all([
      invoke<ClientInfo>('app.info'),
      invoke<Settings>('settings.get'),
      invoke<ToolStatus[]>('settings.toolStatuses'),
    ])
    info.value = nextInfo
    settings.value = nextSettings
    tools.value = nextTools
    accentDraft.value = nextSettings.customAccent || customAccent.value
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}

function choose(pref: ThemePref) {
  if (pref === 'custom' && !accentValid.value) {
    message.error('请先填写合法的十六进制颜色，例如 #a34b2a')
    return
  }
  setTheme(pref, pref === 'custom' ? normalizeAccent(accentDraft.value) : undefined)
  persistTheme(pref)
}

async function persistTheme(pref: ThemePref) {
  if (!settings.value) return
  try {
    settings.value = await invoke<Settings>('settings.setTheme', {
      mode: pref,
      accent: pref === 'custom' ? normalizeAccent(accentDraft.value) : '',
      expectedRevision: settings.value.revision,
    })
    message.success('外观已保存')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
    await load()
  }
}

function applyAccentNow() {
  if (!accentValid.value) {
    message.error('颜色格式应为 #rgb 或 #rrggbb')
    return
  }
  const accent = normalizeAccent(accentDraft.value)
  if (!accent) {
    message.error('颜色格式应为 #rgb 或 #rrggbb')
    return
  }
  setTheme('custom', accent)
}

async function saveAccent() {
  if (!settings.value || !accentValid.value) return
  const accent = normalizeAccent(accentDraft.value)
  if (!accent) return
  await persistTheme('custom')
  accentDraft.value = settings.value?.customAccent ?? accent
}

async function copyPath(path: string) {
  try {
    await navigator.clipboard.writeText(path)
    message.success('路径已复制')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '复制失败')
  }
}

async function openConfig() {
  if (!info.value) return
  try {
    await invoke('app.open', { kind: 'config', target: info.value.configDir })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}

async function addProject() {
  if (!settings.value) return
  const path = projectForm.path.trim()
  if (!path) {
    message.error('请填写源项目目录')
    return
  }
  savingProject.value = true
  errorText.value = ''
  try {
    settings.value = await invoke<Settings>('settings.addSourceProject', {
      path,
      label: projectForm.label.trim(),
      note: projectForm.note.trim(),
      binding: projectForm.binding,
      expectedRevision: settings.value.revision,
    })
    projectForm.path = ''
    projectForm.label = ''
    projectForm.note = ''
    message.success('源项目已登记')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    savingProject.value = false
  }
}

function removeProject(item: SourceProject) {
  if (!settings.value) return
  Modal.confirm({
    title: '移除这个源项目？',
    content: `只会从本机设置中移除 ${item.path}；不会删除磁盘上的任何文件。`,
    okText: '移除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        const result = await invoke<SettingsResult>('settings.removeSourceProject', {
          path: item.path,
          expectedRevision: settings.value!.revision,
        })
        settings.value = result.settings
        message.success(result.removed ? '源项目已移除' : '该源项目已不存在')
      } catch (error) {
        errorText.value = error instanceof Error ? error.message : String(error)
      }
    },
  })
}

const importInput = { current: null as HTMLInputElement | null }

function openImportDialog() {
  importInput.current?.click()
}

async function onImportFile(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = ''
  if (!file) return
  let text = ''
  try {
    text = await file.text()
  } catch {
    message.error('无法读取所选文件')
    return
  }
  let parsed: { digest?: string; label?: string } | undefined
  try {
    parsed = JSON.parse(text)
  } catch {
    message.error('这不是有效的 JSON 快照文件')
    return
  }
  if (!parsed || !parsed.digest) {
    message.error('缺少摘要字段，拒绝导入')
    return
  }
  Modal.confirm({
    title: '导入会话快照？',
    content: `摘要 ${parsed.digest.slice(0, 12)}；导入会写入本机设置并备份原始文件。相同摘要重复导入不会产生新记录。`,
    okText: '备份并导入',
    cancelText: '取消',
    onOk: async () => {
      if (!settings.value) return
      try {
        const result = await invoke<SettingsResult>('settings.importSnapshot', {
          payload: text,
          confirm: true,
          confirmBackup: true,
          expectedRevision: settings.value.revision,
        })
        settings.value = result.settings
        accentDraft.value = result.settings.customAccent || ''
        setTheme(result.settings.theme, result.settings.customAccent)
        message.success(result.record?.applied ? `已应用 ${result.record.applied} 项，备份 ${result.record.skipped} 项` : '该快照此前已导入')
      } catch (error) {
        errorText.value = error instanceof Error ? error.message : String(error)
      }
    },
  })
}
function exportSnapshot() {
  Modal.confirm({
    title: '导出会话快照',
    content: '默认只导出共享项目与外观偏好；不含 token、Git 身份选择或本机私有路径。可在下方勾选包含本机绑定项目。',
    okText: '仅共享项目',
    cancelText: '取消',
    onOk: async () => downloadSnapshot(false),
  })
}

async function downloadSnapshot(includeLocal: boolean) {
  try {
    const result = await invoke<SnapshotPayload>('settings.exportSnapshot', {
      label: 'dev-toolbox-settings',
      includeSharedProjects: includeLocal,
    })
    const blob = new Blob([result.payload], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `dev-toolbox-snapshot-${result.snapshot.digest.slice(0, 12)}.json`
    anchor.click()
    URL.revokeObjectURL(url)
    message.success('快照已导出')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  }
}

function bindingLabel(binding: string): string {
  return binding === 'shared' ? '共享' : '本机'
}

function toolLabel(item: ToolStatus): string {
  if (!item.available) return '未探测到'
  return item.version || 'unknown'
}
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />
  <div class="doc">
    <section class="doc-section">
      <h2>外观</h2>
      <p class="doc-note">跟随系统时，和 Windows 的浅色或深色一起变。自定义强调色只影响本客户端的界面配色。</p>
      <div class="theme-pick" role="radiogroup" aria-label="外观">
        <button v-for="item in themeOptions" :key="item.id" type="button" role="radio" :aria-checked="theme === item.id" :class="{ on: theme === item.id }" @click="choose(item.id)">{{ item.label }}</button>
      </div>
      <div v-if="theme === 'custom'" class="theme-custom">
        <span class="theme-custom-label">强调色</span>
        <input v-model="accentDraft" class="theme-swatch" type="color" :aria-label="'强调色，当前 ' + accentDisplay" :value="accentDisplay" @input="applyAccentNow" />
        <a-input v-model:value="accentDraft" :allow-clear="true" style="max-width: 120px" placeholder="#a34b2a" @blur="saveAccent" @press-enter="saveAccent" />
        <span v-if="!accentValid" class="doc-note" style="color: var(--accent-strong)">格式应为 #rgb 或 #rrggbb</span>
      </div>
    </section>

    <section class="doc-section">
      <h2>源项目清单</h2>
      <p class="doc-note">
        「本机」条目只在本机有意义；「共享」条目可随快照分发，不能指向用户主目录内。清单不含身份选择或凭据。
      </p>
      <div class="toolbar">
        <a-tag color="blue">本机 {{ localCount }}</a-tag>
        <a-tag color="green">共享 {{ sharedCount }}</a-tag>
      </div>
      <div v-if="sourceProjects.length" class="source-list">
        <div v-for="item in sourceProjects" :key="item.path" class="source-row">
          <span class="source-path">{{ item.path }}</span>
          <span class="source-meta">
            <a-tag :color="item.binding === 'shared' ? 'green' : 'blue'">{{ bindingLabel(item.binding) }}</a-tag>
            <span v-if="item.label" class="source-meta">{{ item.label }}</span>
          </span>
          <span class="kv-actions">
            <button type="button" @click="copyPath(item.path)">复制</button>
            <button type="button" @click="removeProject(item)">移除</button>
          </span>
        </div>
      </div>
      <p v-else class="source-empty">还没有登记源项目。</p>
      <div class="source-form">
        <a-input v-model:value="projectForm.path" class="source-form-path" placeholder="源项目目录的完整路径" />
        <a-input v-model:value="projectForm.label" class="source-form-label" placeholder="备注名（可选）" />
        <a-select v-model:value="projectForm.binding" class="source-form-binding" :options="[{ value: 'local', label: '本机' }, { value: 'shared', label: '共享' }]" />
        <a-button type="primary" :loading="savingProject" @click="addProject">登记</a-button>
      </div>
    </section>

    <section class="doc-section">
      <h2>运行时工具状态</h2>
      <p class="doc-note">只读探测：读取已登记安装与 PATH 上的可执行文件，不会执行工具本身，也不切换全局默认或修改真实 PATH。</p>
      <div class="tool-status-list">
        <div v-for="item in tools" :key="item.tool" class="tool-status-row">
          <span class="tool-status-name">{{ item.tool }}</span>
          <a-tag :color="item.available ? 'green' : 'default'">{{ item.available ? '可用' : '未探测到' }}</a-tag>
          <span class="tool-status-version">{{ toolLabel(item) }}</span>
          <span class="tool-status-note">{{ item.configurable ? '可写入工具原生配置' : (item.configNote || '只读') }}</span>
        </div>
      </div>
    </section>

    <section class="doc-section">
      <h2>会话快照</h2>
      <p class="doc-note">导出只包含外观偏好与共享项目；导入前会备份原始文件，同一摘要重复导入不会新增记录。</p>
      <div class="snapshot-bar">
        <a-button @click="exportSnapshot">导出快照</a-button>
        <a-button @click="downloadSnapshot(true)">导出（含本机条目）</a-button>
        <a-button type="primary" @click="openImportDialog">导入快照</a-button>
        <input :ref="(el: any) => (importInput.current = el)" class="snapshot-file" type="file" accept="application/json,.json" @change="onImportFile" />
      </div>
      <div v-if="snapshots.length" class="kv" style="margin-top: 12px">
        <div v-for="item in snapshots" :key="item.sourceDigest" class="kv-row">
          <span class="kv-k">快照</span>
          <span class="kv-v">{{ item.label || item.sourceDigest.slice(0, 12) }}</span>
          <span class="kv-actions"><button type="button" @click="copyPath(item.sourceDigest)">复制摘要</button></span>
        </div>
      </div>
    </section>

    <section class="doc-section">
      <h2>本机数据</h2>
      <p class="doc-note">登记、运行时和 Git 身份都写在这个目录，不进项目仓库。</p>
      <div v-if="info" class="kv">
        <div class="kv-row"><span class="kv-k">配置目录</span><span class="kv-v">{{ info.configDir }}</span><span class="kv-actions"><button type="button" @click="copyPath(info.configDir)">复制</button><button type="button" @click="openConfig">打开</button></span></div>
        <div class="kv-row"><span class="kv-k">项目登记</span><span class="kv-v">{{ info.projectStore }}</span><button type="button" @click="copyPath(info.projectStore)">复制</button></div>
        <div class="kv-row"><span class="kv-k">运行时状态</span><span class="kv-v">{{ info.runtimeState }}</span><button type="button" @click="copyPath(info.runtimeState)">复制</button></div>
        <div class="kv-row"><span class="kv-k">Git 身份</span><span class="kv-v">{{ info.gitIdentity }}</span><button type="button" @click="copyPath(info.gitIdentity)">复制</button></div>
        <div class="kv-row"><span class="kv-k">版本</span><span class="kv-v">{{ info.version }} / 扫描器 {{ info.scanToolVersion }}</span><span class="kv-actions"><a-tag>{{ info.os }}-{{ info.arch }}</a-tag></span></div>
      </div>
    </section>
  </div>
</template>
