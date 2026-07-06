// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import type { AgentTransfer } from "@/types/transfers";
import { useAuthStore } from "./auth";

const mocks = vi.hoisted(() => ({
  chatbotService: {
    listTransfers: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  chatbotService: mocks.chatbotService,
}));

import { useTransfersStore } from "./transfers";

function makeTransfer(overrides: Partial<AgentTransfer> = {}): AgentTransfer {
  return {
    id: overrides.id ?? "transfer-1",
    contact_id: overrides.contact_id ?? "contact-1",
    contact_name: overrides.contact_name ?? "Customer",
    phone_number: overrides.phone_number ?? "201234567890",
    whatsapp_account: overrides.whatsapp_account ?? "Support",
    status: overrides.status ?? "active",
    source: overrides.source ?? "manual",
    transferred_at: overrides.transferred_at ?? new Date().toISOString(),
    sla_breached: overrides.sla_breached ?? false,
    escalation_level: overrides.escalation_level ?? 0,
    agent_id: overrides.agent_id,
    team_id: overrides.team_id,
    instance_id: overrides.instance_id,
  };
}

function setTransferReader(permissions: string[] = ["transfers:read"]) {
  const authStore = useAuthStore();
  authStore.user = {
    id: "user-1",
    email: "agent@example.com",
    full_name: "Agent",
    organization_id: "org-1",
    role: {
      id: "role-1",
      name: "Agent",
      permissions,
    },
  };
}

describe("useTransfersStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("skips active transfer fetches when the user cannot read transfers", async () => {
    const store = useTransfersStore();
    store.transfers = [makeTransfer()];
    store.generalQueueCount = 2;
    store.teamQueueCounts = { teamA: 3 };
    store.totalCount = 5;

    await store.fetchTransfers({ status: "active" });

    expect(mocks.chatbotService.listTransfers).not.toHaveBeenCalled();
    expect(store.transfers).toEqual([]);
    expect(store.generalQueueCount).toBe(0);
    expect(store.teamQueueCounts).toEqual({});
    expect(store.totalCount).toBe(0);
  });

  it("loads transfers when the user has transfer read permission", async () => {
    setTransferReader();
    const store = useTransfersStore();

    mocks.chatbotService.listTransfers.mockResolvedValueOnce({
      data: {
        data: {
          transfers: [makeTransfer({ id: "transfer-live" })],
          general_queue_count: 1,
          team_queue_counts: { teamA: 2 },
          total_count: 3,
        },
      },
    });

    await store.fetchTransfers({ status: "active" });

    expect(mocks.chatbotService.listTransfers).toHaveBeenCalledWith({
      status: "active",
    });
    expect(store.transfers).toHaveLength(1);
    expect(store.transfers[0].id).toBe("transfer-live");
    expect(store.generalQueueCount).toBe(1);
    expect(store.teamQueueCounts).toEqual({ teamA: 2 });
    expect(store.totalCount).toBe(3);
  });

  it("clears transfer history state when the user cannot read transfers", async () => {
    const store = useTransfersStore();
    store.historyTransfers = [
      makeTransfer({ id: "history-1", status: "resumed" }),
    ];
    store.historyTotalCount = 4;

    await store.fetchHistory();

    expect(mocks.chatbotService.listTransfers).not.toHaveBeenCalled();
    expect(store.historyTransfers).toEqual([]);
    expect(store.historyTotalCount).toBe(0);
  });
});
