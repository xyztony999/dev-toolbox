export type ScanStatus = 'complete' | 'partial' | 'cancelled'
export type RequirementStatus = 'resolved' | 'unknown' | 'conflict' | 'unsupported'

export interface RuntimeRequirement {
  runtime: string
  rawConstraint?: string
  normalizedConstraint?: string
  sourcePath?: string
  confidence: string
  status: RequirementStatus
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
  from?: string
  to?: string
}

export interface GitApplyTargetResult {
  target: string
  status: 'success' | 'failed' | 'skipped'
  changes?: GitConfigChange[]
  error?: string
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

export interface GitDoctorReport {
  findings: GitDoctorFinding[]
  bindings: Array<{ profileId: string; dir: string; dirExists: boolean; snippetOk: boolean; drift?: GitSnippetDrift }>
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
