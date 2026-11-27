export type ScanStatus = 'complete' | 'partial' | 'cancelled'
export type RequirementStatus = 'resolved' | 'unknown' | 'conflict' | 'unsupported'

export interface Evidence {
  sourcePath: string
  locator: string
  rawValue: string
  parserId: string
  observedAt?: string
}

export interface RuntimeRequirement {
  runtime: string
  rawConstraint?: string
  normalizedConstraint?: string
  sourcePath?: string
  confidence: string
  status: RequirementStatus
  evidence?: Evidence[]
}

export interface Repository {
  id: string
  path: string
  kind: 'normal' | 'worktree' | 'nested' | 'unknown'
  source: string
  branch?: string
  remoteName?: string
  remoteUrl?: string
  userName?: string
  userEmail?: string
}

export interface Module {
  id: string
  relativePath: string
  buildSystems: string[]
  requirements: RuntimeRequirement[]
}

export interface ScanIssue {
  code: string
  severity: string
  relativePath?: string
  message: string
  recoverable: boolean
}

export interface Candidate {
  id: string
  displayName: string
  rootPath: string
  evidenceDigest: string
  repositories: Repository[]
  modules: Module[]
  issues: ScanIssue[]
  status: ScanStatus
  scannedAt: string
}

export interface Project {
  id: string
  displayName: string
  rootPath: string
  repositories: Repository[]
  modules: Module[]
  issues?: ScanIssue[]
  scanStatus: ScanStatus
  createdAt: string
  lastScanAt: string
  /** 落盘时使用的扫描器版本；旧快照无此字段（视为过期，界面提示重扫） */
  scanToolVersion?: string
  revision: number
}

export interface NativeRequest {
  requestId: string
  operation: string
  payload: Record<string, unknown>
}

export interface NativeResponse<T> {
  requestId: string
  ok: boolean
  data?: T
  error?: { code: string; message: string }
}

export interface InstallMirror {
  id: string
  label: string
}

export interface InstallOption {
  tool: string
  providerId: string
  mirrors: InstallMirror[]
  versionSelectors: string[]
  supportedPlatforms: string[]
  supportedArchs: string[]
  unsupported?: string
}

export interface InstallPackageInfo {
  name: string
  fileName: string
  checksumAlgorithm: string
  checksum: string
  estimatedBytes?: number
}

export interface InstallPlan {
  planId: string
  planDigest: string
  tool: string
  providerId: string
  mirror: string
  requestedSelector: string
  resolvedVersion: string
  os: string
  arch: string
  packages: InstallPackageInfo[]
  targetHome: string
  estimatedBytes?: number
  targetExists: boolean
  expiresAt: string
}

export interface InstallTaskError {
  code: string
  message: string
}

export interface InstallProgress {
  phase: string
  package?: string
  receivedBytes?: number
  totalBytes?: number
}

export interface InstallPackageResult {
  name: string
  fileName: string
  state: 'pending' | 'done' | 'reused' | 'failed'
  error?: InstallTaskError
}

export interface InstallTask {
  taskId: string
  planId: string
  planDigest: string
  state: 'confirmed' | 'downloading' | 'verifying' | 'extracting' | 'registering' | 'complete' | 'failed' | 'cancelled' | 'partial'
  progress: InstallProgress
  packages: InstallPackageResult[]
  installationId?: string
  error?: InstallTaskError
}

export interface InstallEvent {
  sequence: number
  taskId: string
  task: InstallTask
}

export interface GitInsteadOf {
  original: string
  url: string
}

export interface GitAccount {
  host: string
  username: string
}

export interface GitProfile {
  id: string
  name: string
  email: string
  signingKey?: string
  gpgSign?: boolean
  gpgFormat?: string
  customKv?: Record<string, string>
  insteadOf?: GitInsteadOf[]
  accounts?: GitAccount[]
  gitdirs?: string[]
}

export interface GitProfileMarks {
  globalId?: string
  localIds?: string[]
  boundDirs: number
}

export interface GitConfigChange {
  action: string
  key: string
  scope: string
  target?: string
  /** 后端为 *string：未设置时是 null，键存在但为空时是空串 */
  from?: string | null
  to?: string | null
}

export interface GitApplyTargetResult {
  target: string
  status: 'success' | 'failed' | 'skipped' // skipped = 预检通过（dryRun 未写入）
  changes?: GitConfigChange[]
  error?: string
}

/** git.applyProfile 在 scope=global 时（预览与应用同构）的返回 */
export interface GitGlobalApplyResult {
  scope: 'global'
  changes: GitConfigChange[] | null
}

export interface GitKeyOrigin {
  key: string
  value?: string
  source: 'local' | 'includeIf' | 'global' | 'none'
}

