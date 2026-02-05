# Freshdesk Theme Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a theme switcher with a Freshdesk-inspired theme that reduces clutter, improves contrast, and hides the conversation list when a ticket is open.

**Architecture:** CSS-only theme layer activated by a root class, plus a Vue composable for theme state and two small behaviour toggles (hide list on ticket open, collapse sidebar by default). Only 3 upstream files get minimal edits.

**Tech Stack:** Vue 3, Tailwind CSS, SCSS, Shadcn/Radix components, localStorage

---

### Task 1: Create the theme composable

**Files:**
- Create: `frontend/src/composables/useTheme.js`

**Step 1: Create the composable**

```javascript
import { ref, computed, watchEffect } from 'vue'

const STORAGE_KEY = 'libredesk-theme'
const THEMES = ['default', 'freshdesk']

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

  const hideListOnTicketOpen = computed(() => currentTheme.value === 'freshdesk')
  const collapseSidebarByDefault = computed(() => currentTheme.value === 'freshdesk')

  return {
    currentTheme,
    themeClass,
    setTheme,
    hideListOnTicketOpen,
    collapseSidebarByDefault,
    THEMES
  }
}
```

**Step 2: Verify file exists**

Run: `cat frontend/src/composables/useTheme.js`
Expected: File contents shown

**Step 3: Commit**

```bash
git add frontend/src/composables/useTheme.js
git commit -m "feat: add theme composable with localStorage persistence"
```

---

### Task 2: Create the Freshdesk theme SCSS file

**Files:**
- Create: `frontend/src/assets/styles/themes/freshdesk.scss`

**Step 1: Create the theme file with CSS variable overrides**

The file should contain a `.theme-freshdesk` selector that overrides all relevant CSS variables from `main.scss`. Key values:

```scss
.theme-freshdesk {
  // --- Core palette ---
  --background: 0 0% 100%;
  --foreground: 220 14% 10%;
  --card: 0 0% 100%;
  --card-foreground: 220 14% 10%;
  --popover: 0 0% 100%;
  --popover-foreground: 220 14% 10%;
  --primary: 174 62% 33%;
  --primary-foreground: 0 0% 100%;
  --secondary: 210 20% 96%;
  --secondary-foreground: 220 14% 10%;
  --muted: 210 20% 96%;
  --muted-foreground: 220 9% 40%;
  --accent: 174 40% 95%;
  --accent-foreground: 174 62% 25%;
  --destructive: 0 84.2% 60.2%;
  --destructive-foreground: 0 0% 98%;
  --border: 220 13% 89%;
  --input: 220 13% 89%;
  --ring: 174 62% 33%;
  --radius: 0.5rem;
  --canvas: 210 20% 96%;
  --private: 35 90% 94%;

  // --- Dark sidebar (icon rail) ---
  --sidebar-background: 220 15% 16%;
  --sidebar-foreground: 210 10% 85%;
  --sidebar-primary: 174 62% 50%;
  --sidebar-primary-foreground: 0 0% 100%;
  --sidebar-accent: 220 15% 22%;
  --sidebar-accent-foreground: 210 10% 92%;
  --sidebar-border: 220 15% 20%;
  --sidebar-ring: 174 62% 50%;
}

// --- Typography & density improvements ---
.theme-freshdesk {
  // Conversation list items: bolder names, lighter metadata
  .conversation-list-item,
  [data-conversation-list] {
    .contact-name,
    .font-semibold {
      font-weight: 600;
    }
    .text-muted-foreground {
      font-size: 0.8125rem;
    }
  }

  // Better card separation in conversation list
  .conversation-list-item + .conversation-list-item,
  [data-radix-scroll-area-viewport] > div > div > div + div {
    border-top: 1px solid hsl(var(--border));
  }

  // Status badges
  .badge {
    font-weight: 500;
  }
}
```

Note: The exact CSS selectors will need to be verified against the actual conversation list component markup. Read `ConversationList.vue` and `ConversationListItem.vue` (or similar) to find the right selectors. Use broad Tailwind utility selectors where component classes don't exist.

**Step 2: Import the theme in main.scss**

At the end of `frontend/src/assets/styles/main.scss`, add:

```scss
@import './themes/freshdesk.scss';
```

**Step 3: Verify import works**

