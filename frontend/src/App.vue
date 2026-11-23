<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { message, Modal, theme } from 'ant-design-vue'
import { invoke, isDemoClient, isNativeWindow } from './bridge'
import type { Candidate, Project, RuntimeRequirement } from './types'
import RuntimeSettings from './RuntimeSettings.vue'
import GitIdentity from './GitIdentity.vue'
import RemoteManager from './RemoteManager.vue'
import SessionManager from './SessionManager.vue'
import WindowControls from './WindowControls.vue'
import SettingsView from './SettingsView.vue'
import AboutView from './AboutView.vue'
import RuntimeMark from './RuntimeMark.vue'
import BrandMark from './BrandMark.vue'
import { themePref } from './theme'

const prefersDark = window.matchMedia('(prefers-color-scheme: dark)')
const schemeDark = ref(prefersDark.matches)
const onSchemeChange = (event: MediaQueryListEvent) => { schemeDark.value = event.matches }
onMounted(() => prefersDark.addEventListener('change', onSchemeChange))
onBeforeUnmount(() => prefersDark.removeEventListener('change', onSchemeChange))
const isDark = computed(() => themePref.value === 'dark' || (themePref.value === 'system' && schemeDark.value))

const dragArm = { x: 0, y: 0, armed: false }

function endDragArm() {
  dragArm.armed = false
  window.removeEventListener('mousemove', onDragMove)
  window.removeEventListener('mouseup', endDragArm)
}

function onDragMove(event: MouseEvent) {
  if (!dragArm.armed) return
  if (Math.abs(event.screenX - dragArm.x) < 4 && Math.abs(event.screenY - dragArm.y) < 4) return
  endDragArm()
  void window.devToolbox?.window?.drag()
}

function dragWindow(event: MouseEvent) {
  if (!isNativeWindow || event.button !== 0) return
  const target = event.target instanceof Element ? event.target : null
  if (target?.closest('button, a, input, textarea, .win-controls')) return
  dragArm.x = event.screenX
  dragArm.y = event.screenY
  dragArm.armed = true
  window.addEventListener('mousemove', onDragMove)
  window.addEventListener('mouseup', endDragArm)
}

function toggleWindow(event: MouseEvent) {
  if (!isNativeWindow) return
  const target = event.target instanceof Element ? event.target : null
  if (target?.closest('button, a, input, textarea, .win-controls')) return
  void window.devToolbox?.window?.toggleMaximize()
}

function blockBrowserChrome(event: KeyboardEvent) {
  if (!isNativeWindow) return
  if (event.key === 'F5' || ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'r')) event.preventDefault()
}

function blockBrowserMenu(event: MouseEvent) {
  if (!isNativeWindow) return
  const target = event.target instanceof Element ? event.target : null
  if (!target?.closest('input, textarea, [contenteditable="true"]')) event.preventDefault()
}

onMounted(() => {
  document.addEventListener('keydown', blockBrowserChrome, true)
  document.addEventListener('contextmenu', blockBrowserMenu)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', blockBrowserChrome, true)
  document.removeEventListener('contextmenu', blockBrowserMenu)
  endDragArm()
})
const antTheme = computed(() => ({
  algorithm: isDark.value ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: isDark.value ? '#e39a6a' : '#a34b2a',
    colorInfo: isDark.value ? '#e39a6a' : '#a34b2a',
    colorBgBase: isDark.value ? '#1c1a17' : '#f6f3ec',
    colorBorder: isDark.value ? '#3d372f' : '#ddd4c6',
    colorTextBase: isDark.value ? '#f4efe6' : '#221e19',
    borderRadius: 4,
    fontFamily: "'IBM Plex Sans', 'Microsoft YaHei UI', 'Segoe UI Variable Text', 'Segoe UI', sans-serif",
    fontSize: 13,
    controlHeight: 30,
    boxShadow: 'none',
    boxShadowSecondary: isDark.value ? '0 12px 32px rgba(0,0,0,0.45)' : '0 12px 32px rgba(48, 32, 16, 0.14)',
  },
}))

