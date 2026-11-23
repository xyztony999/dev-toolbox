import { ref } from 'vue'

export type ThemePref = 'system' | 'light' | 'dark'

const storageKey = 'devtoolbox.theme'

function readTheme(): ThemePref {
  const saved = localStorage.getItem(storageKey)
  return saved === 'light' || saved === 'dark' ? saved : 'system'
}

export const themePref = ref<ThemePref>(readTheme())

function paint(pref: ThemePref) {
  if (pref === 'system') document.documentElement.removeAttribute('data-theme')
  else document.documentElement.dataset.theme = pref
}

paint(themePref.value)

export function setTheme(pref: ThemePref) {
  themePref.value = pref
  localStorage.setItem(storageKey, pref)
  paint(pref)
}
