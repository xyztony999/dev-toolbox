<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { Modal } from 'ant-design-vue'
import type { FormInstance, Rule } from 'ant-design-vue/es/form'
import type { GitProfile } from './types'
import KeygenPanel from './KeygenPanel.vue'

// 档案编辑抽屉：单列分组表单，label 用人话、键名作注解。
// 只负责草稿与校验；保存由父组件调用服务端，错误经 error 回显在抽屉内
//（页面顶部的错误条会被抽屉遮住）。
const props = defineProps<{ open: boolean; profile?: GitProfile; saving: boolean; error?: string }>()
const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'save', profile: GitProfile): void
  (e: 'applied', profile: GitProfile): void
}>()

const keygenOpen = ref(false)
const keygenMode = ref<'generate' | 'select'>('generate')

type Pair = { left: string; right: string }
const blankPair = (): Pair => ({ left: '', right: '' })

const formRef = ref<FormInstance>()
const form = reactive({ id: '', name: '', email: '', gpgSign: false, signingKey: '', gpgFormat: undefined as string | undefined })
const rewrites = ref<Pair[]>([blankPair()])
const accounts = ref<Pair[]>([blankPair()])
const pairError = ref('')
let baseline = ''

const idLocked = computed(() => !!props.profile)
const gpgFormatOptions = [
  { value: 'openpgp', label: 'GPG（openpgp）' },
  { value: 'ssh', label: 'SSH 密钥（ssh）' },
  { value: 'x509', label: 'X.509 证书（x509）' },
]

// 与服务端 Profile.Validate 同源：ID 与邮箱用同一套规则，避免前端拦住后端接受的值。
const rules: Record<string, Rule[]> = {
  id: [
    { required: true, message: '档案需要一个 ID，例如 work、personal', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9][A-Za-z0-9._-]*$/, message: '只能用字母、数字、点、下划线和短横线，并以字母或数字开头', trigger: 'blur' },
  ],
  name: [{ required: true, whitespace: true, message: '请填写提交时显示的姓名', trigger: 'blur' }],
  email: [
    { required: true, whitespace: true, message: '请填写提交时显示的邮箱', trigger: 'blur' },
    { pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/, message: '这个邮箱格式看起来不对', trigger: 'blur' },
  ],
}

function cleanPairs(pairs: Pair[]): Array<{ left: string; right: string }> {
  return pairs
    .map((pair) => ({ left: pair.left.trim(), right: pair.right.trim() }))
    .filter((pair) => pair.left || pair.right)
}

function snapshot(): string {
  return JSON.stringify([
    form.id.trim(), form.name.trim(), form.email.trim(), form.gpgSign,
    form.signingKey.trim(), form.gpgFormat ?? '', cleanPairs(rewrites.value), cleanPairs(accounts.value),
  ])
}

const dirty = computed(() => snapshot() !== baseline)

function loadDraft() {
  const profile = props.profile
  form.id = profile?.id ?? ''
  form.name = profile?.name ?? ''
  form.email = profile?.email ?? ''
  form.gpgSign = !!profile?.gpgSign
  form.signingKey = profile?.signingKey ?? ''
  form.gpgFormat = profile?.gpgFormat || undefined
  rewrites.value = profile?.insteadOf?.length
    ? profile.insteadOf.map((item) => ({ left: item.original, right: item.url }))
    : [blankPair()]
  accounts.value = profile?.accounts?.length
    ? profile.accounts.map((item) => ({ left: item.host, right: item.username }))
    : [blankPair()]
  pairError.value = ''
  baseline = snapshot()
}

watch(() => props.open, async (open) => {
  if (!open) {
    keygenOpen.value = false
    return
  }
  loadDraft()
  await nextTick()
  formRef.value?.clearValidate()
})

function openKeygen(mode: 'generate' | 'select') {
  if (!idLocked.value || !form.id.trim()) {
    Modal.info({
      title: '请先保存档案',
      content: '生成或选用密钥需要已有档案 ID。先保存档案，再回来操作。',
      okText: '知道了',
      zIndex: 1100,
    })
    return
  }
  keygenMode.value = mode
  keygenOpen.value = true
}

/** 密钥服务已写档案签名字段：回填表单，并只把签名三字段同步进 baseline，保留其它脏字段。 */
function onKeygenApplied(profile: GitProfile) {
  form.signingKey = profile.signingKey ?? ''
  form.gpgFormat = profile.gpgFormat || undefined
  form.gpgSign = !!profile.gpgSign
  try {
    const parts = JSON.parse(baseline) as unknown[]
    parts[3] = form.gpgSign
    parts[4] = form.signingKey.trim()
    parts[5] = form.gpgFormat ?? ''
    baseline = JSON.stringify(parts)
  } catch {
    baseline = snapshot()
  }
  emit('applied', profile)
}

