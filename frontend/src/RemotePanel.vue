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
const editingName = ref('')
const editUrl = ref('')
const renameFrom = ref('')
const renameTo = ref('')
const mirrorName = ref('')
const mirrorUrl = ref('')

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
    onOk: () => guard(async () => {
      remotes.value = await invoke<GitRemoteInfo[]>('git.remoteRemove', { repo: props.repoPath, name: item.name, confirm: true })
      message.success('remote 已删除')
    }),
  })
}

function renameRemote(item: GitRemoteInfo) {
  renameFrom.value = item.name
  renameTo.value = item.name
}

const confirmRename = () => guard(async () => {
  remotes.value = await invoke<GitRemoteInfo[]>('git.remoteRename', { repo: props.repoPath, from: renameFrom.value, to: renameTo.value, confirm: true })
  message.success('remote 已重命名')
  renameFrom.value = ''
  renameTo.value = ''
})

function startEditUrl(item: GitRemoteInfo) {
  editingName.value = item.name
  editUrl.value = item.url ?? ''
}

const confirmSetUrl = () => guard(async () => {
  remotes.value = await invoke<GitRemoteInfo[]>('git.remoteSetUrl', { repo: props.repoPath, name: editingName.value, url: editUrl.value, confirm: true })
  message.success('fetch 地址已更新')
  editingName.value = ''
  editUrl.value = ''
})

const setMirror = (item: GitRemoteInfo) => guard(async () => {
  await invoke('git.mirrorSet', { repo: props.repoPath, remote: item.name, mirrorUrl: mirrorUrl.value, confirm: true })
  mirrorUrl.value = ''
  message.success('镜像推送已开启')
  await load()
})

const unsetMirror = (item: GitRemoteInfo) => guard(async () => {
  await invoke('git.mirrorUnset', { repo: props.repoPath, remote: item.name, confirm: true })
  message.success('镜像推送已取消')
  await load()
})

onMounted(load)
</script>

<template>
  <div class="remote-panel">
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
    <a-skeleton v-if="loading" active :paragraph="{ rows: 2 }" />
    <template v-else>
      <div v-for="item in remotes" :key="item.name" class="remote-row">
        <div class="remote-main">
          <strong>{{ item.name }}<a-tag v-if="item.mirror" color="blue">镜像推送</a-tag></strong>
          <small>{{ item.url || '（无 fetch 地址）' }}<template v-if="item.account">，账号 {{ item.account }}</template></small>
          <small v-if="item.pushUrls?.length">push：{{ item.pushUrls.join('；') }}</small>
        </div>
        <div class="remote-actions">
          <a-button size="small" @click="startEditUrl(item)">改地址</a-button>
          <a-button size="small" @click="renameRemote(item)">重命名</a-button>
          <a-button size="small" danger @click="removeRemote(item)">删除</a-button>
          <template v-if="!item.mirror">
            <a-input v-if="mirrorName === item.name" v-model:value="mirrorUrl" placeholder="镜像地址" size="small" style="width: 200px" />
            <a-button v-if="mirrorName === item.name" size="small" type="primary" :loading="busy" :disabled="!mirrorUrl.trim()" @click="setMirror(item)">开启镜像</a-button>
            <a-button v-else size="small" @click="mirrorName = item.name">镜像…</a-button>
          </template>
          <a-button v-else size="small" @click="unsetMirror(item)">取消镜像</a-button>
        </div>
        <div v-if="editingName === item.name" class="remote-edit">
          <a-input v-model:value="editUrl" placeholder="新的 fetch 地址" size="small" style="max-width: 360px" />
          <a-button size="small" type="primary" :loading="busy" @click="confirmSetUrl">保存</a-button>
          <a-button size="small" @click="editingName = ''">取消</a-button>
        </div>
        <div v-if="renameFrom === item.name" class="remote-edit">
          <a-input v-model:value="renameTo" placeholder="新名字" size="small" style="width: 160px" />
          <a-button size="small" type="primary" :loading="busy" :disabled="renameTo === item.name" @click="confirmRename">保存</a-button>
          <a-button size="small" @click="renameFrom = ''">取消</a-button>
        </div>
      </div>
      <a-empty v-if="!remotes.length" description="尚未配置 remote" />
      <div class="remote-add">
        <a-input v-model:value="newName" placeholder="名字（字母数字._-）" size="small" style="width: 160px" />
        <a-input v-model:value="newUrl" placeholder="地址（https/ssh/git）" size="small" style="width: 300px" />
        <a-button size="small" type="primary" :loading="busy" :disabled="!newName.trim() || !newUrl.trim()" @click="addRemote">新增 Remote</a-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.remote-panel{margin-top:10px;padding:12px 14px;border-radius:8px;background:var(--bg-soft)}
.remote-row{padding:8px 0;border-top:1px solid var(--border-soft)}
.remote-main{display:flex;flex-direction:column;gap:2px}
.remote-main strong{display:block}
.remote-main small{color:var(--text-muted);overflow-wrap:anywhere}
.remote-actions{display:flex;gap:6px;flex-wrap:wrap;margin-top:6px}
.remote-edit{display:flex;gap:6px;margin-top:6px;flex-wrap:wrap}
.remote-add{display:flex;gap:8px;margin-top:12px;flex-wrap:wrap}
</style>
