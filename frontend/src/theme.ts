import { ref, watch } from 'vue'

export type ThemePref = 'system' | 'light' | 'dark' | 'custom'

const THEME_KEY = 'dev-toolbox.theme'
const ACCENT_KEY = 'dev-toolbox.customAccent'
const DEFAULT_ACCENT = '#a34b2a'

function readTheme(): ThemePref {
  const saved = localStorage.getItem(THEME_KEY)
  return saved === 'light' || saved === 'dark' || saved === 'custom' ? saved : 'system'
}

function readAccent(): string {
  return localStorage.getItem(ACCENT_KEY) ?? DEFAULT_ACCENT
}

/** 与后端 clientsettings.NormalizeAccent 保持一致的强调色校验。 */
export function validAccent(value: string): boolean {
  return /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(value.trim())
}

/** 归一化 #abc 形式并统一小写；非法输入返回空串。 */
export function normalizeAccent(value: string): string {
  const trimmed = value.trim()
  if (!validAccent(trimmed)) return ''
  const hex = trimmed.toLowerCase()
  if (hex.length === 4) {
    return `#${hex[1]}${hex[1]}${hex[2]}${hex[2]}${hex[3]}${hex[3]}`
  }
  return hex
}

function parseHex(value: string): [number, number, number] {
  const hex = normalizeAccent(value) || DEFAULT_ACCENT
  return [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16),
  ]
}

function toHex(rgb: [number, number, number]): string {
  return `#${rgb.map((channel) => Math.max(0, Math.min(255, Math.round(channel))).toString(16).padStart(2, '0')).join('')}`
}

/** 按比例把两种颜色混合，用于推导柔和的强调底色。 */
function mix(a: string, b: string, ratio: number): string {
  const left = parseHex(a)
  const right = parseHex(b)
  return toHex([
    left[0] + (right[0] - left[0]) * ratio,
    left[1] + (right[1] - left[1]) * ratio,
    left[2] + (right[2] - left[2]) * ratio,
  ])
}

/** 提亮用于深色底，压暗用于浅色底，保证文字对比度。 */
function shade(hex: string, ratio: number): string {
  const rgb = parseHex(hex)
  return toHex([
    rgb[0] + (255 - rgb[0]) * ratio,
    rgb[1] + (255 - rgb[1]) * ratio,
    rgb[2] + (255 - rgb[2]) * ratio,
  ])
}

export function resolveDark(pref: ThemePref, systemDark: boolean): boolean {
  return pref === 'dark' || ((pref === 'system' || pref === 'custom') && systemDark)
}

export const themePref = ref<ThemePref>(readTheme())
export const customAccent = ref<string>(readAccent())

function paint(pref: ThemePref) {
  if (pref === 'system') document.documentElement.removeAttribute('data-theme')
  else document.documentElement.dataset.theme = pref
}

/** 自定义主题下把强调色及其派生色写进 :root，让所有 --accent* 消费方一致变色。 */
function paintAccent(pref: ThemePref, accent: string, systemDark: boolean) {
  const root = document.documentElement
  if (pref !== 'custom' || !accent) {
    root.style.removeProperty('--accent')
    root.style.removeProperty('--accent-strong')
    root.style.removeProperty('--accent-soft')
    root.style.removeProperty('--accent-mark')
    delete root.dataset.accent
    return
  }
  const base = systemDark ? '#23201c' : '#f7f4ee'
  root.dataset.accent = accent
  root.style.setProperty('--accent', accent)
  root.style.setProperty('--accent-mark', accent)
  root.style.setProperty('--accent-strong', systemDark ? shade(accent, 0.18) : shade(accent, -0.28))
  root.style.setProperty('--accent-soft', mix(accent, base, 0.82))
}

const schemeQuery = window.matchMedia('(prefers-color-scheme: dark)')

function syncAccent() {
  paintAccent(themePref.value, customAccent.value, schemeQuery.matches)
}

paint(themePref.value)
syncAccent()
schemeQuery.addEventListener('change', syncAccent)

export function setTheme(pref: ThemePref, accent?: string): void {
  themePref.value = pref
  localStorage.setItem(THEME_KEY, pref)
  if (typeof accent === 'string' && accent) {
    customAccent.value = accent
    localStorage.setItem(ACCENT_KEY, accent)
  }
  paint(pref)
  syncAccent()
  window.dispatchEvent(new CustomEvent('dev-toolbox-theme-changed', { detail: { pref } }))
}

watch([themePref, customAccent], syncAccent)
