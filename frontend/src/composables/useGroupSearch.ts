import { ref, computed } from "vue";
import {
  campaignsService,
  groupDirectoryService,
} from "@/services/api";
import { toast } from "vue-sonner";

export interface DirectoryGroup {
  id: string;
  group_jid: string;
  name: string;
  description: string;
  country: string;
  language: string;
  category: string;
  image_url: string;
  join_link: string;
  participant_count: number;
  created_at: string;
}

export interface InstanceGroup {
  jid: string;
  name: string;
  participant_count: number;
}

export function useGroupSearch() {
  const activeTab = ref<"my-groups" | "directory">("my-groups");

  // My Groups state
  const selectedInstanceId = ref("");
  const myGroups = ref<InstanceGroup[]>([]);
  const isLoadingMyGroups = ref(false);
  const myGroupsSearchQuery = ref("");

  // Directory state
  const directoryGroups = ref<DirectoryGroup[]>([]);
  const isLoadingDirectory = ref(false);
  const totalDirectoryResults = ref(0);
  const directoryPage = ref(1);
  const directoryLimit = ref(20);
  const directorySearchQuery = ref("");
  const directoryCountry = ref("");
  const directoryCategory = ref("");
  const categories = ref<string[]>([]);
  const countries = ref<string[]>([]);

  // Selection
  const selectedMyGroupIds = ref<Set<string>>(new Set());
  const selectedDirectoryIds = ref<Set<string>>(new Set());

  const selectedMyCount = computed(() => selectedMyGroupIds.value.size);
  const selectedDirectoryCount = computed(() => selectedDirectoryIds.value.size);
  const myGroupsCount = computed(() => myGroups.value.length);
  const directoryCount = computed(() => directoryGroups.value.length);

  // Invite link preview state
  const previewedGroup = ref<{
    jid: string;
    name: string;
    participant_count: number;
    invite_link: string;
  } | null>(null);
  const isPreviewing = ref(false);
  const previewError = ref("");

  function getGroupInitials(name: string): string {
    if (!name) return "?";
    return name
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w.charAt(0))
      .join("")
      .toUpperCase();
  }

  function getGroupJoinUrl(jid: string): string {
    if (!jid) return "#";
    const code = jid.replace("@g.us", "");
    return `https://chat.whatsapp.com/${code}`;
  }

  function toggleMyGroup(jid: string) {
    const s = new Set(selectedMyGroupIds.value);
    if (s.has(jid)) s.delete(jid);
    else s.add(jid);
    selectedMyGroupIds.value = s;
  }

  function toggleAllMyGroups() {
    if (selectedMyGroupIds.value.size === myGroups.value.length) {
      selectedMyGroupIds.value = new Set();
    } else {
      selectedMyGroupIds.value = new Set(myGroups.value.map((g) => g.jid));
    }
  }

  function toggleDirectoryGroup(id: string) {
    const s = new Set(selectedDirectoryIds.value);
    if (s.has(id)) s.delete(id);
    else s.add(id);
    selectedDirectoryIds.value = s;
  }

  function toggleAllDirectoryGroups() {
    if (selectedDirectoryIds.value.size === directoryGroups.value.length) {
      selectedDirectoryIds.value = new Set();
    } else {
      selectedDirectoryIds.value = new Set(directoryGroups.value.map((g) => g.id));
    }
  }

  function clearMySelection() {
    selectedMyGroupIds.value = new Set();
  }

  function clearDirectorySelection() {
    selectedDirectoryIds.value = new Set();
  }

  async function fetchMyGroups() {
    if (!selectedInstanceId.value) return;
    isLoadingMyGroups.value = true;
    try {
      const { data } = await campaignsService.listInstanceGroups(
        selectedInstanceId.value,
        myGroupsSearchQuery.value || undefined,
      );
      myGroups.value = data ?? [];
    } catch {
      toast.error("Failed to load groups");
    } finally {
      isLoadingMyGroups.value = false;
    }
  }

  async function fetchDirectoryGroups() {
    isLoadingDirectory.value = true;
    try {
      const { data } = await groupDirectoryService.search({
        q: directorySearchQuery.value || undefined,
        country: directoryCountry.value || undefined,
        category: directoryCategory.value || undefined,
        page: directoryPage.value,
        limit: directoryLimit.value,
      });
      directoryGroups.value = data?.data ?? [];
      totalDirectoryResults.value = data?.total ?? 0;
    } catch {
      toast.error("Failed to search directory");
    } finally {
      isLoadingDirectory.value = false;
    }
  }

  async function fetchCategories() {
    try {
      const { data } = await groupDirectoryService.getCategories();
      categories.value = data ?? [];
    } catch {
      /* ignore */
    }
  }

  async function fetchCountries() {
    try {
      const { data } = await groupDirectoryService.getCountries();
      countries.value = data ?? [];
    } catch {
      /* ignore */
    }
  }

  async function previewGroupLink(instanceId: string, inviteLink: string) {
    isPreviewing.value = true;
    previewError.value = "";
    previewedGroup.value = null;
    try {
      const { data } = await groupDirectoryService.previewFromLink(
        instanceId,
        inviteLink,
      );
      previewedGroup.value = data ?? null;
    } catch (e: any) {
      previewError.value =
        e?.response?.data?.message || "Failed to preview group link";
      toast.error(previewError.value);
    } finally {
      isPreviewing.value = false;
    }
  }

  async function importToCampaign(campaignId: string) {
    if (activeTab.value === "my-groups" && selectedMyCount.value > 0) {
      const groups = myGroups.value
        .filter((g) => selectedMyGroupIds.value.has(g.jid))
        .map((g) => ({
          jid: g.jid,
          name: g.name,
          participant_count: g.participant_count,
        }));
      try {
        await campaignsService.addGroups(campaignId, groups);
        toast.success(`Imported ${groups.length} groups to campaign`);
        return groups.length;
      } catch {
        toast.error("Failed to import groups");
        return 0;
      }
    }

    if (activeTab.value === "directory" && selectedDirectoryCount.value > 0) {
      const ids = Array.from(selectedDirectoryIds.value);
      try {
        const { data } = await groupDirectoryService.importToCampaign(campaignId, ids);
        toast.success(
          `Imported ${data?.added_count ?? ids.length} groups to campaign`,
        );
        return data?.added_count ?? ids.length;
      } catch {
        toast.error("Failed to import groups");
        return 0;
      }
    }

    toast.error("No groups selected");
    return 0;
  }

  function exportSelectedCSV() {
    let rows: string[][] = [];
    let filename = "groups.csv";

    if (activeTab.value === "my-groups") {
      rows = [
        ["JID", "Name", "Members"],
        ...myGroups.value
          .filter((g) => selectedMyGroupIds.value.has(g.jid))
          .map((g) => [g.jid, g.name, String(g.participant_count)]),
      ];
      filename = "my-groups.csv";
    } else {
      rows = [
        ["Name", "Country", "Language", "Category", "Members", "Join Link"],
        ...directoryGroups.value
          .filter((g) => selectedDirectoryIds.value.has(g.id))
          .map((g) => [
            g.name,
            g.country,
            g.language,
            g.category,
            String(g.participant_count),
            g.join_link ?? "",
          ]),
      ];
      filename = "directory-groups.csv";
    }

    if (rows.length <= 1) {
      toast.error("No groups selected");
      return;
    }

    const csv = rows.map((r) => r.map((c) => `"${c.replace(/"/g, '""')}"`).join(",")).join("\n");
    const blob = new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  return {
    activeTab,
    selectedInstanceId,
    myGroups,
    isLoadingMyGroups,
    myGroupsSearchQuery,
    directoryGroups,
    isLoadingDirectory,
    totalDirectoryResults,
    directoryPage,
    directoryLimit,
    directorySearchQuery,
    directoryCountry,
    directoryCategory,
    categories,
    countries,
    selectedMyGroupIds,
    selectedDirectoryIds,
    selectedMyCount,
    selectedDirectoryCount,
    myGroupsCount,
    directoryCount,
    previewedGroup,
    isPreviewing,
    previewError,
    getGroupInitials,
    getGroupJoinUrl,
    toggleMyGroup,
    toggleAllMyGroups,
    toggleDirectoryGroup,
    toggleAllDirectoryGroups,
    clearMySelection,
    clearDirectorySelection,
    fetchMyGroups,
    fetchDirectoryGroups,
    fetchCategories,
    fetchCountries,
    previewGroupLink,
    importToCampaign,
    exportSelectedCSV,
  };
}
