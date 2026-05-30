import { ref } from "vue";
import {
  whatsappFilterService,
  type WhatsAppFilterBatch,
  type WhatsAppFilterResult,
} from "@/services/api";

export function useWhatsAppFilter() {
  const batches = ref<WhatsAppFilterBatch[]>([]);
  const totalBatches = ref(0);
  const isLoadingBatches = ref(false);

  const activeBatch = ref<WhatsAppFilterBatch | null>(null);
  const activeBatchResults = ref<WhatsAppFilterResult[]>([]);
  const totalResults = ref(0);
  const isLoadingResults = ref(false);
  const isSubmitting = ref(false);

  const error = ref<string | null>(null);

  async function fetchBatches(page = 1, limit = 20) {
    isLoadingBatches.value = true;
    error.value = null;
    try {
      const res = await whatsappFilterService.list({ page, limit });
      const payload = (res.data as any).data ?? res.data;
      batches.value = payload.data || [];
      totalBatches.value = payload.total || 0;
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to load verification campaigns.";
      console.error("fetchBatches error", err);
    } finally {
      isLoadingBatches.value = false;
    }
  }

  async function fetchBatchDetails(id: string) {
    error.value = null;
    try {
      const res = await whatsappFilterService.get(id);
      // fastglue wraps: { data: <batch>, status: "success" } — unwrap .data
      const batch = (res.data as any).data ?? res.data;
      activeBatch.value = batch;
      return batch;
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to load campaign details.";
      console.error("fetchBatchDetails error", err);
      return null;
    }
  }

  async function fetchBatchResults(
    id: string,
    params: {
      page: number;
      limit: number;
      status: "all" | "valid" | "invalid";
      q?: string;
    }
  ) {
    isLoadingResults.value = true;
    error.value = null;
    try {
      const res = await whatsappFilterService.listResults(id, params);
      const payload = (res.data as any).data ?? res.data;
      activeBatchResults.value = payload.data || [];
      totalResults.value = payload.total || 0;
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to load campaign results.";
      console.error("fetchBatchResults error", err);
    } finally {
      isLoadingResults.value = false;
    }
  }

  async function createCampaignJSON(connectionId: string, phones: string[], names?: string[]) {
    isSubmitting.value = true;
    error.value = null;
    try {
      const res = await whatsappFilterService.createJSON({
        connection_id: connectionId,
        phones,
        names,
      });
      // fastglue wraps: { data: <batch>, status: "success" } — unwrap .data
      const batch = (res.data as any).data ?? res.data;
      activeBatch.value = batch;
      return batch;
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to create verification campaign.";
      throw err;
    } finally {
      isSubmitting.value = false;
    }
  }

  async function createCampaignCSV(connectionId: string, file: File) {
    isSubmitting.value = true;
    error.value = null;
    try {
      const res = await whatsappFilterService.createCSV(connectionId, file);
      // fastglue wraps: { data: <batch>, status: "success" } — unwrap .data
      const batch = (res.data as any).data ?? res.data;
      activeBatch.value = batch;
      return batch;
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to parse or upload CSV campaign.";
      throw err;
    } finally {
      isSubmitting.value = false;
    }
  }

  async function deleteCampaign(id: string) {
    error.value = null;
    try {
      await whatsappFilterService.delete(id);
      batches.value = batches.value.filter((b) => b.id !== id);
      if (activeBatch.value?.id === id) {
        activeBatch.value = null;
        activeBatchResults.value = [];
      }
    } catch (err: any) {
      error.value = err.response?.data?.message || "Failed to delete verification campaign.";
      throw err;
    }
  }

  async function downloadResults(id: string, status: "all" | "valid" | "invalid", query?: string) {
    try {
      const res = await whatsappFilterService.exportCSV(id, { status, q: query });
      const blob = new Blob([res.data], { type: "text/csv" });
      const link = document.createElement("a");
      link.href = window.URL.createObjectURL(blob);
      link.download = `whatsapp_filter_results_${id.slice(0, 8)}.csv`;
      link.click();
      window.URL.revokeObjectURL(link.href);
    } catch (err) {
      console.error("downloadResults error", err);
      throw err;
    }
  }

  return {
    batches,
    totalBatches,
    isLoadingBatches,
    activeBatch,
    activeBatchResults,
    totalResults,
    isLoadingResults,
    isSubmitting,
    error,
    fetchBatches,
    fetchBatchDetails,
    fetchBatchResults,
    createCampaignJSON,
    createCampaignCSV,
    deleteCampaign,
    downloadResults,
  };
}
