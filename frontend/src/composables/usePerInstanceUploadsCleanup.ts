import { useQuery, useMutation, useQueryClient } from "@tanstack/vue-query";
import { instancesService } from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";
import { computed, type Ref } from "vue";

export interface InstanceUploadsCleanupSettings {
  instance_id: string;
  inherit: boolean;
  retention_days: number | null;
  effective_retention_days: number | null;
  effective_source: "custom" | "default" | "disabled";
  last_run_date: string | null;
}

export interface InstanceUploadsCleanupHistoryEntry {
  id: string;
  created_at: string;
  actor_email: string | null;
  old_inherit: boolean | null;
  new_inherit: boolean;
  old_retention_days: number | null;
  new_retention_days: number | null;
  reason: string | null;
}

export interface InstanceUploadsCleanupRunResult {
  instance_id: string;
  instance_name: string;
  deleted_files: number;
  retention_used: number;
}

export function useInstanceUploadsCleanup(instanceId: Ref<string>) {
  const query = useQuery({
    queryKey: computed(() => ["instance-uploads-cleanup", instanceId.value]),
    queryFn: async () => {
      const res = await instancesService.getInstanceUploadsCleanup(instanceId.value);
      return unwrapResponse<InstanceUploadsCleanupSettings>(res);
    },
    enabled: computed(() => !!instanceId.value),
  });

  return query;
}

export function useUpdateInstanceUploadsCleanup(instanceId: Ref<string>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      inherit: boolean;
      retention_days?: number;
      reason?: string;
    }) => {
      const res = await instancesService.updateInstanceUploadsCleanup(
        instanceId.value,
        data,
      );
      return unwrapResponse<InstanceUploadsCleanupSettings>(res);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["instance-uploads-cleanup", instanceId.value],
      });
      queryClient.invalidateQueries({
        queryKey: ["instance-uploads-cleanup-history", instanceId.value],
      });
      queryClient.invalidateQueries({
        queryKey: ["org-uploads-cleanup-overview"],
      });
    },
  });
}

export function useInstanceUploadsCleanupHistory(
  instanceId: Ref<string>,
  limit: number = 5,
) {
  return useQuery({
    queryKey: computed(() => [
      "instance-uploads-cleanup-history",
      instanceId.value,
      limit,
    ]),
    queryFn: async () => {
      const res = await instancesService.getInstanceUploadsCleanupHistory(
        instanceId.value,
        { limit },
      );
      return unwrapResponse<{
        entries: InstanceUploadsCleanupHistoryEntry[];
        total: number;
      }>(res);
    },
    enabled: computed(() => !!instanceId.value),
  });
}

export function useRunInstanceUploadsCleanup(instanceId: Ref<string>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const res = await instancesService.runInstanceUploadsCleanup(
        instanceId.value,
      );
      return unwrapResponse<InstanceUploadsCleanupRunResult>(res);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["instance-uploads-cleanup", instanceId.value],
      });
    },
  });
}

export function useOrgUploadsCleanupOverview(
  params?: Ref<{
    limit?: number;
    offset?: number;
    q?: string;
    source?: string;
  }>,
) {
  return useQuery({
    queryKey: computed(() => [
      "org-uploads-cleanup-overview",
      params?.value?.limit,
      params?.value?.offset,
      params?.value?.q,
      params?.value?.source,
    ]),
    queryFn: async () => {
      const res = await instancesService.getOrgUploadsCleanupOverview(
        params?.value,
      );
      return unwrapResponse<{
        items: Array<{
          instance_id: string;
          instance_name: string;
          effective_retention_days: number | null;
          effective_source: string;
          last_run_date: string | null;
        }>;
        total: number;
        limit: number;
        offset: number;
      }>(res);
    },
  });
}