const projects = ref<Project[]>([])
const activeId = ref('')
const candidate = ref<Candidate>()
const rootPath = ref('')
const displayName = ref('')
const busy = ref(false)
const picking = ref(false)
const remoteManager = ref<{ open: boolean; path: string; name: string }>({ open: false, path: '', name: '' })
const sessionOpen = ref(false)
const sessionRepoPaths = computed(() => {
  const paths = [selected.value?.rootPath ?? '', ...(selected.value?.repositories.map((repo) => repo.path) ?? [])]
  return paths.filter(Boolean)
})
const sessionDefaultPath = computed(() => selected.value?.rootPath ?? '.')
const errorText = ref('')
const selected = computed(() => projects.value.find((item) => item.id === activeId.value))
const view = ref<'projects' | 'runtime' | 'git' | 'settings' | 'about'>('projects')
const placeLabel = computed(() => {
  switch (view.value) {
    case 'runtime': return '运行时'
    case 'git': return 'Git'
    case 'settings': return '设置'
    case 'about': return '关于'
    default: return '项目'
  }
})
const viewMeta = computed(() => {
  switch (view.value) {
    case 'runtime':
      return { title: '运行时', hint: '安装、默认版本与环境变量', path: false }
    case 'git':
      return { title: 'Git 身份', hint: '档案、生效范围与仓库审计', path: false }
    case 'settings':
      return { title: '设置', hint: '外观和本机数据位置', path: false }
    case 'about':
      return { title: '关于', hint: '版本、宿主和来源', path: false }
    default:
      if (selected.value) {
        return {
          title: selected.value.displayName,
          hint: `${selected.value.rootPath}  ·  扫描于 ${new Date(selected.value.lastScanAt).toLocaleString()}`,
          path: true,
        }
      }
      return { title: '登记项目', hint: '扫描目录后登记为项目上下文', path: false }
  }
})

function projectMark(name: string): string {
  const trimmed = name.trim()
  return trimmed ? trimmed.slice(0, 1).toUpperCase() : '·'
}

