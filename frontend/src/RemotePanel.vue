<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type { GitRemoteInfo } from './types'

const props = defineProps<{ repoPath: string }>()

const remotes = ref<GitRemoteInfo[]>([])
const loading = ref(true)
const busy = ref(false)
const errorText = ref('')

const newName = ref('')
const newUrl = ref('')
// 编辑态互斥：同一时刻只展开一个（改地址 / 重命名 / 镜像）
const editing = ref<{ kind: 'url' | 'rename' | 'mirror'; remote: string; value: string }>()

function openEdit(kind: 'url' | 'rename' | 'mirror', item: GitRemoteInfo, value: string) {
  editing.value = { kind, remote: item.name, value }
}

async function load() {
  loading.value = true
  errorText.value = ''
  try {
    remotes.value = await invoke<GitRemoteInfo[]>('git.remoteList', { repo: props.repoPath })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { loading.value = false }
}

async function guard(action: () => Promise<void>) {
  busy.value = true
  errorText.value = ''
  try { await action() } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { busy.value = false }
}

async function applyRemoteOp(op: () => Promise<GitRemoteInfo[]>, done: string) {
  await guard(async () => {
    remotes.value = await op()
    editing.value = undefined
    message.success(done)
  })
}

const addRemote = () => guard(async () => {
  remotes.value = await invoke<GitRemoteInfo[]>('git.remoteAdd', { repo: props.repoPath, name: newName.value, url: newUrl.value, confirm: true })
  newName.value = ''
  newUrl.value = ''
  message.success('remote 已新增')
})

function removeRemote(item: GitRemoteInfo) {
  Modal.confirm({
    title: `删除 remote「${item.name}」？`,
    content: `其地址 ${item.url || '（无）'} 将从本地配置移除；不影响其他 remote。`,
    okText: '删除',
    cancelText: '取消',
    onOk: () => applyRemoteOp(
      () => invoke<GitRemoteInfo[]>('git.remoteRemove', { repo: props.repoPath, name: item.name, confirm: true }),
      'remote 已删除'),
  })
}

const saveSetUrl = (item: GitRemoteInfo) => applyRemoteOp(
  () => invoke<GitRemoteInfo[]>('git.remoteSetUrl', { repo: props.repoPath, name: item.name, url: editing.value?.value, confirm: true }),
  'fetch 地址已更新')

const saveRename = (item: GitRemoteInfo) => applyRemoteOp(
  () => invoke<GitRemoteInfo[]>('git.remoteRename', { repo: props.repoPath, from: item.name, to: editing.value?.value, confirm: true }),
  'remote 已重命名')

const pushTarget = (item: GitRemoteInfo) => item.pushUrls?.[0] || item.url || ''

const setMirror = (item: GitRemoteInfo) => applyRemoteOp(async () => {
  await invoke('git.mirrorSet', { repo: props.repoPath, remote: item.name, mirrorUrl: editing.value?.value, confirm: true })
  return await invoke<GitRemoteInfo[]>('git.remoteList', { repo: props.repoPath })
}, '镜像推送已开启')

const unsetMirror = (item: GitRemoteInfo) => applyRemoteOp(async () => {
  await invoke('git.mirrorUnset', { repo: props.repoPath, remote: item.name, confirm: true })
  return await invoke<GitRemoteInfo[]>('git.remoteList', { repo: props.repoPath })
}, '镜像推送已取消')

onMounted(load)
</script>

<template>
  <div class="remote-panel">
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
    <a-skeleton v-if="loading" active :paragraph="{ rows: 2 }" />
    <template v-else>
      <a-empty v-if="!remotes.length" description="尚未配置 remote" />

      <div v-for="item in remotes" :key="item.name" class="remote-item">
        <div class="remote-head">
          <div class="remote-title">
            <strong>{{ item.name }}</strong>
            <a-tag v-if="item.mirror" color="blue">镜像推送</a-tag>
            <a-tag v-if="item.account">账号 {{ item.account }}</a-tag>
          </div>
          <div class="remote-actions">
            <a-button size="small" type="text" @click="openEdit('url', item, item.url ?? '')">改地址</a-button>
            <a-button size="small" type="text" @click="openEdit('rename', item, item.name)">重命名</a-button>
            <a-button size="small" type="text" @click="openEdit('mirror', item, '')">镜像…</a-button>
            <a-button size="small" type="text" danger @click="removeRemote(item)">删除</a-button>
          </div>
        </div>

        <div class="remote-lines">
          <div class="remote-line"><span class="line-kind">fetch</span><code>{{ item.url || '（无）' }}</code></div>
          <div v-for="(push, index) in item.pushUrls ?? []" :key="push + index" class="remote-line">
            <span class="line-kind">push</span><code>{{ push }}</code>
            <a-tag v-if="item.mirror && index === 0" class="line-tag">原始目标</a-tag>
            <a-tag v-else-if="item.mirror" class="line-tag" color="blue">镜像</a-tag>
          </div>
          <div v-if="!item.mirror && !(item.pushUrls?.length)" class="remote-line muted-line">
            <span class="line-kind">push</span><code>{{ item.url || '（无）' }}</code><small>（未单独配置，随 fetch）</small>
          </div>
        </div>

        <div v-if="editing?.remote === item.name && editing.kind === 'url'" class="remote-edit">
          <span class="edit-label">fetch 地址</span>
          <a-input v-model:value="editing.value" size="small" style="max-width: 380px" @press-enter="saveSetUrl(item)" />
          <a-button size="small" type="primary" :loading="busy" @click="saveSetUrl(item)">保存</a-button>
          <a-button size="small" @click="editing = undefined">取消</a-button>
        </div>
        <div v-else-if="editing?.remote === item.name && editing.kind === 'rename'" class="remote-edit">
          <span class="edit-label">新名字</span>
          <a-input v-model:value="editing.value" size="small" style="width: 180px" @press-enter="saveRename(item)" />
          <a-button size="small" type="primary" :loading="busy" :disabled="editing.value === item.name" @click="saveRename(item)">保存</a-button>
          <a-button size="small" @click="editing = undefined">取消</a-button>
        </div>
        <div v-else-if="editing?.remote === item.name && editing.kind === 'mirror'" class="remote-edit">
          <template v-if="!item.mirror">
            <span class="edit-label">镜像地址</span>
            <a-input v-model:value="editing.value" placeholder="如 https://gitee.com/user/repo.git" size="small" style="max-width: 320px" @press-enter="setMirror(item)" />
            <a-button size="small" type="primary" :loading="busy" :disabled="!editing.value.trim() || editing.value.trim() === pushTarget(item)" @click="setMirror(item)">开启镜像推送</a-button>
          </template>
          <template v-else>
            <span class="edit-label">当前镜像</span>
            <code class="mirror-current">{{ item.pushUrls?.[item.pushUrls.length - 1] ?? '' }}</code>
            <a-button size="small" :loading="busy" @click="unsetMirror(item)">取消镜像推送</a-button>
          </template>
          <a-button size="small" @click="editing = undefined">收起</a-button>
        </div>
      </div>

      <div class="remote-add">
        <a-input v-model:value="newName" placeholder="名字（字母数字._-）" size="small" style="width: 180px" @press-enter="newUrl ? addRemote() : undefined" />
        <a-input v-model:value="newUrl" placeholder="地址（https/ssh/git）" size="small" style="flex: 1; min-width: 220px" @press-enter="newName ? addRemote() : undefined" />
        <a-button size="small" type="primary" :loading="busy" :disabled="!newName.trim() || !newUrl.trim()" @click="addRemote">新增 Remote</a-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.remote-panel{margin-top:10px;padding:14px;border-radius:8px;background:var(--bg-soft)}
.remote-item{padding:10px 0;border-top:1px solid var(--border-soft)}
.remote-item:first-of-type{border-top:0}
.remote-head{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}
.remote-title{display:flex;align-items:center;gap:8px}
.remote-title strong{font-size:13.5px}
.remote-actions{display:flex;gap:2px;flex-wrap:wrap}
.remote-actions :deep(.ant-btn){padding:2px 7px}
.remote-lines{margin-top:6px;display:flex;flex-direction:column;gap:3px}
.remote-line{display:flex;align-items:center;gap:8px;min-width:0}
.remote-line code{overflow-wrap:anywhere;color:var(--text-code);font-size:12px}
.line-kind{flex:none;width:38px;color:var(--text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.04em}
.line-tag{margin-left:4px;flex:none}
.muted-line small{color:var(--text-muted)}
.remote-edit{display:flex;align-items:center;gap:8px;margin-top:8px;flex-wrap:wrap}
.edit-label{color:var(--text-muted);font-size:12px;flex:none}
.mirror-current{overflow-wrap:anywhere;color:var(--text-code);font-size:12px}
.remote-add{display:flex;gap:8px;margin-top:14px;flex-wrap:wrap}
</style>
