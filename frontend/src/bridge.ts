import type { NativeRequest, NativeResponse } from './types'

declare global {
  interface Window {
    devToolbox?: {
      invoke: (request: NativeRequest) => Promise<{ accepted: boolean; requestId: string }>
      cancel: (requestId: string) => void
    }
    __devToolboxResolve?: (requestId: string, response: NativeResponse<unknown>) => void
    __devToolboxEvent?: (event: unknown) => void
  }
}

// The host acknowledges invokes immediately, processes them off the UI
// thread, and resolves the matching promise through __devToolboxResolve.
const pending = new Map<string, { resolve: (value: unknown) => void; reject: (error: Error) => void }>()

function requestId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export async function invoke<T>(operation: string, payload: Record<string, unknown> = {}): Promise<T> {
  const id = requestId()
  const bridge = window.devToolbox
  if (!bridge && (import.meta.env.DEV || import.meta.env.MODE === 'demo')) {
    const { mockInvoke } = await import('./dev-mock')
    return (await mockInvoke(operation, payload)) as T
  }
  if (!bridge) throw new Error('桌面桥接尚未连接；请从 DevToolbox 桌面端打开此页面。')
  const response = await new Promise<NativeResponse<T>>((resolve, reject) => {
    pending.set(id, { resolve: resolve as (value: unknown) => void, reject })
    bridge.invoke({ requestId: id, operation, payload }).catch((error) => {
      pending.delete(id)
      reject(error instanceof Error ? error : new Error(String(error)))
    })
  })
  if (!response || response.requestId !== id) throw new Error('桥接响应关联标识无效。')
  if (!response.ok) throw new Error(response.error?.message ?? response.error?.code ?? '操作失败')
  return response.data as T
}

window.__devToolboxResolve = (id, response) => {
  const waiter = pending.get(id)
  if (!waiter) return
  pending.delete(id)
  // resolve with the FULL response so invoke can verify the correlation id
  if (response.ok) waiter.resolve(response)
  else waiter.reject(new Error(response.error?.message ?? response.error?.code ?? '操作失败'))
}

export const isDemoClient = (import.meta.env.DEV || import.meta.env.MODE === 'demo') && typeof window !== 'undefined' && !window.devToolbox

export function cancel(requestId: string): void {
  window.devToolbox?.cancel(requestId)
}