async function loadProjects() {
  busy.value = true
  errorText.value = ''
  try {
    projects.value = await invoke<Project[]>('project.list')
    if (!activeId.value && projects.value.length) activeId.value = projects.value[0].id
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function scanProject() {
  if (!rootPath.value.trim()) return
  busy.value = true
  candidate.value = undefined
  errorText.value = ''
  try {
    candidate.value = await invoke<Candidate>('project.scanPreview', { rootPath: rootPath.value.trim() })
    displayName.value = candidate.value.displayName
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function registerProject() {
  if (!candidate.value) return
  busy.value = true
  try {
    const added = await invoke<Project>('project.register', {
      candidateId: candidate.value.id,
      expectedEvidenceDigest: candidate.value.evidenceDigest,
      rootPath: candidate.value.rootPath,
      displayName: displayName.value,
    })
    activeId.value = added.id
    candidate.value = undefined
    rootPath.value = ''
    await loadProjects()
    message.success('项目上下文已登记')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function rescan() {
  if (!selected.value) return
  busy.value = true
  try {
    const updated = await invoke<Project>('project.rescan', {
      projectId: selected.value.id,
      expectedRevision: selected.value.revision,
    })
    projects.value = projects.value.map((item) => item.id === updated.id ? updated : item)
    message.success('扫描完成')
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

function removeProject() {
  const current = selected.value
  if (!current) return
  Modal.confirm({
    title: `移除“${current.displayName}”？`,
    content: '只移除 DevToolbox 中的登记，不会删除项目目录或修改 Git 配置。',
    okText: '移除登记',
    cancelText: '取消',
    onOk: async () => {
      await invoke('project.unregister', { projectId: current.id, expectedRevision: current.revision, confirm: true })
      activeId.value = ''
      await loadProjects()
    },
  })
}

function requirementLabel(requirement: RuntimeRequirement): string {
  if (requirement.status === 'unknown') return `${requirement.runtime}: 未发现版本要求`
  return `${requirement.runtime}: ${requirement.rawConstraint ?? 'unknown'}`
}

// 详情页功能区导航：登记目录下嵌套多项目时，避免仓库与模块混在一个长列表里滚动。
const detailTab = ref<'overview' | 'repos' | 'modules' | 'issues'>('overview')
const repoFilter = ref('')
const moduleFilter = ref('')
const onlyPendingModules = ref(false)

watch(activeId, () => {
  detailTab.value = 'overview'
  repoFilter.value = ''
  moduleFilter.value = ''
  onlyPendingModules.value = false
})

const filteredRepos = computed(() => {
  const keyword = repoFilter.value.trim().toLowerCase()
  if (!selected.value) return []
  if (!keyword) return selected.value.repositories
  return selected.value.repositories.filter((repo) => repo.path.toLowerCase().includes(keyword))
})

const moduleHasPending = (module: { requirements: RuntimeRequirement[] }): boolean =>
  module.requirements.some((item) => item.status === 'unknown' || item.status === 'conflict')

const filteredModules = computed(() => {
  const keyword = moduleFilter.value.trim().toLowerCase()
  if (!selected.value) return []
  let list = selected.value.modules
  if (onlyPendingModules.value) list = list.filter((module) => moduleHasPending(module))
  if (!keyword) return list
  return list.filter((module) => module.relativePath.toLowerCase().includes(keyword))
})

const requirementSummary = computed(() => {
  const summary = { resolved: 0, unknown: 0, conflict: 0, unsupported: 0 }
  for (const module of selected.value?.modules ?? []) {
    for (const requirement of module.requirements) {
      if (requirement.status in summary) summary[requirement.status as keyof typeof summary] += 1
    }
  }
  return summary
})

const requirementStatusMeta: Record<RuntimeRequirement['status'], { color: string; label: string; explain: string }> = {
  resolved: { color: 'blue', label: '已声明', explain: '构建清单中找到了明确的版本声明，可直接作为选择依据。' },
  unknown: { color: 'default', label: '未声明', explain: '构建清单存在，但没有发现版本声明。产品只如实呈现，不会擅自使用 latest；你可以在清单中补齐版本后再重新扫描。' },
  conflict: { color: 'red', label: '冲突', explain: '同一个运行时有多个互斥的版本声明（如同一份 pom 里两个不同的 Java 版本）。产品不自动替你选一个，请以证据为准人工决定。' },
  unsupported: { color: 'orange', label: '不支持', explain: '该构建系统当前不在识别范围内。' },
}

function constraintText(requirement: RuntimeRequirement): string {
  return requirement.rawConstraint || '未声明'
}

function evidenceLines(requirement: RuntimeRequirement): string[] {
  const evidence = requirement.evidence ?? []
  if (!evidence.length) return requirement.sourcePath ? [requirement.sourcePath] : []
  return evidence.map((item) => `${item.sourcePath} · ${item.locator} = ${item.rawValue}（${item.parserId}）`)
}

async function pickFolder() {
  picking.value = true
  try {
    const path = await invoke<string>('app.pickFolder', {})
    if (path) rootPath.value = path
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    if (!msg.includes('CANCELLED')) errorText.value = msg
  } finally { picking.value = false }
}

onMounted(async () => {
  await loadProjects()
  // 扫描器当前版本：与快照里落盘的 scanToolVersion 比对，判定旧快照
  // （扫描判定已演进、界面仍是旧结论时给出重扫提示）。
  try {
    currentScanVersion.value = (await invoke<{ version: string }>('project.scanToolVersion')).version
  } catch { currentScanVersion.value = '' }
})

// 快照是否产自旧版扫描器：旧快照（无 scanToolVersion 字段）也视为过期。
const currentScanVersion = ref('')
const scanOutdated = computed(() =>
  !!selected.value?.lastScanAt && !!currentScanVersion.value && selected.value.scanToolVersion !== currentScanVersion.value,
)
</script>

<template>
  <a-config-provider :theme="antTheme" :auto-insert-space-in-button="false">
    <div class="desk">
      <header class="chrome-bar" :class="{ native: isNativeWindow }" @mousedown="dragWindow" @dblclick="toggleWindow">
        <div class="wordmark">
          <BrandMark class="mark" :size="16" />
          <span>DevToolbox</span>
        </div>
        <span class="chrome-rule" aria-hidden="true"></span>
        <span class="chrome-where">{{ placeLabel }}</span>
        <span v-if="view === 'projects' && selected" class="chrome-where path">{{ selected.displayName }}</span>
        <span class="chrome-spacer"></span>
        <WindowControls v-if="isNativeWindow" />
      </header>

      <div class="workbench">
        <nav class="rail" aria-label="功能">
          <button type="button" class="rail-btn" :class="{ on: view === 'projects' }" @click="view = 'projects'">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" d="M3 7.5A1.5 1.5 0 0 1 4.5 6H9l2 2h8.5A1.5 1.5 0 0 1 21 9.5v8A1.5 1.5 0 0 1 19.5 19h-15A1.5 1.5 0 0 1 3 17.5z"/></svg>
            <span>项目</span>
          </button>
          <button type="button" class="rail-btn" :class="{ on: view === 'runtime' }" @click="view = 'runtime'">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" d="M12 3 4 7.5 12 12l8-4.5zM4 12.2 12 16.7l8-4.5M4 16.5 12 21l8-4.5"/></svg>
            <span>运行时</span>
          </button>
          <button type="button" class="rail-btn" :class="{ on: view === 'git' }" @click="view = 'git'">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true"><circle cx="6" cy="6" r="2.1" fill="none" stroke="currentColor" stroke-width="1.7"/><circle cx="6" cy="18" r="2.1" fill="none" stroke="currentColor" stroke-width="1.7"/><circle cx="18" cy="8" r="2.1" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M6 8.2v7.6M18 10.2a6.5 6.5 0 0 1-6.2 6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>
            <span>Git</span>
          </button>
          <span class="rail-spacer"></span>
          <button type="button" class="rail-btn" :class="{ on: view === 'settings' }" @click="view = 'settings'">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.7"/><path fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" d="M12 3.5v2.2M12 18.3V20.5M4.8 6.8l1.6 1.6M17.6 15.6l1.6 1.6M3.5 12h2.2M18.3 12H20.5M4.8 17.2l1.6-1.6M17.6 8.4l1.6-1.6"/></svg>
            <span>设置</span>
          </button>
          <button type="button" class="rail-btn" :class="{ on: view === 'about' }" @click="view = 'about'">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M12 11v5" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/><circle cx="12" cy="8" r="0.9" fill="currentColor"/></svg>
            <span>关于</span>
          </button>
        </nav>

        <aside v-if="view === 'projects'" class="pane" aria-label="已登记项目">
          <div class="pane-head"><span>项目</span></div>
          <div class="pane-scroll">
            <button type="button" class="proj register" :class="{ on: !activeId }" :aria-current="!activeId ? 'page' : undefined" @click="activeId = ''">
              <span class="proj-mark plus" aria-hidden="true">+</span>
              <span class="proj-copy"><span class="proj-name">登记项目</span><span class="proj-meta">扫描一个目录</span></span>
            </button>
            <button v-for="item in projects" :key="item.id" type="button" class="proj" :class="{ on: activeId === item.id }" :aria-current="activeId === item.id ? 'page' : undefined" @click="activeId = item.id; view = 'projects'">
              <span class="proj-mark" aria-hidden="true">{{ projectMark(item.displayName) }}</span>
              <span class="proj-copy">
                <span class="proj-name">{{ item.displayName }}</span>
                <span class="proj-meta">{{ item.modules.length }} 个模块</span>
              </span>
            </button>
            <p v-if="!projects.length && !busy" class="pane-empty">登记后会出现在这里</p>
          </div>
        </aside>

        <main class="stage">
          <div class="commandbar">
            <div class="command-copy">
              <div class="command-title-row">
                <h1>{{ viewMeta.title }}</h1>
                <a-tag v-if="view === 'projects' && selected" :color="selected.scanStatus === 'complete' ? 'green' : 'orange'">{{ selected.scanStatus }}</a-tag>
              </div>
              <p class="command-hint" :class="{ path: viewMeta.path }" :title="viewMeta.hint">{{ viewMeta.hint }}</p>
            </div>
            <div v-if="view === 'projects' && selected" class="command-actions">
              <a-button type="primary" size="small" @click="sessionOpen = true">在此打开终端</a-button>
              <a-button size="small" :loading="busy" @click="rescan">重新扫描</a-button>
              <a-button size="small" danger @click="removeProject">移除登记</a-button>
            </div>
          </div>

          <div class="canvas">
            <a-alert v-if="isDemoClient" type="info" show-icon message="本地演示客户端：使用虚构数据，扫描和登记不会访问磁盘或写入设置。" class="page-alert" />
            <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />

            <RuntimeSettings v-if="view === 'runtime'" />
            <GitIdentity v-else-if="view === 'git'" />
            <SettingsView v-else-if="view === 'settings'" />
            <AboutView v-else-if="view === 'about'" />
            <section v-else-if="!selected" class="register-panel">
              <div class="address-bar">
                <span class="address-k">目录</span>
                <a-input v-model:value="rootPath" placeholder="项目目录的完整路径" @press-enter="scanProject" />
                <a-button :loading="picking" @click="pickFolder">浏览…</a-button>
                <a-button type="primary" :loading="busy" @click="scanProject">扫描预览</a-button>
              </div>
              <p v-if="!candidate" class="address-note">先扫描再登记。扫描只读取构建清单和 Git 状态，不会改动项目文件。</p>
              <template v-if="candidate">
                <div class="preview-sheet">
                  <div class="preview-heading">
                    <div><h3>{{ candidate.displayName }}</h3><span class="muted">{{ candidate.rootPath }}</span></div>
                    <a-tag :color="candidate.status === 'complete' ? 'green' : 'orange'">{{ candidate.status }}</a-tag>
                  </div>
                  <a-input v-model:value="displayName" aria-label="项目名称" placeholder="项目显示名称" class="name-input" />
                  <div class="metric-strip">
                    <div class="metric"><strong>{{ candidate.repositories.length }}</strong><span>Git 工作树</span></div>
                    <div class="metric"><strong>{{ candidate.modules.length }}</strong><span>构建模块</span></div>
                    <div class="metric"><strong>{{ candidate.issues.length }}</strong><span>扫描提示</span></div>
                  </div>
                  <div v-for="module in candidate.modules" :key="module.id" class="module-row">
                    <div class="module-path">{{ module.relativePath }}</div>
                    <a-space wrap><a-tag v-for="build in module.buildSystems" :key="build">{{ build }}</a-tag></a-space>
                    <div class="requirement-list">
                      <span v-for="(requirement, index) in module.requirements" :key="`${requirement.runtime}-${index}`" class="requirement-chip">
                        {{ requirementLabel(requirement) }}<small v-if="requirement.sourcePath"> · {{ requirement.sourcePath }}</small>
                      </span>
                    </div>
                  </div>
                  <a-alert v-for="issue in candidate.issues" :key="`${issue.code}-${issue.relativePath}`" :message="issue.message" :description="issue.relativePath" type="warning" show-icon class="issue-alert" />
                  <div class="form-actions"><a-button @click="candidate = undefined">取消</a-button><a-button type="primary" :loading="busy" @click="registerProject">确认登记</a-button></div>
                </div>
              </template>
            </section>

            <template v-else>
              <a-alert v-if="scanOutdated" type="warning" show-icon class="stale-banner"
                message="扫描器已更新——以下判定产自旧版扫描"
                :description="`当前扫描器 v${currentScanVersion} 改进了判定逻辑（Android compileSdk/minSdk 不再误报冲突、Gradle sourceCompatibility/targetCompatibility 可识别）。点击右上角「重新扫描」刷新结论。`" />
              <a-tabs v-model:activeKey="detailTab" class="detail-tabs">
                <a-tab-pane key="overview" tab="概览">
                  <div class="metric-strip">
                    <div class="metric"><strong>{{ selected.repositories.length }}</strong><span>Git 工作树</span></div>
                    <div class="metric"><strong>{{ selected.modules.length }}</strong><span>构建模块</span></div>
                    <div class="metric"><strong>{{ selected.issues?.length ?? 0 }}</strong><span>扫描提示</span></div>
                  </div>
                  <div class="section">
                    <div class="section-title"><div><h2>运行时版本声明</h2><p>按构建模块汇总，状态含义见各条目。</p></div></div>
                    <div class="requirement-legend">
                      <a-tag color="blue">已声明 {{ requirementSummary.resolved }}</a-tag>
                      <a-tag color="red">冲突 {{ requirementSummary.conflict }}</a-tag>
                      <a-tag>未声明 {{ requirementSummary.unknown }}</a-tag>
                      <a-tag v-if="requirementSummary.unsupported" color="orange">不支持 {{ requirementSummary.unsupported }}</a-tag>
                    </div>
                    <div class="requirement-legend-text">
                      <p><strong>未声明</strong>：构建清单存在（如 pom.xml、build.gradle、package.json），但其中没有写版本号。产品只如实呈现，不会擅自使用 latest。</p>
                      <p><strong>冲突</strong>：同一个运行时出现了多个互斥的版本声明。产品不自动替你选一个；构建模块卡片里列出了每处声明的文件与属性，改哪个由你决定。</p>
                    </div>
                  </div>
                </a-tab-pane>
                <a-tab-pane key="repos" :tab="`Git 仓库 ${selected.repositories.length}`">
                  <div class="toolbar">
                    <a-input v-model:value="repoFilter" class="filter-input" allow-clear placeholder="按路径关键字过滤仓库" />
                    <span class="toolbar-count">显示 {{ filteredRepos.length }} / {{ selected.repositories.length }}</span>
                  </div>
                  <a-empty v-if="filteredRepos.length === 0" :description="selected.repositories.length ? '没有匹配的仓库' : '未发现 Git 工作树'" />
                  <div v-for="repo in filteredRepos" :key="repo.id" class="repo-row repo-card">
                    <div>
                      <strong>{{ repo.path }}</strong>
                      <small>{{ repo.source }}</small>
                      <small v-if="repo.branch">分支 {{ repo.branch }}<template v-if="repo.remoteName">，remote {{ repo.remoteName }}</template><template v-if="repo.userName">，{{ repo.userName }}<template v-if="repo.userEmail"> &lt;{{ repo.userEmail }}&gt;</template></template>（项目级）</small>
                      <small v-else-if="repo.remoteUrl">remote {{ repo.remoteName }}，{{ repo.remoteUrl }}</small>
                    </div>
                    <a-space>
                      <a-button size="small" @click="sessionOpen = true">终端</a-button>
                      <a-button size="small" @click="remoteManager = { open: true, path: repo.path, name: repo.path.split(/[\/]/).filter(Boolean).at(-1) || repo.kind }">管理 Remote</a-button>
                      <a-tag>{{ repo.kind }}</a-tag>
                    </a-space>
                  </div>
                </a-tab-pane>
                <a-tab-pane key="modules" :tab="`构建模块 ${selected.modules.length}`">
                  <div class="toolbar">
                    <a-input v-model:value="moduleFilter" class="filter-input" allow-clear placeholder="按模块路径关键字过滤" />
                    <a-checkbox v-model:checked="onlyPendingModules">仅看待处理（未声明/冲突）</a-checkbox>
                    <span class="toolbar-count">显示 {{ filteredModules.length }} / {{ selected.modules.length }}</span>
                  </div>
                  <a-empty v-if="filteredModules.length === 0" :description="selected.modules.length ? '没有匹配的构建模块' : '未发现支持的构建清单'" />
                  <div v-for="module in filteredModules" :key="module.id" class="module-card">
                    <div class="module-header"><strong>{{ module.relativePath }}</strong><a-space wrap><a-tag v-for="build in module.buildSystems" :key="build">{{ build }}</a-tag></a-space></div>
                    <div v-if="module.requirements.length" class="requirement-list">
                      <div v-for="(requirement, index) in module.requirements" :key="`${requirement.runtime}-${index}`" class="requirement-detail">
                        <div>
                          <strong class="tool-heading"><RuntimeMark :name="requirement.runtime" :size="16" /><span>{{ requirement.runtime }}</span></strong>
                          <a-popover overlay-class-name="requirement-popover">
                            <template #content>
                              <p class="popover-title">{{ requirementStatusMeta[requirement.status].label }}：{{ requirementStatusMeta[requirement.status].explain }}</p>
                              <p v-for="line in evidenceLines(requirement)" :key="line" class="popover-evidence"><code>{{ line }}</code></p>
                              <p class="popover-evidence">置信度 {{ requirement.confidence }}</p>
                            </template>
                            <a-tag :color="requirementStatusMeta[requirement.status].color" class="requirement-status-tag">{{ requirementStatusMeta[requirement.status].label }}</a-tag>
                          </a-popover>
                        </div>
                        <code>{{ constraintText(requirement) }}</code>
                        <small>{{ requirement.sourcePath || '未发现来源' }}</small>
                      </div>
                    </div>
                    <a-empty v-else description="未发现运行时版本声明" :image-style="{ height: '32px' }" />
                  </div>
                </a-tab-pane>
                <a-tab-pane key="issues" :tab="`扫描提示 ${selected.issues?.length ?? 0}`">
                  <a-alert v-if="!selected.issues?.length" type="info" show-icon message="没有扫描提示" description="本次扫描未发现权限受限、深度超限或其他可恢复问题。" />
                  <a-alert v-for="issue in selected.issues" :key="`issue-${issue.code}-${issue.relativePath}`" :message="issue.message" :description="issue.relativePath" type="warning" show-icon class="issue-alert" />
                </a-tab-pane>
              </a-tabs>
            </template>
            <RemoteManager v-model:open="remoteManager.open" :repo-path="remoteManager.path" :repo-name="remoteManager.name" />
            <SessionManager v-model:open="sessionOpen" :repo-paths="sessionRepoPaths" :default-path="sessionDefaultPath" />
          </div>
        </main>
      </div>

      <footer class="statusbar">
        <span class="status-item"><span class="status-dot"></span>{{ isDemoClient ? '演示数据' : '本机模式' }}</span>
        <span v-if="busy" class="status-item">正在处理</span>
        <span v-else-if="view === 'projects' && selected" class="status-item">{{ selected.modules.length }} 个模块 · {{ selected.repositories.length }} 个仓库</span>
        <span class="status-spacer"></span>
        <span v-if="currentScanVersion" class="status-item">扫描器 {{ currentScanVersion }}</span>
      </footer>
    </div>
  </a-config-provider>
</template>
