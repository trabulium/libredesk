import { ref, computed } from 'vue'

const STORAGE_KEY = 'libredesk-theme'
const THEMES = ['default', 'fresh']

// Migrate old theme name
const storedTheme = localStorage.getItem(STORAGE_KEY)
if (storedTheme === 'freshdesk') {
  localStorage.setItem(STORAGE_KEY, 'fresh')
}
const currentTheme = ref(localStorage.getItem(STORAGE_KEY) || 'default')

export function useTheme() {
  function setTheme(name) {
    if (!THEMES.includes(name)) return
    currentTheme.value = name
    localStorage.setItem(STORAGE_KEY, name)
  }

  const themeClass = computed(() =>
    currentTheme.value === 'default' ? '' : `theme-${currentTheme.value}`
  )

  const hideListOnTicketOpen = computed(() => currentTheme.value === 'fresh')
  const collapseSidebarByDefault = computed(() => currentTheme.value === 'fresh')

  return {
    currentTheme,
    themeClass,
    setTheme,
    hideListOnTicketOpen,
    collapseSidebarByDefault,
    THEMES
  }
}
