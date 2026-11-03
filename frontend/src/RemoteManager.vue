<script setup lang="ts">
import { ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { invoke } from './bridge'
import type { GitRemoteInfo } from './types'

const props = defineProps<{ open: boolean; repoPath: string; repoName: string }>()
const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()

const remotes = ref<GitRemoteInfo[]>([])
const loading = ref(false)
const busy = ref(false)
const errorText = ref('')

// 模态表单状态（同一时刻只开一个）
const modal = ref<'none' | 'add' | 'setUrl' | 'rename' | 'mirror'>('none')
const formName = ref('')
const formUrl = ref('')
const formFrom = ref('')
const formTo = ref('')

const close = () => emit('update:open', false)

async function load() {
  if (!props.repoPath) return
  loading.value = true
  errorText.value = ''
  try {
    remotes.value = await invoke<GitRemoteInfo[]>('git.remoteList', { repo: props.repoPath })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { loading.value = false }
}

watch(() => [props.open, props.repoPath], ([open]) => { if (open) load() })

async function guard(action: () => Promise<void>) {
  busy.value = true
  errorText.value = ''
  try { await action() } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error)
  } finally { busy.value = false }
}

function openAdd() {
  formName.value = ''
  formUrl.value = ''
  modal.value = 'add'
}

function openSetUrl(item: GitRemoteInfo) {
  formFrom.value = item.name
  formUrl.value = item.url ?? ''
  modal.value = 'setUrl'
}

function openRename(item: GitRemoteInfo) {
  formFrom.value = item.name
  formTo.value = item.name
  modal.value = 'rename'
}

function openMirror(item: GitRemoteInfo) {
  formFrom.value = item.name
  formUrl.value = ''
  modal.value = 'mirror'
}

const submit = () => guard(async () => {
  if (modal.value === 'add') {
    remotes.value = await invoke<GitRemoteInfo[]>('git.remoteAdd', { repo: props.repoPath, name: formName.value, url: formUrl.value, confirm: true })
    message.success(`已新增 remote「${formName.value}」`)
  } else if (modal.value === 'setUrl') {
    remotes.value = await invoke<GitRemoteInfo[]>('git.remoteSetUrl', { repo: props.repoPath, name: formFrom.value, url: formUrl.value, confirm: true })
    message.success(`「${formFrom.value}」fetch 地址已更新`)
  } else if (modal.value === 'rename') {
    remotes.value = await invoke<GitRemoteInfo[]>('git.remoteRename', { repo: props.repoPath, from: formFrom.value, to: formTo.value, confirm: true })
    message.success(`已重命名为「${formTo.value}」`)
  } else if (modal.value === 'mirror') {
    await invoke('git.mirrorSet', { repo: props.repoPath, remote: formFrom.value, mirrorUrl: formUrl.value, confirm: true })
    message.success('镜像推送已开启')
    await load()
  }
  modal.value = 'none'
})

function confirmRemove(item: GitRemoteInfo) {
  Modal.confirm({
    title: `删除 remote「${item.name}」？`,
    content: `将移除 ${item.url || '（无地址）'}；不影响其他 remote 与凭据。`,
    okText: '删除',
    okType: 'danger',
    zIndex: 1100,
    cancelText: '取消',
    onOk: () => guard(async () => {
      remotes.value = await invoke<GitRemoteInfo[]>('git.remoteRemove', { repo: props.repoPath, name: item.name, confirm: true })
      message.success(`已删除「${item.name}」`)
    }),
  })
}

function confirmUnmirror(item: GitRemoteInfo) {
  Modal.confirm({
    title: `取消「${item.name}」的镜像推送？`,
    content: 'push 将回到仅使用 fetch 地址。',
    okText: '取消镜像',
    cancelText: '返回',
    zIndex: 1100,
    onOk: () => guard(async () => {
      await invoke('git.mirrorUnset', { repo: props.repoPath, remote: item.name, confirm: true })
      message.success('镜像推送已取消')
      await load()
    }),
  })
}

async function copyUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    message.success('已复制')
  } catch {
    message.error('复制失败')
  }
}

function onMenu(item: GitRemoteInfo, key: string) {
  if (key === 'setUrl') openSetUrl(item)
  else if (key === 'rename') openRename(item)
  else if (key === 'mirror') openMirror(item)
  else if (key === 'unmirror') confirmUnmirror(item)
  else if (key === 'remove') confirmRemove(item)
}

</script>

