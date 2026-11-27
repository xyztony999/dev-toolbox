import type { Candidate, Project, RuntimeRequirement } from './types'

const now = () => new Date().toISOString()
const projects: Project[] = [
  {
    id: 'demo-multistack-project',
    displayName: '多技术栈示例工程',
    rootPath: 'D:/Demo/ToolboxFixture',
    repositories: [
      { id: 'demo-repo-main', path: 'D:/Demo/ToolboxFixture', kind: 'normal', source: '.git metadata' },
      { id: 'demo-repo-docs', path: 'D:/Demo/ToolboxFixture/docs-site', kind: 'nested', source: '.git metadata' },
    ],
    modules: [
      {
        id: 'demo-module-web', relativePath: 'web', buildSystems: ['node'],
        requirements: [{ runtime: 'node', rawConstraint: '>=22.12.0', sourcePath: 'web/package.json', confidence: 'high', status: 'resolved' }],
      },
      {
        id: 'demo-module-crawler', relativePath: 'services/crawler', buildSystems: ['python'],
        requirements: [{ runtime: 'python', rawConstraint: '>=3.11,<3.12', sourcePath: 'services/crawler/pyproject.toml', confidence: 'high', status: 'resolved' }],
      },
      {
        id: 'demo-module-api', relativePath: 'services/api', buildSystems: ['maven', 'maven-wrapper'],
        requirements: [{ runtime: 'java', rawConstraint: '21', sourcePath: 'services/api/pom.xml', confidence: 'high', status: 'resolved' }],
      },
      {
        id: 'demo-module-mobile', relativePath: 'mobile', buildSystems: ['android-gradle', 'gradle-wrapper'],
        requirements: [
          { runtime: 'java', rawConstraint: '19', sourcePath: 'mobile/build.gradle.kts', confidence: 'medium', status: 'resolved' },
          { runtime: 'androidsdk', rawConstraint: '34', sourcePath: 'mobile/build.gradle.kts', confidence: 'medium', status: 'resolved' },
        ],
      },
    ],
    issues: [{ code: 'PROJECT_SCAN_DEPTH_LIMIT', severity: 'info', relativePath: 'web/.cache', message: '依赖与缓存目录已跳过', recoverable: true }],
    scanStatus: 'complete', createdAt: now(), lastScanAt: now(), scanToolVersion: '2', revision: 1,
  },
]

const userRuntimeState = {
  schema: 1,
  revision: 1,
  globalSelections: {
    node: { installationId: 'demo-node-22', status: 'active', source: 'legacy' },
    java: { installationId: 'demo-java-21', status: 'active', source: 'legacy' },
    maven: { status: 'empty', source: '' },
    gradle: { status: 'empty', source: '' },
    androidsdk: { status: 'empty', source: '' },
    golang: { installationId: 'demo-go-1.27', status: 'active', source: 'user' },
    python: { status: 'empty', source: '' },
    rust: { status: 'empty', source: '' },
  } as Record<string, { installationId?: string; unresolvedHome?: string; status: string; source: string }>,
  installations: [
    { id: 'demo-node-22', runtime: 'node', version: '22.14.0', major: '22', localHome: 'D:/Tools/node-v22.14.0', source: 'downloaded', label: 'Node LTS', vendor: '', available: true },
    { id: 'demo-java-21', runtime: 'java', version: '21.0.6', major: '21', localHome: 'D:/Tools/jdk-21', source: 'imported', label: '', vendor: 'Temurin', available: true },
    { id: 'demo-go-1.27', runtime: 'golang', version: '1.27.3', major: '1.27', localHome: 'C:/Users/demo/go-1.27.3', source: 'downloaded', label: 'Go 1.27.3', vendor: 'Google Go', available: true },
    { id: 'demo-go-1.24', runtime: 'golang', version: '1.24.0', major: '1.24', localHome: 'D:/Tools/go-1.24', source: 'user', label: '', vendor: '', available: false },
  ],
  legacyImports: [] as Array<{ sourceDigest: string; importedAt: string; backupPath: string; created: number; linked: number; unresolved: number; conflicts: number }>,
}

const legacyPreview = {
  sourceId: 'demo-devswitch-state',
  sourcePath: 'C:/Users/demo/AppData/Roaming/devswitch/state.json',
  sourceSchema: 1,
  sourceDigest: 'demo-state-digest-0001',
  selections: userRuntimeState.globalSelections,
  installations: userRuntimeState.installations,
  unresolved: ['golang'],
  unsupported: [],
  duplicateCount: 0,
  backupDirectory: 'C:/Users/demo/AppData/Roaming/DevToolbox/backups/devswitch',
}

// digestString 复用字符串摘要，演示版只需稳定即可。
function digestString(value: string): string {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) | 0
  }
  return Math.abs(hash).toString(16).padStart(8, "0") + value.length.toString(16)
}
function copy<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function nameFromPath(path: string): string {
  return path.replace(/[\\/]+$/, '').split(/[\\/]/).filter(Boolean).at(-1) || '新项目'
}

function createCandidate(rootPath: string): Candidate {
  const name = nameFromPath(rootPath)
  const normalized = rootPath.replace(/\\/g, '/')
  const requirements: RuntimeRequirement[] = [
    { runtime: 'node', rawConstraint: '>=22.12.0', sourcePath: 'frontend/package.json', confidence: 'high', status: 'resolved' },
    { runtime: 'python', rawConstraint: '>=3.11,<3.12', sourcePath: 'services/crawler/pyproject.toml', confidence: 'high', status: 'resolved' },
  ]
  return {
    id: `candidate-${encodeURIComponent(normalized.toLowerCase())}`,
    displayName: name,
    rootPath: normalized,
    evidenceDigest: `demo-digest-${encodeURIComponent(normalized.toLowerCase())}`,
    repositories: [{ id: `repo-${name}`, path: normalized, kind: 'normal', source: '.git metadata' }],
    modules: [
      { id: `module-${name}-web`, relativePath: 'frontend', buildSystems: ['node'], requirements: [requirements[0]] },
      { id: `module-${name}-crawler`, relativePath: 'services/crawler', buildSystems: ['python'], requirements: [requirements[1]] },
      { id: `module-${name}-service`, relativePath: 'services/api', buildSystems: ['maven', 'maven-wrapper'], requirements: [{ runtime: 'java', rawConstraint: '21', sourcePath: 'services/api/pom.xml', confidence: 'high', status: 'resolved' }] },
    ],
    issues: [],
    status: 'complete',
    scannedAt: now(),
  }
}