export interface GitEffectiveIdentity {
  path: string
  repoRoot?: string
  resolved: string
  profileId?: string
  keys: GitKeyOrigin[]
  insteadOf?: GitInsteadOf[]
  accounts?: GitAccount[]
  promptLine?: string
}

export interface GitSnippetDrift {
  id: string
  drift: boolean
  missing: boolean
  unknown: boolean
  keys?: string[]
}

export interface GitDoctorFinding {
  check: string
  level: 'info' | 'warn' | 'error'
  detail: string
}

export interface GitDoctorBinding {
  profileId: string
  dir: string
  dirExists: boolean
  snippetOk: boolean
  snippetMissing?: boolean
  drift?: GitSnippetDrift
}

export interface GitDoctorReport {
  findings: GitDoctorFinding[]
  bindings: GitDoctorBinding[]
  orphanSnippets: string[]
  hookPresent: boolean
}

export interface GitRemoteInfo {
  name: string
  url?: string
  pushUrls?: string[]
  mirror: boolean
  account?: string
}

export interface GitScanRow {
  path: string
  relative: string
  identityId?: string
  status: 'ok' | 'missing' | 'manual'
  hasLocal: boolean
  localName?: string
  localEmail?: string
  source: string
  remotes: GitRemoteInfo[]
}

export interface GitScanResult {
  root: string
  rows: GitScanRow[]
  summary: { total: number; withLocal: number; withIncludeIf: number; missing: number; manual: number; mirrored: number }
}

export interface GitAuditReport {
  repo: string
  expectedProfileId?: string
  authors: Array<{ name: string; email: string; count: number; profileId?: string; polluted?: boolean }>
  mailmapSuggestion?: string
  strictPassed?: boolean
  empty: boolean
}

export interface GitLegacyPreview {
  found: boolean
  sourcePath?: string
  readable: boolean
  valid: boolean
  sourceDigest?: string
  profiles: GitProfile[]
  idConflicts: string[]
  duplicateEmails: string[]
  backupDirectory?: string
}

export interface GitImportRecord {
  sourceKind: string
  sourceDigest: string
  backupPath: string
  importedAt: string
  created: number
  skipped: number
  conflicts: number
}

export interface GitCredentialEntry {
  host: string
  username?: string
}

/** git-gpg-keygen：密钥类型 / 算法 / 有效期（与 common-enums 一致） */
export type GitKeyType = 'openpgp' | 'ssh'
export type GitKeyAlgorithm = 'ed25519' | 'rsa4096' | 'rsa3072' | 'nistp256' | 'ecdsa256'
export type GitKeyExpiry = 'never' | '1y' | '2y' | '5y'

export interface KeygenToolStatus {
  keyType: GitKeyType
  available: boolean
  executablePath?: string
  version?: string
}

export interface KeygenOptions {
  tools: KeygenToolStatus[]
  algorithms: Array<{ keyType: GitKeyType; values: GitKeyAlgorithm[] }>
  expiries: Array<{ keyType: GitKeyType; values: GitKeyExpiry[]; default: GitKeyExpiry }>
  defaults: {
    openpgp: { algorithm: GitKeyAlgorithm; expiry: GitKeyExpiry }
    ssh: { algorithm: GitKeyAlgorithm; expiry: GitKeyExpiry }
  }
  /** 演示/宿主可选：SSH 默认私钥路径提示（无后缀） */
  sshDefaultPathHint?: string
}

export interface KeygenCreateRequest {
  profileId: string
  keyType: GitKeyType
  algorithm?: GitKeyAlgorithm
  expiry?: GitKeyExpiry
  passphrase?: string
  enableGpgSign?: boolean
  comment?: string
  outputPath?: string
  dryRun?: boolean
  confirm?: boolean
}

export interface KeygenPlan {
  profileId: string
  keyType: GitKeyType
  toolPath: string
  commandSummary: string[]
  profilePatch: { signingKey: string; gpgFormat: string; gpgSign: boolean }
  warnings: string[]
  wouldFail?: string
}

export interface KeygenResult {
  profile: GitProfile
  keyType: GitKeyType
  signingKey: string
  fingerprint?: string
  publicKeyPath?: string
  publicKeyPreview?: string
  localKeyCreated: boolean
  profileUpdated: boolean
  warning?: string
}

export interface LocalKeySummary {
  keyType: GitKeyType
  id: string
  fingerprint: string
  uidOrComment: string
  publicKeyPath?: string
  createdAt?: string
}

export interface KeyExportRequest {
  keyType: GitKeyType
  id: string
  writePath?: string
  overwrite?: boolean
  confirm?: boolean
}

export interface PublicKeyExport {
  text: string
  writtenPath?: string
}
