<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type {
  GitApplyTargetResult, GitAuditReport, GitConfigChange, GitCredentialEntry, GitDoctorBinding,
  GitDoctorReport, GitEffectiveIdentity, GitGlobalApplyResult, GitImportRecord, GitLegacyPreview,
  GitProfile, GitScanResult,
} from './types'
import ProfileEditor from './ProfileEditor.vue'
import ChangeTable from './ChangeTable.vue'

const profiles = ref<GitProfile[]>([])
const globalId = ref('')
const booting = ref(true)
const tab = ref('profiles')
// 正在执行的动作；同一时刻只允许一个请求，其余按钮置灰而不是一起转圈。
const pending = ref('')
const errorText = ref('')
const busy = computed(() => !!pending.value)

const editorOpen = ref(false)
const editingProfile = ref<GitProfile>()
const editorError = ref('')

// 生效范围：全局默认 / 目录规则 / 单个仓库
const changingGlobal = ref(false)
const globalPick = ref('')
const globalPreview = ref<GitConfigChange[]>()
const bindingHealth = ref<GitDoctorBinding[]>()
const newRuleDir = ref('')
const newRuleProfile = ref('')
const queryPath = ref('')
const effective = ref<GitEffectiveIdentity>()
const repoApplyId = ref('')
const repoPaths = ref<string[]>([])
const repoPreview = ref<GitApplyTargetResult[]>()
// 预览与实际应用的结果用同一块区域展示；执行过应用后不再出现「确认应用」。
const repoApplied = ref(false)

const scanRoot = ref('')
const scanDepth = ref(4)
const scan = ref<GitScanResult>()
const auditRepo = ref('')
// 空串会被下拉当成已选值而不显示占位文字；未选时保持 undefined。
const auditExpect = ref<string>()
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

const sourceLabel: Record<GitEffectiveIdentity['keys'][number]['source'], string> = {
  local: '仓库设置', includeIf: '目录规则', global: '全局默认', none: '未设置',
}
const applyStatusLabel: Record<GitApplyTargetResult['status'], string> = {
  skipped: '可以应用', failed: '无法应用', success: '已应用',
}

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

// 服务端错误是「错误码: 英文说明」。能落到具体做法的几条翻成人话，其余原样显示。
function friendlyError(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error)
  const bound = raw.match(/GIT_DIRECTORY_BOUND:.*bound to (\S+)/)
  if (bound) return `这个目录已经绑定了「${bound[1]}」。要换身份，先解绑它。`
  if (raw.includes('GIT_NOT_A_REPOSITORY')) return '这个路径不是 Git 仓库'
  if (raw.includes('GIT_REPO_NOT_FOUND')) return '找不到这个仓库，请确认路径'
  if (raw.includes('GIT_PROFILE_NOT_FOUND')) return '找不到这个档案，请重新选择'
  if (raw.includes('GIT_DIR_INVALID')) return '请先填写要绑定的目录'
  if (raw.includes('CONFIRMATION_REQUIRED')) return '这个操作需要确认后才能执行'
  return raw
}

async function guard(key: string, action: () => Promise<void>) {
  if (pending.value) return
  pending.value = key
  errorText.value = ''
  try {
    await action()
  } catch (error) {
    errorText.value = friendlyError(error)
  } finally {
    pending.value = ''
  }
}

const profileOptions = computed(() => profiles.value.map((profile) => ({ label: profile.id, value: profile.id })))

// 这个身份现在管哪些地方：全局默认，以及绑定到的目录树。
function scopeOf(profile: GitProfile): string[] {
  const scopes: string[] = []
  if (profile.id === globalId.value) scopes.push('所有仓库的默认身份')
  for (const dir of profile.gitdirs ?? []) scopes.push(`${dir} 及其子目录`)
  return scopes
}

function openEditor(profile?: GitProfile) {
  editingProfile.value = profile
  editorError.value = ''
  editorOpen.value = true
}

const saveProfile = (profile: GitProfile) => guard('save', async () => {
  try {
    await invoke('git.saveProfile', { profile, confirm: true })
  } catch (error) {
    // 抽屉遮住了页面顶部的错误条：保存失败要回显在抽屉里。
    editorError.value = friendlyError(error)
    return
  }
  editorOpen.value = false
  message.success('档案已保存')
  await reloadProfiles()
})

function removeProfile(profile: GitProfile) {
  const ruleCount = profile.gitdirs?.length ?? 0
  Modal.confirm({
    title: `删除档案「${profile.id}」？`,
    content: ruleCount
      ? `它还绑定着 ${ruleCount} 个目录，删除时这些目录规则和对应的配置片段会一并移除。已经写进 git 的其它配置不会被改动。`
      : '只删除档案本身；已经写进 git 的配置不会被改动。',
    okText: '删除',
    okButtonProps: { danger: true },
    cancelText: '取消',
    zIndex: 1100,
    onOk: () => guard('remove', async () => {
      await invoke('git.removeProfile', { id: profile.id, confirm: true })
      message.success('档案已删除')
      await reloadProfiles()
      await loadBindingHealth()
    }),
  })
}

