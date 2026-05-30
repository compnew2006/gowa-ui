<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { toast } from "vue-sonner";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Checkbox } from "@/components/ui/checkbox";
import {
  PageHeader,
  SearchInput,
  PaginationControls,
} from "@/components/shared";
import { useGroupSearch } from "@/composables/useGroupSearch";
import { instancesService, campaignsService } from "@/services/api";
import {
  Download,
  Users,
  Globe,
  Send,
  Loader2,
  ExternalLink,
  Check,
  Search,
} from "lucide-vue-next";

const { t } = useI18n();
const route = useRoute();
const gs = useGroupSearch();

// Set default tab to directory (like kingmaster)
gs.activeTab.value = "directory";

const instances = ref<Array<{ id: string; name: string }>>([]);
const isLoadingInstances = ref(false);

// Invite link preview state
const previewInstanceId = ref("");
const previewLink = ref("");

// Import dialog
const showImportDialog = ref(false);
const importCampaignId = ref("");
const campaignList = ref<Array<{ id: string; name: string }>>([]);
const isLoadingCampaigns = ref(false);
const isImporting = ref(false);

async function loadInstances() {
  isLoadingInstances.value = true;
  try {
    const response = await instancesService.list();
    const data = (response.data as any)?.data || response.data;
    instances.value = Array.isArray(data) ? data : data?.instances || [];
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("groupSearch.failedToLoadInstances"));
  } finally {
    isLoadingInstances.value = false;
  }
}

async function loadCampaigns() {
  isLoadingCampaigns.value = true;
  try {
    const { data } = await campaignsService.list({ status: "draft", limit: 100 });
    campaignList.value = data?.data ?? [];
  } catch {
    /* ignore */
  } finally {
    isLoadingCampaigns.value = false;
  }
}

function openImportDialog() {
  importCampaignId.value = route.query.campaign as string ?? "";
  loadCampaigns();
  showImportDialog.value = true;
}

async function confirmImport() {
  if (!importCampaignId.value) return;
  isImporting.value = true;
  await gs.importToCampaign(importCampaignId.value);
  isImporting.value = false;
  showImportDialog.value = false;
}

watch(gs.selectedInstanceId, () => gs.fetchMyGroups());
watch(gs.myGroupsSearchQuery, () => gs.fetchMyGroups());
watch(
  [gs.directorySearchQuery, gs.directoryCountry, gs.directoryCategory, gs.directoryPage],
  () => gs.fetchDirectoryGroups(),
);

onMounted(() => {
  loadInstances();
  gs.fetchCategories();
  gs.fetchCountries();
});
</script>