<template>
  <a-drawer :open="open" width="560" :title="`${repoName} · Remote 管理`" @close="close">
    <a-alert v-if="errorText" type="error" show-icon closable :message="errorText" class="page-alert" />
    <a-skeleton v-if="loading" active :paragraph="{ rows: 3 }" />
    <template v-else>
      <a-empty v-if="!remotes.length" description="尚未配置 remote">
        <a-button type="primary" @click="openAdd">新增 Remote</a-button>
      </a-empty>
      <template v-else>
        <div v-for="item in remotes" :key="item.name" class="remote-item">
          <div class="remote-head">
            <strong>{{ item.name }}</strong>
            <a-dropdown :trigger="['click']">
              <a-button size="small" type="text" aria-label="remote 操作菜单">⋯</a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="setUrl" @click="onMenu(item, 'setUrl')">改 fetch 地址</a-menu-item>
                  <a-menu-item key="rename" @click="onMenu(item, 'rename')">重命名</a-menu-item>
                  <a-menu-item v-if="!item.mirror" key="mirror" @click="onMenu(item, 'mirror')">设置镜像推送</a-menu-item>
                  <a-menu-item v-else key="unmirror" @click="onMenu(item, 'unmirror')">取消镜像推送</a-menu-item>
                  <a-menu-item key="remove" danger @click="onMenu(item, 'remove')">删除 remote</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </div>
          <div class="remote-line">
            <span class="remote-k">fetch</span>
            <code class="remote-url" @click="copyUrl(item.url || '')">{{ item.url || '（无）' }}</code>
          </div>
          <div v-if="item.pushUrls?.length" class="remote-line">
            <span class="remote-k">push</span>
            <div class="remote-push">
              <div v-for="(pushUrl, index) in item.pushUrls" :key="pushUrl + index" class="remote-push-row">
                <span class="remote-index">{{ index + 1 }}</span>
                <code class="remote-url" @click="copyUrl(pushUrl)">{{ pushUrl }}</code>
                <a-tag v-if="item.mirror && index === 0" class="remote-tag">原目标</a-tag>
                <a-tag v-else-if="item.mirror" class="remote-tag" color="blue">镜像</a-tag>
              </div>
            </div>
          </div>
          <div v-if="item.account" class="remote-line">
            <span class="remote-k">账号</span>
            <a-tag class="remote-tag">{{ item.account }}</a-tag>
          </div>
        </div>
        <a-button type="primary" block class="add-button" @click="openAdd">+ 新增 Remote</a-button>
      </template>
    </template>
  </a-drawer>

  <a-modal :open="modal === 'add'" title="新增 Remote" :z-index="1100" :confirm-loading="busy"
    ok-text="新增" cancel-text="取消" @ok="submit" @cancel="modal = 'none'">
    <a-form layout="vertical">
      <a-form-item label="名字" extra="字母数字与 . _ -，不含空格">
        <a-input v-model:value="formName" placeholder="如 gitee" />
      </a-form-item>
      <a-form-item label="地址" extra="https / ssh / git 协议">
        <a-input v-model:value="formUrl" placeholder="https://gitee.com/user/repo.git" />
      </a-form-item>
    </a-form>
  </a-modal>

  <a-modal :open="modal === 'setUrl'" :title="`修改「${formFrom}」的 fetch 地址`" :z-index="1100" :confirm-loading="busy"
    ok-text="保存" cancel-text="取消" @ok="submit" @cancel="modal = 'none'">
    <a-form layout="vertical">
      <a-form-item label="fetch 地址">
        <a-input v-model:value="formUrl" />
      </a-form-item>
    </a-form>
  </a-modal>

  <a-modal :open="modal === 'rename'" :title="`重命名「${formFrom}」`" :z-index="1100" :confirm-loading="busy"
    ok-text="保存" cancel-text="取消" @ok="submit" @cancel="modal = 'none'">
    <a-form layout="vertical">
      <a-form-item label="新名字">
        <a-input v-model:value="formTo" @press-enter="submit" />
      </a-form-item>
    </a-form>
  </a-modal>

  <a-modal :open="modal === 'mirror'" :title="`为「${formFrom}」设置镜像推送`" :z-index="1100" :confirm-loading="busy"
    ok-text="开启镜像" cancel-text="取消" @ok="submit" @cancel="modal = 'none'">
    <a-form layout="vertical">
      <a-form-item label="镜像地址" extra="开启后 git push 会同时推送到原目标与镜像地址">
        <a-input v-model:value="formUrl" placeholder="https://gitee.com/user/repo.git" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped>
.remote-item{padding:14px 0;border-top:1px solid var(--border-soft)}
.remote-item:first-child{border-top:0}
.remote-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:8px}
.remote-head strong{font-size:14px}
.remote-line{display:flex;gap:10px;margin:5px 0;align-items:flex-start}
.remote-k{flex:none;width:44px;color:var(--text-muted);font-size:12px;padding-top:3px}
.remote-url{flex:1;min-width:0;padding:2px 8px;border-radius:6px;background:var(--bg-chip);color:var(--text-code);font-size:12px;overflow-wrap:anywhere;cursor:pointer}
.remote-push{flex:1;min-width:0;display:flex;flex-direction:column;gap:4px}
.remote-push-row{display:flex;gap:8px;align-items:center}
.remote-index{flex:none;width:16px;height:16px;border-radius:50%;background:var(--bg-chip);color:var(--text-muted);font-size:11px;display:grid;place-items:center}
.remote-tag{margin-left:6px}
.add-button{margin-top:14px}
</style>