const importGlobal = () => guard('importGlobal', async () => {
  await invoke('git.importGlobalIdentity', { asId: '', confirm: true })
  message.success('已收编当前全局身份')
  await reloadProfiles()
})

// ---- 生效范围 ----

const globalProfile = computed(() => profiles.value.find((profile) => profile.id === globalId.value))

type DirectoryRule = { profileId: string; dir: string; health?: GitDoctorBinding }

const normalizeDir = (dir: string) => dir.replace(/[\\/]+$/, '').toLowerCase()
const directoryRules = computed<DirectoryRule[]>(() => profiles.value
  .flatMap((profile) => (profile.gitdirs ?? []).map((dir) => ({
    profileId: profile.id,
    dir,
    health: bindingHealth.value?.find((item) => item.profileId === profile.id && normalizeDir(item.dir) === normalizeDir(dir)),
  })))
  .sort((a, b) => a.dir.localeCompare(b.dir)))

function ruleWarning(rule: DirectoryRule): { label: string; detail: string } | undefined {
  const health = rule.health
  if (!health) return undefined
  if (!health.dirExists) return { label: '目录不存在', detail: '这个目录现在不在磁盘上。规则会保留，目录重新出现后继续生效。' }
  if (health.drift?.drift) return { label: '配置被手改', detail: `这个目录的身份配置片段被手动改过（${(health.drift.keys ?? []).join('、') || '内容与档案不一致'}），实际生效的可能不是「${rule.profileId}」。` }
  if (!health.snippetOk) return { label: '配置片段缺失', detail: '全局配置里指向这个目录的片段丢了，规则目前不生效。可以解绑后重新绑定。' }
  return undefined
}

// 绑定健康来自只读体检；读不到时规则表照常显示，只是不带状态标记。
// 结果只给规则表用，不写进「Git 体检」的展示状态——那里的结果应由用户点「开始体检」得出。
async function loadBindingHealth() {
  try {
    const report = await invoke<GitDoctorReport>('git.gitDoctor', {})
    bindingHealth.value = report.bindings ?? []
  } catch { /* 健康标记是补充信息，失败不阻塞 */ }
}

watch(tab, (value) => {
  if (value === 'scope' && !bindingHealth.value) void loadBindingHealth()
})

function startGlobalChange() {
  globalPick.value = ''
  globalPreview.value = undefined
  changingGlobal.value = true
}

const previewGlobalChange = () => guard('globalPreview', async () => {
  const result = await invoke<GitGlobalApplyResult>('git.applyProfile', { id: globalPick.value, scope: 'global', repos: [], dryRun: true })
  globalPreview.value = result.changes ?? []
})

function confirmGlobalChange() {
  Modal.confirm({
    title: `把全局默认换成「${globalPick.value}」？`,
    content: '这会改写全局 Git 配置。按预览显示的内容写入，档案里没有的签名、URL 重写和账号选择器会被清除。',
    okText: '确认更换',
    cancelText: '取消',
    zIndex: 1100,
    onOk: () => guard('globalApply', async () => {
      await invoke('git.applyProfile', { id: globalPick.value, scope: 'global', repos: [], dryRun: false, confirm: true })
      message.success(`全局默认已更换为「${globalPick.value}」`)
      changingGlobal.value = false
      globalPreview.value = undefined
      await reloadProfiles()
    }),
  })
}

const canBind = computed(() => !!newRuleDir.value.trim() && !!newRuleProfile.value)

const bindDirectory = () => guard('bind', async () => {
  await invoke('git.bindProfile', { id: newRuleProfile.value, dir: newRuleDir.value.trim(), confirm: true })
  message.success(`已绑定「${newRuleDir.value.trim()}」，它下面的仓库将使用「${newRuleProfile.value}」`)
  newRuleDir.value = ''
  await reloadProfiles()
  await loadBindingHealth()
})

function unbindDirectory(rule: DirectoryRule) {
  const warning = ruleWarning(rule)
  Modal.confirm({
    title: `解除「${rule.dir}」的目录规则？`,
    content: warning?.label === '配置被手改'
      ? '这个目录的配置片段被手动改过，解绑会把它回收。之后它下面的仓库改用全局默认；仓库里已有的本地配置不受影响。'
      : `解绑后，这个目录下的仓库不再自动使用「${rule.profileId}」，会回落到全局默认。仓库里已有的本地配置不受影响。`,
    okText: '解绑',
    okButtonProps: { danger: true },
    cancelText: '取消',
    zIndex: 1100,
    onOk: () => guard('unbind', async () => {
      await invoke('git.unbindProfile', { id: rule.profileId, dir: rule.dir, confirm: true })
      message.success(`已解绑「${rule.dir}」`)
      await reloadProfiles()
      await loadBindingHealth()
    }),
  })
}