<template>
  <div class="flex h-full flex-col gap-6 p-6">
    <PageHeader :title="t('groupSearch.title')" :subtitle="t('groupSearch.subtitle')" />

    <Tabs v-model="gs.activeTab.value" class="w-full">
      <TabsList>
        <TabsTrigger value="my-groups">
          <Users class="w-4 h-4 mr-1.5" />
          {{ t("groupSearch.myGroupsTab") }}
        </TabsTrigger>
        <TabsTrigger value="directory">
          <Globe class="w-4 h-4 mr-1.5" />
          {{ t("groupSearch.directoryTab") }}
        </TabsTrigger>
      </TabsList>

      <!-- My Groups Tab -->
      <TabsContent value="my-groups" class="mt-4 space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="gs.selectedInstanceId.value">
            <SelectTrigger class="w-[220px]">
              <SelectValue :placeholder="t('groupSearch.selectInstance')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="inst in instances" :key="inst.id" :value="inst.id">
                {{ inst.name }}
              </SelectItem>
            </SelectContent>
          </Select>
          <SearchInput
            v-model="gs.myGroupsSearchQuery.value"
            :placeholder="t('groupSearch.searchMyGroups')"
            class="w-[260px]"
          />
        </div>

        <div v-if="gs.isLoadingMyGroups.value" class="flex justify-center py-12">
          <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
        </div>

        <div v-else-if="!gs.selectedInstanceId.value" class="flex justify-center py-12 text-muted-foreground">
          {{ t("groupSearch.noInstanceSelected") }}
        </div>

        <div v-else-if="gs.myGroups.value.length === 0" class="flex justify-center py-12 text-muted-foreground">
          {{ t("groupSearch.noGroupsFound") }}
        </div>

        <template v-else>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Checkbox
                :checked="gs.selectedMyCount.value === gs.myGroups.value.length && gs.myGroups.value.length > 0"
                @update:checked="gs.toggleAllMyGroups()"
              />
              <span class="text-sm text-muted-foreground">
                {{ t("groupSearch.selectedCount", { count: gs.selectedMyCount.value }) }}
              </span>
            </div>
            <span class="text-sm text-muted-foreground">
              {{ t("groupSearch.foundCount", { count: gs.myGroupsCount.value }) }}
            </span>
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Card
              v-for="group in gs.myGroups.value"
              :key="group.jid"
              class="cursor-pointer transition-colors hover:bg-muted/50"
              :class="gs.selectedMyGroupIds.value.has(group.jid) ? 'ring-2 ring-primary' : ''"
              @click="gs.toggleMyGroup(group.jid)"
            >
              <CardContent class="flex items-center gap-3 p-4">
                <Checkbox
                  :checked="gs.selectedMyGroupIds.value.has(group.jid)"
                  @click.stop
                  @update:checked="gs.toggleMyGroup(group.jid)"
                />
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                  {{ gs.getGroupInitials(group.name) }}
                </div>
                <div class="min-w-0 flex-1">
                  <p class="truncate font-medium">{{ group.name }}</p>
                  <p class="text-sm text-muted-foreground">
                    {{ t("groupSearch.memberCount", { count: group.participant_count }) }}
                  </p>
                </div>
                <a
                  :href="gs.getGroupJoinUrl(group.jid)"
                  target="_blank"
                  rel="noopener"
                  class="inline-flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground hover:bg-primary/90"
                  @click.stop
                >
                  <ExternalLink class="w-3 h-3" />
                  {{ t("groupSearch.joinLink") }}
                </a>
              </CardContent>
            </Card>
          </div>
        </template>

        <!-- Actions bar -->
        <div v-if="gs.selectedMyCount.value > 0" class="flex items-center gap-2 border-t pt-4">
          <Button variant="outline" size="sm" @click="gs.exportSelectedCSV()">
            <Download class="w-4 h-4 mr-1.5" />
            {{ t("groupSearch.exportCSV") }}
          </Button>
          <Button size="sm" @click="openImportDialog()">
            <Send class="w-4 h-4 mr-1.5" />
            {{ t("groupSearch.importToCampaign") }}
          </Button>
        </div>
      </TabsContent>

      <!-- Directory Tab -->
      <TabsContent value="directory" class="mt-4 space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <SearchInput
            v-model="gs.directorySearchQuery.value"
            :placeholder="t('groupSearch.searchDirectory')"
            class="w-[260px]"
          />
          <Select v-model="gs.directoryCountry.value">
            <SelectTrigger class="w-[160px]">
              <SelectValue :placeholder="t('groupSearch.allCountries')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in gs.countries.value" :key="c" :value="c">
                {{ c }}
              </SelectItem>
            </SelectContent>
          </Select>
          <Select v-model="gs.directoryCategory.value">
            <SelectTrigger class="w-[160px]">
              <SelectValue :placeholder="t('groupSearch.allCategories')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="cat in gs.categories.value" :key="cat" :value="cat">
                {{ cat }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Invite Link Preview Section -->
        <div class="rounded-lg border bg-card p-4 space-y-3">
          <div class="text-sm font-medium">{{ t("groupSearch.previewFromLink") }}</div>
          <div class="flex flex-wrap items-center gap-3">
            <Select v-model="previewInstanceId">
              <SelectTrigger class="w-[220px]">
                <SelectValue :placeholder="t('groupSearch.selectInstance')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="inst in instances" :key="inst.id" :value="inst.id">
                  {{ inst.name }}
                </SelectItem>
              </SelectContent>
            </Select>
            <div class="flex items-center gap-2 flex-1 min-w-[260px]">
              <input
                v-model="previewLink"
                type="text"
                :placeholder="t('groupSearch.pasteInviteLink')"
                class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                @keydown.enter="gs.previewGroupLink(previewInstanceId, previewLink)"
              />
              <Button
                :disabled="!previewInstanceId || !previewLink || gs.isPreviewing.value"
                @click="gs.previewGroupLink(previewInstanceId, previewLink)"
              >
                <Search v-if="!gs.isPreviewing.value" class="w-4 h-4 mr-1.5" />
                <Loader2 v-else class="w-4 h-4 mr-1.5 animate-spin" />
                {{ t("groupSearch.preview") }}
              </Button>
            </div>
          </div>

          <!-- Preview Result -->
          <div v-if="gs.previewedGroup.value" class="rounded-md border p-3">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                {{ gs.getGroupInitials(gs.previewedGroup.value.name) }}
              </div>
              <div class="flex-1 min-w-0">
                <p class="font-medium">{{ gs.previewedGroup.value.name }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("groupSearch.memberCount", { count: gs.previewedGroup.value.participant_count }) }}
                </p>
              </div>
              <a
                :href="gs.previewedGroup.value.invite_link"
                target="_blank"
                rel="noopener"
                class="inline-flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground hover:bg-primary/90"
              >
                <ExternalLink class="w-3 h-3" />
                {{ t("groupSearch.joinLink") }}
              </a>
            </div>
          </div>

          <div v-if="gs.previewError.value" class="text-sm text-destructive">
            {{ gs.previewError.value }}
          </div>
        </div>

        <div v-if="gs.isLoadingDirectory.value" class="flex justify-center py-12">
          <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
        </div>

        <div v-else-if="gs.directoryGroups.value.length === 0" class="flex justify-center py-12 text-muted-foreground">
          {{ t("groupSearch.noDirectoryGroups") }}
        </div>

        <template v-else>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Checkbox
                :checked="gs.selectedDirectoryCount.value === gs.directoryGroups.value.length && gs.directoryGroups.value.length > 0"
                @update:checked="gs.toggleAllDirectoryGroups()"
              />
              <span class="text-sm text-muted-foreground">
                {{ t("groupSearch.selectedCount", { count: gs.selectedDirectoryCount.value }) }}
              </span>
            </div>
            <span class="text-sm text-muted-foreground">
              {{ t("groupSearch.foundCount", { count: gs.totalDirectoryResults.value }) }}
            </span>
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Card
              v-for="group in gs.directoryGroups.value"
              :key="group.id"
              class="cursor-pointer transition-colors hover:bg-muted/50"
              :class="gs.selectedDirectoryIds.value.has(group.id) ? 'ring-2 ring-primary' : ''"
              @click="gs.toggleDirectoryGroup(group.id)"
            >
              <CardContent class="p-4">
                <div class="flex items-start gap-3">
                  <Checkbox
                    :checked="gs.selectedDirectoryIds.value.has(group.id)"
                    class="mt-0.5"
                    @click.stop
                    @update:checked="gs.toggleDirectoryGroup(group.id)"
                  />
                  <div class="min-w-0 flex-1">
                    <p class="font-medium">{{ group.name }}</p>
                    <p v-if="group.description" class="mt-1 line-clamp-2 text-sm text-muted-foreground">
                      {{ group.description }}
                    </p>
                    <div class="mt-2 flex flex-wrap items-center gap-1.5">
                      <span v-if="group.country" class="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs">
                        {{ group.country }}
                      </span>
                      <span v-if="group.language" class="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs">
                        {{ group.language }}
                      </span>
                      <span v-if="group.category" class="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                        {{ group.category }}
                      </span>
                      <span class="text-xs text-muted-foreground">
                        {{ t("groupSearch.memberCount", { count: group.participant_count }) }}
                      </span>
                    </div>
                    <a
                      v-if="group.join_link"
                      :href="group.join_link"
                      target="_blank"
                      rel="noopener"
                      class="mt-2 inline-flex items-center gap-1 text-xs text-primary hover:underline"
                      @click.stop
                    >
                      <ExternalLink class="w-3 h-3" />
                      {{ t("groupSearch.joinLink") }}
                    </a>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <PaginationControls
            :current-page="gs.directoryPage.value"
            :total-pages="gs.directoryLimit.value ? Math.ceil(gs.totalDirectoryResults.value / gs.directoryLimit.value) : 1"
            :total-items="gs.totalDirectoryResults.value"
            :page-size="gs.directoryLimit.value"
            @update:current-page="gs.directoryPage.value = $event"
          />
        </template>

        <!-- Actions bar -->
        <div v-if="gs.selectedDirectoryCount.value > 0" class="flex items-center gap-2 border-t pt-4">
          <Button variant="outline" size="sm" @click="gs.exportSelectedCSV()">
            <Download class="w-4 h-4 mr-1.5" />
            {{ t("groupSearch.exportCSV") }}
          </Button>
          <Button size="sm" @click="openImportDialog()">
            <Send class="w-4 h-4 mr-1.5" />
            {{ t("groupSearch.importToCampaign") }}
          </Button>
        </div>
      </TabsContent>
    </Tabs>

    <!-- Import Dialog -->
    <Dialog :open="showImportDialog" @update:open="showImportDialog = $event">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t("groupSearch.importDialogTitle") }}</DialogTitle>
          <DialogDescription>{{ t("groupSearch.importDialogDesc") }}</DialogDescription>
        </DialogHeader>
        <Select v-model="importCampaignId">
          <SelectTrigger>
            <SelectValue :placeholder="t('groupSearch.selectCampaign')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="c in campaignList" :key="c.id" :value="c.id">
              {{ c.name }}
            </SelectItem>
          </SelectContent>
        </Select>
        <DialogFooter>
          <Button variant="outline" @click="showImportDialog = false">Cancel</Button>
          <Button :disabled="!importCampaignId || isImporting" @click="confirmImport">
            <Loader2 v-if="isImporting" class="w-4 h-4 mr-1.5 animate-spin" />
            <Check v-else class="w-4 h-4 mr-1.5" />
            {{ t("groupSearch.importToCampaign") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
