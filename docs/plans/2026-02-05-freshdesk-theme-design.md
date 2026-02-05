# Freshdesk-Inspired Theme Design

## Goal

Add a theme system to Libredesk with a Freshdesk-inspired theme that reduces visual clutter, improves contrast, and optimises the layout for smaller screens — while keeping the ability to switch back to the default upstream theme for testing compatibility.

## Architecture

Two-layer approach to minimise merge conflicts with upstream:

### Layer 1: CSS-Only Theme File

A single SCSS file (`themes/freshdesk.scss`) activated by a `.theme-freshdesk` class on the root element. Contains:

- CSS variable overrides (colors, spacing, radius)
- Targeted CSS rules for typography hierarchy, density, borders
- No changes to any `.vue` component templates

### Layer 2: Minimal Behaviour Toggles

A Vue composable (`useTheme.js`) that:

- Stores theme choice in localStorage (`libredesk-theme`)
- Applies CSS class to root element
- Exposes reactive flags: `hideListOnTicketOpen`, `collapseSidebarByDefault`
- Uses `provide/inject` so components read flags without import changes

## Visual Design

### Color Palette (Freshdesk-inspired teal)

**Light mode:**
- Primary: teal `#1E9E99` (HSL 174 62% 33%)
- Background: pure white
- Canvas (gaps): subtle cool gray `HSL 210 20% 96%`
- Borders: cooler, more visible `HSL 220 13% 91%`
- Foreground: near-black `HSL 220 14% 10%`
- Muted text: darker than default for better contrast (~5.5:1 ratio)

**Icon sidebar (dark rail like Freshdesk):**
- Background: dark charcoal `HSL 220 15% 16%`
- Text: light gray, teal accent for active items

### Typography & Density

- Conversation list: bolder contact names, lighter/smaller metadata
- More padding between list items (clearer card separation)
- Status badges get colored backgrounds
- Slightly smaller font in dense areas (14px vs 16px)

## Behaviour Changes

### 1. Hide conversation list when ticket is open

When `route.params.uuid` exists, the conversation list panel gets `v-show="false"`. The ticket detail takes full width. A back button appears to return to the list.

### 2. Sidebar collapsed by default

The existing `collapsible="offcanvas"` sidebar defaults to closed. Icon sidebar always visible. Text sidebar slides out as overlay on click.

### 3. Back button

Small back arrow in ticket header when list is hidden. Navigates to inbox list route.

## Theme Switcher

- Location: icon sidebar footer (palette icon with dropdown)
- Options: Default, Freshdesk
- Storage: localStorage only (no backend changes)
- Instant application, no reload

## Files

**New files (zero conflict risk):**
- `src/assets/styles/themes/freshdesk.scss`
- `src/composables/useTheme.js`
- `src/components/sidebar/ThemeSwitcher.vue`

**Minimal edits (~10 lines total across 3 files):**
- `App.vue` — import composable, bind theme class
- `InboxLayout.vue` — v-show on list panel + back button
- `Sidebar.vue` — read collapse default from theme
