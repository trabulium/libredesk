<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="group relative block px-4 p-4 transition-all duration-200 ease-in-out cursor-pointer hover:bg-accent/20 dark:hover:bg-accent/60"
        :class="{
          'bg-accent/60': conversation.uuid === currentConversation?.uuid,
          'bg-primary/5': isItemSelected && conversation.uuid !== currentConversation?.uuid
        }"
      >
        <div class="flex items-start gap-3">
          <!-- Checkbox -->
          <div class="flex items-center pt-3" @click.prevent.stop="handleCheckboxClick">
            <Checkbox
              :checked="isItemSelected"
            />
          </div>

          <!-- Avatar -->
          <Avatar class="w-12 h-12 rounded-full shadow shrink-0">
            <AvatarImage
              :src="conversation.contact.avatar_url || ''"
              class="object-cover"
              v-if="conversation.contact.avatar_url || ''"
            />
            <AvatarFallback>
              {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
            </AvatarFallback>
          </Avatar>

          <!-- Content (left) -->
          <div class="flex-1 min-w-0 space-y-1.5">
            <!-- Subject + Reference Number row -->
            <div class="flex items-center gap-1.5 min-w-0" v-if="conversation.subject || conversation.reference_number">
              <span
                v-if="conversation.reference_number"
                class="text-xs font-medium text-muted-foreground whitespace-nowrap"
              >#{{ conversation.reference_number }}</span>
              <h3 class="text-sm font-semibold truncate conversation-subject">
                {{ conversation.subject || 'No subject' }}
              </h3>
            </div>

            <!-- Contact name + badges -->
            <div class="flex items-center gap-1.5 min-w-0">
              <span class="text-xs text-muted-foreground truncate">
                {{ contactFullName }}
              </span>
              <Pencil
                v-if="hasDraftForConversation"
                class="w-3 h-3 text-muted-foreground flex-shrink-0"
              />
              <!-- Status badge -->
              <span
                v-if="conversation.status"
                class="conversation-status-badge text-[10px] font-medium px-1.5 py-0.5 rounded-full whitespace-nowrap"
                :class="statusClass"
              >{{ conversation.status }}</span>
              <!-- Priority badge -->
              <span
                v-if="conversation.priority && conversation.priority !== 'None'"
                class="text-[10px] font-medium px-1.5 py-0.5 rounded-full whitespace-nowrap"
                :class="priorityClass"
              >{{ conversation.priority }}</span>
            </div>

            <!-- Inbox name -->
            <p class="text-xs text-gray-400 flex items-center gap-1.5">
              <Mail class="w-3.5 h-3.5 text-gray-400/80" />
              <span>{{ conversation.inbox_name }}</span>
            </p>

            <!-- Message preview -->
            <div
              class="text-sm flex items-center gap-1.5 break-all text-gray-600 dark:text-gray-300"
            >
              <Reply
                class="text-green-600 flex-shrink-0"
                size="15"
                v-if="conversation.last_message_sender === 'agent'"
              />
              {{ trimmedLastMessage }}
            </div>

            <!-- SLA Badges -->
            <div class="flex items-center">
              <div :class="getSlaClass(frdStatus)">
                <SlaBadge
                  :dueAt="conversation.first_response_deadline_at"
                  :actualAt="conversation.first_reply_at"
                  :label="'FRD'"
                  :showExtra="false"
                  @status="frdStatus = $event"
                  :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
                />
              </div>
              <div :class="getSlaClass(rdStatus)">
                <SlaBadge
                  :dueAt="conversation.resolution_deadline_at"
                  :actualAt="conversation.resolved_at"
                  :label="'RD'"
                  :showExtra="false"
                  @status="rdStatus = $event"
                  :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
                />
              </div>
              <div :class="getSlaClass(nrdStatus)">
                <SlaBadge
                  :dueAt="conversation.next_response_deadline_at"
                  :actualAt="conversation.next_response_met_at"
                  :label="'NRD'"
                  :showExtra="false"
                  @status="nrdStatus = $event"
                  :key="`${conversation.uuid}-${conversation.next_response_deadline_at}-${conversation.next_response_met_at}`"
                />
              </div>
            </div>
          </div>

          <!-- Right column: 2x2 grid — assignments left, time+unread right -->
          <div class="shrink-0 grid grid-cols-[auto_auto] gap-x-3 gap-y-1.5 items-center pt-1" @click.prevent.stop>
            <!-- Row 1: Agent | Time -->
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  class="text-xs flex items-center gap-1 py-1 px-1 justify-end hover:text-foreground transition-colors cursor-pointer"
                  :class="conversation.assigned_user_name ? 'text-muted-foreground' : 'text-orange-500 dark:text-orange-400'"
                >
                  <User class="w-3 h-3" />
                  {{ conversation.assigned_user_name || 'Unassigned' }}
                  <ChevronDown class="w-2.5 h-2.5 opacity-50" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="max-h-60 overflow-y-auto">
                <DropdownMenuItem
                  v-if="conversation.assigned_user_name"
                  @click="unassignAgent"
                  class="text-xs text-muted-foreground"
                >
                  None
                </DropdownMenuItem>
                <DropdownMenuSeparator v-if="conversation.assigned_user_name" />
                <DropdownMenuItem
                  v-for="agent in usersStore.options"
                  :key="'agent-' + agent.value"
                  @click="assignAgent(agent)"
                  class="text-xs"
                >
                  {{ agent.label }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <span
              class="text-xs text-gray-400 whitespace-nowrap text-right"
            >
              {{ conversation.last_message_at ? relativeLastMessageTime : '' }}
            </span>

            <!-- Row 2: Team | Unread -->
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  class="text-xs flex items-center gap-1 py-1 px-1 justify-end hover:text-foreground transition-colors cursor-pointer"
                  :class="conversation.assigned_team_name ? 'text-muted-foreground' : 'text-muted-foreground/50'"
                >
                  <Users class="w-3 h-3" />
                  {{ conversation.assigned_team_name || 'No team' }}
                  <ChevronDown class="w-2.5 h-2.5 opacity-50" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="max-h-60 overflow-y-auto">
                <DropdownMenuItem
                  v-if="conversation.assigned_team_name"
                  @click="unassignTeam"
                  class="text-xs text-muted-foreground"
                >
                  None
                </DropdownMenuItem>
                <DropdownMenuSeparator v-if="conversation.assigned_team_name" />
                <DropdownMenuItem
                  v-for="team in teamsStore.options"
                  :key="'team-' + team.value"
                  @click="assignTeam(team)"
                  class="text-xs"
                >
                  {{ team.label }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <div class="flex justify-end">
              <div
                v-if="conversation.unread_message_count > 0"
                class="flex items-center justify-center w-6 h-6 bg-green-600 text-white text-xs font-medium rounded-full"
              >
                {{ conversation.unread_message_count }}
              </div>
            </div>
          </div>
        </div>
      </router-link>
    </ContextMenuTrigger>
    <ContextMenuContent>
      <ContextMenuItem @click="handleMarkAsUnread">
        <MailOpen class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.markAsUnread') }}
      </ContextMenuItem>
    </ContextMenuContent>
  </ContextMenu>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { getRelativeTime } from '@/utils/datetime'
import { Mail, Reply, Pencil, MailOpen, User, Users, ChevronDown } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@/components/ui/context-menu'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { Checkbox } from '@/components/ui/checkbox'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

let timer = null
const now = ref(new Date())
const route = useRoute()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const frdStatus = ref('')
const rdStatus = ref('')
const nrdStatus = ref('')

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const handleMarkAsUnread = () => {
  conversationStore.markAsUnread(props.conversation.uuid)
}

const statusClass = computed(() => {
  const s = (props.conversation.status || '').toLowerCase()
  switch (s) {
    case 'open':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'replied':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'resolved':
      return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    case 'closed':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
    case 'snoozed':
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
})

const priorityClass = computed(() => {
  const p = (props.conversation.priority || '').toLowerCase()
  switch (p) {
    case 'urgent':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    case 'high':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'medium':
      return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'low':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
})

const conversationRoute = computed(() => {
  const baseRoute = route.name.includes('team')
    ? 'team-inbox-conversation'
    : route.name.includes('view')
      ? 'view-inbox-conversation'
      : 'inbox-conversation'
  return {
    name: baseRoute,
    params: {
      uuid: props.conversation.uuid,
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
    },
    query: props.conversation.mentioned_message_uuid
      ? { scrollTo: props.conversation.mentioned_message_uuid }
      : {}
  }
})

onMounted(() => {
  timer = setInterval(() => {
    now.value = new Date()
  }, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const trimmedLastMessage = computed(() => {
  const message = props.conversation.last_message || ''
  return message.length > 100 ? message.slice(0, 100) + '...' : message
})

const getSlaClass = (status) => (['overdue', 'remaining'].includes(status) ? 'mr-2' : '')

const relativeLastMessageTime = computed(() => {
  return props.conversation.last_message_at
    ? getRelativeTime(props.conversation.last_message_at, now.value)
    : ''
})

const hasDraftForConversation = computed(() => {
  return conversationStore.hasDraft(props.conversation.uuid)
})

const isItemSelected = computed(() => {
  return conversationStore.isSelected(props.conversation.uuid)
})

const handleCheckboxClick = (event) => {
  conversationStore.toggleSelect(props.conversation.uuid, event.shiftKey)
}

const assignAgent = async (agent) => {
  try {
    await api.updateAssignee(props.conversation.uuid, 'user', { assignee_id: parseInt(agent.value) })
    props.conversation.assigned_user_name = agent.label
  } catch (error) {
    handleHTTPError(error)
  }
}

const unassignAgent = async () => {
  try {
    await api.removeAssignee(props.conversation.uuid, 'user')
    props.conversation.assigned_user_name = null
  } catch (error) {
    handleHTTPError(error)
  }
}

const assignTeam = async (team) => {
  try {
    await api.updateAssignee(props.conversation.uuid, 'team', { assignee_id: parseInt(team.value) })
    props.conversation.assigned_team_name = team.label
  } catch (error) {
    handleHTTPError(error)
  }
}

const unassignTeam = async () => {
  try {
    await api.removeAssignee(props.conversation.uuid, 'team')
    props.conversation.assigned_team_name = null
  } catch (error) {
    handleHTTPError(error)
  }
}
</script>
