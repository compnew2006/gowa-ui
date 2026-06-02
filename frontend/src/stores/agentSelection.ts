import { defineStore } from "pinia";
import { ref } from "vue";
import {
  agentSelectionService,
  type AgentSelectionAuditEvent,
  type AgentSelectionMenuPreview,
  type AgentSelectionOption,
  type AgentSelectionParticipant,
  type AgentSelectionSession,
  type AgentSelectionSettings,
} from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";

export const useAgentSelectionStore = defineStore("agentSelection", () => {
  const settings = ref<AgentSelectionSettings | null>(null);
  const participants = ref<AgentSelectionParticipant[]>([]);
  const options = ref<AgentSelectionOption[]>([]);
  const preview = ref<AgentSelectionMenuPreview | null>(null);
  const sessions = ref<AgentSelectionSession[]>([]);
  const auditEvents = ref<AgentSelectionAuditEvent[]>([]);
  const loading = ref(false);

  async function fetchSettings(instanceId?: string) {
    loading.value = true;
    try {
      const response = await agentSelectionService.getSettings(
        instanceId ? { instance_id: instanceId } : undefined,
      );
      const data = unwrapResponse<{ settings: AgentSelectionSettings }>(
        response,
      );
      settings.value = data.settings;
      return data.settings;
    } finally {
      loading.value = false;
    }
  }

  async function saveSettings(data: Partial<AgentSelectionSettings>) {
    const response = await agentSelectionService.updateSettings(data);
    const payload = unwrapResponse<{ settings: AgentSelectionSettings }>(
      response,
    );
    settings.value = payload.settings;
    return payload.settings;
  }

  async function fetchParticipants(settingsId?: string) {
    const response = await agentSelectionService.listParticipants(
      settingsId ? { settings_id: settingsId } : undefined,
    );
    const data = unwrapResponse<{ participants: AgentSelectionParticipant[] }>(
      response,
    );
    participants.value = data.participants || [];
    return participants.value;
  }

  async function createParticipant(
    data: Omit<
      AgentSelectionParticipant,
      "id" | "organization_id" | "user"
    >,
  ) {
    const response = await agentSelectionService.createParticipant(data);
    const payload = unwrapResponse<{ participant: AgentSelectionParticipant }>(
      response,
    );
    participants.value = [...participants.value, payload.participant].sort(
      bySortOrderThenName,
    );
    return payload.participant;
  }

  async function updateParticipant(
    id: string,
    data: Partial<AgentSelectionParticipant>,
  ) {
    const response = await agentSelectionService.updateParticipant(id, data);
    const payload = unwrapResponse<{ participant: AgentSelectionParticipant }>(
      response,
    );
    participants.value = participants.value
      .map((participant) =>
        participant.id === id ? payload.participant : participant,
      )
      .sort(bySortOrderThenName);
    return payload.participant;
  }

  async function deleteParticipant(id: string) {
    await agentSelectionService.deleteParticipant(id);
    participants.value = participants.value.filter((item) => item.id !== id);
  }

  async function fetchOptions(settingsId?: string) {
    const response = await agentSelectionService.listOptions(
      settingsId ? { settings_id: settingsId } : undefined,
    );
    const data = unwrapResponse<{ options: AgentSelectionOption[] }>(response);
    options.value = data.options || [];
    return options.value;
  }

  async function createOption(
    data: Omit<AgentSelectionOption, "id" | "organization_id">,
  ) {
    const response = await agentSelectionService.createOption(data);
    const payload = unwrapResponse<{ option: AgentSelectionOption }>(response);
    options.value = [...options.value, payload.option].sort(
      bySortOrderThenLabel,
    );
    return payload.option;
  }

  async function updateOption(id: string, data: Partial<AgentSelectionOption>) {
    const response = await agentSelectionService.updateOption(id, data);
    const payload = unwrapResponse<{ option: AgentSelectionOption }>(response);
    options.value = options.value
      .map((option) => (option.id === id ? payload.option : option))
      .sort(bySortOrderThenLabel);
    return payload.option;
  }

  async function deleteOption(id: string) {
    await agentSelectionService.deleteOption(id);
    options.value = options.value.filter((option) => option.id !== id);
  }

  async function fetchPreview(settingsId?: string) {
    const response = await agentSelectionService.preview({
      settings_id: settingsId,
    });
    const data = unwrapResponse<{ menu: AgentSelectionMenuPreview }>(response);
    preview.value = data.menu;
    return data.menu;
  }

  async function testSend(contactId: string, settingsId?: string) {
    const response = await agentSelectionService.testSend({
      settings_id: settingsId,
      contact_id: contactId,
    });
    return unwrapResponse<{
      sent: boolean;
      whatsapp_account: string;
      contact_id: string;
      menu_text: string;
      option_count: number;
      outbound_message_id?: string;
    }>(response);
  }

  async function fetchSessions(params?: { status?: string }) {
    const response = await agentSelectionService.listSessions(params);
    const data = unwrapResponse<{ sessions: AgentSelectionSession[] }>(
      response,
    );
    sessions.value = data.sessions || [];
    return sessions.value;
  }

  async function fetchAudit(params?: { event_type?: string }) {
    const response = await agentSelectionService.listAudit(params);
    const data = unwrapResponse<{ events: AgentSelectionAuditEvent[] }>(
      response,
    );
    auditEvents.value = data.events || [];
    return auditEvents.value;
  }

  async function cancelSession(id: string) {
    const response = await agentSelectionService.cancelSession(id);
    const data = unwrapResponse<{ session: AgentSelectionSession }>(response);
    sessions.value = sessions.value.map((session) =>
      session.id === id ? data.session : session,
    );
    return data.session;
  }

  return {
    settings,
    participants,
    options,
    preview,
    sessions,
    auditEvents,
    loading,
    fetchSettings,
    saveSettings,
    fetchParticipants,
    createParticipant,
    updateParticipant,
    deleteParticipant,
    fetchOptions,
    createOption,
    updateOption,
    deleteOption,
    fetchPreview,
    testSend,
    fetchSessions,
    fetchAudit,
    cancelSession,
  };
});

function bySortOrderThenName(
  a: AgentSelectionParticipant,
  b: AgentSelectionParticipant,
) {
  return a.sort_order - b.sort_order || a.display_name.localeCompare(b.display_name);
}

function bySortOrderThenLabel(a: AgentSelectionOption, b: AgentSelectionOption) {
  return a.sort_order - b.sort_order || a.label.localeCompare(b.label);
}
