<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { message, Modal, theme } from 'ant-design-vue'
import { invoke, isDemoClient } from './bridge'
import type { Candidate, Project, RuntimeRequirement } from './types'
import RuntimeSettings from './RuntimeSettings.vue'
import GitIdentity from './GitIdentity.vue'

const prefersDark = window.matchMedia('(prefers-color-scheme: dark)')
const isDark = ref(prefersDark.matches)
const onSchemeChange = (event: MediaQueryListEvent) => { isDark.value = event.matches }
onMounted(() => prefersDark.addEventListener('change', onSchemeChange))
onBeforeUnmount(() => prefersDark.removeEventListener('change', onSchemeChange))
const antTheme = computed(() => ({
  algorithm: isDark.value ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: { colorPrimary: isDark.value ? '#58b8a4' : '#3d8f7c', colorInfo: isDark.value ? '#58b8a4' : '#3d8f7c', borderRadius: 6 },
}))

const projects = ref<Project[]>([])
const activeId = ref('')
const candidate = ref<Candidate>()
const rootPath = ref('')
const displayName = ref('')
const busy = ref(false)
const errorText = ref('')
const selected = computed(() => projects.value.find((item) => item.id === activeId.value))
const view = ref<'projects' | 'runtime' | 'git'>('projects')
const viewMeta = computed(() => {
  switch (view.value) {
    case 'runtime':
      return { title: '用户级运行时设置' }
    case 'git':
      return { title: 'Git 身份' }
    default:
      return { title: selected.value?.displayName ?? '登记项目' }
  }
})

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

onMounted(loadProjects)
</script>