const installMirrors = {
  node: [{ id: 'official', label: 'Node.js 官方源' }, { id: 'npmmirror', label: 'npmmirror 镜像' }, { id: 'tuna', label: '清华 TUNA 镜像' }],
  java: [{ id: 'official', label: 'Adoptium 官方源' }, { id: 'tuna', label: '清华 TUNA 镜像' }],
  maven: [{ id: 'official', label: 'Apache 官方源' }, { id: 'tuna', label: '清华 TUNA 镜像' }],
  gradle: [{ id: 'official', label: 'Gradle 官方源' }, { id: 'tencent', label: '腾讯云镜像' }],
  androidsdk: [{ id: 'official', label: 'Google 官方源' }, { id: 'tencent', label: '腾讯云镜像' }],
  golang: [{ id: 'official', label: 'Go 官方中国站' }],
  python: [{ id: 'official', label: 'npmmirror python-build-standalone 镜像' }],
} as Record<string, Array<{ id: string; label: string }>>

interface DemoInstallTask {
  taskId: string
  planId: string
  planDigest: string
  state: string
  progress: { phase: string; package?: string; receivedBytes?: number; totalBytes?: number }
  packages: Array<{ name: string; fileName: string; state: string; error?: { code: string; message: string } }>
  installationId?: string
  error?: { code: string; message: string }
}

const installOptions: Array<{ tool: string; providerId: string; mirrors: Array<{ id: string; label: string }>; versionSelectors: string[]; supportedPlatforms: string[]; supportedArchs: string[] }> =
  Object.entries(installMirrors).map(([tool, mirrors]) => ({
    tool,
    providerId: `demo-${tool}`,
    mirrors,
    versionSelectors: tool === 'androidsdk' ? ['latest'] : ['latest', 'major'],
    supportedPlatforms: ['windows', 'linux'],
    supportedArchs: ['amd64', 'arm64'],
  }))

const installTasks = new Map<string, DemoInstallTask>()
let installTaskSeq = 0

function demoPlan(tool: string, selector: string, mirror: string) {
  return {
    planId: `demo-plan-${tool}-${selector}-${mirror}`,
    planDigest: `demo-digest-${tool}-${selector}-${mirror}`,
    tool,
    providerId: `demo-${tool}`,
    mirror: mirror || 'official',
    requestedSelector: selector,
    resolvedVersion: tool === 'androidsdk' ? '13114758' : `1.0.0-demo-${selector}`,
    os: 'windows',
    arch: 'amd64',
    packages: [{ name: tool, fileName: `${tool}-demo.zip`, checksumAlgorithm: 'sha256', checksum: 'demo-checksum', estimatedBytes: 32 * 1024 * 1024 }],
    targetHome: 'C:/Users/demo/AppData/Local/DevToolbox/runtimes/demo',
    estimatedBytes: 32 * 1024 * 1024,
    targetExists: false,
    expiresAt: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
  }
}

function advanceDemoTask(task: DemoInstallTask) {
  const phases = ['downloading', 'verifying', 'extracting', 'registering', 'complete']
  const index = phases.indexOf(task.progress.phase)
  const next = phases[Math.min(index + 1, phases.length - 1)]
  task.progress = { phase: next, package: task.packages[0]?.name }
  if (next === 'complete') {
    task.state = 'complete'
    task.installationId = 'demo-installed-runtime'
    const state = userRuntimeState
    if (!state.installations.some((item) => item.id === task.installationId)) {
      state.installations.push({ id: task.installationId!, runtime: demoPlanTool(task.planId), version: '1.0.0-demo', major: '1', localHome: 'C:/Users/demo/AppData/Local/DevToolbox/runtimes/demo', source: 'downloaded', label: '', vendor: '', available: true })
      state.revision += 1
    }
  } else {
    task.state = next as DemoInstallTask['state']
  }
}

function demoPlanTool(planId: string): string {
  return planId.split('-')[2] ?? 'node'
}

const gitProfiles: Array<{ id: string; name: string; email: string; signingKey?: string; gpgSign?: boolean; gpgFormat?: string; insteadOf?: Array<{ original: string; url: string }>; accounts?: Array<{ host: string; username: string }>; gitdirs?: string[] }> = [
  {
    id: 'work', name: '张三', email: 'z@corp.com', signingKey: 'KEY123', gpgSign: true, gpgFormat: 'ssh',
    insteadOf: [{ original: 'git@github.com:', url: 'git@github.com-work:' }],
    accounts: [{ host: 'github.com', username: 'corp-zhang' }], gitdirs: ['D:/work/'],
  },
  { id: 'personal', name: 'tony', email: 'tony@me.dev' },
]

const SCAN_TOOL_VERSION = '3'

const demoGoValues: Record<string, string> = {
  GOPROXY: 'https://proxy.golang.org,direct', GOPATH: 'C:/Users/demo/go', GOBIN: '',
  GOROOT: 'C:/Users/demo/go-1.27.3', GOMODCACHE: 'C:/Users/demo/go/pkg/mod',
  GOOS: 'windows', GOARCH: 'amd64', GOVERSION: 'go1.27.3',
}
function demoGoReport() {
  return {
    goBinary: 'C:/Users/demo/go-1.27.3/bin/go.exe', goVersion: demoGoValues.GOVERSION,
    envFile: 'C:/Users/demo/AppData/Roaming/go/env', values: { ...demoGoValues },
    editable: ['GOPROXY', 'GOPATH', 'GOBIN'],
    presets: [
      { id: 'default', label: '官方 proxy.golang.org', value: 'https://proxy.golang.org,direct' },
      { id: 'cn', label: 'goproxy.cn（国内）', value: 'https://goproxy.cn,direct' },
      { id: 'aliyun', label: '阿里云镜像', value: 'https://mirrors.aliyun.com/goproxy/,direct' },
      { id: 'direct', label: 'direct（离线/直连）', value: 'direct' },
    ],
  }
}

