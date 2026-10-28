<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type {
  GitAuditReport, GitCredentialEntry, GitDoctorReport, GitEffectiveIdentity,
  GitImportRecord, GitLegacyPreview, GitProfile, GitScanResult,
} from './types'

const profiles = ref<GitProfile[]>([])
const globalId = ref('')
const booting = ref(true)
const editing = ref<GitProfile>()
const rewriteOriginal = ref('')
const rewriteUrl = ref('')
const accountHost = ref('')
const accountUsername = ref('')
const busy = ref(false)
const errorText = ref('')

const applyId = ref('')
const applyScope = ref<'global' | 'repos'>('global')
const applyRepo = ref('')
const applyResults = ref<Array<{ target: string; status: string; changes?: Array<{ key: string; action: string; from?: string; to?: string }>; error?: string }>>([])
const effective = ref<GitEffectiveIdentity>()
const effectivePath = ref('.')
const bindDir = ref('')

const scanRoot = ref('')
const scanDepth = ref(4)
const scan = ref<GitScanResult>()
const auditRepo = ref('')
const auditExpect = ref('')
const auditStrict = ref(false)
const audit = ref<GitAuditReport>()
const doctor = ref<GitDoctorReport>()

const legacy = ref<GitLegacyPreview>()
const legacyRecord = ref<GitImportRecord>()
const credentials = ref<GitCredentialEntry[]>([])
const credHost = ref('github.com')
const credUsername = ref('')
const credToken = ref('')
const mirrorRepo = ref('.')
const mirrorRemote = ref('origin')
const mirrorUrl = ref('')

async function reloadProfiles() {
  const result = await invoke<{ profiles: GitProfile[]; marks: { globalId?: string } }>('git.listProfiles', {})
  profiles.value = result.profiles
  globalId.value = result.marks.globalId ?? ''
}

async function bootstrap() {
  const result = await invoke<{ profiles: GitProfile[]; marks: { globalId?: string }; credentials: GitCredentialEntry[]; settings: { scanRoot: string; scanDepth: number } }>('git.bootstrap', {})
  profiles.value = result.profiles
  globalId.value = result.marks.globalId ?? ''
  credentials.value = result.credentials
  if (result.settings.scanRoot) scanRoot.value = result.settings.scanRoot
  if (result.settings.scanDepth > 0) scanDepth.value = result.settings.scanDepth
}