async function browse(target: 'rule' | 'query' | 'repo') {
  try {
    const path = await invoke<string>('app.pickFolder', {})
    if (!path) return
    if (target === 'rule') newRuleDir.value = path
    else if (target === 'query') queryPath.value = path
    else if (!repoPaths.value.includes(path)) repoPaths.value = [...repoPaths.value, path]
  } catch (error) {
    const text = error instanceof Error ? error.message : String(error)
    if (!text.includes('CANCELLED')) errorText.value = text
  }
}

const lookupRepository = () => guard('lookup', async () => {
  effective.value = await invoke<GitEffectiveIdentity>('git.effectiveIdentity', { path: queryPath.value.trim() })
})

const effectiveTitle = computed(() => {
  const result = effective.value
  if (!result) return ''
  if (result.resolved === 'none') return '这个位置没有配置身份'
  if (result.resolved === 'manual') return '用的是仓库里手动配置的身份，不在任何档案里'
  return `用的是「${result.profileId || result.resolved}」`
})

const effectiveNote = computed(() => {
  const result = effective.value
  if (!result) return ''
  if (result.resolved === 'none') return '在这里提交之前，需要先给它配置一个身份。'
  const name = result.keys.find((item) => item.key === 'user.name')?.value
  const email = result.keys.find((item) => item.key === 'user.email')?.value
  const who = [name, email ? `<${email}>` : ''].filter(Boolean).join(' ')
  const origin = sourceLabel[result.keys.find((item) => item.key === 'user.name')?.source ?? 'none']
  if (result.resolved === 'manual') return who
  return [who, origin === '未设置' ? '' : `来自${origin}`].filter(Boolean).join('，')
})

const canPreviewRepo = computed(() => !!repoApplyId.value && repoPaths.value.some((path) => path.trim()))

const previewRepoApply = () => guard('repoPreview', async () => {
  repoPreview.value = await invoke<GitApplyTargetResult[]>('git.applyProfile', {
    id: repoApplyId.value, scope: 'repos', repos: repoPaths.value.map((path) => path.trim()).filter(Boolean), dryRun: true,
  })
})

// 预检里有无法应用的仓库时不允许执行，先把原因处理掉。
const canConfirmRepo = computed(() => !!repoPreview.value?.length && repoPreview.value.every((item) => item.status !== 'failed'))

function confirmRepoApply() {
  Modal.confirm({
    title: `把「${repoApplyId.value}」写进这 ${repoPreview.value?.length ?? 0} 个仓库的设置？`,
    content: '按预览显示的内容写入每个仓库自己的配置，会覆盖它们已有的身份配置。全局默认和目录规则不变。',
    okText: '确认应用',
    cancelText: '取消',
    zIndex: 1100,
    onOk: () => guard('repoApply', async () => {
      const results = await invoke<GitApplyTargetResult[]>('git.applyProfile', {
        id: repoApplyId.value, scope: 'repos', repos: repoPaths.value.map((path) => path.trim()).filter(Boolean), dryRun: false, confirm: true,
      })
      repoPreview.value = results
      repoApplied.value = true
      const failed = results.filter((item) => item.status === 'failed').length
      if (failed) message.warning(`已应用 ${results.length - failed} 个仓库，${failed} 个未能应用`)
      else message.success(`已应用到 ${results.length} 个仓库`)
      await reloadProfiles()
    }),
  })
}

function resetRepoApply() {
  repoPreview.value = undefined
  repoApplied.value = false
}

// ---- 仓库与体检 / 迁移与凭据 ----

const runScan = () => guard('scan', async () => {
  scan.value = await invoke<GitScanResult>('git.scanRepos', { root: scanRoot.value, depth: scanDepth.value })
})

const adoptScanned = (path: string) => guard('adopt', async () => {
  await invoke('git.importScanFindings', { root: scan.value?.root ?? '', depth: scanDepth, paths: [path], dryRun: false, confirm: true })
  message.success('已收编手配身份')
  await reloadProfiles()
})

const runAudit = () => guard('audit', async () => {
  audit.value = await invoke<GitAuditReport>('git.auditRepo', { repo: auditRepo.value || '.', expectedProfileId: auditExpect.value ?? '', strict: auditStrict.value })
})

const runDoctor = () => guard('doctor', async () => {
  doctor.value = await invoke<GitDoctorReport>('git.gitDoctor', {})
})

const previewLegacy = () => guard('legacyPreview', async () => {
  legacy.value = await invoke<GitLegacyPreview>('git.legacyPreview', {})
})