type DemoConfigItem = { key: string; label: string; value: string; editable: boolean; source: string; presets?: Array<{ id: string; label: string; value: string }>; note?: string }
const demoToolConfigs: Record<string, { binary?: string; note?: string; items: DemoConfigItem[] }> = {
  golang: {
    binary: 'C:/Users/demo/go-1.27.3/bin/go.exe',
    items: [
      { key: 'GOPROXY', label: 'GOPROXY', value: demoGoValues.GOPROXY, editable: true, source: 'go env', presets: demoGoReport().presets },
      { key: 'GOPATH', label: 'GOPATH', value: demoGoValues.GOPATH, editable: true, source: 'go env' },
      { key: 'GOBIN', label: 'GOBIN', value: '', editable: true, source: 'go env' },
      { key: 'GOROOT', label: 'GOROOT', value: demoGoValues.GOROOT, editable: false, source: 'go env（只读）' },
    ],
  },
  node: {
    binary: 'C:/demo/node/npm.cmd',
    items: [{ key: 'registry', label: 'npm registry', value: 'https://registry.npmjs.org/', editable: true, source: 'npm config（~/.npmrc）', presets: [{ id: 'npmjs', label: '官方 registry.npmjs.org', value: 'https://registry.npmjs.org/' }, { id: 'npmmirror', label: 'npmmirror（国内）', value: 'https://registry.npmmirror.com/' }] }],
  },
  python: {
    binary: 'C:/Users/demo/python/python.exe',
    items: [{ key: 'global.index-url', label: 'pip index-url', value: '', editable: true, source: 'pip config（用户级）', presets: [{ id: 'pypi', label: '官方 pypi.org', value: 'https://pypi.org/simple' }, { id: 'tsinghua', label: '清华 TUNA', value: 'https://pypi.tuna.tsinghua.edu.cn/simple' }, { id: 'aliyun', label: '阿里云镜像', value: 'https://mirrors.aliyun.com/pypi/simple/' }] }],
  },
  maven: {
    items: [{ key: 'mirror.url', label: '中央仓库镜像', value: '', editable: true, source: '~/.m2/settings.xml（托管 id=dev-toolbox）', presets: [{ id: 'aliyun', label: '阿里云镜像', value: 'https://maven.aliyun.com/repository/public' }, { id: 'tencent', label: '腾讯云镜像', value: 'https://mirrors.cloud.tencent.com/nexus/repository/maven-public/' }], note: '仅托管 id=dev-toolbox 的 mirror；写入前备份 settings.xml。' }],
  },
  java: { items: [{ key: 'JAVA_HOME', label: 'JAVA_HOME', value: 'D:/Tools/jdk-21', editable: false, source: '只读（默认安装推导）', note: '切换默认版本即改变；会话注入以此为准。' }] },
  androidsdk: { items: [{ key: 'ANDROID_HOME', label: 'ANDROID_HOME', value: 'E:/Android/Sdk', editable: false, source: '进程环境 ANDROID_HOME', note: '产品不写系统环境变量。' }] },
  gradle: { items: [{ key: 'GRADLE_USER_HOME', label: 'GRADLE_USER_HOME', value: 'C:/Users/demo/.gradle', editable: false, source: '只读（默认 ~/.gradle）', note: 'Gradle 无安全的全局镜像开关。' }] },
  rust: { items: [{ key: 'cargo.mirror', label: 'cargo 镜像（replace-with）', value: '', editable: false, source: '~/.cargo/config.toml（只读）', note: 'cargo 镜像需手工编辑 config.toml 的 [source] 段。' }] },
}

const clientSettings = {
  schema: 1,
  revision: 1,
  theme: 'system',
  customAccent: '',
  sourceProjects: [
    { path: 'D:/Demo/ToolboxFixture', label: '演示源项目', binding: 'local', note: '' },
  ] as Array<{ path: string; label?: string; binding: string; note?: string }>,
  snapshots: [] as Array<{ sourceDigest: string; label?: string; backupPath?: string; importedAt: string; applied: number; skipped: number }>,
}


