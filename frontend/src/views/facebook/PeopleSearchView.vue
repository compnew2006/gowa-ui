<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { watchDebounced } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import { PageHeader } from "@/components/shared";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Search,
  Users,
  UserCheck,
  Loader2,
  ChevronLeft,
  ChevronRight,
  Database,
  Sparkles,
  CheckSquare,
  Square,
  Import,
  Facebook,
  Award,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { fbPeopleSearchService, api } from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";
import { cn, getAvatarGradient, getInitials } from "@/lib/utils";

const { t, locale } = useI18n();
const isRTL = computed(() => locale.value === "ar");

// State variables
const campaigns = ref<string[]>([]);
const selectedCampaign = ref<string>("");
const loadingCampaigns = ref(false);

const results = ref<Array<{ name: string; page_id: string; followers_count: string }>>([]);
const loadingResults = ref(false);
const searchQuery = ref("");
const page = ref(1);
const perPage = ref(25);
const total = ref(0);
const totalPages = ref(0);

const selectedRows = ref<Set<string>>(new Set());
const selectedData = ref<Record<string, { identifier: string; name: string }>>({});

const isImportModalOpen = ref(false);
const importListName = ref("");
const importing = ref(false);

// Computed variables
const selectedCount = computed(() => selectedRows.value.size);

const isAllSelected = computed(() => {
  if (results.value.length === 0) return false;
  return results.value.every((row) => selectedRows.value.has(row.page_id));
});

// Load campaigns on mount
onMounted(async () => {
  await fetchCampaigns();
});

// Watch campaign change
watch(selectedCampaign, () => {
  page.value = 1;
  selectedRows.value.clear();
  selectedData.value = {};
  fetchResults();
});

// Watch pagination changes
watch(page, () => {
  fetchResults();
});

// Watch per page changes
watch(perPage, () => {
  page.value = 1;
  fetchResults();
});

// Debounced search
watchDebounced(
  searchQuery,
  () => {
    page.value = 1;
    fetchResults();
  },
  { debounce: 350 }
);

async function fetchCampaigns() {
  loadingCampaigns.value = true;
  try {
    const response = await api.get("/facebook/people-search/campaigns");
    const payload = unwrapResponse<{ campaigns: string[] }>(response);
    campaigns.value = payload.campaigns || [];
    if (campaigns.value.length > 0 && !selectedCampaign.value) {
      selectedCampaign.value = campaigns.value[0];
    }
  } catch (error: any) {
    toast.error(error.response?.data?.message || "Failed to load campaigns list");
  } finally {
    loadingCampaigns.value = false;
  }
}

async function fetchResults() {
  if (!selectedCampaign.value) {
    results.value = [];
    total.value = 0;
    totalPages.value = 0;
    return;
  }
  loadingResults.value = true;
  try {
    const response = await fbPeopleSearchService.search({
      campaign_id: selectedCampaign.value,
      page: page.value,
      per_page: perPage.value,
      q: searchQuery.value || undefined,
    });
    const payload = unwrapResponse<{
      data: Array<{ name: string; page_id: string; followers_count: string }>;
      total: number;
      total_pages: number;
    }>(response);

    results.value = payload.data || [];
    total.value = payload.total || 0;
    totalPages.value = payload.total_pages || 0;
  } catch (error: any) {
    toast.error(error.response?.data?.message || "Failed to fetch search results");
  } finally {
    loadingResults.value = false;
  }
}

function toggleSelectAll() {
  if (isAllSelected.value) {
    // Unselect all visible rows
    results.value.forEach((row) => {
      selectedRows.value.delete(row.page_id);
      delete selectedData.value[row.page_id];
    });
  } else {
    // Select all visible rows
    results.value.forEach((row) => {
      selectedRows.value.add(row.page_id);
      selectedData.value[row.page_id] = {
        identifier: row.page_id,
        name: row.name || row.followers_count || row.page_id,
      };
    });
  }
}

