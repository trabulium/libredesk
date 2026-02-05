<template>
  <div v-if="!isSearchRoute" class="h-screen w-full flex flex-col">

    <!-- Freshdesk theme: show one panel at a time (list OR detail) -->
    <template v-if="hideListOnTicketOpen">
      <!-- Back button when viewing a ticket -->
      <div
        v-if="hasConversationOpen"
        class="flex items-center px-3 py-1.5 border-b bg-background shrink-0"
      >
        <button
          @click="goBack"
          class="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft class="h-4 w-4" />
          Back to conversations
        </button>
      </div>

      <!-- Full-width conversation list (no ticket selected) -->
      <div v-show="!hasConversationOpen" class="flex-1 overflow-y-auto">
        <ConversationList />
      </div>

      <!-- Full-width ticket detail (ticket selected) -->
      <!-- Always mounted via v-show so InboxView fetches data -->
      <div v-show="hasConversationOpen" class="flex-1 overflow-hidden">
        <router-view v-slot="{ Component }">
          <component :is="Component" />
        </router-view>
      </div>

      <!-- Hidden: InboxView must always be mounted to fetch list data -->
      <div v-show="false" v-if="!hasConversationOpen">
        <router-view />
      </div>
    </template>

    <!-- Default theme: original resizable panels -->
    <template v-else>
      <ResizablePanelGroup
        direction="horizontal"
        class="flex-1"
        @layout="onLayoutChange"
      >
        <ResizablePanel
          :default-size="panelSizes[0]"
          :min-size="25"
          :max-size="35"
          class="overflow-y-auto"
        >
          <ConversationList />
        </ResizablePanel>

        <ResizableHandle />

        <ResizablePanel :default-size="panelSizes[1]" :min-size="30">
          <router-view v-slot="{ Component }">
            <component :is="Component" />
          </router-view>
        </ResizablePanel>
      </ResizablePanelGroup>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import { useTheme } from '@/composables/useTheme'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import { ArrowLeft } from 'lucide-vue-next'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@/components/ui/resizable'

const route = useRoute()
const router = useRouter()
const { hideListOnTicketOpen } = useTheme()

const isSearchRoute = computed(() => route.name === 'search')
const hasConversationOpen = computed(() => !!route.params.uuid)

const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}

function goBack() {
  const routeName = route.name
  if (routeName === 'team-inbox-conversation') {
    router.push({ name: 'team-inbox', params: { teamID: route.params.teamID } })
  } else if (routeName === 'view-inbox-conversation') {
    router.push({ name: 'view-inbox', params: { viewID: route.params.viewID } })
  } else {
    router.push({ name: 'inbox', params: { type: route.params.type || 'assigned' } })
  }
}
</script>
