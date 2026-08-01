import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'
import { contactsService } from '@/services/api'

export interface UseChatLifecycleOptions {
  /** i18n translator. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string; assigned_user_id?: string } | null
    selectedContactIds: Set<string>
    setCurrentContact: (c: any) => void
    fetchContacts: () => Promise<void>
    claimChat: (id: string) => Promise<void>
    joinChat: (id: string) => Promise<void>
    leaveChat: (id: string) => Promise<void>
    releaseChat: (id: string) => Promise<void>
    closeChat: (id: string) => Promise<void>
    reopenChat: (id: string) => Promise<void>
    bulkReleaseChats: (ids: string[]) => Promise<unknown>
    inviteCollaborator: (id: string, userId: string) => Promise<void>
    removeCollaborator: (id: string, userId: string) => Promise<void>
  }
  /** Auth store reactive surface. */
  authStore: {
    userRole: string | undefined
    user?: { id: string; full_name?: string } | null
    hasPermission: (resource: string, action: string) => boolean
  }
  /** Users store reactive surface. */
  usersStore: {
    users: Array<{ id: string; full_name: string; email: string; is_active: boolean; role?: { name?: string } }>
    fetchUsers: () => Promise<unknown>
  }
}

/**
 * Chat lifecycle actions (claim / join / leave / release / close / reopen),
 * bulk release, collaborator invite/remove, and contact assignment. Owns the
 * busy flags and the Assign/Invite dialog state shared by both dialogs.
 *
 * @example
 * ```ts
 * const life = useChatLifecycle({ contactsStore, authStore, usersStore, t })
 * ```
 */
export function useChatLifecycle(options: UseChatLifecycleOptions) {
  const { t, contactsStore, authStore } = options

  // ─── Chat lifecycle: claim & collaboration ───
  const isClaiming = ref(false)
  const isJoining = ref(false)
  const isAssignDialogOpen = ref(false)
  const isInviteDialogOpen = ref(false)

  // Search state for assignment dialog
  const assignSearchQuery = ref('')

  async function handleClaim() {
    if (!contactsStore.currentContact) return
    isClaiming.value = true
    try {
      await contactsStore.claimChat(contactsStore.currentContact.id)
    } catch (error: any) {
      if (error.response?.status === 409) {
        console.error('Chat already assigned:', error.response.data?.message)
      }
    } finally {
      isClaiming.value = false
    }
  }

  async function handleJoin() {
    if (!contactsStore.currentContact) return
    isJoining.value = true
    try {
      await contactsStore.joinChat(contactsStore.currentContact.id)
    } catch {
      console.error('Failed to join chat')
    } finally {
      isJoining.value = false
    }
  }

  async function handleLeave() {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.leaveChat(contactsStore.currentContact.id)
    } catch {
      console.error('Failed to leave chat')
    }
  }

  // Release returns the conversation to pending without closing it. Mirrors
  // handleLeave's shape (currentContact guard + try/catch).
  async function handleRelease() {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.releaseChat(contactsStore.currentContact.id)
    } catch {
      console.error('Failed to release chat')
    }
  }

  async function handleClose() {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.closeChat(contactsStore.currentContact.id)
    } catch {
      console.error('Failed to close chat')
    }
  }

  async function handleReopen() {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.reopenChat(contactsStore.currentContact.id)
    } catch {
      console.error('Failed to reopen chat')
    }
  }

  // Bulk release (M4). Wraps the store action with the standard try/catch +
  // error log used by the other lifecycle handlers.
  async function handleBulkRelease() {
    const ids = Array.from(contactsStore.selectedContactIds)
    if (!ids.length) return
    try {
      await contactsStore.bulkReleaseChats(ids)
    } catch {
      console.error('Failed to bulk release chats')
    }
  }

  async function handleInvite(userId: string) {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.inviteCollaborator(contactsStore.currentContact.id, userId)
    } catch {
      console.error('Failed to invite collaborator')
    }
  }

  async function handleRemoveCollaborator(userId: string) {
    if (!contactsStore.currentContact) return
    try {
      await contactsStore.removeCollaborator(contactsStore.currentContact.id, userId)
    } catch {
      console.error('Failed to remove collaborator')
    }
  }

  // Check if current user can assign contacts (admin or manager only)
  const canAssignContacts = computed(() => {
    // Try store first, then fallback to localStorage
    let role = authStore.userRole
    if (!role || role === 'agent') {
      try {
        const storedUser = localStorage.getItem('user')
        if (storedUser) {
          const user = JSON.parse(storedUser)
          role = user.role?.name || user.role // Support both old and new format
        }
      } catch {
        // ignore
      }
    }
    return role === 'admin' || role === 'manager'
  })

  // Get list of users for assignment
  const assignableUsers = computed(() => {
    return options.usersStore.users.filter(u => u.is_active)
  })

  // Filtered users for assignment dialog (shared by Assign + Invite dialogs)
  const filteredAssignableUsers = computed(() => {
    const query = assignSearchQuery.value.toLowerCase().trim()
    if (!query) return assignableUsers.value
    return assignableUsers.value.filter(u =>
      u.full_name.toLowerCase().includes(query) ||
      u.email.toLowerCase().includes(query)
    )
  })

  async function assignContactToUser(userId: string | null) {
    if (!contactsStore.currentContact) return

    try {
      await contactsService.assign(contactsStore.currentContact.id, userId)
      toast.success(userId ? t('chat.contactAssigned') : t('chat.contactUnassigned'))
      // Update current contact with new assignment
      contactsStore.setCurrentContact({
        ...contactsStore.currentContact,
        assigned_user_id: userId || undefined
      })
      // Refresh contacts list
      await contactsStore.fetchContacts()
    } catch (error: any) {
      const message = error.response?.data?.message || t('chat.assignFailed')
      toast.error(message)
    }
  }

  return {
    // State
    isClaiming,
    isJoining,
    isAssignDialogOpen,
    isInviteDialogOpen,
    assignSearchQuery,
    canAssignContacts,
    filteredAssignableUsers,
    // Actions
    handleClaim,
    handleJoin,
    handleLeave,
    handleRelease,
    handleClose,
    handleReopen,
    handleBulkRelease,
    handleInvite,
    handleRemoveCollaborator,
    assignContactToUser,
  }
}