function toggleSelect(row: { name: string; page_id: string; followers_count: string }) {
  if (selectedRows.value.has(row.page_id)) {
    selectedRows.value.delete(row.page_id);
    delete selectedData.value[row.page_id];
  } else {
    selectedRows.value.add(row.page_id);
    selectedData.value[row.page_id] = {
      identifier: row.page_id,
      name: row.name || row.followers_count || row.page_id,
    };
  }
}

function openImportModal() {
  if (selectedCount.value === 0) {
    toast.warning(isRTL.value ? "يرجى تحديد صف واحد على الأقل" : "Please select at least one contact to import");
    return;
  }
  const dateStr = new Date().toLocaleString(locale.value === "ar" ? "ar-EG" : "en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  });
  importListName.value = isRTL.value
    ? `قائمة بحث فيسبوك ${dateStr}`
    : `FB Search List ${dateStr}`;
  isImportModalOpen.value = true;
}

async function submitImport() {
  if (!importListName.value.trim()) {
    toast.warning(isRTL.value ? "اسم قائمة جهات الاتصال مطلوب" : "List name is required");
    return;
  }
  importing.value = true;
  try {
    const list = Object.values(selectedData.value);
    const response = await fbPeopleSearchService.addContacts({
      name: importListName.value.trim(),
      data: list,
    });
    const payload = unwrapResponse<{
      success: boolean;
      created: number;
      updated: number;
      total: number;
    }>(response);

    toast.success(
      isRTL.value
        ? `تم الاستيراد بنجاح! تم إنشاء ${payload.created} وتحديث ${payload.updated} جهة اتصال.`
        : `Successfully imported! Created ${payload.created} and updated ${payload.updated} contacts.`
    );

    // Clear selections and close modal
    selectedRows.value.clear();
    selectedData.value = {};
    isImportModalOpen.value = false;
  } catch (error: any) {
    toast.error(error.response?.data?.message || "Failed to import contacts");
  } finally {
    importing.value = false;
  }
}

function avatarGradient(name: string) {
  return `bg-gradient-to-br ${getAvatarGradient(name)} text-white`;
}
</script>

<template>
  <div :dir="isRTL ? 'rtl' : 'ltr'" class="relative min-h-screen bg-slate-950/40 text-slate-100 flex flex-col justify-between overflow-hidden">
    <!-- Glowing background decorative meshes -->
    <div class="absolute -top-40 -left-40 w-96 h-96 bg-blue-600/10 rounded-full blur-3xl pointer-events-none animate-pulse"></div>
    <div class="absolute -bottom-40 -right-40 w-[24rem] h-[24rem] bg-indigo-600/10 rounded-full blur-3xl pointer-events-none animate-pulse duration-5000"></div>

    <div class="flex-1 flex flex-col p-4 md:p-8 relative z-10 max-w-7xl w-full mx-auto pb-24">
      <PageHeader
        :title="t('nav.facebookPeopleSearch')"
        :subtitle="t('nav.facebookPeopleSearchDesc')"
        :breadcrumbs="[
          { label: t('nav.facebookTools'), href: '/facebook/page-search' },
          { label: t('nav.facebookPeopleSearch') },
        ]"
      >
        <template #actions>
          <Badge variant="outline" class="bg-blue-950/40 border-blue-500/30 text-blue-400 flex items-center gap-1.5 py-1 px-3 rounded-full">
            <Award class="h-3 w-3 text-yellow-400 animate-bounce" />
            <span>v1.0 Premium</span>
          </Badge>
        </template>
      </PageHeader>

      <div class="grid gap-6 mt-6">
        <!-- Campaign Selector Card -->
        <Card class="bg-slate-900/40 border-slate-800/80 backdrop-blur-xl shadow-xl overflow-hidden rounded-2xl relative">
          <div class="absolute top-0 inset-x-0 h-[2px] bg-gradient-to-r from-blue-500 via-indigo-500 to-purple-500"></div>
          <CardHeader class="pb-4">
            <CardTitle class="text-base font-medium flex items-center gap-2">
              <Database class="h-4.5 w-4.5 text-blue-400" />
              {{ isRTL ? 'اختر حملة البحث' : 'Select Search Campaign' }}
            </CardTitle>
            <CardDescription>
              {{ isRTL ? 'حدد حملة بحث عن الأشخاص لعرض النتائج واستيرادها' : 'Choose a people search campaign to view results and import' }}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div class="flex flex-col sm:flex-row gap-4 items-end">
              <div class="flex-1 min-w-[240px]">
                <Label for="campaign-select" class="text-xs text-slate-400 mb-1.5 block">
                  {{ isRTL ? 'الحملات المتاحة' : 'Available Campaigns' }}
                </Label>
                <div v-if="loadingCampaigns" class="h-10 flex items-center px-3 border border-slate-800 bg-slate-950/40 rounded-xl">
                  <Loader2 class="h-4 w-4 animate-spin text-blue-400 mr-2" />
                  <span class="text-xs text-slate-500">{{ isRTL ? 'جاري تحميل الحملات...' : 'Loading campaigns...' }}</span>
                </div>
                <Select v-else v-model="selectedCampaign" id="campaign-select">
                  <SelectTrigger class="w-full bg-slate-950/40 border-slate-800 rounded-xl text-slate-200">
                    <SelectValue :placeholder="isRTL ? 'اختر حملة...' : 'Select a campaign...'" />
                  </SelectTrigger>
                  <SelectContent class="bg-slate-900 border-slate-800 text-slate-200">
                    <SelectItem v-for="camp in campaigns" :key="camp" :value="camp" class="hover:bg-slate-800 focus:bg-slate-800">
                      {{ camp }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Button variant="outline" class="border-slate-800 bg-slate-950/40 hover:bg-slate-800 rounded-xl" @click="fetchCampaigns" :disabled="loadingCampaigns">
                <Loader2 v-if="loadingCampaigns" class="h-4 w-4 animate-spin" />
                <span v-else>{{ isRTL ? 'تحديث القائمة' : 'Refresh' }}</span>
              </Button>
            </div>
          </CardContent>
        </Card>

        <!-- No campaign chosen view -->
        <div v-if="!selectedCampaign" class="flex-1 flex flex-col items-center justify-center p-12 text-center border border-dashed border-slate-800 rounded-2xl bg-slate-900/10 min-h-[300px]">
          <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-blue-500/10 border border-blue-500/20 text-blue-400 shadow-md">
            <Sparkles class="h-7 w-7 animate-pulse" />
          </div>
          <h3 class="mt-4 text-lg font-semibold text-slate-200">{{ isRTL ? 'لم يتم تحديد حملة' : 'No Campaign Selected' }}</h3>
          <p class="mt-2 text-sm text-slate-500 max-w-sm">
            {{ isRTL ? 'يرجى اختيار حملة بحث من القائمة أعلاه لعرض البيانات واستخراجها.' : 'Please select a campaign from the selector above to load target profiles.' }}
          </p>
        </div>

        <!-- Campaign results view -->
        <Card v-else class="bg-slate-900/40 border-slate-800/80 backdrop-blur-xl shadow-xl overflow-hidden rounded-2xl">
          <!-- Toolbar -->
          <CardHeader class="pb-3 border-b border-slate-800/60 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div>
              <CardTitle class="text-base font-semibold flex items-center gap-2">
                <Facebook class="h-4.5 w-4.5 text-blue-500" />
                {{ isRTL ? 'نتائج البحث عن الأشخاص' : 'Search Profiles' }}
              </CardTitle>
              <CardDescription class="mt-1">
                {{ isRTL ? `وجدنا ${total} ملف شخصي مطابق` : `Found ${total} matched profiles` }}
              </CardDescription>
            </div>

            <!-- Filters -->
            <div class="flex items-center gap-3 w-full md:w-auto">
              <div class="relative w-full md:w-64">
                <Search class="absolute left-3 top-2.5 h-4 w-4 text-slate-500" :class="isRTL ? 'left-auto right-3' : ''" />
                <Input
                  v-model="searchQuery"
                  :placeholder="isRTL ? 'بحث بالاسم أو المعرف...' : 'Search by name or ID...'"
                  class="bg-slate-950/40 border-slate-800 pl-9 rounded-xl text-slate-200"
                  :class="isRTL ? 'pl-3 pr-9' : ''"
                />
              </div>
            </div>
          </CardHeader>

          <!-- Table Container -->
          <CardContent class="p-0">
            <div v-if="loadingResults" class="h-64 flex flex-col items-center justify-center gap-3">
              <Loader2 class="h-8 w-8 animate-spin text-blue-500" />
              <span class="text-sm text-slate-500">{{ isRTL ? 'جاري تحميل النتائج...' : 'Loading results...' }}</span>
            </div>

            <div v-else-if="results.length === 0" class="h-64 flex flex-col items-center justify-center text-center p-8">
              <Users class="h-10 w-10 text-slate-600 mb-2" />
              <h4 class="text-sm font-semibold text-slate-300">{{ isRTL ? 'لا توجد نتائج' : 'No records found' }}</h4>
              <p class="text-xs text-slate-500 mt-1 max-w-xs">{{ isRTL ? 'لم نجد أي ملفات تطابق شروط البحث الحالية.' : 'No profiles match the filter criteria.' }}</p>
            </div>

            <div v-else class="overflow-x-auto w-full">
              <table class="w-full text-sm text-left text-slate-300 border-collapse" :class="isRTL ? 'text-right' : ''">
                <thead>
                  <tr class="bg-slate-950/60 border-b border-slate-800/80 text-slate-400 font-semibold text-xs tracking-wider uppercase">
                    <th scope="col" class="px-6 py-4 w-[60px] text-center">
                      <button @click="toggleSelectAll" class="text-slate-400 hover:text-slate-100 transition-colors">
                        <CheckSquare v-if="isAllSelected" class="h-4.5 w-4.5 text-blue-500" />
                        <Square v-else class="h-4.5 w-4.5" />
                      </button>
                    </th>
                    <th scope="col" class="px-6 py-4">{{ isRTL ? 'الاسم' : 'Profile Name' }}</th>
                    <th scope="col" class="px-6 py-4">{{ isRTL ? 'المعرف' : 'Profile ID' }}</th>
                    <th scope="col" class="px-6 py-4">{{ isRTL ? 'مؤشرات / المتابعين' : 'Followers / Data' }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-800/40">
                  <tr
                    v-for="row in results"
                    :key="row.page_id"
                    class="hover:bg-slate-800/20 transition-colors cursor-pointer"
                    @click="toggleSelect(row)"
                  >
                    <td class="px-6 py-4 text-center" @click.stop="toggleSelect(row)">
                      <button class="text-slate-400 transition-colors">
                        <CheckSquare v-if="selectedRows.has(row.page_id)" class="h-4.5 w-4.5 text-blue-500" />
                        <Square v-else class="h-4.5 w-4.5" />
                      </button>
                    </td>
                    <td class="px-6 py-4">
                      <div class="flex items-center gap-3">
                        <span
                          :class="cn(
                            'flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold',
                            avatarGradient(row.name || '?')
                          )"
                        >
                          {{ getInitials(row.name || "?") }}
                        </span>
                        <span class="font-medium text-slate-200">{{ row.name || 'Anonymous' }}</span>
                      </div>
                    </td>
                    <td class="px-6 py-4 font-mono text-slate-400 text-xs">{{ row.page_id }}</td>
                    <td class="px-6 py-4">
                      <Badge variant="secondary" class="bg-slate-800/60 text-slate-300 border border-slate-700/30">
                        {{ row.followers_count || 'N/A' }}
                      </Badge>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>

          <!-- Table Footer / Pagination -->
          <div v-if="results.length > 0 && totalPages > 1" class="border-t border-slate-800/60 px-6 py-4 flex items-center justify-between text-xs text-slate-500">
            <div>
              {{ isRTL ? `عرض الصفحة ${page} من ${totalPages}` : `Showing page ${page} of ${totalPages}` }}
            </div>
            <div class="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                class="hover:bg-slate-800 rounded-lg p-1.5"
                :disabled="page <= 1 || loadingResults"
                @click="page = page - 1"
              >
                <ChevronLeft class="h-4 w-4" :class="isRTL ? 'rotate-180' : ''" />
              </Button>
              <div class="flex gap-1">
                <Button
                  v-for="p in totalPages"
                  :key="p"
                  variant="ghost"
                  size="sm"
                  class="rounded-lg h-7 w-7 text-xs font-semibold"
                  :class="p === page ? 'bg-blue-600 hover:bg-blue-500 text-white' : 'hover:bg-slate-800 text-slate-400'"
                  @click="page = p"
                >
                  {{ p }}
                </Button>
              </div>
              <Button
                variant="ghost"
                size="sm"
                class="hover:bg-slate-800 rounded-lg p-1.5"
                :disabled="page >= totalPages || loadingResults"
                @click="page = page + 1"
              >
                <ChevronRight class="h-4 w-4" :class="isRTL ? 'rotate-180' : ''" />
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- Floating import action bar at the bottom -->
    <transition
      enter-active-class="transition ease-out duration-300 transform"
      enter-from-class="translate-y-full opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition ease-in duration-200 transform"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="translate-y-full opacity-0"
    >
      <div v-if="selectedCount > 0" class="fixed bottom-6 inset-x-4 max-w-2xl mx-auto z-50 bg-slate-900/90 border border-slate-800 backdrop-blur-md rounded-2xl shadow-2xl p-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <div class="h-8 w-8 bg-blue-500/10 border border-blue-500/30 text-blue-400 rounded-lg flex items-center justify-center">
            <UserCheck class="h-4.5 w-4.5" />
          </div>
          <div>
            <span class="text-sm font-semibold text-slate-100">
              {{ isRTL ? `تم تحديد ${selectedCount} صف` : `Selected ${selectedCount} profiles` }}
            </span>
            <span class="text-[10px] text-slate-500 block mt-0.5">
              {{ isRTL ? 'جاهز للاستيراد كجهات اتصال' : 'Ready to import to contacts' }}
            </span>
          </div>
        </div>

        <Button class="bg-blue-600 hover:bg-blue-500 text-white rounded-xl shadow-lg shadow-blue-600/20 px-4 py-2 flex items-center gap-2 font-medium" @click="openImportModal">
          <Import class="h-4 w-4" />
          <span>{{ isRTL ? 'استيراد جهات الاتصال' : 'Import Contacts' }}</span>
        </Button>
      </div>
    </transition>

    <!-- Confirm Import Dialog -->
    <Dialog v-model:open="isImportModalOpen">
      <DialogContent class="bg-slate-900 border-slate-800 text-slate-100 max-w-md rounded-2xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-lg font-bold">
            <Users class="h-5 w-5 text-blue-400" />
            {{ isRTL ? 'حفظ جهات الاتصال' : 'Import Contacts List' }}
          </DialogTitle>
          <DialogDescription class="text-slate-400 text-xs">
            {{ isRTL ? `سيتم حفظ ${selectedCount} جهة اتصال في النظام باسم القائمة المحدد` : `Will save ${selectedCount} selected profiles as Whatomate contacts under the specified list tag.` }}
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-3">
          <div class="space-y-1.5">
            <Label for="list-name" class="text-xs text-slate-400">
              {{ isRTL ? 'اسم قائمة جهات الاتصال' : 'Contacts List Name / Tag' }}
            </Label>
            <Input
              id="list-name"
              v-model="importListName"
              placeholder="e.g. Leads Campaign 2026"
              class="bg-slate-950 border-slate-800 rounded-xl text-slate-200"
            />
          </div>
        </div>

        <DialogFooter class="gap-2 sm:gap-0">
          <Button variant="ghost" class="hover:bg-slate-800 rounded-xl text-slate-400" @click="isImportModalOpen = false" :disabled="importing">
            {{ isRTL ? 'إلغاء' : 'Cancel' }}
          </Button>
          <Button class="bg-blue-600 hover:bg-blue-500 text-white rounded-xl" @click="submitImport" :disabled="importing">
            <Loader2 v-if="importing" class="h-4 w-4 animate-spin mr-2" />
            <span>{{ isRTL ? 'استيراد الآن' : 'Import Now' }}</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<style scoped>
.animate-pulse {
  animation: pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
@keyframes pulse {
  0%, 100% {
    opacity: 0.15;
  }
  50% {
    opacity: 0.35;
  }
}
</style>