async function guard(action: () => Promise<void>) {
  busy.value = true
  errorText.value = ''
  try {
    await action()
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

function startEdit(profile: GitProfile) {
  editing.value = JSON.parse(JSON.stringify(profile)) as GitProfile
  rewriteOriginal.value = profile.insteadOf?.[0]?.original ?? ''
  rewriteUrl.value = profile.insteadOf?.[0]?.url ?? ''
  accountHost.value = profile.accounts?.[0]?.host ?? ''
  accountUsername.value = profile.accounts?.[0]?.username ?? ''
}

function newProfile() {
  editing.value = { id: '', name: '', email: '', insteadOf: [], accounts: [] }
  rewriteOriginal.value = ''
  rewriteUrl.value = ''
  accountHost.value = ''
  accountUsername.value = ''
}

const saveProfile = () => guard(async () => {
  const profile = editing.value
  if (!profile) return
  profile.insteadOf = rewriteOriginal.value && rewriteUrl.value
    ? [{ original: rewriteOriginal.value, url: rewriteUrl.value }]
    : []
  profile.accounts = accountHost.value && accountUsername.value
    ? [{ host: accountHost.value, username: accountUsername.value }]
    : []
  await invoke('git.saveProfile', { profile, confirm: true })
  editing.value = undefined
  message.success('身份档案已保存')
  await reloadProfiles()
})

const removeProfile = (id: string) => guard(async () => {
  await invoke('git.removeProfile', { id, confirm: true })
  message.success('档案已删除（已写入的 git 配置不受影响）')
  await reloadProfiles()
})

const importGlobal = () => guard(async () => {
  await invoke('git.importGlobalIdentity', { asId: '', confirm: true })
  message.success('已收编当前全局身份')
  await reloadProfiles()
})

const previewApply = () => guard(async () => {
  applyResults.value = await invoke('git.applyProfile', { id: applyId.value, scope: applyScope.value === 'global' ? 'global' : 'repos', repos: applyScope.value === 'global' ? [] : [applyRepo.value || '.'], dryRun: true })
})

const confirmApply = () => guard(async () => {
  applyResults.value = await invoke('git.applyProfile', { id: applyId.value, scope: applyScope.value === 'global' ? 'global' : 'repos', repos: applyScope.value === 'global' ? [] : [applyRepo.value || '.'], dryRun: false, confirm: true })
  message.success('应用完成；逐仓结果如下')
  await reloadProfiles()
})

const loadEffective = () => guard(async () => {
  effective.value = await invoke<GitEffectiveIdentity>('git.effectiveIdentity', { path: effectivePath.value || '.' })
})

const doBind = () => guard(async () => {
  await invoke('git.bindProfile', { id: applyId.value, dir: bindDir.value, confirm: true })
  message.success('目录绑定完成，目录树下仓库将经 includeIf 生效')
  await reloadProfiles()
})

const doUnbind = () => guard(async () => {
  const result = await invoke<{ profile: GitProfile; drift: { drift: boolean; keys?: string[] } | null }>('git.unbindProfile', { id: applyId.value, dir: bindDir.value, confirm: true })
  if (result.drift?.drift) message.warning(`片段有手改漂移（${(result.drift.keys ?? []).join('、')}），已如实回收`)
  else message.success('解绑完成')
  await reloadProfiles()
})

const runScan = () => guard(async () => {
  scan.value = await invoke<GitScanResult>('git.scanRepos', { root: scanRoot.value, depth: scanDepth.value })
})

const runAudit = () => guard(async () => {
  audit.value = await invoke<GitAuditReport>('git.auditRepo', { repo: auditRepo.value || '.', expectedProfileId: auditExpect.value, strict: auditStrict.value })
})

const runDoctor = () => guard(async () => {
  doctor.value = await invoke<GitDoctorReport>('git.gitDoctor', {})
})

const previewLegacy = () => guard(async () => {
  legacy.value = await invoke<GitLegacyPreview>('git.legacyPreview', {})
})

const importLegacy = () => guard(async () => {
  const snapshot = legacy.value
  if (!snapshot?.sourceDigest) return
  legacyRecord.value = await invoke<GitImportRecord>('git.legacyImport', { sourceDigest: snapshot.sourceDigest, profileIds: [], confirmBackup: true, confirmImport: true })
  message.success('gitid 档案导入完成')
  await reloadProfiles()
})

const reloadCredentials = () => guard(async () => {
  credentials.value = await invoke<GitCredentialEntry[]>('git.credentialList', { host: '' })
})

const credentialLogin = (host: string) => guard(async () => {
  await invoke('git.credentialLogin', { host })
  message.success(`helper 登录完成：${host}`)
  await reloadCredentials()
})

const credentialSet = () => guard(async () => {
  await invoke('git.credentialSet', { host: credHost.value, username: credUsername.value, token: credToken.value, confirm: true })
  credToken.value = ''
  message.success('凭据已写入 helper（token 未留存）')
  await reloadCredentials()
})

const credentialRemove = (host: string, username?: string) => guard(async () => {
  await invoke('git.credentialRemove', { host, username, confirm: true })
  message.success('凭据已请求 helper 擦除')
  await reloadCredentials()
})

const mirrorSet = () => guard(async () => {
  await invoke('git.mirrorSet', { repo: mirrorRepo.value, remote: mirrorRemote.value, mirrorUrl: mirrorUrl.value, confirm: true })
  message.success('镜像推送已配置')
})

const mirrorUnset = () => guard(async () => {
  await invoke('git.mirrorUnset', { repo: mirrorRepo.value, remote: mirrorRemote.value, confirm: true })
  message.success('镜像推送已取消')
})

onMounted(() => guard(async () => {
  await bootstrap()
  booting.value = false
}))
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
  <div v-if="booting" class="section boot-panel">
    <a-skeleton active :paragraph="{ rows: 4 }" />
    <p class="muted">正在读取身份档案与凭据账号…</p>
  </div>
  <template v-else>

  <section class="section">
    <div class="section-title">
      <div><h2>身份档案</h2><p>token 不入档案；★ 为当前全局生效。删除档案不改动已写入的 git 配置。</p></div>
      <a-space>
        <a-button :loading="busy" @click="importGlobal">收编当前全局身份</a-button>
        <a-button type="primary" @click="newProfile">新建档案</a-button>
      </a-space>
    </div>
    <a-empty v-if="!profiles.length" description="尚无身份档案" />
    <div v-for="profile in profiles" :key="profile.id" class="profile-row">
      <div>
        <strong>{{ profile.id }} <a-tag v-if="profile.id === globalId" color="blue">★ 全局生效</a-tag></strong>
        <small>{{ profile.name }} &lt;{{ profile.email }}&gt;<template v-if="profile.signingKey"> · 签名 {{ profile.gpgFormat || 'openpgp' }}</template></small>
        <small v-if="profile.insteadOf?.length || profile.accounts?.length">
          重写 {{ profile.insteadOf?.length ?? 0 }} 条 · 账号 {{ profile.accounts?.map((a) => a.host).join('、') }}
        </small>
      </div>
      <a-space>
        <a-button size="small" @click="startEdit(profile)">编辑</a-button>
        <a-button size="small" danger @click="removeProfile(profile.id)">删除</a-button>
      </a-space>
    </div>
    <template v-if="editing">
      <a-divider />
      <a-form layout="vertical" class="profile-form">
        <a-space wrap>
          <a-input v-model:value="editing.id" placeholder="id（字母数字._-）" :disabled="!!profiles.some((p) => p.id === editing!.id)" style="width: 160px" />
          <a-input v-model:value="editing.name" placeholder="user.name" style="width: 160px" />
          <a-input v-model:value="editing.email" placeholder="user.email" style="width: 220px" />
          <a-input v-model:value="editing.signingKey" placeholder="signingkey（可空）" style="width: 200px" />
          <a-checkbox v-model:checked="editing.gpgSign">gpgsign</a-checkbox>
          <a-select v-model:value="editing.gpgFormat" allow-clear placeholder="gpg.format" style="width: 120px"
            :options="['openpgp', 'ssh', 'x509'].map((v) => ({ label: v, value: v }))" />
        </a-space>
        <a-space wrap>
          <a-input v-model:value="rewriteOriginal" placeholder="insteadOf 原始地址" style="width: 220px" />
          <a-input v-model:value="rewriteUrl" placeholder="替换为（url 前缀）" style="width: 220px" />
          <a-input v-model:value="accountHost" placeholder="账号 host" style="width: 160px" />
          <a-input v-model:value="accountUsername" placeholder="用户名选择器" style="width: 160px" />
        </a-space>
        <div class="form-actions">
          <a-button @click="editing = undefined">取消</a-button>
          <a-button type="primary" :loading="busy" @click="saveProfile">保存档案</a-button>
        </div>
      </a-form>
    </template>
  </section>

  <section class="section">
    <div class="section-title"><div><h2>应用与目录绑定</h2><p>应用是全量覆盖：档案缺失的受管键会被清除；先 dry-run 预览再确认执行。</p></div></div>
    <a-space wrap>
      <a-select v-model:value="applyId" placeholder="选择档案" style="width: 180px" :options="profiles.map((p) => ({ label: p.id, value: p.id }))" />
      <a-radio-group v-model:value="applyScope">
        <a-radio-button value="global">全局</a-radio-button>
        <a-radio-button value="repos">指定仓库</a-radio-button>
      </a-radio-group>
      <a-input v-if="applyScope === 'repos'" v-model:value="applyRepo" placeholder="仓库路径" style="width: 280px" />
      <a-button :loading="busy" :disabled="!applyId" @click="previewApply">dry-run 预览</a-button>
      <a-button type="primary" :loading="busy" :disabled="!applyId" @click="confirmApply">确认应用</a-button>
    </a-space>
    <div v-for="result in applyResults" :key="result.target" class="apply-row">
      <div><strong>{{ result.target }}</strong><a-tag :color="result.status === 'success' ? 'green' : result.status === 'failed' ? 'red' : 'default'">{{ result.status }}</a-tag></div>
      <small v-if="result.error">{{ result.error }}</small>
      <div v-else><code v-for="change in result.changes" :key="change.key + change.action" class="change-chip">{{ change.action }} {{ change.key }}<template v-if="change.to"> → {{ change.to }}</template></code></div>
    </div>
    <a-divider />
    <a-space wrap>
      <a-input v-model:value="effectivePath" placeholder="目录或仓库路径" style="width: 300px" />
      <a-button :loading="busy" @click="loadEffective">查看生效身份</a-button>
      <a-input v-model:value="bindDir" placeholder="绑定目录树（可不存在）" style="width: 260px" />
      <a-button :loading="busy" :disabled="!applyId" @click="doBind">绑定</a-button>
      <a-button :loading="busy" :disabled="!applyId" @click="doUnbind">解绑</a-button>
    </a-space>
    <div v-if="effective" class="apply-row">
      <div><strong>{{ effective.resolved }}</strong><small>每键来源</small></div>
      <code v-for="key in effective.keys" :key="key.key" class="change-chip">{{ key.key }} = {{ key.value || '∅' }}（{{ key.source }}）</code>
    </div>
  </section>

  <section class="section">
    <div class="section-title"><div><h2>仓库审计与 Git 体检</h2><p>扫描与审计只读；strict 是提交前门禁，本产品不做自动修复、不写 .mailmap。</p></div></div>
    <div class="toolbar">
      <a-input v-model:value="scanRoot" placeholder="扫描根目录" style="width: 260px" />
      <a-input-number v-model:value="scanDepth" :min="1" :max="12" />
      <a-button :loading="busy" @click="runScan">扫描</a-button>
      <a-input v-model:value="auditRepo" placeholder="审计仓库路径" style="width: 220px" />
      <a-select v-model:value="auditExpect" allow-clear placeholder="期望档案" style="width: 150px" :options="profiles.map((p) => ({ label: p.id, value: p.id }))" />
      <a-checkbox v-model:checked="auditStrict">strict</a-checkbox>
      <a-button :loading="busy" @click="runAudit">提交审计</a-button>
      <a-button :loading="busy" @click="runDoctor">Git 体检</a-button>
    </div>
    <div v-if="scan" class="apply-row">
      <div><strong>扫描 {{ scan.summary.total }} 个仓库</strong><small>本地覆盖 {{ scan.summary.withLocal }} · 绑定生效 {{ scan.summary.withIncludeIf }} · 缺失 {{ scan.summary.missing }} · 手配 {{ scan.summary.manual }} · 镜像 {{ scan.summary.mirrored }}</small></div>
      <div v-for="row in scan.rows" :key="row.path" class="scan-row">
        <code>{{ row.relative }}</code>
        <a-tag :color="row.status === 'ok' ? 'green' : row.status === 'missing' ? 'red' : 'orange'">{{ row.status === 'ok' ? '✓ 一致' : row.status === 'missing' ? '✗ 缺失' : '⚠ 手配' }}</a-tag>
        <small>{{ row.source }}<template v-if="row.identityId"> · {{ row.identityId }}</template><template v-if="row.remotes.some((r) => r.mirror)"> · ⊕ 镜像</template></small>
        <a-button v-if="row.status === 'manual'" size="small" @click="guard(async () => { await invoke('git.importScanFindings', { root: scan!.root, depth: scanDepth, paths: [row.path], dryRun: false, confirm: true }); message.success('已收编手配身份'); await reloadProfiles() })">收编</a-button>
      </div>
    </div>
    <div v-if="audit" class="apply-row">
      <div><strong>作者分布（{{ audit.repo }}）</strong><a-tag v-if="audit.strictPassed === false" color="red">strict 未通过</a-tag><a-tag v-else-if="audit.strictPassed" color="green">strict 通过</a-tag></div>
      <div v-for="author in audit.authors" :key="author.email + author.name" class="scan-row">
        <span>{{ author.name }} &lt;{{ author.email }}&gt;</span>
        <a-tag :color="author.polluted === true ? 'red' : author.polluted === false ? 'green' : 'default'">{{ author.count }} 次提交<template v-if="author.polluted"> · {{ author.polluted ? '污染' : '一致' }}</template></a-tag>
      </div>
      <p v-if="audit.mailmapSuggestion" class="muted">mailmap 建议（不写入）：{{ audit.mailmapSuggestion }}</p>
    </div>
    <div v-if="doctor" class="apply-row">
      <div><strong>体检</strong><a-tag :color="doctor.hookPresent ? 'green' : 'orange'">include 钩子 {{ doctor.hookPresent ? '就位' : '缺失' }}</a-tag></div>
      <div v-for="(finding, index) in doctor.findings" :key="index" class="scan-row">
        <a-tag :color="finding.level === 'error' ? 'red' : finding.level === 'warn' ? 'orange' : 'blue'">{{ finding.level }}</a-tag>
        <small>{{ finding.check }}：{{ finding.detail }}</small>
      </div>
    </div>
  </section>

  <section class="section">
    <div class="section-title"><div><h2>迁移与凭据</h2><p>gitid 旧档案只读探测、备份后幂等导入；token 经 helper 直通，界面不显示任何密钥。</p></div></div>
    <a-space wrap>
      <a-button :loading="busy" @click="previewLegacy">探测 gitid 档案</a-button>
    </a-space>
    <div v-if="legacy" class="apply-row">
      <template v-if="legacy.found && legacy.valid">
        <div><strong>{{ legacy.sourcePath }}</strong><a-tag>schema 有效</a-tag></div>
        <small>可导入 {{ legacy.profiles.length }} 个；冲突 {{ legacy.idConflicts.length }}；重复邮箱 {{ legacy.duplicateEmails.length }}；备份目录 {{ legacy.backupDirectory }}</small>
        <div class="form-actions"><a-button type="primary" :loading="busy" @click="importLegacy">备份并导入</a-button></div>
      </template>
      <a-empty v-else-if="!legacy.found" description="未发现 gitid 档案" />
      <a-alert v-else type="warning" message="来源存在但格式无效，保留原文件并先处理。" show-icon />
    </div>
    <div v-if="legacyRecord" class="apply-row"><small>导入记录：新建 {{ legacyRecord.created }} · 跳过 {{ legacyRecord.skipped }} · 冲突 {{ legacyRecord.conflicts }} · 备份 {{ legacyRecord.backupPath }}</small></div>
    <a-divider />
    <a-space wrap>
      <a-button :loading="busy" @click="reloadCredentials">刷新凭据账号</a-button>
      <a-input v-model:value="credHost" placeholder="host" style="width: 180px" />
      <a-input v-model:value="credUsername" placeholder="用户名" style="width: 160px" />
      <a-input-password v-model:value="credToken" placeholder="token（提交后不保留）" style="width: 220px" />
      <a-button :loading="busy" @click="credentialSet">写入 helper</a-button>
    </a-space>
    <div v-for="entry in credentials" :key="entry.host + entry.username" class="scan-row">
      <code>{{ entry.host }}</code><span>{{ entry.username }}</span>
      <a-button size="small" @click="credentialLogin(entry.host)">helper 登录</a-button>
      <a-button size="small" danger @click="credentialRemove(entry.host, entry.username)">擦除</a-button>
    </div>
    <a-divider />
    <a-space wrap>
      <a-input v-model:value="mirrorRepo" placeholder="仓库路径" style="width: 240px" />
      <a-input v-model:value="mirrorRemote" placeholder="remote 名" style="width: 120px" />
      <a-input v-model:value="mirrorUrl" placeholder="镜像 URL" style="width: 280px" />
      <a-button :loading="busy" @click="mirrorSet">开启镜像推送</a-button>
      <a-button :loading="busy" @click="mirrorUnset">取消镜像推送</a-button>
    </a-space>
  </section>
  </template>
</template>

<style scoped>
.profile-row{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;padding:12px 0;border-top:1px solid var(--border-soft)}
.profile-row strong{display:block}
.profile-row small,.apply-row small{display:block;margin-top:4px;color:var(--text-muted);overflow-wrap:anywhere}
.profile-form{margin-top:8px}
.apply-row{padding:10px 0;border-top:1px solid var(--border-soft)}
.apply-row strong{display:block}
.change-chip{display:inline-block;margin:3px 8px 3px 0;padding:2px 8px;border-radius:6px;background:var(--bg-soft);color:var(--text-code);font-size:12px;overflow-wrap:anywhere}
.scan-row{display:flex;align-items:center;gap:10px;padding:4px 0;flex-wrap:wrap}
</style>
