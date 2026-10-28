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
    scanStatus: 'complete', createdAt: now(), lastScanAt: now(), revision: 1,
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
    golang: { unresolvedHome: 'D:/Tools/go-1.24', status: 'unresolved', source: 'legacy' },
    python: { status: 'empty', source: '' },
  } as Record<string, { installationId?: string; unresolvedHome?: string; status: string; source: string }>,
  installations: [
    { id: 'demo-node-22', runtime: 'node', version: '22.14.0', major: '22', localHome: 'D:/Tools/node-v22.14.0', source: 'downloaded', label: 'Node LTS', vendor: '', available: true },
    { id: 'demo-java-21', runtime: 'java', version: '21.0.6', major: '21', localHome: 'D:/Tools/jdk-21', source: 'imported', label: '', vendor: 'Temurin', available: true },
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

export async function mockInvoke(operation: string, payload: Record<string, unknown>): Promise<unknown> {
  switch (operation) {
    case 'project.list':
      return copy(projects)
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
      const candidates = root ? [] : [{ candidateId: 'demo-node-runtime', digest: 'demo-node-runtime', tool: 'node', version: '22.12.0', major: '22', home: 'C:/demo/node', binary: 'C:/demo/node/node.exe', source: 'path', evidence: ['directory-name'], availability: 'available', confidence: 'exact' }]
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
      const target = scope === 'global' ? 'global' : String((payload.repos as string[])?.[0] ?? '.')
      return [{
        target, status: payload.dryRun ? 'skipped' : 'success',
        changes: [
          { action: 'set', key: 'user.name', scope, to: '张三' },
          { action: 'set', key: 'user.email', scope, to: 'z@corp.com' },
          { action: 'unset', key: 'gpg.format', scope },
        ],
      }]
    }
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
    default:
      throw new Error(`演示桥接不支持操作：${operation}`)
  }
}
