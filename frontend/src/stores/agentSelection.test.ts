// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  agentSelectionService: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    listParticipants: vi.fn(),
    addParticipant: vi.fn(),
    deleteParticipant: vi.fn(),
    listOptions: vi.fn(),
    addOption: vi.fn(),
    deleteOption: vi.fn(),
    preview: vi.fn(),
    listSessions: vi.fn(),
    cancelSession: vi.fn(),
    listAudit: vi.fn(),
    testSend: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  agentSelectionService: mocks.agentSelectionService,
}));

import { useAgentSelectionStore } from "./agentSelection";

describe("agentSelection store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("fetchSettings stores the settings payload", async () => {
    const payload = {
      settings: {
        id: "s-1",
        organization_id: "org-1",
        instance_id: null,
        allowed_instance_ids: [],
        enabled: true,
        trigger_mode: "first_pending_message" as const,
        trigger_keywords: [],
        prompt_delay_minutes: 3,
        selection_timeout_minutes: 10,
        max_invalid_attempts: 3,
        menu_header_text: "Header",
        menu_footer_text: "",
        invalid_reply_text: "Invalid",
        timeout_response_text: "",
        unavailable_agent_text: "",
        custom_final_option_enabled: false,
        custom_final_option_text: "",
        hide_unavailable_agents: true,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    };
    mocks.agentSelectionService.getSettings.mockResolvedValueOnce({
      data: payload,
    });

    const store = useAgentSelectionStore();
    await store.fetchSettings();

    expect(mocks.agentSelectionService.getSettings).toHaveBeenCalledWith(
      undefined,
    );
    expect(store.settings?.id).toBe("s-1");
    expect(store.loading).toBe(false);
  });

  it("updateSettings forwards payload and stores the new settings", async () => {
    const newSettings = {
      id: "s-1",
      organization_id: "org-1",
      instance_id: null,
      allowed_instance_ids: ["wa-1"],
      enabled: true,
      trigger_mode: "keyword" as const,
      trigger_keywords: ["support"],
      prompt_delay_minutes: 5,
      selection_timeout_minutes: 15,
      max_invalid_attempts: 5,
      menu_header_text: "H",
      menu_footer_text: "F",
      invalid_reply_text: "X",
      timeout_response_text: "T",
      unavailable_agent_text: "U",
      custom_final_option_enabled: true,
      custom_final_option_text: "C",
      hide_unavailable_agents: false,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-02T00:00:00Z",
    };
    mocks.agentSelectionService.updateSettings.mockResolvedValueOnce({
      data: { settings: newSettings },
    });

    const store = useAgentSelectionStore();
    const updatePayload = {
      enabled: true,
      trigger_mode: "keyword" as const,
      trigger_keywords: ["support"],
      prompt_delay_minutes: 5,
      selection_timeout_minutes: 15,
      max_invalid_attempts: 5,
      menu_header_text: "H",
      menu_footer_text: "F",
      invalid_reply_text: "X",
      timeout_response_text: "T",
      unavailable_agent_text: "U",
      custom_final_option_enabled: true,
      custom_final_option_text: "C",
      allowed_instance_ids: ["wa-1"],
      hide_unavailable_agents: false,
    };
    await store.saveSettings(updatePayload);

    expect(mocks.agentSelectionService.updateSettings).toHaveBeenCalledWith(
      updatePayload,
    );
    expect(store.settings?.trigger_mode).toBe("keyword");
    expect(store.settings?.allowed_instance_ids).toEqual(["wa-1"]);
  });

  it("testSend calls the service and returns the result", async () => {
    const expected = {
      sent: true,
      whatsapp_account: "wa-1",
      contact_id: "c-1",
      menu_text: "1) Agent A\n2) Agent B",
      option_count: 2,
      outbound_message_id: "msg-1",
    };
    mocks.agentSelectionService.testSend.mockResolvedValueOnce({
      data: expected,
    });

    const store = useAgentSelectionStore();
    const result = await store.testSend("c-1", "s-1");

    expect(mocks.agentSelectionService.testSend).toHaveBeenCalledWith({
      contact_id: "c-1",
      settings_id: "s-1",
    });
    expect(result).toEqual(expected);
  });

  it("testSend propagates service errors", async () => {
    const err = new Error("no active account");
    mocks.agentSelectionService.testSend.mockRejectedValueOnce(err);

    const store = useAgentSelectionStore();
    await expect(store.testSend("c-1")).rejects.toBe(err);
    expect(mocks.agentSelectionService.testSend).toHaveBeenCalledWith({
      contact_id: "c-1",
      settings_id: undefined,
    });
  });

  it("fetchAudit and fetchSessions store their lists", async () => {
    const sessions = [
      {
        id: "ses-1",
        organization_id: "org-1",
        instance_id: null,
        contact_id: "c-1",
        whatsapp_account: "wa-1",
        status: "menu_sent" as const,
        prompt_due_at: "2026-01-01T00:00:00Z",
        expires_at: "2026-01-01T00:10:00Z",
        menu_snapshot: null,
        selected_option_id: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];
    const events = [
      {
        id: "ev-1",
        organization_id: "org-1",
        instance_id: null,
        session_id: "ses-1",
        event_type: "menu_sent",
        actor_type: "system",
        actor_id: null,
        metadata: {},
        created_at: "2026-01-01T00:00:00Z",
      },
    ];
    mocks.agentSelectionService.listSessions.mockResolvedValueOnce({
      data: { sessions },
    });
    mocks.agentSelectionService.listAudit.mockResolvedValueOnce({
      data: { events },
    });

    const store = useAgentSelectionStore();
    await store.fetchSessions();
    await store.fetchAudit();

    expect(store.sessions).toHaveLength(1);
    expect(store.sessions[0].id).toBe("ses-1");
    expect(store.auditEvents).toHaveLength(1);
    expect(store.auditEvents[0].event_type).toBe("menu_sent");
  });
});