function requestClose() {
  if (!dirty.value) {
    emit('update:open', false)
    return
  }
  Modal.confirm({
    title: '放弃未保存的修改？',
    content: '刚才填写的内容不会保存。',
    okText: '放弃修改',
    okButtonProps: { danger: true },
    cancelText: '继续编辑',
    zIndex: 1100,
    onOk: () => emit('update:open', false),
  })
}

async function submit() {
  pairError.value = ''
  const halfFilled = [...rewrites.value, ...accounts.value]
    .some((pair) => [pair.left, pair.right].some((v) => v.trim()) && [pair.left, pair.right].some((v) => !v.trim()))
  if (halfFilled) {
    pairError.value = 'URL 重写和账号选择器每一行都要填完整；不需要的行请留空或删掉。'
    return
  }
  try {
    await formRef.value?.validate()
  } catch {
    await nextTick()
    // 把第一个出错字段滚到视野里：抽屉可能停在表单下半部分，红字提示在上方看不到。
    formRef.value?.$el?.querySelector('.ant-form-item-has-error')?.scrollIntoView({ block: 'center' })
    return
  }
  emit('save', {
    id: form.id.trim(),
    name: form.name.trim(),
    email: form.email.trim(),
    gpgSign: form.gpgSign,
    signingKey: form.signingKey.trim() || undefined,
    gpgFormat: form.gpgFormat || undefined,
    insteadOf: cleanPairs(rewrites.value).map((pair) => ({ original: pair.left, url: pair.right })),
    accounts: cleanPairs(accounts.value).map((pair) => ({ host: pair.left, username: pair.right })),
    // 这两项不在本表单编辑，原样带回，避免保存时丢掉。
    customKv: props.profile?.customKv,
    gitdirs: props.profile?.gitdirs,
  })
}
</script>