Run: `cd /home/ubuntu/libredesk/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/assets/styles/themes/freshdesk.scss frontend/src/assets/styles/main.scss
git commit -m "feat: add Freshdesk theme CSS variable overrides"
```

---

### Task 3: Wire up theme class in App.vue

**Files:**
- Modify: `frontend/src/App.vue`

**Step 1: Add theme binding to root div**

In `App.vue`, import the composable and bind the theme class to the root div:

In `<script setup>`, add:
```javascript
import { useTheme } from '@/composables/useTheme'
const { themeClass } = useTheme()
```

In the template, change the root div from:
```html
<div class="flex w-full h-screen text-foreground bg-canvas p-1.5">
```
to:
```html
<div :class="['flex w-full h-screen text-foreground bg-canvas p-1.5', themeClass]">
```

**Step 2: Verify the app still loads**

Run: `cd /home/ubuntu/libredesk/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/App.vue
git commit -m "feat: bind theme class to root element in App.vue"
```

---

### Task 4: Create ThemeSwitcher component

**Files:**
- Create: `frontend/src/components/sidebar/ThemeSwitcher.vue`

**Step 1: Create the component**

A small dropdown in the icon sidebar footer. Uses the existing Shadcn DropdownMenu components. Shows a palette icon that opens a menu with theme options.

```vue
<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <SidebarMenuButton>
        <Palette />
      </SidebarMenuButton>
    </DropdownMenuTrigger>
    <DropdownMenuContent side="right" align="end">
      <DropdownMenuItem
        v-for="t in THEMES"
        :key="t"
        @click="setTheme(t)"
        :class="{ 'bg-accent': currentTheme === t }"
      >
        <Check v-if="currentTheme === t" class="mr-2 h-4 w-4" />
        <span v-else class="mr-2 h-4 w-4" />
        {{ t.charAt(0).toUpperCase() + t.slice(1) }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { useTheme } from '@/composables/useTheme'
import { Palette, Check } from 'lucide-vue-next'
import { SidebarMenuButton } from '@/components/ui/sidebar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'

const { currentTheme, setTheme, THEMES } = useTheme()
</script>
```

**Step 2: Add ThemeSwitcher to App.vue icon sidebar footer**

In `App.vue`, import ThemeSwitcher and add it as a SidebarMenuItem in the SidebarFooter, before the NotificationBell:

```javascript
import ThemeSwitcher from '@/components/sidebar/ThemeSwitcher.vue'
```

```html
<SidebarMenuItem>
  <Tooltip>
    <TooltipTrigger as-child>
      <ThemeSwitcher />
    </TooltipTrigger>
    <TooltipContent side="right">
      <p>Theme</p>
    </TooltipContent>
  </Tooltip>
</SidebarMenuItem>
```

**Step 3: Verify build**

Run: `cd /home/ubuntu/libredesk/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/components/sidebar/ThemeSwitcher.vue frontend/src/App.vue
git commit -m "feat: add theme switcher to icon sidebar"
```

---

### Task 5: Hide conversation list when ticket is open

**Files:**
- Modify: `frontend/src/layouts/inbox/InboxLayout.vue`

**Step 1: Read the current InboxLayout.vue**

Current structure is a `ResizablePanelGroup` with two panels: ConversationList (25%) and detail (75%).

**Step 2: Add conditional hiding and back button**

Import the theme composable and route:

```javascript
import { useTheme } from '@/composables/useTheme'
const { hideListOnTicketOpen } = useTheme()

const hasConversationOpen = computed(() => !!route.params.uuid)
const showList = computed(() => !hideListOnTicketOpen.value || !hasConversationOpen.value)
```

Wrap the ConversationList panel with `v-show="showList"` and the ResizableHandle similarly. When the list is hidden, the detail panel should use 100% width.

Add a back button above or inside the detail panel when `hideListOnTicketOpen && hasConversationOpen`:

```html
<div v-if="hideListOnTicketOpen && hasConversationOpen" class="flex items-center px-3 py-2 border-b bg-background">
  <button @click="goBack" class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
    <ArrowLeft class="h-4 w-4" />
    Back to conversations
  </button>
</div>
```

The `goBack` function navigates to the parent inbox route (removes the uuid param):