const importLegacy = () => guard('legacyImport', async () => {
  const snapshot = legacy.value
  if (!snapshot?.sourceDigest) return
  legacyRecord.value = await invoke<GitImportRecord>('git.legacyImport', { sourceDigest: snapshot.sourceDigest, profileIds: [], confirmBackup: true, confirmImport: true })
  message.success('gitid 档案导入完成')
  await reloadProfiles()
})

const reloadCredentials = () => guard('credentials', async () => {
  credentials.value = await invoke<GitCredentialEntry[]>('git.credentialList', { host: '' })
})

const credentialLogin = (host: string) => guard('credentialLogin', async () => {
  await invoke('git.credentialLogin', { host })
  message.success(`helper 登录完成：${host}`)
  await reloadCredentials()
})

const credentialSet = () => guard('credentialSet', async () => {
  await invoke('git.credentialSet', { host: credHost.value, username: credUsername.value, token: credToken.value, confirm: true })
  credToken.value = ''
  message.success('凭据已写入 helper')
  await reloadCredentials()
})

const credentialRemove = (host: string, username?: string) => guard('credentialRemove', async () => {
  await invoke('git.credentialRemove', { host, username, confirm: true })
  message.success('凭据已请求 helper 擦除')
  await reloadCredentials()
})

const mirrorSet = () => guard('mirrorSet', async () => {
  await invoke('git.mirrorSet', { repo: mirrorRepo.value, remote: mirrorRemote.value, mirrorUrl: mirrorUrl.value, confirm: true })
  message.success('镜像推送已配置')
})

const mirrorUnset = () => guard('mirrorUnset', async () => {
  await invoke('git.mirrorUnset', { repo: mirrorRepo.value, remote: mirrorRemote.value, confirm: true })
  message.success('镜像推送已取消')
})

onMounted(() => guard('bootstrap', async () => {
  await bootstrap()
  booting.value = false
}))
</script>