<template>
  <a-drawer :open="open" width="min(560px, 100vw)" :title="idLocked ? '编辑档案' : '新建档案'"
    :mask-closable="false" @close="requestClose">
    <a-alert v-if="error" type="error" show-icon :message="error" class="editor-alert" />
    <a-form ref="formRef" layout="vertical" :model="form" :rules="rules" class="editor-form" @finish="submit">
      <section class="field-group">
        <h3>档案</h3>
        <a-form-item name="id">
          <template #label>档案 ID<span class="key-note">用于目录规则引用</span></template>
          <a-input v-model:value="form.id" :disabled="idLocked" :maxlength="64" autocomplete="off" />
          <template v-if="idLocked" #extra>已有档案不能改 ID。</template>
          <template v-else #extra>字母、数字、点、下划线或短横线，例如 work、personal。</template>
        </a-form-item>
        <a-form-item name="name">
          <template #label>姓名<span class="key-note">user.name</span></template>
          <a-input v-model:value="form.name" :maxlength="120" autocomplete="off" />
        </a-form-item>
        <a-form-item name="email">
          <template #label>邮箱<span class="key-note">user.email</span></template>
          <a-input v-model:value="form.email" :maxlength="254" autocomplete="off" />
        </a-form-item>
      </section>

      <section class="field-group">
        <h3>提交签名</h3>
        <a-form-item>
          <a-checkbox v-model:checked="form.gpgSign">提交时签名<span class="key-note">commit.gpgsign</span></a-checkbox>
        </a-form-item>
        <a-form-item>
          <template #label>签名密钥<span class="key-note">user.signingkey</span></template>
          <a-input v-model:value="form.signingKey" placeholder="留空则不设置" :maxlength="256" autocomplete="off" />
        </a-form-item>
        <a-form-item>
          <template #label>签名格式<span class="key-note">gpg.format</span></template>
          <a-select v-model:value="form.gpgFormat" :options="gpgFormatOptions" allow-clear placeholder="留空则不设置" />
        </a-form-item>
        <div class="signing-actions">
          <a-button size="small" :disabled="!idLocked" @click="openKeygen('generate')">生成密钥</a-button>
          <a-button size="small" :disabled="!idLocked" @click="openKeygen('select')">选用本机密钥</a-button>
        </div>
        <p v-if="!idLocked" class="group-note">新建档案需先保存，才能生成或选用密钥。x509 / 手工指纹仍可直接填写。</p>
      </section>

      <section class="field-group">
        <h3>URL 重写</h3>
        <p class="group-note">Git 访问以左边地址开头的仓库时，改用右边的地址。常用来让同一主机的不同账号走不同密钥。<span class="key-note">url.&lt;替换为&gt;.insteadOf</span></p>
        <div class="pair-head" aria-hidden="true"><span>原始地址</span><span></span><span>替换为</span><span></span></div>
        <div v-for="(pair, index) in rewrites" :key="index" class="pair-row">
          <a-input v-model:value="pair.left" placeholder="原始地址" aria-label="URL 重写的原始地址" autocomplete="off" />
          <span class="pair-arrow" aria-hidden="true">→</span>
          <a-input v-model:value="pair.right" placeholder="替换为" aria-label="URL 重写的替换地址" autocomplete="off" />
          <a-button type="text" size="small" class="pair-remove" @click="rewrites.splice(index, 1)">删除</a-button>
        </div>
        <a-button type="link" size="small" class="pair-add" :disabled="rewrites.length >= 8" @click="rewrites.push(blankPair())">再加一条</a-button>
      </section>

      <section class="field-group">
        <h3>账号选择器</h3>
        <p class="group-note">推送到这个主机时使用这个用户名，配合 URL 重写可以区分同一主机上的多个账号。<span class="key-note">credential.&lt;主机&gt;.username</span></p>
        <div class="pair-head account" aria-hidden="true"><span>主机</span><span>用户名</span><span></span></div>
        <div v-for="(pair, index) in accounts" :key="index" class="pair-row account">
          <a-input v-model:value="pair.left" placeholder="主机，如 github.com" aria-label="账号选择器的主机" autocomplete="off" />
          <a-input v-model:value="pair.right" placeholder="用户名" aria-label="账号选择器的用户名" autocomplete="off" />
          <a-button type="text" size="small" class="pair-remove" @click="accounts.splice(index, 1)">删除</a-button>
        </div>
        <a-button type="link" size="small" class="pair-add" :disabled="accounts.length >= 8" @click="accounts.push(blankPair())">再加一条</a-button>
      </section>

      <a-alert v-if="pairError" type="error" show-icon :message="pairError" class="editor-alert" />
    </a-form>

    <template #footer>
      <div class="editor-footer">
        <a-button @click="requestClose">取消</a-button>
        <a-button type="primary" :loading="saving" @click="submit">保存档案</a-button>
      </div>
    </template>
  </a-drawer>

  <KeygenPanel
    v-model:open="keygenOpen"
    :mode="keygenMode"
    :profile-id="form.id.trim()"
    :profile-name="form.name"
    :profile-email="form.email"
    :current-signing-key="form.signingKey"
    @applied="onKeygenApplied"
  />
</template>

<style scoped>
.editor-alert{margin-bottom:14px}
.editor-form .field-group{padding:4px 0 20px;border-top:1px solid var(--border-soft)}
.editor-form .field-group:first-child{border-top:0;padding-top:0}
.editor-form .field-group:last-of-type{padding-bottom:4px}
.field-group h3{margin:0 0 14px;color:var(--text-title);font:600 13.5px/1.4 var(--sans)}
.field-group .ant-form-item{margin-bottom:14px}
.key-note{margin-left:8px;color:var(--text-muted);font-weight:400}
.field-group .key-note{font:400 12px ui-monospace,Consolas,'Cascadia Mono',monospace}
.group-note{margin:-4px 0 12px;color:var(--text-muted);font-size:12.5px;line-height:1.6}
.pair-head,.pair-row{display:grid;grid-template-columns:minmax(0,1fr) 20px minmax(0,1fr) 48px;gap:8px;align-items:center}
.pair-head.account,.pair-row.account{grid-template-columns:minmax(0,1fr) minmax(0,1fr) 48px}
.pair-head{padding:0 0 6px;color:var(--text-muted);font-size:12px}
.pair-row{margin-bottom:8px}
.pair-arrow{text-align:center;color:var(--text-muted)}
.pair-remove{justify-self:end}
.pair-add{padding-left:0}
.signing-actions{display:flex;flex-wrap:wrap;gap:8px;margin:-4px 0 8px}
.editor-footer{display:flex;justify-content:flex-end;gap:9px}
@media(max-width:440px){
  .pair-head{display:none}
  .pair-row{grid-template-columns:minmax(0,1fr) 48px}
  .pair-arrow{display:none}
  .pair-row>:nth-child(3){grid-column:1}
}
</style>