```javascript
import { ArrowLeft } from 'lucide-vue-next'

function goBack() {
  // Navigate to parent route (inbox list without uuid)
  const parentRoute = route.matched[route.matched.length - 2]
  if (parentRoute) {
    router.push({ name: parentRoute.name, params: route.params })
  } else {
    router.back()
  }
}
```

**Step 3: Verify build**

Run: `cd /home/ubuntu/libredesk/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/layouts/inbox/InboxLayout.vue
git commit -m "feat: hide conversation list when ticket is open in Freshdesk theme"
```

---

### Task 6: Collapse sidebar by default in Freshdesk theme

**Files:**
- Modify: `frontend/src/components/sidebar/Sidebar.vue`

**Step 1: Read the current default**

Currently: `const sidebarOpen = useStorage('mainSidebarOpen', true)`

**Step 2: Use theme preference for initial default**

Import the theme composable and adjust the default:

```javascript
import { useTheme } from '@/composables/useTheme'
const { collapseSidebarByDefault } = useTheme()
```

Change the sidebar default to respect the theme. The simplest approach: if the user hasn't explicitly set a preference yet AND the theme wants collapsed, start collapsed. Since `useStorage` persists to localStorage, only override the initial default — not force it every time.

One approach: use a separate storage key that tracks if the user explicitly toggled, or simply change the default:

```javascript
const sidebarOpen = useStorage('mainSidebarOpen', !collapseSidebarByDefault.value)
```

Note: This sets the default once. If the user switches themes, they may need to manually toggle. This is acceptable — the sidebar toggle button already exists.

**Step 3: Verify build**

Run: `cd /home/ubuntu/libredesk/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/components/sidebar/Sidebar.vue
git commit -m "feat: collapse sidebar by default in Freshdesk theme"
```

---

### Task 7: Refine theme CSS with real component selectors

**Files:**
- Modify: `frontend/src/assets/styles/themes/freshdesk.scss`

**Step 1: Read actual conversation list components**

Read the conversation list item components to find the actual CSS classes/selectors used:
- `frontend/src/features/conversation/list/ConversationList.vue`
- `frontend/src/features/conversation/list/ConversationListItem.vue` (or similar filename)

**Step 2: Update theme SCSS with accurate selectors**

Adjust the typography, density, and separation rules to target the real class names and structure found in step 1. Focus on:
- Contact name weight (bolder)
- Metadata size (smaller)
- List item padding/spacing
- Active/selected item highlight using teal accent
- Separator lines between items

Also read `ConversationDetailView.vue` or the ticket header area to find where to style the ticket view for better spacing.

**Step 3: Build and verify visually**

Run: `cd /home/ubuntu/libredesk/frontend && NODE_OPTIONS='--max-old-space-size=4096' pnpm build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/assets/styles/themes/freshdesk.scss
git commit -m "refine: update Freshdesk theme with accurate component selectors"
```

---

### Task 8: Build, deploy, and test

**Step 1: Build frontend**

```bash
cd /home/ubuntu/libredesk/frontend
NODE_OPTIONS='--max-old-space-size=4096' pnpm build
```

**Step 2: Build Go binary**

```bash
cd /home/ubuntu/libredesk
VERSION=$(git describe --tags --always)
CGO_ENABLED=0 go build -ldflags "-s -w -X 'main.buildString=$VERSION' -X 'main.versionString=$VERSION'" -o libredesk ./cmd/
```

**Step 3: Stuff assets**

```bash
/home/ubuntu/go/bin/stuffbin -a stuff -in libredesk -out libredesk frontend/dist i18n schema.sql static
```

**Step 4: Deploy**

```bash
docker compose build --no-cache app
docker compose up -d --force-recreate app
```

**Step 5: Test**

1. Open the app — should load with default theme
2. Click the palette icon in the sidebar footer
3. Switch to "Freshdesk" theme — verify:
   - Teal color scheme applied
   - Dark sidebar rail
   - Better contrast on text
   - Conversation list hides when clicking a ticket
   - Back button appears and works
   - Sidebar starts collapsed
4. Switch back to "Default" — verify everything reverts
5. Refresh page — verify theme persists

**Step 6: Commit any fixes**

```bash
git add -A
git commit -m "fix: theme deployment adjustments"
```