<template>
  <a-config-provider :theme="antTheme">
  <a-layout class="shell">
    <a-layout-sider class="sidebar" width="264" breakpoint="lg" collapsed-width="0">
      <div class="brand"><span class="brand-mark"><svg width="22" height="22" viewBox="0 0 512 512" aria-hidden="true"><g stroke="#80e0c0" stroke-width="28.8" stroke-linecap="round" fill="none"><line x1="168" y1="140" x2="168" y2="297"/><path d="M340 168 v24 a129.6 129.6 0 0 1 -129.6 129.6"/></g><circle cx="168" cy="140" r="52" fill="#80e0c0"/><circle cx="340" cy="168" r="43.2" fill="#80e0c0"/><circle cx="168" cy="340" r="43.2" fill="#80e0c0"/></svg></span><div><strong>DevToolbox</strong><small>开发环境与 Git 身份</small></div></div>
      <div class="side-heading">全局能力</div>
      <a-menu :selected-keys="view === 'runtime' ? ['runtime-settings'] : view === 'git' ? ['git-identity'] : ['projects-home']" mode="inline" class="project-menu">
        <a-menu-item key="projects-home" @click="activeId = ''; view = 'projects'">
          <span class="menu-icon"><svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true"><g fill="none" stroke="#80e0c0" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></g><circle cx="12" cy="13.5" r="2.4" fill="#80e0c0"/></svg></span>
          <span class="menu-text"><span class="menu-project">项目上下文</span>
          <span class="menu-meta">登记并扫描开发项目</span></span>
        </a-menu-item>
        <a-menu-item key="runtime-settings" @click="view = 'runtime'">
          <span class="menu-icon"><svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true"><g fill="none" stroke="#80e0c0" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 3 3 8 12 13 21 8"/><polyline points="3 12.5 12 17.5 21 12.5"/><polyline points="3 17 12 22 21 17"/></g></svg></span>
          <span class="menu-text"><span class="menu-project">用户级运行时设置</span>
          <span class="menu-meta">获取运行时、全局默认与导入</span></span>
        </a-menu-item>
        <a-menu-item key="git-identity" @click="view = 'git'">
          <span class="menu-icon"><svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true"><g fill="none" stroke="#80e0c0" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="6" r="2.6"/><circle cx="6" cy="18" r="2.6"/><circle cx="18" cy="8" r="2.6"/><path d="M6 8.6v6.8"/><path d="M18 10.6a8 8 0 0 1-8 7.4"/></g></svg></span>
          <span class="menu-text"><span class="menu-project">Git 身份</span>
          <span class="menu-meta">档案、目录绑定与仓库审计</span></span>
        </a-menu-item>
      </a-menu>
      <template v-if="projects.length">
        <div class="side-heading">已登记项目</div>
        <a-menu :selected-keys="view === 'projects' && activeId ? [activeId] : []" mode="inline" class="project-menu project-menu-list">
          <a-menu-item v-for="item in projects" :key="item.id" @click="activeId = item.id; view = 'projects'">
            <span class="menu-project">{{ item.displayName }}</span>
            <span class="menu-meta">{{ item.modules.length }} 个模块</span>
          </a-menu-item>
        </a-menu>
      </template>
      <div v-else-if="!busy" class="side-empty">登记项目后会显示在这里</div>
      <div class="side-bottom"><span class="status-dot"></span>本机模式</div>
    </a-layout-sider>

    <a-layout class="content-layout">
      <a-layout-content class="page-content">
        <div class="page-head">
          <div><h1>{{ viewMeta.title }}</h1><div v-if="view === 'projects' && selected" class="page-sub">{{ selected.rootPath }}</div><div v-else-if="view === 'projects'" class="page-sub">扫描只读取项目清单，不运行脚本或 Wrapper。</div></div>
          <a-space v-if="view === 'projects' && selected">
            <a-button @click="rescan" :loading="busy">重新扫描</a-button>
            <a-button danger @click="removeProject">移除登记</a-button>
          </a-space>
        </div>
        <a-alert v-if="isDemoClient" type="info" show-icon message="本地演示客户端：使用虚构数据，扫描和登记不会访问磁盘或写入设置。" class="page-alert" />
        <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />

        <RuntimeSettings v-if="view === 'runtime'" />
        <GitIdentity v-else-if="view === 'git'" />
        <section v-else-if="!selected || candidate" class="section add-panel">
          <div class="section-title"><div><h2>登记项目</h2><p>扫描只读取支持的项目清单，不运行脚本或 Wrapper。</p></div></div>
          <div class="path-entry">
            <a-input v-model:value="rootPath" placeholder="输入项目目录的完整路径" @press-enter="scanProject" />
            <a-button type="primary" :loading="busy" @click="scanProject">扫描预览</a-button>
          </div>
          <template v-if="candidate">
            <div class="preview-heading">
              <div><h3>{{ candidate.displayName }}</h3><span class="muted">{{ candidate.rootPath }}</span></div>
              <a-tag :color="candidate.status === 'complete' ? 'green' : 'orange'">{{ candidate.status }}</a-tag>
            </div>
            <a-input v-model:value="displayName" aria-label="项目名称" placeholder="项目显示名称" class="name-input" />
            <div class="preview-grid">
              <div class="preview-stat"><strong>{{ candidate.repositories.length }}</strong><span>Git 工作树</span></div>
              <div class="preview-stat"><strong>{{ candidate.modules.length }}</strong><span>构建模块</span></div>
              <div class="preview-stat"><strong>{{ candidate.issues.length }}</strong><span>扫描提示</span></div>
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
          </template>
        </section>

        <template v-else-if="selected">
          <section class="summary-grid">
            <div class="section summary-card"><span class="summary-label">项目目录</span><strong class="path-value">{{ selected.rootPath }}</strong></div>
            <div class="section summary-card"><span class="summary-label">最近扫描</span><strong>{{ new Date(selected.lastScanAt).toLocaleString() }}</strong></div>
            <div class="section summary-card"><span class="summary-label">扫描状态</span><a-tag :color="selected.scanStatus === 'complete' ? 'green' : 'orange'">{{ selected.scanStatus }}</a-tag></div>
          </section>
          <section class="section">
            <div class="section-title"><div><h2>仓库</h2><p>依据 Git 元数据识别，不读取身份配置。</p></div></div>
            <a-empty v-if="selected.repositories.length === 0" description="未发现 Git 工作树" />
            <div v-for="repo in selected.repositories" :key="repo.id" class="repo-row"><div><strong>{{ repo.path }}</strong><small>{{ repo.source }}</small></div><a-tag>{{ repo.kind }}</a-tag></div>
          </section>
          <section class="section">
            <div class="section-title"><div><h2>构建模块</h2><p>版本要求保留原始约束与来源；未声明时显示 unknown。</p></div></div>
            <a-empty v-if="selected.modules.length === 0" description="未发现支持的构建清单" />
            <div v-for="module in selected.modules" :key="module.id" class="module-card">
              <div class="module-header"><strong>{{ module.relativePath }}</strong><a-space wrap><a-tag v-for="build in module.buildSystems" :key="build">{{ build }}</a-tag></a-space></div>
              <div v-if="module.requirements.length" class="requirement-list">
                <div v-for="(requirement, index) in module.requirements" :key="`${requirement.runtime}-${index}`" class="requirement-detail">
                  <div><strong>{{ requirement.runtime }}</strong><a-tag :color="requirement.status === 'conflict' ? 'red' : requirement.status === 'unknown' ? 'default' : 'blue'">{{ requirement.status }}</a-tag></div>
                  <code>{{ requirement.rawConstraint || '未声明' }}</code>
                  <small>{{ requirement.sourcePath || '未发现来源' }} · 置信度 {{ requirement.confidence }}</small>
                </div>
              </div>
              <a-empty v-else description="未发现运行时版本声明" :image-style="{ height: '32px' }" />
            </div>
          </section>
          <section v-if="selected.issues?.length" class="section">
            <div class="section-title"><div><h2>扫描提示</h2><p>部分路径可能未能读取。</p></div></div>
            <a-alert v-for="issue in selected.issues" :key="`${issue.code}-${issue.relativePath}`" :message="issue.message" :description="issue.relativePath" type="warning" show-icon class="issue-alert" />
          </section>
        </template>
      </a-layout-content>
    </a-layout>
  </a-layout>
  </a-config-provider>
</template>