<template>
  <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" @close="errorText = ''" />
  <div v-if="booting" class="section boot-panel">
    <a-skeleton active :paragraph="{ rows: 4 }" />
  </div>
  <a-tabs v-else v-model:activeKey="tab" class="git-tabs">

    <a-tab-pane key="profiles" tab="身份档案">
      <div class="group-actions">
        <a-button :disabled="busy" @click="importGlobal">收编当前全局身份</a-button>
        <a-button type="primary" @click="openEditor()">新建档案</a-button>
      </div>
      <a-empty v-if="!profiles.length" description="还没有身份档案。点「收编当前全局身份」把现在 git 里的全局身份存成档案，或者「新建档案」。" />
      <div v-for="profile in profiles" :key="profile.id" class="identity-row">
        <span class="avatar" aria-hidden="true">{{ profile.name.trim().charAt(0).toUpperCase() || '?' }}</span>
        <div class="identity-main">
          <div class="identity-title">
            <strong>{{ profile.id }}</strong>
            <a-tag v-if="profile.id === globalId" class="global-mark">全局默认</a-tag>
          </div>
          <div class="identity-contact">{{ profile.name }} &lt;{{ profile.email }}&gt;</div>
          <div v-if="profile.signingKey || profile.insteadOf?.length || profile.accounts?.length" class="identity-meta">
            <span v-if="profile.signingKey">{{ profile.gpgSign ? '提交签名' : '签名密钥（未启用）' }}</span>
            <span v-if="profile.insteadOf?.length">URL 重写 {{ profile.insteadOf.length }} 条</span>
            <span v-if="profile.accounts?.length">账号 {{ profile.accounts.map((item) => item.host).join('、') }}</span>
          </div>
          <div class="scope-line">
            <template v-if="scopeOf(profile).length">
              <span>生效范围</span>
              <span v-for="scope in scopeOf(profile)" :key="scope" class="scope-chip">{{ scope }}</span>
            </template>
            <span v-else>还没用在任何地方。可以设为全局默认，或绑定一个目录。</span>
          </div>
        </div>
        <div class="identity-actions">
          <a-button size="small" @click="openEditor(profile)">编辑</a-button>
          <a-button size="small" danger @click="removeProfile(profile)">删除</a-button>
        </div>
      </div>
      <ProfileEditor v-model:open="editorOpen" :profile="editingProfile" :saving="pending === 'save'" :error="editorError" @save="saveProfile" />
    </a-tab-pane>

    <a-tab-pane key="scope" tab="生效范围">
      <p class="scope-intro">身份按范围从大到小排列：全局默认管所有仓库，目录规则覆盖某个目录下的仓库，仓库自己的设置优先级最高。</p>

      <section class="layer">
        <div class="layer-head"><h3>全局默认</h3></div>
        <p class="layer-hint">没被目录规则或仓库自己的设置覆盖时，所有仓库都用这个身份。</p>
        <div class="global-card">
          <span class="avatar" aria-hidden="true">{{ globalProfile?.name.trim().charAt(0).toUpperCase() || '?' }}</span>
          <div class="global-main">
            <strong>{{ globalProfile?.id ?? '全局身份不在任何档案里' }}</strong>
            <small v-if="globalProfile">{{ globalProfile.name }} &lt;{{ globalProfile.email }}&gt;</small>
            <small v-else>现在 git 里的全局姓名和邮箱不匹配任何档案。到「身份档案」点「收编当前全局身份」可以把它存成档案。</small>
          </div>
          <a-button size="small" :disabled="busy" @click="startGlobalChange">更换…</a-button>
        </div>
        <div v-if="changingGlobal" class="result-panel">
          <div class="inline-form">
            <span class="inline-label">换成</span>
            <a-select v-model:value="globalPick" :options="profileOptions" placeholder="选择档案" aria-label="新的全局默认档案" class="pick" @change="globalPreview = undefined" />
            <a-button :loading="pending === 'globalPreview'" :disabled="!globalPick" @click="previewGlobalChange">预览变更</a-button>
            <a-button type="text" @click="changingGlobal = false">取消</a-button>
          </div>
          <template v-if="globalPreview">
            <p class="layer-hint">会改写全局 Git 配置。档案里没有的签名、URL 重写和账号选择器会被清除。</p>
            <ChangeTable v-if="globalPreview.length" :changes="globalPreview" />
            <p v-else class="layer-hint">全局配置已经和「{{ globalPick }}」一致，不需要更换。</p>
            <div class="form-actions">
              <a-button type="primary" :disabled="!globalPreview.length" :loading="pending === 'globalApply'" @click="confirmGlobalChange">确认更换</a-button>
            </div>
          </template>
        </div>
      </section>

      <section class="layer nested">
        <div class="layer-head"><h3>目录规则</h3></div>
        <p class="layer-hint">{{ directoryRules.length ? '绑定一个目录后，它下面的所有仓库（包括以后克隆的）自动用指定身份。' : '还没有目录规则。绑定一个目录后，它下面的所有仓库（包括以后克隆的）自动用指定身份。' }}</p>
        <div v-if="directoryRules.length" class="rules">
          <div v-for="rule in directoryRules" :key="rule.profileId + rule.dir" class="rule-row">
            <code class="rule-dir">{{ rule.dir }}</code>
            <div class="rule-target">
              <span class="rule-verb">用</span>
              <span class="rule-profile">{{ rule.profileId }}</span>
              <a-tooltip v-if="ruleWarning(rule)" :title="ruleWarning(rule)?.detail">
                <a-tag color="warning">{{ ruleWarning(rule)?.label }}</a-tag>
              </a-tooltip>
              <a-button type="text" size="small" danger class="rule-action" :disabled="busy" @click="unbindDirectory(rule)">解绑</a-button>
            </div>
          </div>
        </div>
        <div class="inline-form rule-add">
          <a-input v-model:value="newRuleDir" class="grow" placeholder="目录路径" aria-label="要绑定的目录路径" @press-enter="bindDirectory" />
          <a-button :disabled="busy" @click="browse('rule')">浏览…</a-button>
          <span class="rule-verb">用</span>
          <a-select v-model:value="newRuleProfile" :options="profileOptions" placeholder="选择档案" aria-label="目录规则使用的档案" class="pick" />
          <a-button :type="directoryRules.length ? 'default' : 'primary'" :loading="pending === 'bind'" :disabled="!canBind" @click="bindDirectory">绑定目录</a-button>
        </div>
      </section>

      <section class="layer nested deeper">
        <div class="layer-head"><h3>查某个仓库用的是谁</h3></div>
        <p class="layer-hint">填一个仓库路径，看它现在实际用的身份，以及每一项是哪一层给的。</p>
        <div class="inline-form">
          <a-input v-model:value="queryPath" class="grow" placeholder="仓库路径" aria-label="要查询的仓库路径" @press-enter="lookupRepository" />
          <a-button :disabled="busy" @click="browse('query')">浏览…</a-button>
          <a-button type="primary" :loading="pending === 'lookup'" :disabled="!queryPath.trim()" @click="lookupRepository">查看</a-button>
        </div>
        <div v-if="effective" class="result-panel">
          <div class="result-head">
            <strong>{{ effectiveTitle }}</strong>
            <span>{{ effectiveNote }}</span>
          </div>
          <div class="origin-scroll">
            <table class="origin-table">
              <thead><tr><th scope="col">配置项</th><th scope="col">生效值</th><th scope="col">来自</th></tr></thead>
              <tbody>
                <tr v-for="key in effective.keys" :key="key.key">
                  <td><code>{{ key.key }}</code></td>
                  <td :class="{ unset: !key.value }">{{ key.value || '未设置' }}</td>
                  <td><a-tag v-if="key.source !== 'none'" class="origin-tag">{{ sourceLabel[key.source] }}</a-tag><span v-else class="unset">—</span></td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-if="!effective.keys.length" class="layer-hint">没有读到任何身份配置项。</p>
        </div>
      </section>

      <section class="layer nested deeper">
        <div class="layer-head"><h3>把档案写进仓库设置</h3></div>
        <p class="layer-hint">很少需要，大多数情况用目录规则就够了。它只影响列出的仓库，优先级最高，会覆盖这些仓库里已有的身份配置。</p>
        <div class="inline-form">
          <a-select v-model:value="repoApplyId" :options="profileOptions" placeholder="选择档案" aria-label="要写入仓库的档案" class="pick" @change="resetRepoApply" />
          <a-select v-model:value="repoPaths" mode="tags" :open="false" :token-separators="[',', '\n']" class="grow" placeholder="仓库路径，可以填多个" aria-label="仓库路径，可以填多个" @change="resetRepoApply" />
          <a-button :disabled="busy" @click="browse('repo')">浏览…</a-button>
          <a-button :loading="pending === 'repoPreview'" :disabled="!canPreviewRepo" @click="previewRepoApply">预览变更</a-button>
        </div>
        <template v-if="repoPreview">
          <a-alert v-if="!repoPreview.length" type="info" show-icon message="没有可预览的仓库：请先填写至少一个仓库路径。" class="result-panel" />
          <div v-for="result in repoPreview" :key="result.target" class="result-panel">
            <div class="result-head">
              <code>{{ result.target }}</code>
              <a-tag :color="result.status === 'failed' ? 'red' : result.status === 'success' ? 'green' : undefined">{{ applyStatusLabel[result.status] }}</a-tag>
            </div>
            <a-alert v-if="result.error" type="error" show-icon :message="friendlyError(new Error(result.error))" />
            <ChangeTable v-else :changes="result.changes ?? []" />
          </div>
          <div v-if="repoPreview.length" class="form-actions">
            <a-button @click="resetRepoApply">{{ repoApplied ? '关闭' : '取消' }}</a-button>
            <a-button v-if="!repoApplied" type="primary" :disabled="!canConfirmRepo" @click="confirmRepoApply">确认应用</a-button>
          </div>
        </template>
      </section>
    </a-tab-pane>

    <a-tab-pane key="audit" tab="仓库与体检">
      <div class="group-label">扫描目录下的仓库</div>
      <div class="toolbar">
        <a-input v-model:value="scanRoot" class="grow" placeholder="扫描根目录" />
        <a-input-number v-model:value="scanDepth" :min="1" :max="12" style="width: 90px" />
        <a-button :loading="busy" @click="runScan">扫描</a-button>
      </div>
      <div v-if="scan" class="apply-row">
        <div><strong>扫描 {{ scan.summary.total }} 个仓库</strong><small>本地覆盖 {{ scan.summary.withLocal }}，绑定生效 {{ scan.summary.withIncludeIf }}，缺失 {{ scan.summary.missing }}，手配 {{ scan.summary.manual }}，镜像 {{ scan.summary.mirrored }}</small></div>
        <div v-for="row in scan.rows" :key="row.path" class="scan-row">
          <code>{{ row.relative }}</code>
          <a-tag :color="row.status === 'ok' ? 'green' : row.status === 'missing' ? 'red' : 'orange'">{{ row.status === 'ok' ? '一致' : row.status === 'missing' ? '缺失身份' : '手配身份' }}</a-tag>
          <small>{{ row.source }}<template v-if="row.identityId">，{{ row.identityId }}</template><template v-if="row.remotes.some((r) => r.mirror)">，镜像推送</template></small>
          <a-button v-if="row.status === 'manual'" size="small" @click="adoptScanned(row.path)">收编为档案</a-button>
        </div>
      </div>
      <a-divider />
      <div class="group-label">提交审计</div>
      <div class="toolbar">
        <a-input v-model:value="auditRepo" class="grow" placeholder="仓库路径" />
        <a-select v-model:value="auditExpect" allow-clear placeholder="期望档案" style="width: 160px" :options="profileOptions" />
        <a-checkbox v-model:checked="auditStrict">strict 门禁</a-checkbox>
        <a-button :loading="busy" @click="runAudit">审计</a-button>
      </div>
      <div v-if="audit" class="apply-row">
        <div><strong>{{ audit.repo }}</strong><a-tag v-if="audit.strictPassed === false" color="red">strict 未通过</a-tag><a-tag v-else-if="audit.strictPassed" color="green">strict 通过</a-tag></div>
        <div v-for="author in audit.authors" :key="author.email + author.name" class="scan-row">
          <span>{{ author.name }} &lt;{{ author.email }}&gt;</span>
          <a-tag :color="author.polluted === true ? 'red' : author.polluted === false ? 'green' : 'default'">{{ author.count }} 次提交<template v-if="author.polluted">，{{ author.polluted ? '污染' : '一致' }}</template></a-tag>
        </div>
        <p v-if="audit.mailmapSuggestion" class="muted">mailmap 建议（不会写入）：{{ audit.mailmapSuggestion }}</p>
      </div>
      <a-divider />
      <div class="group-label">Git 体检</div>
      <div class="toolbar"><a-button :loading="busy" @click="runDoctor">开始体检</a-button></div>
      <div v-if="doctor" class="apply-row">
        <div><a-tag :color="doctor.hookPresent ? 'green' : 'orange'">include 钩子 {{ doctor.hookPresent ? '就位' : '缺失' }}</a-tag></div>
        <div v-for="(finding, index) in doctor.findings" :key="index" class="scan-row">
          <a-tag :color="finding.level === 'error' ? 'red' : finding.level === 'warn' ? 'orange' : 'blue'">{{ finding.level }}</a-tag>
          <small>{{ finding.check }}：{{ finding.detail }}</small>
        </div>
      </div>
    </a-tab-pane>

    <a-tab-pane key="settings" tab="迁移与凭据">
      <div class="group-label">从 gitid 导入旧档案</div>
      <div class="toolbar"><a-button :loading="busy" @click="previewLegacy">探测 gitid 档案</a-button></div>
      <div v-if="legacy" class="apply-row">
        <template v-if="legacy.found && legacy.valid">
          <div><strong>{{ legacy.sourcePath }}</strong><a-tag>schema 有效</a-tag></div>
          <small>可导入 {{ legacy.profiles.length }} 个，冲突 {{ legacy.idConflicts.length }}，重复邮箱 {{ legacy.duplicateEmails.length }}，备份目录 {{ legacy.backupDirectory }}</small>
          <div class="form-actions"><a-button type="primary" :loading="busy" @click="importLegacy">备份并导入</a-button></div>
        </template>
        <a-empty v-else-if="!legacy.found" description="未发现 gitid 档案" />
        <a-alert v-else type="warning" message="来源存在但格式无效，原文件保持不动。" show-icon />
      </div>
      <div v-if="legacyRecord" class="apply-row"><small>导入记录：新建 {{ legacyRecord.created }}，跳过 {{ legacyRecord.skipped }}，冲突 {{ legacyRecord.conflicts }}，备份 {{ legacyRecord.backupPath }}</small></div>
      <a-divider />
      <div class="group-label">凭据账号</div>
      <div class="toolbar">
        <a-input v-model:value="credHost" placeholder="host（如 github.com）" style="width: 200px" />
        <a-input v-model:value="credUsername" placeholder="用户名" style="width: 160px" />
        <a-input-password v-model:value="credToken" placeholder="token（提交后不保留）" style="width: 220px" />
        <a-button :loading="busy" @click="credentialSet">写入 helper</a-button>
        <a-button :loading="busy" @click="reloadCredentials">刷新</a-button>
      </div>
      <div v-for="entry in credentials" :key="entry.host + entry.username" class="scan-row">
        <code>{{ entry.host }}</code><span>{{ entry.username }}</span>
        <a-button size="small" @click="credentialLogin(entry.host)">helper 登录</a-button>
        <a-button size="small" danger @click="credentialRemove(entry.host, entry.username)">擦除</a-button>
      </div>
      <a-divider />
      <div class="group-label">镜像推送（当前仓库）</div>
      <div class="toolbar">
        <a-input v-model:value="mirrorRepo" placeholder="仓库路径" style="width: 240px" />
        <a-input v-model:value="mirrorRemote" placeholder="remote 名" style="width: 120px" />
        <a-input v-model:value="mirrorUrl" placeholder="镜像 URL" style="width: 280px" />
        <a-button :loading="busy" @click="mirrorSet">开启</a-button>
        <a-button :loading="busy" @click="mirrorUnset">取消</a-button>
      </div>
    </a-tab-pane>
  </a-tabs>
