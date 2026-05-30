import { ref, computed } from "vue";
import {
  groupParticipantsService,
  instancesService,
  campaignsService,
  type GroupParticipant,
} from "@/services/api";
import { toast } from "vue-sonner";

export interface InstanceOption {
  id: string;
  name: string;
}

export interface GroupOption {
  jid: string;
  name: string;
  participant_count: number;
}

export function useGroupParticipants() {
  // Instance & Group selection
  const instances = ref<InstanceOption[]>([]);
  const selectedInstanceId = ref("");
  const groups = ref<GroupOption[]>([]);
  const selectedGroupJid = ref("");
  const isLoadingInstances = ref(false);
  const isLoadingGroups = ref(false);

  // Participants state
  const participants = ref<GroupParticipant[]>([]);
  const isLoadingParticipants = ref(false);
  const searchQuery = ref("");

  // Selection
  const selectedJids = ref<Set<string>>(new Set());
  const selectedCount = computed(() => selectedJids.value.size);

  // Filtered participants
  const filteredParticipants = computed(() => {
    if (!searchQuery.value) return participants.value;
    const q = searchQuery.value.toLowerCase();
    return participants.value.filter(
      (p) =>
        p.jid.toLowerCase().includes(q) ||
        p.jid.replace(/@.*$/, "").includes(q),
    );
  });

  // Fetch instances
  async function fetchInstances() {
    isLoadingInstances.value = true;
    try {
      const response = await instancesService.list();
      const raw = (response.data as any)?.data ?? response.data ?? [];
      const list = Array.isArray(raw) ? raw : (raw.instances ?? []);
      instances.value = list.map((i: any) => ({
        id: i.id,
        name: i.name || i.phone_number || i.id,
      }));
    } catch {
      toast.error("Failed to load instances");
    } finally {
      isLoadingInstances.value = false;
    }
  }

  // Fetch groups for selected instance
  async function fetchGroups() {
    if (!selectedInstanceId.value) {
      groups.value = [];
      return;
    }
    isLoadingGroups.value = true;
    try {
      const response = await campaignsService.listInstanceGroups(selectedInstanceId.value);
      const raw = (response.data as any)?.data ?? response.data ?? [];
      const list = Array.isArray(raw) ? raw : [];
      groups.value = list.map((g: any) => ({
        jid: g.jid,
        name: g.name || g.jid,
        participant_count: g.participant_count ?? 0,
      }));
    } catch {
      toast.error("Failed to load groups");
    } finally {
      isLoadingGroups.value = false;
    }
  }

  // Fetch participants for selected group
  async function fetchParticipants() {
    if (!selectedInstanceId.value || !selectedGroupJid.value) {
      participants.value = [];
      return;
    }
    isLoadingParticipants.value = true;
    selectedJids.value = new Set();
    try {
      const response = await groupParticipantsService.list(
        selectedInstanceId.value,
        selectedGroupJid.value,
      );
      const raw = (response.data as any)?.data ?? response.data ?? {};
      const list = raw?.participants ?? (Array.isArray(raw) ? raw : []);
      participants.value = list;
    } catch {
      toast.error("Failed to load participants");
    } finally {
      isLoadingParticipants.value = false;
    }
  }

  // Toggle selection
  function toggleParticipant(jid: string) {
    const s = new Set(selectedJids.value);
    if (s.has(jid)) s.delete(jid);
    else s.add(jid);
    selectedJids.value = s;
  }

  function toggleAll() {
    if (selectedJids.value.size === filteredParticipants.value.length) {
      selectedJids.value = new Set();
    } else {
      selectedJids.value = new Set(filteredParticipants.value.map((p) => p.jid));
    }
  }

  function clearSelection() {
    selectedJids.value = new Set();
  }

  // Add participants
  async function addParticipants(rawInput: string) {
    if (!selectedInstanceId.value || !selectedGroupJid.value) {
      toast.error("Select an instance and group first");
      return;
    }
    const jids = parsePhoneInput(rawInput);
    if (jids.length === 0) {
      toast.error("No valid phone numbers provided");
      return;
    }
    try {
      const response = await groupParticipantsService.add(
        selectedInstanceId.value,
        selectedGroupJid.value,
        jids,
      );
      const raw = (response.data as any)?.data ?? response.data ?? {};
      toast.success(`Added ${raw?.affected ?? jids.length} participant(s)`);
      await fetchParticipants();
    } catch (e: any) {
      toast.error(e?.response?.data?.message || "Failed to add participants");
    }
  }

  // Remove participants
  async function removeSelected() {
    if (selectedJids.value.size === 0) {
      toast.error("No participants selected");
      return;
    }
    try {
      const response = await groupParticipantsService.remove(
        selectedInstanceId.value,
        selectedGroupJid.value,
        Array.from(selectedJids.value),
      );
      const raw = (response.data as any)?.data ?? response.data ?? {};
      toast.success(`Removed ${raw?.affected ?? selectedJids.value.size} participant(s)`);
      selectedJids.value = new Set();
      await fetchParticipants();
    } catch (e: any) {
      toast.error(e?.response?.data?.message || "Failed to remove participants");
    }
  }

  // Promote participants
  async function promoteSelected() {
    if (selectedJids.value.size === 0) {
      toast.error("No participants selected");
      return;
    }
    try {
      const response = await groupParticipantsService.promote(
        selectedInstanceId.value,
        selectedGroupJid.value,
        Array.from(selectedJids.value),
      );
      const raw = (response.data as any)?.data ?? response.data ?? {};
      toast.success(`Promoted ${raw?.affected ?? selectedJids.value.size} participant(s)`);
      selectedJids.value = new Set();
      await fetchParticipants();
    } catch (e: any) {
      toast.error(e?.response?.data?.message || "Failed to promote participants");
    }
  }

  // Demote participants
  async function demoteSelected() {
    if (selectedJids.value.size === 0) {
      toast.error("No participants selected");
      return;
    }
    try {
      const response = await groupParticipantsService.demote(
        selectedInstanceId.value,
        selectedGroupJid.value,
        Array.from(selectedJids.value),
      );
      const raw = (response.data as any)?.data ?? response.data ?? {};
      toast.success(`Demoted ${raw?.affected ?? selectedJids.value.size} participant(s)`);
      selectedJids.value = new Set();
      await fetchParticipants();
    } catch (e: any) {
      toast.error(e?.response?.data?.message || "Failed to demote participants");
    }
  }

  // Helper: parse comma/space/newline separated phone numbers to JIDs
  function parsePhoneInput(input: string): string[] {
    const raw = input
      .split(/[,;\n\r\t ]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    const jids: string[] = [];
    for (const num of raw) {
      const cleaned = num.replace(/[^0-9+]/g, "").replace(/^\+/, "");
      if (cleaned.length >= 7 && cleaned.length <= 15) {
        jids.push(`${cleaned}@s.whatsapp.net`);
      }
    }
    return [...new Set(jids)];
  }

  return {
    // State
    instances,
    selectedInstanceId,
    groups,
    selectedGroupJid,
    isLoadingInstances,
    isLoadingGroups,
    participants,
    isLoadingParticipants,
    searchQuery,
    selectedJids,
    selectedCount,
    filteredParticipants,
    // Methods
    fetchInstances,
    fetchGroups,
    fetchParticipants,
    toggleParticipant,
    toggleAll,
    clearSelection,
    addParticipants,
    removeSelected,
    promoteSelected,
    demoteSelected,
  };
}