export async function mockInvoke(operation: string, payload: Record<string, unknown>): Promise<unknown> {
  switch (operation) {
    case 'project.list':
      return copy(projects)
    case 'project.scanToolVersion':
      return { version: SCAN_TOOL_VERSION }
    case 'app.info':
      return {
        version: '0.2.0',
        scanToolVersion: SCAN_TOOL_VERSION,
        configDir: 'C:/Users/demo/AppData/Roaming/DevToolbox',
        projectStore: 'C:/Users/demo/AppData/Roaming/DevToolbox/projects.json',
        runtimeState: 'C:/Users/demo/AppData/Roaming/DevToolbox/user-runtime-state.json',
        gitIdentity: 'C:/Users/demo/AppData/Roaming/DevToolbox/git-identity.json',
        os: 'windows',
        arch: 'amd64',
      }
    case 'app.open':
      return {}
    case 'project.scanPreview': {
      const rootPath = String(payload.rootPath ?? '').trim()
      if (!rootPath) throw new Error('请输入项目目录。')
      return createCandidate(rootPath)
    }
    case 'project.register': {
      const rootPath = String(payload.rootPath ?? '')
      const existing = projects.find((item) => item.rootPath.toLowerCase() === rootPath.toLowerCase())
      if (existing) return copy(existing)
      const candidate = createCandidate(rootPath)
      if (candidate.id !== payload.candidateId || candidate.evidenceDigest !== payload.expectedEvidenceDigest) {
        throw new Error('PROJECT_SCAN_STALE: 预览已过期，请重新扫描。')
      }
      const created: Project = {
        id: `project-${Date.now()}`,
        displayName: String(payload.displayName || candidate.displayName),
        rootPath: candidate.rootPath,
        repositories: candidate.repositories,
        modules: candidate.modules,
        issues: candidate.issues,
        scanStatus: candidate.status,
        createdAt: now(),
        lastScanAt: candidate.scannedAt,
        scanToolVersion: SCAN_TOOL_VERSION,
        revision: 1,
      }
      projects.unshift(created)
      return copy(created)
    }
    case 'project.get': {
      const found = projects.find((item) => item.id === payload.projectId)
      if (!found) throw new Error('PROJECT_NOT_FOUND')
      return copy(found)
    }
    case 'project.rescan': {
      const found = projects.find((item) => item.id === payload.projectId)
      if (!found) throw new Error('PROJECT_NOT_FOUND')
      if (found.revision !== payload.expectedRevision) throw new Error('PROJECT_REVISION_CONFLICT')
      found.lastScanAt = now()
      found.scanToolVersion = SCAN_TOOL_VERSION
      found.revision += 1
      found.scanStatus = 'complete'
      return copy(found)
    }
    case 'project.unregister': {
      const index = projects.findIndex((item) => item.id === payload.projectId)
      if (index < 0) return { removed: false }
      if (projects[index].revision !== payload.expectedRevision) throw new Error('PROJECT_REVISION_CONFLICT')
      projects.splice(index, 1)
      return { removed: true }
    }
    case 'runtime.getUserState':
      return copy(userRuntimeState)
    case 'runtime.discover': {
      const root = String((payload.roots as string[] | undefined)?.[0] ?? '').trim()
      const candidates = root ? [] : [{ candidateId: 'demo-rust-stable', digest: 'demo-rust-stable', tool: 'rust', version: 'unknown', major: '', home: 'C:/Users/demo/.rustup/toolchains/stable-x86_64-pc-windows-msvc', binary: 'C:/Users/demo/.rustup/toolchains/stable-x86_64-pc-windows-msvc/bin/rustc.exe', source: 'rustup', evidence: ['rustup channel:stable', 'rustup default'], availability: 'available', confidence: 'unknown' }, { candidateId: 'demo-node-runtime', digest: 'demo-node-runtime', tool: 'node', version: '22.12.0', major: '22', home: 'C:/demo/node', binary: 'C:/demo/node/node.exe', source: 'path', evidence: ['directory-name'], availability: 'available', confidence: 'exact' }]
      return { scanId: 'demo-scan', status: 'complete', candidates, issues: [] }
    }
    case 'runtime.inspectPath': {
      const home = String(payload.home ?? 'C:/demo/node')
      const tool = String(payload.tool ?? 'node')
      return { candidateId: `demo-${tool}-${home}`, digest: `demo-${tool}-${home}`, tool, version: 'unknown', major: '', home, binary: `${home}/node.exe`, source: 'user', evidence: ['demo fixture'], availability: 'available', confidence: 'unknown' }
    }
    case 'runtime.register': {
      if (Number(payload.expectedRevision) !== userRuntimeState.revision) throw new Error('USER_RUNTIME_REVISION_CONFLICT')
      const selected = payload.candidates as Array<{candidateId: string; digest: string; tool: string; home: string; source: string}>
      let created = 0
      for (const candidate of selected ?? []) {
        if (userRuntimeState.installations.some((entry) => entry.runtime === candidate.tool && entry.localHome === candidate.home)) continue
        userRuntimeState.installations.push({ id: candidate.candidateId, runtime: candidate.tool, version: '', major: '', localHome: candidate.home, source: candidate.source, label: '', vendor: '', available: true })
        created += 1
      }
      if (created) userRuntimeState.revision += 1
      return { state: copy(userRuntimeState), counts: { created, linked: (selected?.length ?? 0) - created, skipped: 0 } }
    }
    case 'legacy.inspectDevSwitch':
      return { found: true, sourcePath: legacyPreview.sourcePath, readable: true, valid: true, schema: 1, sourceDigest: legacyPreview.sourceDigest }
    case 'legacy.previewDevSwitchImport':
      return copy(legacyPreview)
    case 'legacy.importDevSwitch': {
      if (payload.expectedSourceDigest !== legacyPreview.sourceDigest) throw new Error('LEGACY_SOURCE_CHANGED')
      if (!payload.confirmBackup || !payload.confirmImport) throw new Error('LEGACY_CONFIRMATION_REQUIRED')
      if (!userRuntimeState.legacyImports.some((item) => item.sourceDigest === legacyPreview.sourceDigest)) {
        userRuntimeState.legacyImports.push({ sourceDigest: legacyPreview.sourceDigest, importedAt: now(), backupPath: `${legacyPreview.backupDirectory}/state-demo-state-digest.json`, created: 2, linked: 0, unresolved: 1, conflicts: 0 })
      }
      return copy(userRuntimeState.legacyImports.at(-1))
    }
    case 'runtime.setGlobalDefault': {
      if (Number(payload.expectedRevision) !== userRuntimeState.revision) throw new Error('USER_RUNTIME_REVISION_CONFLICT')
      const item = userRuntimeState.installations.find((entry) => entry.id === payload.installationId && entry.runtime === payload.runtime)
      if (!item) throw new Error('RUNTIME_INSTALLATION_NOT_FOUND')
      if (!item.available) throw new Error('RUNTIME_INSTALLATION_UNAVAILABLE')
      userRuntimeState.globalSelections[item.runtime] = { installationId: item.id, status: 'active', source: 'user' }
      userRuntimeState.revision += 1
      return copy(userRuntimeState)
    }
    case 'runtime.toolConfigRead': {
      const tool = String(payload.tool ?? '')
      const cfg = demoToolConfigs[tool]
      if (!cfg) return { tool, items: [], note: '该工具暂无可管理的环境配置' }
      return { tool, binary: cfg.binary, items: copy(cfg.items), note: cfg.note }
    }
    case 'runtime.toolConfigWrite': {
      const tool = String(payload.tool ?? '')
      const key = String(payload.key ?? '')
      const item = demoToolConfigs[tool]?.items.find((entry) => entry.key === key)
      if (!item?.editable) throw new Error('TOOL_CONFIG_KEY_NOT_EDITABLE')
      if (!payload.dryRun && !payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      const from = item.value
      const to = payload.unset ? '' : String(payload.value ?? '')
      if (!payload.dryRun) item.value = to
      return {
        change: { key, from, to, action: payload.unset ? 'unset' : 'set' },
        report: { tool, binary: demoToolConfigs[tool]?.binary, items: copy(demoToolConfigs[tool]?.items ?? []) },
      }
    }
    case 'runtime.removeInstallation': {
      if (Number(payload.expectedRevision) !== userRuntimeState.revision) throw new Error('USER_RUNTIME_REVISION_CONFLICT')
      const idx = userRuntimeState.installations.findIndex((entry) => entry.id === payload.id)
      if (idx < 0) throw new Error('RUNTIME_INSTALL_NOT_FOUND')
      const inst = userRuntimeState.installations[idx]
      const owned = inst.source === 'downloaded'
      if (payload.deleteFiles && !owned) throw new Error('RUNTIME_REMOVE_NOT_OWNED')
      const preview = {
        id: inst.id, runtime: inst.runtime, version: inst.version, source: inst.source, localHome: inst.localHome,
        ownedByProduct: owned, willDeleteDir: owned && !!payload.deleteFiles,
        isDefault: userRuntimeState.globalSelections[inst.runtime]?.installationId === inst.id,
      }
      if (payload.dryRun) return { preview }
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      userRuntimeState.installations.splice(idx, 1)
      if (userRuntimeState.globalSelections[inst.runtime]?.installationId === inst.id) delete userRuntimeState.globalSelections[inst.runtime]
      userRuntimeState.revision += 1
      return { preview, state: copy(userRuntimeState), removed: true, dirDeleted: preview.willDeleteDir, wasDefault: preview.isDefault }
    }
    case 'runtime.install.options': {
      const tool = String(payload.tool ?? '')
      return tool ? copy(installOptions.filter((item) => item.tool === tool)) : copy(installOptions)
    }
    case 'runtime.install.preview': {
      const tool = String(payload.tool ?? 'node')
      const selector = String(payload.selector ?? 'latest')
      const mirror = String(payload.mirror ?? 'official')
      if (!installMirrors[tool]) throw new Error('RUNTIME_PROVIDER_UNSUPPORTED')
      await new Promise((resolve) => setTimeout(resolve, 300))
      return demoPlan(tool, selector, mirror)
    }
    case 'runtime.install.start': {
      if (!payload.confirm) throw new Error('RUNTIME_CONFIRMATION_REQUIRED')
      const planId = String(payload.planId ?? '')
      const task: DemoInstallTask = {
        taskId: `demo-install-task-${++installTaskSeq}`,
        planId,
        planDigest: String(payload.planDigest ?? ''),
        state: 'downloading',
        progress: { phase: 'downloading', package: demoPlanTool(planId) },
        packages: [{ name: demoPlanTool(planId), fileName: 'demo-archive.zip', state: 'pending' }],
      }
      installTasks.set(task.taskId, task)
      return copy(task)
    }
    case 'runtime.install.status': {
      const task = installTasks.get(String(payload.taskId ?? ''))
      if (!task) throw new Error('RUNTIME_TASK_NOT_FOUND')
      advanceDemoTask(task)
      return copy(task)
    }
    case 'runtime.install.cancel': {
      const task = installTasks.get(String(payload.taskId ?? ''))
      if (!task) throw new Error('RUNTIME_TASK_NOT_FOUND')
      task.state = 'cancelled'
      task.progress = { phase: 'cancelled' }
      return copy(task)
    }
    case 'runtime.install.events':
      return []
    case 'git.listProfiles':
      return { profiles: copy(gitProfiles), marks: { globalId: 'work', localIds: [], boundDirs: 1 } }
    case 'git.saveProfile':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return copy(payload.profile)
    case 'git.removeProfile':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { removed: true }
    case 'git.importGlobalIdentity':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { id: String(payload.asId || 'imported'), name: 'demo', email: 'demo@devtoolbox.local' }
    case 'git.applyProfile': {
      if (!payload.dryRun && !payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      const scope = payload.scope === 'global' ? 'global' : 'repos'
      const changes = [
        { action: 'set', key: 'user.name', scope, from: 'Xinyi Zhang <42613048+xyztony999@users.noreply.github.com>', to: '张三' },
        { action: 'set', key: 'user.email', scope, from: '42613048+xyztony999@users.noreply.github.com', to: 'z@corp.com' },
        { action: 'unset', key: 'user.signingkey', scope, from: '65A1285037EA924A' },
      ]
      // 与真实桥接同构：全局返回 {scope, changes}，仓库返回逐仓结果数组。
      if (scope === 'global') return { scope: 'global', changes }
      const repos = ((payload.repos as string[]) ?? []).filter(Boolean)
      return (repos.length ? repos : ['.']).map((target) => ({
        target, status: payload.dryRun ? 'skipped' : 'success', changes,
      }))
    }
    case 'app.pickFolder':
      return 'D:/Project/demo-repo'
    case 'git.effectiveIdentity':
      return {
        path: String(payload.path ?? '.'), repoRoot: String(payload.path ?? '.'), resolved: 'work', profileId: 'work',
        keys: [
          { key: 'user.name', value: '张三', source: 'includeIf' },
          { key: 'user.email', value: 'z@corp.com', source: 'includeIf' },
          { key: 'user.signingkey', value: 'KEY123', source: 'includeIf' },
          { key: 'commit.gpgsign', value: 'true', source: 'includeIf' },
          { key: 'gpg.format', value: 'ssh', source: 'includeIf' },
        ],
        insteadOf: [{ original: 'git@github.com:', url: 'git@github.com-work:' }],
        accounts: [{ host: 'github.com', username: 'corp-zhang' }],
        promptLine: 'work',
      }
    case 'git.bindProfile':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { id: payload.id, gitdirs: [payload.dir] }
    case 'git.unbindProfile': {
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { profile: { id: payload.id, gitdirs: [] }, drift: null }
    }
    case 'git.clearIdentity':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return [{ action: 'unset', key: 'user.name', scope: payload.scope }]
    case 'git.scanRepos':
      return {
        root: String(payload.root ?? 'D:/Project'),
        rows: [
          { path: 'D:/Project/unigate', relative: 'unigate', identityId: 'work', status: 'ok', hasLocal: true, source: 'local', remotes: [{ name: 'origin', url: 'https://github.com/demo/demo.git', pushUrls: [], mirror: false }] },
          { path: 'D:/Project/pdfjs', relative: 'pdfjs', status: 'missing', hasLocal: false, source: 'none', remotes: [] },
          { path: 'D:/Project/Kulaa', relative: 'Kulaa', status: 'manual', hasLocal: false, source: 'global', remotes: [{ name: 'origin', url: 'https://gitee.com/demo/demo.git', pushUrls: ['https://gitee.com/demo/demo.git', 'https://github.com/demo/demo.git'], mirror: true }] },
        ],
        summary: { total: 3, withLocal: 1, withIncludeIf: 0, missing: 1, manual: 1, mirrored: 1 },
      }
    case 'git.saveScanRoot':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { scanRoot: payload.root, scanDepth: payload.depth }
    case 'git.importScanFindings':
      if (!payload.dryRun && !payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return [{ id: 'imported-demo', name: 'Kulaa Dev', email: 'dev@kulaa.local' }]
    case 'git.auditRepo':
      return {
        repo: String(payload.repo ?? '.'),
        expectedProfileId: payload.expectedProfileId || undefined,
        authors: [
          { name: '张三', email: 'z@corp.com', count: 42, profileId: 'work', polluted: false },
          { name: 'zhangsan', email: 'zhangsan@corp.com', count: 3, polluted: true },
        ],
        mailmapSuggestion: '张三 <z@corp.com>\n张三 <z@corp.com> zhangsan <zhangsan@corp.com>',
        strictPassed: false,
        empty: false,
      }
    case 'git.gitDoctor':
      return {
        findings: [
          { check: 'gitVersion', level: 'info', detail: 'git version 2.56.0.windows.2' },
          { check: 'credentialHelper', level: 'info', detail: 'credential.helper = manager' },
        ],
        bindings: [{ profileId: 'work', dir: 'D:/work/', dirExists: true, snippetOk: true }],
        orphanSnippets: [],
        hookPresent: true,
      }
    case 'git.remoteList':
      return [{ name: 'origin', url: 'https://github.com/demo/demo.git', pushUrls: [], mirror: false, account: 'corp-zhang' }]
    case 'git.remoteAdd':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return [{ name: 'origin', url: 'https://github.com/demo/demo.git', pushUrls: [], mirror: false }, { name: payload.name, url: payload.url, pushUrls: [], mirror: false }]
    case 'git.remoteRemove':
    case 'git.remoteRename':
    case 'git.remoteSetUrl':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return [{ name: payload.to ?? payload.name ?? 'origin', url: payload.url ?? 'https://github.com/demo/demo.git', pushUrls: [], mirror: false }]
    case 'git.mirrorList':
      return [{ name: 'origin', url: 'https://github.com/demo/demo.git', pushUrls: [], mirror: false }]
    case 'git.mirrorSet':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { remote: payload.remote, pushUrls: ['https://github.com/demo/demo.git', payload.mirrorUrl] }
    case 'git.mirrorUnset':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { remote: payload.remote, removed: ['https://github.com/demo/demo.git', payload.mirrorUrl] }
    case 'git.credentialList':
      return [{ host: 'github.com', username: 'corp-zhang' }]
    case 'git.credentialLogin':
      return { host: payload.host, username: 'corp-zhang' }
    case 'git.credentialSet':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { host: payload.host, username: payload.username }
    case 'git.credentialRemove':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { host: payload.host }
    case 'git.legacyPreview':
      return {
        found: true, sourcePath: 'C:/Users/demo/AppData/Roaming/gitid/config.json', readable: true, valid: true,
        sourceDigest: 'demo-gitid-digest', profiles: gitProfiles.filter((p) => p.id === 'personal'),
        idConflicts: [], duplicateEmails: [], backupDirectory: 'C:/Users/demo/AppData/Roaming/DevToolbox/git-identity/backups/gitid',
      }
    case 'git.legacyImport':
      if (!payload.confirmBackup || !payload.confirmImport) throw new Error('GIT_LEGACY_CONFIRMATION_REQUIRED')
      return { sourceKind: 'gitid', sourceDigest: payload.sourceDigest, backupPath: '.../backups/gitid/config-demo.json', importedAt: now(), created: 1, skipped: 0, conflicts: 0 }
    case 'git.keygenOptions':
      return {
        tools: [
          { keyType: 'openpgp', available: true, executablePath: 'C:/Program Files/GnuPG/bin/gpg.exe', version: '2.4.5' },
          { keyType: 'ssh', available: true, executablePath: 'C:/Windows/System32/OpenSSH/ssh-keygen.exe', version: 'OpenSSH_for_Windows_9.5' },
        ],
        algorithms: [
          { keyType: 'openpgp', values: ['ed25519', 'rsa4096', 'rsa3072', 'nistp256'] },
          { keyType: 'ssh', values: ['ed25519', 'rsa4096', 'ecdsa256'] },
        ],
        expiries: [
          { keyType: 'openpgp', values: ['never', '1y', '2y', '5y'], default: 'never' },
          { keyType: 'ssh', values: ['never'], default: 'never' },
        ],
        defaults: {
          openpgp: { algorithm: 'ed25519', expiry: 'never' },
          ssh: { algorithm: 'ed25519', expiry: 'never' },
        },
        sshDefaultPathHint: 'C:/Users/demo/.ssh/devtoolbox_profile',
      }
    case 'git.keygenPreview': {
      const profileId = String(payload.profileId ?? '')
      const profile = gitProfiles.find((item) => item.id === profileId)
      if (!profile) throw new Error('GIT_PROFILE_NOT_FOUND')
      const keyType = String(payload.keyType ?? 'openpgp') as 'openpgp' | 'ssh'
      if (keyType === 'openpgp' && (!profile.name?.trim() || !profile.email?.trim())) {
        throw new Error('GIT_PROFILE_INVALID: 档案需要姓名和邮箱')
      }
      const algorithm = String(payload.algorithm ?? 'ed25519')
      const expiry = keyType === 'ssh' ? 'never' : String(payload.expiry ?? 'never')
      const openpgpAlgos = ['ed25519', 'rsa4096', 'rsa3072', 'nistp256']
      const sshAlgos = ['ed25519', 'rsa4096', 'ecdsa256']
      if (keyType === 'openpgp' && !openpgpAlgos.includes(algorithm)) throw new Error('GIT_KEYGEN_PARAMS_INVALID')
      if (keyType === 'ssh' && !sshAlgos.includes(algorithm)) throw new Error('GIT_KEYGEN_PARAMS_INVALID')
      if (keyType === 'ssh' && expiry !== 'never') throw new Error('GIT_KEYGEN_PARAMS_INVALID')
      const enableGpgSign = payload.enableGpgSign !== false
      const warnings: string[] = []
      if (profile.signingKey) warnings.push('将替换现有签名密钥引用')
      const passphraseSet = !!String(payload.passphrase ?? '')
      return {
        profileId,
        keyType,
        toolPath: keyType === 'ssh' ? 'C:/Windows/System32/OpenSSH/ssh-keygen.exe' : 'C:/Program Files/GnuPG/bin/gpg.exe',
        commandSummary: keyType === 'ssh'
          ? [
            `ssh-keygen -t ${algorithm === 'ecdsa256' ? 'ecdsa' : algorithm === 'rsa4096' ? 'rsa' : 'ed25519'} -f <output>`,
            `口令：${passphraseSet ? '<passphrase>' : '（未设置）'}`,
            `注释：devtoolbox:${profileId}`,
          ]
          : [
            `gpg --batch --generate-key（UID=${profile.name} <${profile.email}>）`,
            `算法 ${algorithm}，有效期 ${expiry}`,
            `口令：${passphraseSet ? '<passphrase>' : '（未设置）'}`,
          ],
        profilePatch: {
          signingKey: '(pending)',
          gpgFormat: keyType === 'ssh' ? 'ssh' : 'openpgp',
          gpgSign: enableGpgSign,
        },
        warnings,
      }
    }
    case 'git.keygenCreate': {
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      const profileId = String(payload.profileId ?? '')
      const profile = gitProfiles.find((item) => item.id === profileId)
      if (!profile) throw new Error('GIT_PROFILE_NOT_FOUND')
      const keyType = String(payload.keyType ?? 'openpgp') as 'openpgp' | 'ssh'
      if (keyType === 'openpgp' && (!profile.name?.trim() || !profile.email?.trim())) {
        throw new Error('GIT_PROFILE_INVALID: 档案需要姓名和邮箱')
      }
      const enableGpgSign = payload.enableGpgSign !== false
      const fingerprint = keyType === 'openpgp'
        ? `DEMO${Date.now().toString(16).toUpperCase().slice(-12)}`
        : undefined
      const outputBase = String(payload.outputPath ?? `C:/Users/demo/.ssh/devtoolbox_${profileId}`).replace(/\.pub$/i, '')
      const publicKeyPath = keyType === 'ssh' ? `${outputBase}.pub` : undefined
      const signingKey = keyType === 'ssh' ? publicKeyPath! : fingerprint!
      profile.signingKey = signingKey
      profile.gpgFormat = keyType === 'ssh' ? 'ssh' : 'openpgp'
      profile.gpgSign = enableGpgSign
      return {
        profile: copy(profile),
        keyType,
        signingKey,
        fingerprint: fingerprint || `SHA256:demo${profileId}`,
        publicKeyPath,
        publicKeyPreview: keyType === 'ssh'
          ? `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5DEMO devtoolbox:${profileId}`
          : undefined,
        localKeyCreated: true,
        profileUpdated: true,
      }
    }
    case 'git.keyList': {
      const keyType = String(payload.keyType ?? 'openpgp') as 'openpgp' | 'ssh'
      if (keyType === 'openpgp') {
        return [
          { keyType: 'openpgp', id: 'ABCDEF0123456789', fingerprint: 'ABCD EF01 2345 6789', uidOrComment: '张三 <z@corp.com>', createdAt: '2025-01-12' },
          { keyType: 'openpgp', id: 'FEDCBA9876543210', fingerprint: 'FEDC BA98 7654 3210', uidOrComment: 'tony <tony@me.dev>', createdAt: '2024-08-03' },
        ]
      }
      return [
        { keyType: 'ssh', id: 'C:/Users/demo/.ssh/devtoolbox_work.pub', fingerprint: 'SHA256:demoworkkey', uidOrComment: 'devtoolbox:work', publicKeyPath: 'C:/Users/demo/.ssh/devtoolbox_work.pub', createdAt: '2025-03-01' },
        { keyType: 'ssh', id: 'C:/Users/demo/.ssh/id_ed25519.pub', fingerprint: 'SHA256:demoided25519', uidOrComment: 'demo@devtoolbox', publicKeyPath: 'C:/Users/demo/.ssh/id_ed25519.pub' },
      ]
    }
    case 'git.keySelect': {
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      const profileId = String(payload.profileId ?? '')
      const profile = gitProfiles.find((item) => item.id === profileId)
      if (!profile) throw new Error('GIT_PROFILE_NOT_FOUND')
      const keyType = String(payload.keyType ?? 'openpgp') as 'openpgp' | 'ssh'
      const id = String(payload.id ?? '')
      if (!id) throw new Error('GIT_KEY_NOT_FOUND')
      const enableGpgSign = payload.enableGpgSign !== false
      const signingKey = id
      profile.signingKey = signingKey
      profile.gpgFormat = keyType === 'ssh' ? 'ssh' : 'openpgp'
      profile.gpgSign = enableGpgSign
      return {
        profile: copy(profile),
        keyType,
        signingKey,
        fingerprint: keyType === 'openpgp' ? id : `SHA256:select${id.length}`,
        publicKeyPath: keyType === 'ssh' ? id : undefined,
        localKeyCreated: false,
        profileUpdated: true,
      }
    }
    case 'git.keyExport': {
      const keyType = String(payload.keyType ?? 'openpgp') as 'openpgp' | 'ssh'
      const id = String(payload.id ?? '')
      if (!id) throw new Error('GIT_KEY_NOT_FOUND')
      const text = keyType === 'ssh'
        ? `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5DEMOEXPORT ${id}`
        : `-----BEGIN PGP PUBLIC KEY BLOCK-----\nVersion: Demo\n\nmQENDemoPublicKey${id.slice(0, 8)}\n-----END PGP PUBLIC KEY BLOCK-----`
      const writePath = String(payload.writePath ?? '').trim()
      if (!writePath) return { text }
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      // 演示：含 "exists" 的路径视为已存在，需 overwrite
      if (/exists/i.test(writePath) && !payload.overwrite) throw new Error('GIT_KEY_EXPORT_EXISTS')
      if (writePath.includes('://') || writePath.length < 2) throw new Error('GIT_KEY_EXPORT_PATH_INVALID')
      return { text, writtenPath: writePath }
    }
    case 'session.preview': {
      const selected = userRuntimeState.installations.filter((item) => userRuntimeState.globalSelections[item.runtime]?.installationId === item.id && item.available)
      const binFor = (runtime: string, home: string) => ['node', 'python'].includes(runtime) ? home : `${home}/bin`
      return {
        cwd: String(payload.path ?? '.'),
        tools: ['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python', 'rust'].map((tool) => {
          const installation = selected.find((item) => item.runtime === tool)
          if (installation) return { tool, status: 'injected', version: installation.version, binDir: binFor(tool, installation.localHome) }
          const selection = userRuntimeState.globalSelections[tool]
          return { tool, status: 'skipped', reason: selection?.status === 'unresolved' ? `待重新关联：${selection.unresolvedHome}` : '未选择' }
        }),
        javaHome: selected.find((item) => item.runtime === 'java')?.localHome,
        pathPrepends: selected.map((item) => binFor(item.runtime, item.localHome)),
        gitIdentity: { resolved: 'personal', profileId: 'personal' },
        terminals: [
          { id: 'powershell', label: 'PowerShell' },
          { id: 'cmd', label: '命令提示符' },
        ],
      }
    }
    case 'session.open':
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      return { terminalId: String(payload.terminalId ?? 'powershell'), pid: 4242 }
    case 'settings.get':
      return copy(clientSettings)
    case 'settings.toolStatuses': {
      const configured = userRuntimeState.installations.filter((item) => userRuntimeState.globalSelections[item.runtime]?.installationId === item.id)
      const configuredTools = configured.map((item) => ({
        tool: item.runtime,
        available: item.available,
        version: item.available ? item.version : 'unknown',
        home: item.localHome,
        source: 'user',
        configurable: ['golang', 'node', 'python', 'maven'].includes(item.runtime),
        configNote: ['golang', 'node', 'python', 'maven'].includes(item.runtime) ? '可由本产品写入工具原生用户配置' : '只读；请使用工具自身的配置文件',
      }))
      const rest = ['node', 'java', 'maven', 'gradle', 'androidsdk', 'golang', 'python', 'rust']
        .filter((tool) => !configured.some((item) => item.runtime === tool))
        .map((tool) => ({
          tool,
          available: false,
          source: 'none',
          configurable: ['golang', 'node', 'python', 'maven'].includes(tool),
          configNote: ['golang', 'node', 'python', 'maven'].includes(tool) ? '可由本产品写入工具原生用户配置' : '只读；请使用工具自身的配置文件',
        }))
      return [...configuredTools, ...rest]
    }
    case 'settings.setTheme': {
      const mode = String(payload.mode ?? 'system')
      if (mode === 'neon') throw new Error('CLIENT_SETTINGS_THEME_INVALID')
      clientSettings.theme = mode
      if (mode === 'custom') {
        const accent = String(payload.accent ?? '')
        if (!/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(accent)) throw new Error('CLIENT_SETTINGS_ACCENT_INVALID')
        clientSettings.customAccent = accent
      } else {
        clientSettings.customAccent = ''
      }
      clientSettings.revision += 1
      return copy(clientSettings)
    }
    case 'settings.addSourceProject': {
      const rawPath = String(payload.path ?? '').trim()
      if (!rawPath || rawPath.includes('://')) throw new Error('CLIENT_SETTINGS_PATH_INVALID')
      const path = rawPath.replace(/\\/g, '/')
      if (clientSettings.sourceProjects.some((item) => item.path.toLowerCase() === path.toLowerCase())) throw new Error('CLIENT_SETTINGS_PATH_INVALID: already registered')
      const binding = String(payload.binding ?? 'local')
      if (!['local', 'shared'].includes(binding)) throw new Error('CLIENT_SETTINGS_BINDING_INVALID')
      clientSettings.sourceProjects.push({ path, label: String(payload.label ?? ''), binding, note: String(payload.note ?? '') })
      clientSettings.revision += 1
      return copy(clientSettings)
    }
    case 'settings.removeSourceProject': {
      const path = String(payload.path ?? '').trim().replace(/\\/g, '/')
      const index = clientSettings.sourceProjects.findIndex((item) => item.path.toLowerCase() === path.toLowerCase())
      if (index < 0) return { settings: copy(clientSettings), removed: false }
      clientSettings.sourceProjects.splice(index, 1)
      clientSettings.revision += 1
      return { settings: copy(clientSettings), removed: true }
    }
    case 'settings.exportSnapshot': {
      const shared = clientSettings.sourceProjects.filter((item) => item.binding === 'shared')
      const draft = {
        schema: 1,
        label: String(payload.label ?? ''),
        exportedAt: now(),
        theme: clientSettings.theme,
        customAccent: clientSettings.customAccent,
        sourceProjects: payload.includeSharedProjects ? clientSettings.sourceProjects : shared,
      }
      const digest = 'demo-' + digestString(JSON.stringify(draft))
      const snapshot = { ...draft, digest }
      return { snapshot, payload: JSON.stringify(snapshot, null, 2) }
    }
    case 'settings.importSnapshot': {
      if (!payload.confirm) throw new Error('CONFIRMATION_REQUIRED')
      const snapshot = JSON.parse(String(payload.payload ?? '{}'))
      if (!snapshot?.schema || !snapshot?.digest) throw new Error('CLIENT_SETTINGS_SNAPSHOT_INVALID')
      const existing = clientSettings.snapshots.find((item) => item.sourceDigest === snapshot.digest)
      if (existing) return { settings: copy(clientSettings), record: existing, backedUp: false }
      clientSettings.theme = snapshot.theme ?? clientSettings.theme
      clientSettings.customAccent = snapshot.customAccent ?? ''
      clientSettings.revision += 1
      const record = { sourceDigest: snapshot.digest, label: snapshot.label ?? '', backupPath: 'demo-backups/' + snapshot.digest, importedAt: now(), applied: 1, skipped: 0 }
      clientSettings.snapshots.push(record)
      return { settings: copy(clientSettings), record, backedUp: !!payload.confirmBackup }
    }
    default:
      throw new Error(`演示桥接不支持操作：${operation}`)
  }
}