</template>

<style scoped>
.group-label{font-weight:600;font-size:13.5px;color:var(--text-title);margin:0 0 10px}
.group-actions{display:flex;justify-content:flex-end;gap:9px;margin-bottom:6px}
.git-tabs :deep(.ant-tabs-tab){padding:8px 4px}

/* 身份档案列表 */
.identity-row{display:flex;align-items:flex-start;gap:14px;padding:14px 0;border-top:1px solid var(--border-soft)}
.avatar{flex:none;display:grid;place-items:center;width:38px;height:38px;border-radius:9px;background:var(--accent-soft);color:var(--accent-strong);font-weight:700;font-size:16px}
.identity-main{flex:1;min-width:0}
.identity-title{display:flex;flex-wrap:wrap;align-items:center;gap:4px 8px;color:var(--text-strong)}
.identity-title strong{font-size:14.5px;overflow-wrap:anywhere}
.identity-title .ant-tag{margin:0}
.global-mark{background:var(--accent);color:#fff;border-color:var(--accent)}
.identity-contact{margin-top:3px;font-size:13px;overflow-wrap:anywhere}
.identity-meta{display:flex;flex-wrap:wrap;gap:4px 16px;margin-top:5px;color:var(--text-muted);font-size:12.5px}
.scope-line{display:flex;flex-wrap:wrap;align-items:center;gap:6px;margin-top:9px;color:var(--text-muted);font-size:12.5px}
.scope-chip{max-width:100%;padding:2px 8px;border-radius:6px;background:var(--bg-chip);color:var(--text-code);font-size:12px;overflow-wrap:anywhere}
.identity-actions{display:flex;flex:none;gap:8px}

/* 生效范围 */
.scope-intro{margin:0 0 4px;color:var(--text-muted);font-size:12.5px;line-height:1.6}
.layer{padding:18px 0 20px;border-top:1px solid var(--border-soft)}
.scope-intro+.layer{padding-top:14px;border-top:0}
.layer.nested{margin-left:9px;padding:4px 0 20px 18px;border:0;border-left:2px solid var(--accent)}
/* 仓库层比目录规则再深一级：缩进深度对应「范围更小、优先级更高」。 */
.layer.nested.deeper{margin-left:36px}
.layer-head{margin-bottom:4px}
.layer-head h3{margin:0;color:var(--text-title);font:600 14.5px 'Segoe UI'}
.layer-hint{margin:0 0 12px;color:var(--text-muted);font-size:12.5px;line-height:1.6}
.inline-form{display:flex;flex-wrap:wrap;align-items:center;gap:10px}
.inline-form .grow{flex:1;min-width:220px}
.inline-form .pick{width:200px}
.inline-label{color:var(--text-muted);font-size:12.5px}
.global-card{display:flex;align-items:center;gap:14px;padding:12px 16px;border-radius:8px;background:var(--bg-soft)}
.global-main{flex:1;min-width:0}
.global-main strong{display:block;color:var(--text-strong);overflow-wrap:anywhere}
.global-main small{display:block;margin-top:3px;color:var(--text-muted);font-size:12.5px;overflow-wrap:anywhere}
.rules{border-bottom:1px solid var(--border-soft)}
.rule-row{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.4fr);gap:4px 16px;align-items:center;padding:9px 0;border-top:1px solid var(--border-soft)}
.rule-dir{color:var(--text-code);font:12.5px ui-monospace,Consolas,'Cascadia Mono',monospace;overflow-wrap:anywhere}
.rule-target{display:flex;flex-wrap:wrap;align-items:center;gap:4px 8px;min-width:0}
.rule-verb{color:var(--text-muted);font-size:12.5px}
.rule-profile{color:var(--text-strong);overflow-wrap:anywhere}
.rule-target .ant-tag{margin:0}
.rule-action{margin-left:auto}
.rule-add{margin-top:14px}
.result-panel{--change-rule:var(--border);margin-top:12px;padding:14px 16px;border-radius:8px;background:var(--bg-soft)}
.result-head{display:flex;flex-wrap:wrap;align-items:center;gap:4px 10px;margin-bottom:8px;color:var(--text-strong)}
.result-head strong{font-weight:600}
.result-head span{color:var(--text-muted);font-size:12.5px}
.result-head code{color:var(--text-code);font:12.5px ui-monospace,Consolas,'Cascadia Mono',monospace;overflow-wrap:anywhere}
.result-head .ant-tag{margin:0}
.origin-scroll{width:100%;overflow-x:auto}
.origin-table{width:100%;min-width:300px;border-collapse:collapse;font-size:12.5px;color:var(--text-strong)}
.origin-table th{padding:6px 10px;text-align:left;font-weight:500;font-size:12px;color:var(--text-muted);border-bottom:1px solid var(--border);white-space:nowrap}
.origin-table td{padding:7px 10px;vertical-align:top;border-bottom:1px solid var(--border);overflow-wrap:anywhere}
.origin-table tr:last-child td{border-bottom:0}
.origin-table code{white-space:nowrap;color:var(--text-code);font:12px ui-monospace,Consolas,'Cascadia Mono',monospace}
.origin-tag{margin:0}
.unset{color:var(--text-muted)}
.result-panel .form-actions{margin-top:14px}
@media(max-width:560px){
  .rule-row{grid-template-columns:minmax(0,1fr)}
  .identity-row{flex-wrap:wrap}
  .identity-actions{width:100%;justify-content:flex-end}
}
</style>
