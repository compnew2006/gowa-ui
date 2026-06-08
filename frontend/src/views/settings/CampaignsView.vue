<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import { useI18n } from "vue-i18n";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Progress } from "@/components/ui/progress";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { RangeCalendar } from "@/components/ui/range-calendar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  campaignsService,
  templatesService,
  accountsService,
  instancesService,
  contactsService,
  savedContentsService,
} from "@/services/api";
import { wsService } from "@/services/websocket";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "vue-sonner";
import {
  PageHeader,
  DataTable,
  DeleteConfirmDialog,
  SearchInput,
  type Column,
} from "@/components/shared";
import { getErrorMessage } from "@/lib/api-utils";
import { useConfigStore } from "@/stores/config";
import {
  Plus,
  Pencil,
  Trash2,
  Megaphone,
  Play,
  Pause,
  XCircle,
  Users,
  CheckCircle,
  Clock,
  AlertCircle,
  Loader2,
  Upload,
  UserPlus,
  Eye,
  FileSpreadsheet,
  UsersRound,
  AlertTriangle,
  Check,
  RefreshCw,
  CalendarIcon,
  MessageSquare,
  BookOpen,
} from "lucide-vue-next";
import { formatDate } from "@/lib/utils";
import { useDebounceFn } from "@vueuse/core";
import WhatsAppRichTextEditor from "@/components/chat/WhatsAppRichTextEditor.vue";
import ContentPickerModal from "@/components/ContentPickerModal.vue";
import type { SavedContent } from "@/services/api";

const { t } = useI18n();
const configStore = useConfigStore();

interface Campaign {
  id: string;
  name: string;
  template_name: string;
  template_id?: string;
  whatsapp_account?: string;
  min_delay_seconds?: number;
  max_delay_seconds?: number;
  header_media_id?: string;
  header_media_filename?: string;
  header_media_mime_type?: string;
  poll_question?: string;
  poll_options?: string[];
  poll_max_selections?: number;
  status:
    | "draft"
    | "scheduled"
    | "running"
    | "paused"
    | "completed"
    | "failed"
    | "queued"
    | "processing"
    | "cancelled";
  total_recipients: number;
  sent_count: number;
  delivered_count: number;
  read_count: number;
  failed_count: number;
  scheduled_at?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

interface Template {
  id: string;
  name: string;
  display_name?: string;
  status: string;
  body_content?: string;
  header_type?: string; // TEXT, IMAGE, DOCUMENT, VIDEO
  header_content?: string;
}

interface CSVRow {
  phone_number: string;
  name: string;
  params: Record<string, string>; // keyed by param name (e.g., {"name": "John"} or {"1": "John"})
  isValid: boolean;
  errors: string[];
}

interface CSVValidation {
  isValid: boolean;
  rows: CSVRow[];
  templateParamNames: string[]; // e.g., ["name", "order_id"] or ["1", "2"]
  csvColumns: string[];
  columnMapping: { csvColumn: string; paramName: string }[]; // Shows how CSV columns map to params
  errors: string[];
  warnings: string[]; // Non-blocking warnings (e.g., mixed param types)
}

interface Account {
  id: string;
  name: string;
  phone_id: string;
}

interface Instance {
  id: string;
  name: string;
  status?: string;
}

interface Recipient {
  id: string;
  phone_number: string;
  recipient_name: string;
  status: string;
  sent_at?: string;
  delivered_at?: string;
  error_message?: string;
}

interface GroupRecipient extends Recipient {
  recipient_type?: string;
  group_jid?: string;
  group_name?: string;
  participant_count?: number;
}

interface ContactRecipient {
  id: string;
  phone_number: string;
  profile_name?: string;
  name?: string;
  created_at?: string;
}

type ContactsImportDateBasis = "created" | "incoming_any";

const campaigns = ref<Campaign[]>([]);
const templates = ref<Template[]>([]);
const templatesLoading = ref(false);
const accounts = ref<Account[]>([]);
const instances = ref<Instance[]>([]);
const isLoading = ref(true);
const isCreating = ref(false);
const showCreateDialog = ref(false);
const editingCampaignId = ref<string | null>(null); // null = create mode, string = edit mode
const isWhatsmeowProvider = computed(() => configStore.isWhatsmeow);

const senderLabel = computed(() =>
  isWhatsmeowProvider.value
    ? t("campaigns.whatsappInstance")
    : t("campaigns.whatsappAccount"),
);
const senderPlaceholder = computed(() =>
  isWhatsmeowProvider.value
    ? t("campaigns.selectInstance")
    : t("campaigns.selectAccount"),
);
const noSendersFoundMessage = computed(() =>
  isWhatsmeowProvider.value
    ? t("campaigns.noInstancesFound")
    : t("campaigns.noAccountsFound"),
);
const senderOptions = computed(() =>
  isWhatsmeowProvider.value
    ? instances.value.map((instance) => ({
        value: instance.id,
        label: instance.name,
      }))
    : accounts.value.map((account) => ({
        value: account.name,
        label: account.name,
      })),
);

const columns = computed<Column<Campaign>[]>(() => [
  { key: "name", label: t("campaigns.campaign"), sortable: true },
  { key: "status", label: t("campaigns.status"), sortable: true },
  { key: "stats", label: t("campaigns.progress") },
  { key: "created_at", label: t("campaigns.created"), sortable: true },
  { key: "actions", label: t("common.actions"), align: "right" },
]);

const sortKey = ref("created_at");
const sortDirection = ref<"asc" | "desc">("desc");
const searchQuery = ref("");

// Pagination state
const currentPage = ref(1);
const totalItems = ref(0);
const pageSize = 20;

function handlePageChange(page: number) {
  currentPage.value = page;
  fetchCampaigns();
}

// Filter state
const filterStatus = ref<string>("all");
type TimeRangePreset = "today" | "7days" | "30days" | "this_month" | "custom";
const selectedRange = ref<TimeRangePreset>("this_month");
const customDateRange = ref<any>({ start: undefined, end: undefined });
const isDatePickerOpen = ref(false);

const statusOptions = computed(() => [
  { value: "all", label: t("campaigns.allStatuses") },
  { value: "draft", label: t("campaigns.draft") },
  { value: "queued", label: t("campaigns.queued") },
  { value: "processing", label: t("campaigns.processing") },
  { value: "completed", label: t("campaigns.completed") },
  { value: "failed", label: t("campaigns.failed") },
  { value: "cancelled", label: t("campaigns.cancelled") },
  { value: "paused", label: t("campaigns.paused") },
]);

// Format date as YYYY-MM-DD in local timezone
const formatDateLocal = (date: Date): string => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const getDateRange = computed(() => {
  const now = new Date();
  let from: Date;
  let to: Date = now;

  switch (selectedRange.value) {
    case "today":
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      break;
    case "7days":
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7);
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      break;
    case "30days":
      from = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 30);
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      break;
    case "this_month":
      from = new Date(now.getFullYear(), now.getMonth(), 1);
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      break;
    case "custom":
      if (customDateRange.value.start && customDateRange.value.end) {
        from = new Date(
          customDateRange.value.start.year,
          customDateRange.value.start.month - 1,
          customDateRange.value.start.day,
        );
        to = new Date(
          customDateRange.value.end.year,
          customDateRange.value.end.month - 1,
          customDateRange.value.end.day,
        );
      } else {
        from = new Date(now.getFullYear(), now.getMonth(), 1);
        to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      }
      break;
    default:
      from = new Date(now.getFullYear(), now.getMonth(), 1);
      to = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  }

  return {
    from: formatDateLocal(from),
    to: formatDateLocal(to),
  };
});



// Recipients state
const showRecipientsDialog = ref(false);
const showAddRecipientsDialog = ref(false);
const selectedCampaign = ref<Campaign | null>(null);
const recipients = ref<Recipient[]>([]);
const isLoadingRecipients = ref(false);
const isAddingRecipients = ref(false);
const recipientsInput = ref("");

// CSV upload state
const csvFile = ref<File | null>(null);
const csvValidation = ref<CSVValidation | null>(null);
const isValidatingCSV = ref(false);
const selectedTemplate = ref<Template | null>(null);
const addRecipientsTab = ref("manual");
const contactsForImport = ref<ContactRecipient[]>([]);
const contactsSearchQuery = ref("");
const contactsDateFrom = ref("");
const contactsDateTo = ref("");
const contactsImportDateBasis = ref<ContactsImportDateBasis>("created");
const contactsImportPage = ref(1);
const contactsImportTotal = ref(0);
const contactsImportPageSize = ref(50);
const contactsImportPageSizeOptions = [25, 50, 100, 500];
const selectedContactsById = ref<Record<string, ContactRecipient>>({});
const isLoadingContactsForImport = ref(false);
const contactsImportSelectionPrunePageSize = 500;

// Group targeting state
const showGroupsDialog = ref(false);
const showAddGroupsDialog = ref(false);
const groupRecipients = ref<GroupRecipient[]>([]);
const isLoadingGroups = ref(false);
const availableGroups = ref<Array<{ jid: string; name: string; participant_count: number }>>([]);
const isLoadingAvailableGroups = ref(false);
const selectedAvailableGroupIds = ref<Set<string>>(new Set());
const groupSourceInstanceId = ref("");


// Media upload state
const campaignMediaFile = ref<File | null>(null);
const campaignFileInput = ref<HTMLInputElement | null>(null);
const campaignMediaInputKey = ref(0);
const maxCampaignMediaSizeBytes = 16 * 1024 * 1024;
const allowedCampaignMediaMimeTypes = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
  "video/mp4",
  "video/3gpp",
  "audio/aac",
  "audio/mp4",
  "audio/mpeg",
  "audio/ogg",
  "application/pdf",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
]);
const allowedCampaignMediaExtensions = [
  ".jpg",
  ".jpeg",
  ".png",
  ".webp",
  ".mp4",
  ".3gp",
  ".aac",
  ".m4a",
  ".mp3",
  ".ogg",
  ".pdf",
  ".xlsx",
  ".docx",
  ".pptx",
];
const campaignMediaAccept = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "video/mp4",
  "video/3gpp",
  "audio/aac",
  "audio/mp4",
  "audio/mpeg",
  "audio/ogg",
  ".pdf",
  ".xlsx",
  ".docx",
  ".pptx",
].join(",");

// Computed: template parameter format hints
const templateParamNames = computed(() => {
  if (!selectedTemplate.value) return [];
  return getTemplateParamNames(selectedTemplate.value);
});

const manualEntryFormat = computed(() => {
  const params = templateParamNames.value;
  if (params.length === 0) {
    return "phone_number";
  }
  return `phone_number, ${params.join(", ")}`;
});

const csvColumnsHint = computed(() => {
  const params = templateParamNames.value;
  if (params.length === 0) {
    return ["phone_number (or phone, mobile, number)"];
  }
  return ["phone_number (or phone, mobile, number)", ...params.map((p) => p)];
});

function formatParamName(param: string): string {
  return `{{${param}}}`;
}

// Dynamic placeholder for recipient input based on template parameters
const recipientPlaceholder = computed(() => {
  const params = templateParamNames.value;
  if (params.length === 0) {
    return `+1234567890
+0987654321
+1122334455`;
  }
  // Generate example values for each parameter
  const exampleValues = params.map((p, i) => {
    if (/^\d+$/.test(p)) {
      return `value${i + 1}`;
    }
    // Use parameter name as hint for example value
    if (p.toLowerCase().includes("name")) return "John Doe";
    if (p.toLowerCase().includes("order")) return "ORD-123";
    if (p.toLowerCase().includes("date")) return "2024-01-15";
    if (p.toLowerCase().includes("amount") || p.toLowerCase().includes("price"))
      return "99.99";
    return `${p}_value`;
  });
  const line1 = `+1234567890, ${exampleValues.join(", ")}`;
  const line2 = `+0987654321, ${exampleValues
    .map((v) => {
      if (v === "John Doe") return "Jane Smith";
      if (v === "ORD-123") return "ORD-456";
      return v;
    })
    .join(", ")}`;
  return `${line1}\n${line2}`;
});

const selectedContactIdSet = computed(
  () => new Set(Object.keys(selectedContactsById.value)),
);

const filteredContactsForImport = computed(() => contactsForImport.value);

const selectedContactsForImport = computed(() =>
  Object.values(selectedContactsById.value),
);

const contactsImportInstanceId = computed(() => {
  if (!isWhatsmeowProvider.value) return undefined;
  const rawInstanceId = selectedCampaign.value?.whatsapp_account?.trim() || "";
  return rawInstanceId || undefined;
});

const selectedCampaignInstanceLabel = computed(() => {
  const instanceId = contactsImportInstanceId.value;
  if (!instanceId) return "";
  const instance = instances.value.find(
    (candidate) => candidate.id === instanceId,
  );
  return instance?.name || instanceId;
});

const contactsImportMissingInstance = computed(
  () => isWhatsmeowProvider.value && !contactsImportInstanceId.value,
);

const contactsImportTotalPages = computed(() => {
  const total = Math.max(0, contactsImportTotal.value);
  return Math.max(1, Math.ceil(total / contactsImportPageSize.value));
});

const hasContactsImportFilters = computed(() =>
  Boolean(
    contactsSearchQuery.value.trim() ||
    contactsDateFrom.value ||
    contactsDateTo.value,
  ),
);

const areAllFilteredContactsSelected = computed(() => {
  if (filteredContactsForImport.value.length === 0) return false;
  return filteredContactsForImport.value.every((contact) =>
    selectedContactIdSet.value.has(contact.id),
  );
});

// Manual input validation
interface ManualInputValidation {
  isValid: boolean;
  totalLines: number;
  validLines: number;
  invalidLines: { lineNumber: number; reason: string }[];
}

const manualInputValidation = computed((): ManualInputValidation => {
  const params = templateParamNames.value;
  const lines = recipientsInput.value
    .trim()
    .split("\n")
    .filter((line) => line.trim());

  if (lines.length === 0) {
    return { isValid: false, totalLines: 0, validLines: 0, invalidLines: [] };
  }

  const invalidLines: { lineNumber: number; reason: string }[] = [];

  for (let i = 0; i < lines.length; i++) {
    const parts = lines[i].split(",").map((p) => p.trim());
    const phone = parts[0]?.replace(/[^\d+]/g, "");

    // Validate phone number
    if (!phone || !phone.match(/^\+?\d{10,15}$/)) {
      invalidLines.push({
        lineNumber: i + 1,
        reason: t("campaigns.invalidPhoneNumber"),
      });
      continue;
    }

    // Validate params count
    const providedParams = parts.slice(1).filter((p) => p.length > 0).length;
    if (params.length > 0 && providedParams < params.length) {
      invalidLines.push({
        lineNumber: i + 1,
        reason: t("campaigns.missingParameters", {
          needed: params.length,
          has: providedParams,
        }),
      });
    }
  }

  return {
    isValid: invalidLines.length === 0 && lines.length > 0,
    totalLines: lines.length,
    validLines: lines.length - invalidLines.length,
    invalidLines,
  };
});

function normalizePhoneNumber(phone: string): string {
  return phone.replace(/[^\d+]/g, "");
}

function isValidRecipientPhone(phone: string): boolean {
  return /^\+?\d{10,15}$/.test(phone);
}

function triggerCampaignFilePicker() {
  campaignFileInput.value?.click();
}

function clearCampaignMediaSelection() {
  campaignMediaFile.value = null;
  campaignMediaInputKey.value += 1;
}

function isAllowedCampaignMediaFile(file: File): boolean {
  const fileType = (file.type || "").toLowerCase();
  if (fileType && allowedCampaignMediaMimeTypes.has(fileType)) {
    return true;
  }

  const fileNameLower = file.name.toLowerCase();
  return allowedCampaignMediaExtensions.some((ext) =>
    fileNameLower.endsWith(ext),
  );
}

function handleCampaignMediaFileSelect(event: Event) {
  const input = event.target as HTMLInputElement | null;
  const selected = input?.files?.[0] ?? null;
  if (!selected) {
    clearCampaignMediaSelection();
    return;
  }

  if (selected.size > maxCampaignMediaSizeBytes) {
    toast.error(t("campaigns.mediaFileTooLarge"));
    clearCampaignMediaSelection();
    return;
  }

  if (!isAllowedCampaignMediaFile(selected)) {
    toast.error(t("campaigns.mediaFileTypeUnsupported"));
    clearCampaignMediaSelection();
    return;
  }

  campaignMediaFile.value = selected;
}

function getContactRecipientDisplayName(contact: ContactRecipient): string {
  return contact.profile_name || contact.name || contact.phone_number;
}

function normalizeContactsForImport(sourceContacts: any[]): ContactRecipient[] {
  return sourceContacts
    .map((contact: any) => ({
      id: String(contact.id || ""),
      phone_number: normalizePhoneNumber(
        String(contact.phone_number || "").trim(),
      ),
      profile_name: contact.profile_name || "",
      name: contact.name || "",
      created_at:
        typeof contact.created_at === "string" ? contact.created_at : "",
    }))
    .filter(
      (contact: ContactRecipient) =>
        contact.id && isValidRecipientPhone(contact.phone_number),
    );
}

function buildContactsForImportParams(options?: {
  page?: number;
  limit?: number;
}) {
  const baseParams: Parameters<typeof contactsService.list>[0] = {
    page: options?.page ?? contactsImportPage.value,
    limit: options?.limit ?? contactsImportPageSize.value,
    search: contactsSearchQuery.value.trim() || undefined,
  };

  if (isWhatsmeowProvider.value) {
    const instanceId = contactsImportInstanceId.value;
    if (!instanceId) {
      return null;
    }

    return {
      ...baseParams,
      instance_id: instanceId,
      date_basis: contactsImportDateBasis.value,
      date_from: contactsDateFrom.value || undefined,
      date_to: contactsDateTo.value || undefined,
    };
  }

  return {
    ...baseParams,
    created_from: contactsDateFrom.value || undefined,
    created_to: contactsDateTo.value || undefined,
  };
}

async function pruneSelectedContactsForImport() {
  const selectedEntries = Object.entries(selectedContactsById.value);
  if (selectedEntries.length === 0) return;

  const selectedIds = new Set(selectedEntries.map(([id]) => id));
  const matchedIds = new Set<string>();
  let page = 1;
  let total = 0;

  while (matchedIds.size < selectedIds.size) {
    const params = buildContactsForImportParams({
      page,
      limit: contactsImportSelectionPrunePageSize,
    });
    if (!params) {
      selectedContactsById.value = {};
      return;
    }

    const response = await contactsService.list(params);
    const payload = (response.data as any).data || response.data;
    const pageContacts = normalizeContactsForImport(
      Array.isArray(payload.contacts) ? payload.contacts : [],
    );

    for (const contact of pageContacts) {
      if (selectedIds.has(contact.id)) {
        matchedIds.add(contact.id);
      }
    }

    const payloadTotal = Number(payload.total);
    total =
      Number.isFinite(payloadTotal) && payloadTotal >= 0
        ? payloadTotal
        : (page - 1) * contactsImportSelectionPrunePageSize +
          pageContacts.length;

    if (
      pageContacts.length === 0 ||
      page * contactsImportSelectionPrunePageSize >= total
    ) {
      break;
    }

    page += 1;
  }

  if (matchedIds.size === selectedIds.size) {
    return;
  }

  selectedContactsById.value = Object.fromEntries(
    selectedEntries.filter(([id]) => matchedIds.has(id)),
  );
}

function toggleContactSelection(contact: ContactRecipient, checked: boolean) {
  const contactId = contact.id;
  if (checked) {
    selectedContactsById.value = {
      ...selectedContactsById.value,
      [contactId]: contact,
    };
    return;
  }
  const next = { ...selectedContactsById.value };
  delete next[contactId];
  selectedContactsById.value = next;
}

function toggleAllFilteredContacts(checked: boolean) {
  if (checked) {
    const next = { ...selectedContactsById.value };
    for (const contact of filteredContactsForImport.value) {
      next[contact.id] = contact;
    }
    selectedContactsById.value = next;
    return;
  }

  const next = { ...selectedContactsById.value };
  for (const contact of filteredContactsForImport.value) {
    delete next[contact.id];
  }
  selectedContactsById.value = next;
}

function goToContactsImportPage(page: number) {
  if (isLoadingContactsForImport.value) return;
  if (page < 1 || page > contactsImportTotalPages.value) return;
  contactsImportPage.value = page;
  void fetchContactsForImport();
}

function updateContactsImportPageSize(value: unknown) {
  if (value === null || value === undefined || typeof value === "boolean")
    return;
  const parsed = Number(value);
  if (
    !Number.isFinite(parsed) ||
    parsed < 1 ||
    parsed === contactsImportPageSize.value
  ) {
    return;
  }

  contactsImportPageSize.value = parsed;
  contactsImportPage.value = 1;
  void fetchContactsForImport();
}

async function fetchContactsForImport(options?: { pruneSelection?: boolean }) {
  isLoadingContactsForImport.value = true;
  try {
    const params = buildContactsForImportParams();
    if (!params) {
      contactsForImport.value = [];
      contactsImportTotal.value = 0;
      if (options?.pruneSelection) {
        selectedContactsById.value = {};
      }
      return;
    }

    const response = await contactsService.list(params);
    const payload = (response.data as any).data || response.data;
    contactsForImport.value = normalizeContactsForImport(
      Array.isArray(payload.contacts) ? payload.contacts : [],
    );

    const total = Number(payload.total);
    contactsImportTotal.value =
      Number.isFinite(total) && total >= 0
        ? total
        : (contactsImportPage.value - 1) * contactsImportPageSize.value +
          contactsForImport.value.length;

    if (options?.pruneSelection) {
      await pruneSelectedContactsForImport();
    }
  } catch (error: any) {
    console.error("Failed to fetch contacts for campaign recipients:", error);
    contactsForImport.value = [];
    contactsImportTotal.value = 0;
    toast.error(
      getErrorMessage(
        error,
        t("common.failedLoad", { resource: t("resources.contacts") }),
      ),
    );
  } finally {
    isLoadingContactsForImport.value = false;
  }
}

// Form state
const newCampaign = ref({
  name: "",
  whatsapp_account: "",
  template_id: "",
  body_content: "",
  min_delay_minutes: 0,
  max_delay_minutes: 0,
  poll_enabled: false,
  poll_question: "",
  poll_options: ["", ""] as string[],
  poll_max_selections: 0,
});
const campaignPlaceholderTokens = [
  "{customer_name}",
  "{chat_id}",
  "{agent_name}",
  "{organization_name}",
  "{contact_name}",
  "{phone_number}",
];

function addPollOption() {
  if (newCampaign.value.poll_options.length >= 12) return;
  newCampaign.value.poll_options = [...newCampaign.value.poll_options, ""];
}

function removePollOption(index: number) {
  if (newCampaign.value.poll_options.length <= 2) return;
  newCampaign.value.poll_options = newCampaign.value.poll_options.filter((_: string, i: number) => i !== index);
}

function appendCampaignPlaceholder(token: string) {
  newCampaign.value.body_content = `${newCampaign.value.body_content || ""}${token}`;
}

const isContentPickerOpen = ref(false);

async function handleContentSelect(content: SavedContent) {
  newCampaign.value.body_content = content.body;
  if (content.media_mime_type && content.id) {
    try {
      const response = await savedContentsService.getMedia(content.id);
      const blob = new Blob([response.data], {
        type: response.headers["content-type"],
      });
      const fileName = content.media_filename || `saved-content-${content.id}`;
      const file = new File([blob], fileName, { type: blob.type });
      campaignMediaFile.value = file;
    } catch {
      campaignMediaFile.value = null;
    }
  }
}

const selectedCreateTemplate = computed(
  () =>
    templates.value.find(
      (template) => template.id === newCampaign.value.template_id,
    ) || null,
);

const canUploadMediaInForm = computed(() => {
  if (isWhatsmeowProvider.value) return true;
  if (!newCampaign.value.template_id) return false;

  const template = selectedCreateTemplate.value;
  if (!template) return true;

  const headerType = String(template.header_type || "").toUpperCase();
  if (!headerType) return true;
  return headerType !== "TEXT" && headerType !== "NONE";
});

// AlertDialog state
const deleteDialogOpen = ref(false);
const cancelDialogOpen = ref(false);
const campaignToDelete = ref<Campaign | null>(null);
const campaignToCancel = ref<Campaign | null>(null);

// WebSocket subscription for real-time stats updates
let unsubscribeCampaignStats: (() => void) | null = null;

onMounted(async () => {
  await configStore.fetchConfig();
  await Promise.all([fetchCampaigns(), fetchSenders()]);

  // Subscribe to campaign stats updates
  unsubscribeCampaignStats = wsService.onCampaignStatsUpdate((payload) => {
    const campaign = campaigns.value.find((c) => c.id === payload.campaign_id);
    if (campaign) {
      campaign.sent_count = payload.sent_count;
      campaign.delivered_count = payload.delivered_count;
      campaign.read_count = payload.read_count;
      campaign.failed_count = payload.failed_count;
      if (payload.status) {
        campaign.status = payload.status;
      }
    }
  });
});

onUnmounted(() => {
  if (unsubscribeCampaignStats) {
    unsubscribeCampaignStats();
  }
  for (const url of Object.values(mediaBlobUrls.value)) {
    URL.revokeObjectURL(url);
  }
});

async function fetchCampaigns() {
  isLoading.value = true;
  try {
    const { from, to } = getDateRange.value;
    const params: Record<string, string | number> = {
      from,
      to,
      page: currentPage.value,
      limit: pageSize,
    };
    if (filterStatus.value && filterStatus.value !== "all") {
      params.status = filterStatus.value;
    }
    if (searchQuery.value) {
      params.search = searchQuery.value;
    }
    const response = await campaignsService.list(params);
    // API returns: { status: "success", data: { campaigns: [...], total: N } }
    const data = response.data.data || response.data;
    campaigns.value = data.campaigns || [];
    totalItems.value = data.total ?? campaigns.value.length;
  } catch (error) {
    console.error("Failed to fetch campaigns:", error);
    campaigns.value = [];
    totalItems.value = 0;
  } finally {
    isLoading.value = false;
  }
}

function applyCustomRange() {
  if (customDateRange.value.start && customDateRange.value.end) {
    isDatePickerOpen.value = false;
    fetchCampaigns();
  }
}

// Debounced search
const debouncedSearch = useDebounceFn(() => {
  currentPage.value = 1;
  fetchCampaigns();
}, 300);

watch(searchQuery, () => debouncedSearch());

// Watch for filter changes
watch([filterStatus, selectedRange], () => {
  currentPage.value = 1;
  if (selectedRange.value !== "custom") {
    fetchCampaigns();
  }
});

const debouncedContactsImportFilters = useDebounceFn(() => {
  contactsImportPage.value = 1;
  void fetchContactsForImport({ pruneSelection: true });
}, 350);

watch(
  [
    contactsSearchQuery,
    contactsDateFrom,
    contactsDateTo,
    contactsImportDateBasis,
  ],
  () => {
    if (!showAddRecipientsDialog.value) return;
    debouncedContactsImportFilters();
  },
);

async function fetchTemplates(account?: string) {
  templatesLoading.value = true;
  try {
    const response = await templatesService.list(
      account ? { account } : undefined,
    );
    const data = (response.data as any).data || response.data;
    templates.value = data.templates || [];
  } catch (error) {
    console.error("Failed to fetch templates:", error);
    templates.value = [];
  } finally {
    templatesLoading.value = false;
  }
}

async function fetchSenders() {
  if (isWhatsmeowProvider.value) {
    await fetchInstances();
    return;
  }
  await fetchAccounts();
}

// Re-fetch templates when account changes
watch(
  () => newCampaign.value.whatsapp_account,
  (account) => {
    newCampaign.value.template_id = "";
    if (isWhatsmeowProvider.value) {
      templates.value = [];
      return;
    }
    if (account) {
      fetchTemplates(account);
    } else {
      templates.value = [];
    }
  },
);

async function fetchAccounts() {
  try {
    const response = await accountsService.list();
    accounts.value = response.data.data?.accounts || [];
    instances.value = [];
  } catch (error) {
    console.error("Failed to fetch accounts:", error);
    accounts.value = [];
  }
}

async function fetchInstances() {
  try {
    const response = await instancesService.list();
    const data = (response.data as any).data || response.data;
    instances.value = Array.isArray(data) ? data : data.instances || [];
    accounts.value = [];
  } catch (error) {
    console.error("Failed to fetch instances:", error);
    instances.value = [];
  }
}

async function createCampaign() {
  const name = newCampaign.value.name.trim();
  const bodyContent = newCampaign.value.body_content.trim();
  const minDelayMinutes = Number(newCampaign.value.min_delay_minutes);
  const maxDelayMinutes = Number(newCampaign.value.max_delay_minutes);

  if (!name) {
    toast.error(t("campaigns.enterCampaignName"));
    return;
  }
  if (!newCampaign.value.whatsapp_account) {
    toast.error(
      isWhatsmeowProvider.value
        ? t("campaigns.selectInstance")
        : t("campaigns.selectWhatsappAccount"),
    );
    return;
  }
  if (!isWhatsmeowProvider.value && !newCampaign.value.template_id) {
    toast.error(t("campaigns.selectTemplateRequired"));
    return;
  }
  if (isWhatsmeowProvider.value && !bodyContent) {
    toast.error(t("campaigns.enterMessageBody"));
    return;
  }
  if (
    !Number.isFinite(minDelayMinutes) ||
    !Number.isFinite(maxDelayMinutes) ||
    minDelayMinutes < 0 ||
    maxDelayMinutes < 0 ||
    minDelayMinutes > maxDelayMinutes
  ) {
    toast.error(t("campaigns.invalidDelayRange"));
    return;
  }

  isCreating.value = true;
  try {
    const payload: Record<string, any> = {
      name,
      whatsapp_account: newCampaign.value.whatsapp_account,
      min_delay_seconds: Math.floor(minDelayMinutes * 60),
      max_delay_seconds: Math.floor(maxDelayMinutes * 60),
    };
    if (isWhatsmeowProvider.value) {
      payload.body_content = bodyContent;
    } else {
      payload.template_id = newCampaign.value.template_id;
    }

    // Include poll data if enabled and valid
    if (newCampaign.value.poll_enabled && newCampaign.value.poll_question.trim()) {
      const validOptions = newCampaign.value.poll_options.filter((o: string) => o.trim() !== "");
      if (validOptions.length >= 2) {
        payload.poll_question = newCampaign.value.poll_question.trim();
        payload.poll_options = validOptions;
        payload.poll_max_selections = newCampaign.value.poll_max_selections || 0;
      }
    }

    const createResponse = await campaignsService.create(payload);
    const createdCampaign =
      (createResponse.data as any).data || createResponse.data;
    const createdCampaignID = createdCampaign?.id
      ? String(createdCampaign.id)
      : "";

    let mediaUploadError: unknown = null;
    if (campaignMediaFile.value) {
      if (!createdCampaignID) {
        mediaUploadError = new Error(
          "Campaign ID was missing after create response",
        );
      } else {
        try {
          await campaignsService.uploadMedia(
            createdCampaignID,
            campaignMediaFile.value,
          );
        } catch (uploadError) {
          mediaUploadError = uploadError;
        }
      }
    }

    if (mediaUploadError) {
      toast.error(
        getErrorMessage(
          mediaUploadError,
          t("campaigns.campaignCreatedMediaUploadFailed"),
        ),
      );
    } else {
      toast.success(
        t("common.createdSuccess", { resource: t("resources.Campaign") }),
      );
    }

    showCreateDialog.value = false;
    resetForm();
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedCreate", { resource: t("resources.campaign") }),
      ),
    );
  } finally {
    isCreating.value = false;
  }
}

function resetForm() {
  newCampaign.value = {
    name: "",
    whatsapp_account: "",
    template_id: "",
    body_content: "",
    min_delay_minutes: 0,
    max_delay_minutes: 0,
    poll_enabled: false,
    poll_question: "",
    poll_options: ["", ""],
    poll_max_selections: 0,
  };
  clearCampaignMediaSelection();
}

async function openEditDialog(campaign: Campaign) {
  editingCampaignId.value = campaign.id;
  clearCampaignMediaSelection();
  newCampaign.value = {
    name: campaign.name,
    whatsapp_account: campaign.whatsapp_account || "",
    template_id: campaign.template_id || "",
    body_content: "",
    min_delay_minutes: Math.max(
      0,
      Math.floor((campaign.min_delay_seconds || 0) / 60),
    ),
    max_delay_minutes: Math.max(
      0,
      Math.floor((campaign.max_delay_seconds || 0) / 60),
    ),
    poll_enabled: !!(campaign.poll_question && campaign.poll_options?.length),
    poll_question: campaign.poll_question || "",
    poll_options: campaign.poll_options?.length ? campaign.poll_options : ["", ""],
    poll_max_selections: campaign.poll_max_selections || 0,
  };

  if (isWhatsmeowProvider.value && campaign.template_id) {
    try {
      const response = await templatesService.get(campaign.template_id);
      const template = (response.data as any).data || response.data;
      newCampaign.value.body_content = template?.body_content || "";
    } catch (error) {
      console.error("Failed to fetch template for campaign edit:", error);
      newCampaign.value.body_content = "";
    }
  }

  showCreateDialog.value = true;
}

function openCreateDialog() {
  editingCampaignId.value = null;
  resetForm();
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  showCreateDialog.value = false;
  editingCampaignId.value = null;
  resetForm();
}

async function saveCampaign() {
  const minDelayMinutes = Number(newCampaign.value.min_delay_minutes);
  const maxDelayMinutes = Number(newCampaign.value.max_delay_minutes);

  if (!newCampaign.value.name.trim()) {
    toast.error(t("campaigns.enterCampaignName"));
    return;
  }
  if (!newCampaign.value.whatsapp_account) {
    toast.error(
      isWhatsmeowProvider.value
        ? t("campaigns.selectInstance")
        : t("campaigns.selectWhatsappAccount"),
    );
    return;
  }
  if (isWhatsmeowProvider.value && !newCampaign.value.body_content.trim()) {
    toast.error(t("campaigns.enterMessageBody"));
    return;
  }
  if (
    !Number.isFinite(minDelayMinutes) ||
    !Number.isFinite(maxDelayMinutes) ||
    minDelayMinutes < 0 ||
    maxDelayMinutes < 0 ||
    minDelayMinutes > maxDelayMinutes
  ) {
    toast.error(t("campaigns.invalidDelayRange"));
    return;
  }

  if (editingCampaignId.value) {
    // Update existing campaign
    isCreating.value = true;
    try {
      const payload: Record<string, any> = {
        name: newCampaign.value.name.trim(),
        whatsapp_account: newCampaign.value.whatsapp_account,
        min_delay_seconds: Math.floor(minDelayMinutes * 60),
        max_delay_seconds: Math.floor(maxDelayMinutes * 60),
      };
      if (isWhatsmeowProvider.value) {
        payload.body_content = newCampaign.value.body_content.trim();
      } else {
        payload.template_id = newCampaign.value.template_id;
      }
      await campaignsService.update(editingCampaignId.value, payload);

      let mediaUploadError: unknown = null;
      if (campaignMediaFile.value) {
        try {
          await campaignsService.uploadMedia(
            editingCampaignId.value,
            campaignMediaFile.value,
          );
        } catch (uploadError) {
          mediaUploadError = uploadError;
        }
      }

      toast.success(
        t("common.updatedSuccess", { resource: t("resources.Campaign") }),
      );
      if (mediaUploadError) {
        toast.error(
          getErrorMessage(mediaUploadError, t("campaigns.mediaUploadFailed")),
        );
      }

      showCreateDialog.value = false;
      editingCampaignId.value = null;
      resetForm();
      await fetchCampaigns();
    } catch (error: any) {
      toast.error(
        getErrorMessage(
          error,
          t("common.failedUpdate", { resource: t("resources.campaign") }),
        ),
      );
    } finally {
      isCreating.value = false;
    }
  } else {
    // Create new campaign
    await createCampaign();
  }
}

async function startCampaign(campaign: Campaign) {
  try {
    await campaignsService.start(campaign.id);
    toast.success(t("campaigns.campaignStarted"));
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.startFailed")));
  }
}

async function pauseCampaign(campaign: Campaign) {
  try {
    await campaignsService.pause(campaign.id);
    toast.success(t("campaigns.campaignPaused"));
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.pauseFailed")));
  }
}

function openCancelDialog(campaign: Campaign) {
  campaignToCancel.value = campaign;
  cancelDialogOpen.value = true;
}

async function confirmCancelCampaign() {
  if (!campaignToCancel.value) return;

  try {
    await campaignsService.cancel(campaignToCancel.value.id);
    toast.success(t("campaigns.campaignCancelled"));
    cancelDialogOpen.value = false;
    campaignToCancel.value = null;
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.cancelFailed")));
  }
}

async function retryFailed(campaign: Campaign) {
  try {
    const response = await campaignsService.retryFailed(campaign.id);
    const result = response.data.data;
    toast.success(
      t("campaigns.retryingFailed", { count: result?.retry_count || 0 }),
    );
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.retryFailedError")));
  }
}

function openDeleteDialog(campaign: Campaign) {
  campaignToDelete.value = campaign;
  deleteDialogOpen.value = true;
}

async function confirmDeleteCampaign() {
  if (!campaignToDelete.value) return;

  try {
    await campaignsService.delete(campaignToDelete.value.id);
    toast.success(
      t("common.deletedSuccess", { resource: t("resources.Campaign") }),
    );
    deleteDialogOpen.value = false;
    campaignToDelete.value = null;
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedDelete", { resource: t("resources.campaign") }),
      ),
    );
  }
}

function getStatusIcon(status: string) {
  switch (status) {
    case "completed":
      return CheckCircle;
    case "running":
    case "processing":
    case "queued":
      return Play;
    case "paused":
      return Pause;
    case "scheduled":
      return Clock;
    case "failed":
    case "cancelled":
      return AlertCircle;
    default:
      return Megaphone;
  }
}

function getStatusClass(status: string): string {
  switch (status) {
    case "completed":
      return "border-primary/40 text-primary";
    case "running":
    case "processing":
    case "queued":
      return "border-primary text-primary";
    case "failed":
    case "cancelled":
      return "border-destructive text-destructive";
    default:
      return "";
  }
}

function getProgressPercentage(campaign: Campaign): number {
  if (campaign.total_recipients === 0) return 0;
  return Math.round((campaign.sent_count / campaign.total_recipients) * 100);
}

// Cache for media blob URLs and loading states
const mediaBlobUrls = ref<Record<string, string>>({});
const mediaLoadingState = ref<Record<string, "loading" | "loaded" | "error">>(
  {},
);

async function loadMediaPreview(campaignId: string) {
  if (mediaLoadingState.value[campaignId]) return; // Already loading or loaded

  mediaLoadingState.value[campaignId] = "loading";
  try {
    const response = await campaignsService.getMedia(campaignId);
    const blob = new Blob([response.data], {
      type: response.headers["content-type"],
    });
    mediaBlobUrls.value[campaignId] = URL.createObjectURL(blob);
    mediaLoadingState.value[campaignId] = "loaded";
  } catch (error) {
    console.error("Failed to load media preview:", error);
    mediaLoadingState.value[campaignId] = "error";
  }
}

function getMediaPreviewUrl(campaignId: string): string {
  if (!mediaLoadingState.value[campaignId]) {
    loadMediaPreview(campaignId);
  }
  return mediaBlobUrls.value[campaignId] || "";
}

// Media preview dialog
const showMediaPreviewDialog = ref(false);
const previewingCampaign = ref<Campaign | null>(null);

// Recipients functions
const deletingRecipientId = ref<string | null>(null);

async function viewRecipients(campaign: Campaign) {
  selectedCampaign.value = campaign;
  showRecipientsDialog.value = true;
  isLoadingRecipients.value = true;
  try {
    const response = await campaignsService.getRecipients(campaign.id);
    recipients.value = response.data.data?.recipients || [];
  } catch (error) {
    console.error("Failed to fetch recipients:", error);
    toast.error(
      t("common.failedLoad", { resource: t("resources.recipients") }),
    );
    recipients.value = [];
  } finally {
    isLoadingRecipients.value = false;
  }
}

async function deleteRecipient(recipientId: string) {
  if (!selectedCampaign.value) return;

  deletingRecipientId.value = recipientId;
  try {
    await campaignsService.deleteRecipient(
      selectedCampaign.value.id,
      recipientId,
    );
    recipients.value = recipients.value.filter((r) => r.id !== recipientId);
    // Update recipient count in selectedCampaign
    selectedCampaign.value.total_recipients = recipients.value.length;
    toast.success(
      t("common.deletedSuccess", { resource: t("resources.Recipient") }),
    );
    await fetchCampaigns(); // Refresh campaigns list
    // Update selectedCampaign with fresh data
    const updated = campaigns.value.find(
      (c) => c.id === selectedCampaign.value?.id,
    );
    if (updated) {
      selectedCampaign.value = updated;
    }
  } catch (error: any) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedDelete", { resource: t("resources.recipient") }),
      ),
    );
  } finally {
    deletingRecipientId.value = null;
  }
}

// Group targeting functions
async function viewGroups(campaign: Campaign) {
  selectedCampaign.value = campaign;
  showGroupsDialog.value = true;
  isLoadingGroups.value = true;
  try {
    const response = await campaignsService.getGroups(campaign.id);
    groupRecipients.value = (response.data.data || []) as GroupRecipient[];
  } catch (error) {
    console.error("Failed to fetch group recipients:", error);
    toast.error(getErrorMessage(error, t("campaigns.groupValidationFailed")));
    groupRecipients.value = [];
  } finally {
    isLoadingGroups.value = false;
  }
}

async function deleteGroupTarget(recipientId: string) {
  if (!selectedCampaign.value) return;
  try {
    await campaignsService.deleteGroup(selectedCampaign.value.id, recipientId);
    groupRecipients.value = groupRecipients.value.filter((r) => r.id !== recipientId);
    selectedCampaign.value.total_recipients = Math.max(0, (selectedCampaign.value.total_recipients || 0) - 1);
    toast.success(t("campaigns.groupTargetRemoved"));
    await fetchCampaigns();
    const updated = campaigns.value.find((c) => c.id === selectedCampaign.value?.id);
    if (updated) selectedCampaign.value = updated;
  } catch (error) {
    toast.error(getErrorMessage(error, t("common.failedDelete", { resource: t("resources.recipient") })));
  }
}

function openAddGroupsDialog() {
  groupSourceInstanceId.value = selectedCampaign.value?.whatsapp_account || "";
  availableGroups.value = [];
  selectedAvailableGroupIds.value = new Set();
  showAddGroupsDialog.value = true;
}

async function loadAvailableGroups() {
  if (!groupSourceInstanceId.value) {
    toast.error(t("campaigns.loadAvailableGroupsHint"));
    return;
  }
  isLoadingAvailableGroups.value = true;
  availableGroups.value = [];
  try {
    const response = await campaignsService.listInstanceGroups(groupSourceInstanceId.value);
    availableGroups.value = response.data.data || [];
  } catch (error) {
    console.error("Failed to load available groups:", error);
    toast.error(getErrorMessage(error, t("campaigns.groupValidationFailed")));
  } finally {
    isLoadingAvailableGroups.value = false;
  }
}

async function addGroups() {
  if (!selectedCampaign.value || selectedAvailableGroupIds.value.size === 0) return;
  const existingJids = new Set(groupRecipients.value.map((r) => r.group_jid));
  const selected = availableGroups.value
    .filter((g) => selectedAvailableGroupIds.value.has(g.jid) && !existingJids.has(g.jid))
    .map((g) => ({ jid: g.jid, name: g.name, participant_count: g.participant_count }));

  if (selected.length === 0) {
    toast.info(t("campaigns.groupAlreadyAdded"));
    return;
  }

  try {
    await campaignsService.addGroups(selectedCampaign.value.id, selected);
    toast.success(t("campaigns.groupTargetAdded"));
    showAddGroupsDialog.value = false;
    const response = await campaignsService.getGroups(selectedCampaign.value.id);
    groupRecipients.value = (response.data.data || []) as GroupRecipient[];
    await fetchCampaigns();
    const updated = campaigns.value.find((c) => c.id === selectedCampaign.value?.id);
    if (updated) selectedCampaign.value = updated;
  } catch (error) {
    toast.error(getErrorMessage(error, t("campaigns.groupValidationFailed")));
  }
}

const alreadyAddedGroupJids = computed(() => new Set(groupRecipients.value.map((r) => r.group_jid)));

async function addRecipients() {
  if (!selectedCampaign.value) return;

  const lines = recipientsInput.value
    .trim()
    .split("\n")
    .filter((line) => line.trim());
  if (lines.length === 0) {
    toast.error(t("campaigns.enterPhoneNumber"));
    return;
  }

  // Get template parameter names for mapping
  const paramNames = templateParamNames.value;

  // Parse CSV/text input - format: phone_number, param1, param2, ...
  // Parameters are mapped to template parameter names in order
  const recipientsList = lines.map((line) => {
    const parts = line.split(",").map((p) => p.trim());
    const recipient: {
      phone_number: string;
      recipient_name?: string;
      template_params?: Record<string, any>;
    } = {
      phone_number: normalizePhoneNumber(parts[0]), // Clean phone number
    };

    // Map values to template parameter names
    const params: Record<string, any> = {};
    for (let i = 1; i < parts.length && i <= paramNames.length; i++) {
      if (parts[i] && parts[i].length > 0) {
        params[paramNames[i - 1]] = parts[i];
      }
    }

    if (Object.keys(params).length > 0) {
      recipient.template_params = params;
    }
    return recipient;
  });

  isAddingRecipients.value = true;
  try {
    const response = await campaignsService.addRecipients(
      selectedCampaign.value.id,
      recipientsList,
    );
    const result = response.data.data;
    toast.success(
      t("campaigns.addedRecipients", {
        count: result?.added_count || recipientsList.length,
      }),
    );
    showAddRecipientsDialog.value = false;
    recipientsInput.value = "";
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.addRecipientsFailed")));
  } finally {
    isAddingRecipients.value = false;
  }
}

async function addRecipientsFromContacts() {
  if (!selectedCampaign.value) return;

  if (selectedContactsForImport.value.length === 0) {
    toast.error(t("campaigns.enterPhoneNumber"));
    return;
  }

  const recipientsList = selectedContactsForImport.value.map((contact) => {
    const phoneNumber = normalizePhoneNumber(contact.phone_number);
    const displayName = getContactRecipientDisplayName(contact).trim();
    const recipient: { phone_number: string; recipient_name?: string } = {
      phone_number: phoneNumber,
    };
    if (displayName && displayName !== phoneNumber) {
      recipient.recipient_name = displayName;
    }
    return recipient;
  });

  isAddingRecipients.value = true;
  try {
    const response = await campaignsService.addRecipients(
      selectedCampaign.value.id,
      recipientsList,
    );
    const result = response.data.data;
    toast.success(
      t("campaigns.addedRecipients", {
        count: result?.added_count || recipientsList.length,
      }),
    );
    showAddRecipientsDialog.value = false;
    selectedContactsById.value = {};
    contactsSearchQuery.value = "";
    contactsDateFrom.value = "";
    contactsDateTo.value = "";
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.addRecipientsFailed")));
  } finally {
    isAddingRecipients.value = false;
  }
}

function getRecipientStatusClass(status: string): string {
  switch (status) {
    case "sent":
    case "delivered":
      return "border-primary/40 text-primary";
    case "failed":
      return "border-destructive text-destructive";
    default:
      return "";
  }
}

// CSV functions
function getTemplateParamNames(template: Template): string[] {
  // Extract parameter names from body_content on-the-fly
  // Supports both positional ({{1}}, {{2}}) and named ({{name}}, {{order_id}}) parameters
  if (!template.body_content) return [];
  const matches = template.body_content.match(/\{\{([^}]+)\}\}/g) || [];
  const seen = new Set<string>();
  const names: string[] = [];
  for (const m of matches) {
    const name = m.replace(/[{}]/g, "").trim();
    if (name && !seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }
  return names;
}

interface TextPart {
  text: string;
  isParam: boolean;
  value?: string;
}

function parseTemplateParams(content: string): TextPart[] {
  if (!content) return [];
  const parts: TextPart[] = [];
  const regex = /\{\{([^}]+)\}\}/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(content)) !== null) {
    if (match.index > lastIndex) {
      parts.push({
        text: content.substring(lastIndex, match.index),
        isParam: false,
      });
    }
    parts.push({ text: match[0], isParam: true, value: match[1] });
    lastIndex = regex.lastIndex;
  }

  if (lastIndex < content.length) {
    parts.push({ text: content.substring(lastIndex), isParam: false });
  }

  return parts;
}

function hasMixedParamTypes(paramNames: string[]): boolean {
  // Check if template has both positional (numeric) and named parameters
  if (paramNames.length === 0) return false;
  const hasPositional = paramNames.some((n) => /^\d+$/.test(n));
  const hasNamed = paramNames.some((n) => !/^\d+$/.test(n));
  return hasPositional && hasNamed;
}

async function openAddRecipientsDialog(campaign: Campaign) {
  selectedCampaign.value = campaign;
  recipientsInput.value = "";
  csvFile.value = null;
  csvValidation.value = null;
  addRecipientsTab.value = "manual";
  selectedTemplate.value = null;
  contactsSearchQuery.value = "";
  contactsDateFrom.value = "";
  contactsDateTo.value = "";
  contactsImportDateBasis.value = "created";
  contactsImportPage.value = 1;
  contactsImportTotal.value = 0;
  selectedContactsById.value = {};
  contactsForImport.value = [];

  // Fetch template details to get body_content
  if (campaign.template_id) {
    try {
      const response = await templatesService.get(campaign.template_id);
      selectedTemplate.value = response.data.data || response.data;
    } catch (error) {
      console.error("Failed to fetch template:", error);
      selectedTemplate.value = null;
    }
  }

  showAddRecipientsDialog.value = true;
  void fetchContactsForImport();
}

function handleCSVFileSelect(event: Event) {
  const input = event.target as HTMLInputElement;
  if (input.files && input.files[0]) {
    csvFile.value = input.files[0];
    validateCSV();
  }
}

async function validateCSV() {
  if (!csvFile.value || !selectedTemplate.value) return;

  isValidatingCSV.value = true;
  csvValidation.value = null;

  try {
    const text = await csvFile.value.text();
    const lines = text.split("\n").filter((line) => line.trim());

    if (lines.length === 0) {
      csvValidation.value = {
        isValid: false,
        rows: [],
        templateParamNames: [],
        csvColumns: [],
        columnMapping: [],
        errors: [t("campaigns.csvEmpty")],
        warnings: [],
      };
      return;
    }

    // Parse header row
    const headerLine = lines[0];
    const headers = parseCSVLine(headerLine).map((h) => h.toLowerCase().trim());

    // Find required columns
    const phoneIndex = headers.findIndex(
      (h) =>
        h === "phone" ||
        h === "phone_number" ||
        h === "phonenumber" ||
        h === "mobile" ||
        h === "number",
    );
    const nameIndex = headers.findIndex(
      (h) =>
        h === "name" ||
        h === "recipient_name" ||
        h === "recipientname" ||
        h === "customer_name",
    );

    // Get template parameter names (e.g., ["name", "order_id"] or ["1", "2"])
    const templateParamNames = getTemplateParamNames(selectedTemplate.value);

    const globalErrors: string[] = [];
    const globalWarnings: string[] = [];

    if (phoneIndex === -1) {
      globalErrors.push(t("campaigns.missingPhoneColumn"));
    }

    // Warn about mixed param types
    if (hasMixedParamTypes(templateParamNames)) {
      globalWarnings.push(t("campaigns.mixedParamTypes"));
    }

    // Map CSV columns to template parameter names
    // Strategy:
    // 1. Try to match CSV headers to template param names directly
    // 2. Fall back to positional mapping for remaining params
    const paramColumnMapping: { csvIndex: number; paramName: string }[] = [];
    const usedCsvIndices = new Set<number>(
      [phoneIndex, nameIndex].filter((i) => i >= 0),
    );
    const mappedParamNames = new Set<string>();

    // First pass: exact matches between CSV headers and template param names
    for (const paramName of templateParamNames) {
      const csvIndex = headers.findIndex(
        (h, idx) =>
          !usedCsvIndices.has(idx) &&
          (h === paramName.toLowerCase() ||
            h === `param${paramName}` ||
            h === `{{${paramName}}}`),
      );
      if (csvIndex !== -1) {
        paramColumnMapping.push({ csvIndex, paramName });
        usedCsvIndices.add(csvIndex);
        mappedParamNames.add(paramName);
      }
    }

    // Second pass: positional mapping for unmapped params
    const remainingParamNames = templateParamNames.filter(
      (n) => !mappedParamNames.has(n),
    );
    const remainingCsvIndices = headers
      .map((_, idx) => idx)
      .filter((idx) => !usedCsvIndices.has(idx))
      .sort((a, b) => a - b);

    for (
      let i = 0;
      i < remainingParamNames.length && i < remainingCsvIndices.length;
      i++
    ) {
      paramColumnMapping.push({
        csvIndex: remainingCsvIndices[i],
        paramName: remainingParamNames[i],
      });
    }

    // Validate CSV columns match template params
    if (templateParamNames.length > 0) {
      // Check for missing columns (params that couldn't be mapped)
      const mappedCount = paramColumnMapping.length;
      if (mappedCount < templateParamNames.length) {
        const unmappedParams = templateParamNames.slice(mappedCount);
        globalErrors.push(
          t("campaigns.missingParamColumns", {
            params: unmappedParams.join(", "),
          }),
        );
      }

      // Warn if named params are being mapped positionally (not by column name)
      const namedParams = templateParamNames.filter((n) => !/^\d+$/.test(n));
      if (namedParams.length > 0) {
        const positionallyMapped = namedParams.filter(
          (n) => !mappedParamNames.has(n),
        );
        if (positionallyMapped.length > 0) {
          globalWarnings.push(
            t("campaigns.paramsMappedPositionally", {
              params: positionallyMapped.join(", "),
            }),
          );
        }
      }
    }

    // Parse data rows
    const rows: CSVRow[] = [];
    const seenPhones = new Map<string, number>(); // phone -> first occurrence row index

    for (let i = 1; i < lines.length; i++) {
      const values = parseCSVLine(lines[i]);
      if (values.length === 0 || (values.length === 1 && !values[0].trim()))
        continue;

      const rowErrors: string[] = [];
      const phone = phoneIndex >= 0 ? values[phoneIndex]?.trim() || "" : "";
      const cleanPhone = phone.replace(/[^\d+]/g, ""); // Normalize for duplicate check
      const name = nameIndex >= 0 ? values[nameIndex]?.trim() || "" : "";

      // Build params object with proper keys
      const params: Record<string, string> = {};
      for (const mapping of paramColumnMapping) {
        const value = values[mapping.csvIndex]?.trim() || "";
        if (value) {
          params[mapping.paramName] = value;
        }
      }

      // Validate phone number
      if (!phone) {
        rowErrors.push(t("campaigns.missingPhoneNumber"));
      } else if (!phone.match(/^\+?\d{10,15}$/)) {
        rowErrors.push(t("campaigns.invalidPhoneFormat"));
      } else {
        // Check for duplicates
        if (seenPhones.has(cleanPhone)) {
          rowErrors.push(
            t("campaigns.duplicatePhone", {
              row: seenPhones.get(cleanPhone)! + 1,
            }),
          );
        } else {
          seenPhones.set(cleanPhone, rows.length);
        }
      }

      // Validate params count if template requires params
      const providedParamCount = Object.keys(params).length;
      if (
        templateParamNames.length > 0 &&
        providedParamCount < templateParamNames.length
      ) {
        rowErrors.push(
          t("campaigns.templateRequiresParamsError", {
            required: templateParamNames.length,
            found: providedParamCount,
          }),
        );
      }

      rows.push({
        phone_number: phone,
        name,
        params,
        isValid: rowErrors.length === 0,
        errors: rowErrors,
      });
    }

    const validRows = rows.filter((r) => r.isValid);

    // Build column mapping for display
    const columnMapping = paramColumnMapping.map((m) => ({
      csvColumn: headers[m.csvIndex],
      paramName: m.paramName,
    }));

    csvValidation.value = {
      isValid: globalErrors.length === 0 && validRows.length > 0,
      rows,
      templateParamNames,
      csvColumns: headers,
      columnMapping,
      errors: globalErrors,
      warnings: globalWarnings,
    };
  } catch (error) {
    console.error("Failed to parse CSV:", error);
    csvValidation.value = {
      isValid: false,
      rows: [],
      templateParamNames: [],
      csvColumns: [],
      columnMapping: [],
      errors: [t("campaigns.parseCsvFailed")],
      warnings: [],
    };
  } finally {
    isValidatingCSV.value = false;
  }
}

function parseCSVLine(line: string): string[] {
  const result: string[] = [];
  let current = "";
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const char = line[i];

    if (char === '"') {
      if (inQuotes && line[i + 1] === '"') {
        current += '"';
        i++;
      } else {
        inQuotes = !inQuotes;
      }
    } else if (char === "," && !inQuotes) {
      result.push(current);
      current = "";
    } else {
      current += char;
    }
  }
  result.push(current);

  return result;
}

async function addRecipientsFromCSV() {
  if (!selectedCampaign.value || !csvValidation.value) return;

  const validRows = csvValidation.value.rows.filter((r) => r.isValid);
  if (validRows.length === 0) {
    toast.error(t("campaigns.noValidRowsToImport"));
    return;
  }

  const recipientsList = validRows.map((row) => {
    const recipient: {
      phone_number: string;
      recipient_name?: string;
      template_params?: Record<string, any>;
    } = {
      phone_number: normalizePhoneNumber(row.phone_number),
    };
    if (row.name) {
      recipient.recipient_name = row.name;
    }
    // Use params directly - already keyed by param name (e.g., {"name": "John"} or {"1": "John"})
    if (Object.keys(row.params).length > 0) {
      recipient.template_params = row.params;
    }
    return recipient;
  });

  isAddingRecipients.value = true;
  try {
    const response = await campaignsService.addRecipients(
      selectedCampaign.value.id,
      recipientsList,
    );
    const result = response.data.data;
    toast.success(
      t("campaigns.addedFromCsv", {
        count: result?.added_count || recipientsList.length,
      }),
    );
    showAddRecipientsDialog.value = false;
    csvFile.value = null;
    csvValidation.value = null;
    await fetchCampaigns();
  } catch (error: any) {
    toast.error(getErrorMessage(error, t("campaigns.addRecipientsFailed")));
  } finally {
    isAddingRecipients.value = false;
  }
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('campaigns.title')"
      :subtitle="$t('campaigns.subtitle')"
      :icon="Megaphone"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="openCreateDialog">
          <Plus class="h-4 w-4 mr-2" />
          {{ $t("campaigns.createCampaign") }}
        </Button>
      </template>
    </PageHeader>

    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{{
            editingCampaignId
              ? $t("campaigns.editCampaign")
              : $t("campaigns.createNewCampaign")
          }}</DialogTitle>
          <DialogDescription>
            {{
              editingCampaignId
                ? $t("campaigns.editDescription")
                : $t("campaigns.createDescription")
            }}
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-3 py-3">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="grid gap-2">
              <Label for="name">{{ $t("campaigns.campaignName") }}</Label>
              <Input
                id="name"
                v-model="newCampaign.name"
                :placeholder="$t('campaigns.campaignNamePlaceholder')"
                :disabled="isCreating"
              />
            </div>
            <div class="grid gap-2">
              <Label for="account">{{ senderLabel }}</Label>
              <Select
                v-model="newCampaign.whatsapp_account"
                :disabled="isCreating"
              >
                <SelectTrigger>
                  <SelectValue :placeholder="senderPlaceholder" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="option in senderOptions"
                    :key="option.value"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p
                v-if="senderOptions.length === 0"
                class="text-xs text-muted-foreground"
              >
                {{ noSendersFoundMessage }}
              </p>
            </div>
          </div>

          <Separator />

          <div>
            <p class="text-sm font-semibold mb-2">
              {{ $t("campaigns.messageSection") }}
            </p>
            <div v-if="!isWhatsmeowProvider" class="grid gap-2">
              <Label for="template">{{ $t("campaigns.messageTemplate") }}</Label>
              <Select v-model="newCampaign.template_id" :disabled="isCreating">
                <SelectTrigger>
                  <SelectValue :placeholder="$t('campaigns.selectTemplate')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="template in templates"
                    :key="template.id"
                    :value="template.id"
                  >
                    {{ template.display_name || template.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p
                v-if="templates.length === 0 && !isCreating"
                class="text-xs text-muted-foreground"
              >
                {{ $t("campaigns.noTemplatesFound") }}
              </p>
            </div>

            <div v-else class="grid gap-2">
              <div class="flex items-center justify-between">
                <Label for="body-content">{{ $t("campaigns.messageBody") }}</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="h-7 px-2 text-xs gap-1"
                  :disabled="isCreating"
                  @click="isContentPickerOpen = true"
                >
                  <BookOpen class="h-3.5 w-3.5" />
                  {{ $t("savedContents.pickerTitle") }}
                </Button>
              </div>
              <div class="flex flex-wrap items-center gap-1.5">
                <Button
                  v-for="token in campaignPlaceholderTokens"
                  :key="token"
                  type="button"
                  variant="outline"
                  size="sm"
                  class="h-7 px-2 text-xs"
                  :disabled="isCreating"
                  @click="appendCampaignPlaceholder(token)"
                >
                  {{ token }}
                </Button>
              </div>
              <WhatsAppRichTextEditor
                id="body-content"
                v-model="newCampaign.body_content"
                :placeholder="$t('campaigns.messageBodyPlaceholder')"
                :rows="3"
                :disabled="isCreating"
              />
              <p class="text-xs text-muted-foreground">
                {{ $t("campaigns.placeholderHint") }}
              </p>
            </div>
          </div>

          <Separator />

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <p class="text-sm font-semibold mb-2">
                {{ $t("campaigns.deliverySection") }}
              </p>
              <div class="grid gap-2">
                <p class="text-sm font-semibold mb-2">{{ $t("campaigns.delayBetweenMessages") }}</p>
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <div class="grid gap-1.5">
                    <Label for="delay-min" class="text-xs text-muted-foreground">
                      {{ $t("campaigns.delayFromMinutes") }}
                    </Label>
                    <Input
                      id="delay-min"
                      v-model.number="newCampaign.min_delay_minutes"
                      type="number"
                      min="0"
                      step="1"
                      :placeholder="$t('campaigns.delayFromMinutes')"
                      :disabled="isCreating"
                    />
                  </div>
                  <div class="grid gap-1.5">
                    <Label for="delay-max" class="text-xs text-muted-foreground">
                      {{ $t("campaigns.delayToMinutes") }}
                    </Label>
                    <Input
                      id="delay-max"
                      v-model.number="newCampaign.max_delay_minutes"
                      type="number"
                      min="0"
                      step="1"
                      :placeholder="$t('campaigns.delayToMinutes')"
                      :disabled="isCreating"
                    />
                  </div>
                </div>
                <p class="text-xs text-muted-foreground">
                  {{ $t("campaigns.delayRangeHint") }}
                </p>
              </div>
            </div>

            <div>
              <p class="text-sm font-semibold mb-2">
                {{ $t("campaigns.mediaSection") }}
              </p>
              <div class="grid gap-2">
                <Label for="media-file"
                  >{{ $t("campaigns.mediaFile") }} ({{
                    $t("common.optional")
                  }})</Label
                >
                <div class="flex items-center gap-2">
                  <input
                    id="media-file"
                    ref="campaignFileInput"
                    type="file"
                    :accept="campaignMediaAccept"
                    :disabled="isCreating || !canUploadMediaInForm"
                    class="hidden"
                    @change="handleCampaignMediaFileSelect"
                  />
                  <Button
                    variant="outline"
                    :disabled="isCreating || !canUploadMediaInForm"
                    @click="triggerCampaignFilePicker"
                  >
                    <Upload class="h-4 w-4 me-2" />
                    {{ campaignMediaFile ? campaignMediaFile.name : $t("campaigns.chooseFile") }}
                  </Button>
                  <Button
                    v-if="campaignMediaFile"
                    variant="ghost"
                    size="icon"
                    class="h-9 w-9"
                    :disabled="isCreating"
                    @click="clearCampaignMediaSelection"
                    :aria-label="$t('common.clear')"
                  >
                    <XCircle class="h-4 w-4" />
                  </Button>
                </div>
                <p class="text-xs text-muted-foreground">
                  {{
                    canUploadMediaInForm
                      ? $t("campaigns.mediaCreateHint")
                      : $t("campaigns.mediaNeedsHeaderTemplate")
                  }}
                </p>
              </div>
            </div>
          </div>

          <!-- Poll Section (whatsmeow only) -->
          <div v-if="isWhatsmeowProvider" class="space-y-3 pt-4 border-t mt-4">
            <div class="flex items-center gap-2">
              <input
                id="poll-toggle"
                type="checkbox"
                v-model="newCampaign.poll_enabled"
                class="h-4 w-4 rounded border-muted-foreground/40"
                :disabled="isCreating"
              />
              <Label for="poll-toggle" class="text-sm font-semibold cursor-pointer">
                Add Poll to Message
              </Label>
            </div>
            <div v-if="newCampaign.poll_enabled" class="space-y-3 pl-6">
              <div class="grid gap-1.5">
                <Label for="poll-question" class="text-xs text-muted-foreground">Question</Label>
                <Input
                  id="poll-question"
                  v-model="newCampaign.poll_question"
                  placeholder="e.g. Did you enjoy this content?"
                  :disabled="isCreating"
                />
              </div>
              <div class="space-y-2">
                <Label class="text-xs text-muted-foreground">Options</Label>
                <div
                  v-for="(_, idx) in newCampaign.poll_options"
                  :key="idx"
                  class="flex items-center gap-2"
                >
                  <Input
                    v-model="newCampaign.poll_options[idx]"
                    :placeholder="'Option ' + (idx + 1)"
                    :disabled="isCreating"
                    class="flex-1"
                  />
                  <Button
                    v-if="newCampaign.poll_options.length > 2"
                    variant="ghost"
                    size="icon"
                    class="h-8 w-8 shrink-0"
                    :disabled="isCreating"
                    @click="removePollOption(idx)"
                  >
                    <XCircle class="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Button
                  v-if="newCampaign.poll_options.length < 12"
                  variant="outline"
                  size="sm"
                  :disabled="isCreating"
                  @click="addPollOption"
                >
                  <Plus class="h-3.5 w-3.5 mr-1" />
                  Add Option
                </Button>
                <p class="text-xs text-muted-foreground">
                  Min 2, max 12 options. Leave empty options blank to exclude them.
                </p>
              </div>
              <div class="grid gap-1.5">
                <Label for="poll-max" class="text-xs text-muted-foreground">
                  Max selections (0 = unlimited)
                </Label>
                <Input
                  id="poll-max"
                  v-model.number="newCampaign.poll_max_selections"
                  type="number"
                  min="0"
                  :max="newCampaign.poll_options.length"
                  :disabled="isCreating"
                  class="w-24"
                />
              </div>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            @click="closeCreateDialog"
            :disabled="isCreating"
          >
            {{ $t("common.cancel") }}
          </Button>
          <Button @click="saveCampaign" :disabled="isCreating">
            <Loader2 v-if="isCreating" class="h-4 w-4 me-2 animate-spin" />
            {{
              editingCampaignId
                ? $t("campaigns.saveChanges")
                : $t("campaigns.createCampaign")
            }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Campaigns List -->
    <ScrollArea class="flex-1">
      <div class="min-h-full">
        <div class="flex items-start justify-between gap-4 px-6 pt-6 pb-3">
          <div class="min-w-0">
            <h2 class="text-lg font-semibold text-foreground">
              {{ $t("campaigns.yourCampaigns") }}
            </h2>
            <p class="text-sm text-muted-foreground">
              {{ $t("campaigns.yourCampaignsDesc") }}
            </p>
          </div>
          <div class="flex items-center gap-3 flex-wrap shrink-0">
            <div class="flex items-center gap-2">
              <Select v-model="filterStatus">
                <SelectTrigger class="w-[130px]">
                  <SelectValue :placeholder="$t('campaigns.allStatuses')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="opt in statusOptions"
                    :key="opt.value"
                    :value="opt.value"
                  >
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <div class="flex items-center gap-1">
                <Select v-model="selectedRange">
                  <SelectTrigger class="w-[130px]">
                    <SelectValue :placeholder="$t('campaigns.selectRange')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="today">{{
                      $t("campaigns.today")
                    }}</SelectItem>
                    <SelectItem value="7days">{{
                      $t("campaigns.last7Days")
                    }}</SelectItem>
                    <SelectItem value="30days">{{
                      $t("campaigns.last30Days")
                    }}</SelectItem>
                    <SelectItem value="this_month">{{
                      $t("campaigns.thisMonth")
                    }}</SelectItem>
                    <SelectItem value="custom">{{
                      $t("campaigns.customRange")
                    }}</SelectItem>
                  </SelectContent>
                </Select>
                <Popover
                  v-if="selectedRange === 'custom'"
                  v-model:open="isDatePickerOpen"
                >
                  <PopoverTrigger as-child>
                    <Button variant="outline" size="sm" class="h-9 px-2">
                      <CalendarIcon class="h-4 w-4" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent class="w-auto p-4" align="end">
                    <div class="space-y-4">
                      <RangeCalendar
                        v-model="customDateRange"
                        :number-of-months="2"
                      />
                      <Button
                        class="w-full"
                        size="sm"
                        @click="applyCustomRange"
                        :disabled="
                          !customDateRange.start || !customDateRange.end
                        "
                      >
                        {{ $t("campaigns.applyRange") }}
                      </Button>
                    </div>
                  </PopoverContent>
                </Popover>
              </div>
            </div>
            <SearchInput
              v-model="searchQuery"
              :placeholder="$t('campaigns.searchCampaigns') + '...'"
              class="w-48"
            />
          </div>
        </div>
        <div class="px-6 pb-6">
          <DataTable
            :items="campaigns"
            :columns="columns"
            :is-loading="isLoading"
            :empty-icon="Megaphone"
            :empty-title="
              searchQuery
                ? $t('campaigns.noMatchingCampaigns')
                : $t('campaigns.noCampaignsYet')
            "
            :empty-description="
              searchQuery
                ? $t('campaigns.noMatchingCampaignsDesc')
                : $t('campaigns.noCampaignsYetDesc')
            "
            v-model:sort-key="sortKey"
            v-model:sort-direction="sortDirection"
            server-pagination
            :current-page="currentPage"
            :total-items="totalItems"
            :page-size="pageSize"
            item-name="campaigns"
            @page-change="handlePageChange"
          >
            <template #cell-name="{ item: campaign }">
              <div>
                <span class="font-medium">{{ campaign.name }}</span>
                <p class="text-xs text-muted-foreground">
                  {{ campaign.template_name || $t("campaigns.noTemplate") }}
                </p>
              </div>
            </template>
            <template #cell-status="{ item: campaign }">
              <Badge
                variant="outline"
                :class="[getStatusClass(campaign.status), 'text-xs']"
              >
                <component
                  :is="getStatusIcon(campaign.status)"
                  class="h-3 w-3 mr-1"
                />
                {{ campaign.status }}
              </Badge>
            </template>
            <template #cell-stats="{ item: campaign }">
              <div class="space-y-1">
                <div
                  v-if="
                    campaign.status === 'running' ||
                    campaign.status === 'processing'
                  "
                  class="w-32"
                >
                  <Progress
                    :model-value="getProgressPercentage(campaign)"
                    class="h-1.5"
                  />
                  <span class="text-xs text-muted-foreground"
                    >{{ getProgressPercentage(campaign) }}%</span
                  >
                </div>
                <div class="flex items-center gap-3 text-xs">
                  <span :title="$t('campaigns.recipients')"
                    ><Users class="h-3 w-3 inline mr-0.5" />{{
                      campaign.total_recipients
                    }}</span
                  >
                  <span class="text-primary" :title="$t('campaigns.delivered')">{{
                    campaign.delivered_count
                  }}</span>
                  <span class="text-primary/70" :title="$t('campaigns.read')">{{
                    campaign.read_count
                  }}</span>
                  <span
                    v-if="campaign.failed_count > 0"
                    class="text-destructive"
                    :title="$t('campaigns.failed')"
                    >{{ campaign.failed_count }}</span
                  >
                </div>
              </div>
            </template>
            <template #cell-created_at="{ item: campaign }">
              <span class="text-muted-foreground text-sm">{{
                formatDate(campaign.created_at)
              }}</span>
            </template>
            <template #cell-actions="{ item: campaign }">
              <div class="flex items-center justify-end gap-0.5">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="viewRecipients(campaign)"
                  :aria-label="$t('campaigns.viewRecipients')"
                >
                  <Eye class="h-4 w-4" />
                </Button>
                <Button
                  v-if="campaign.status === 'draft'"
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="openAddRecipientsDialog(campaign as any)"
                  :aria-label="$t('campaigns.addRecipients')"
                >
                  <UserPlus class="h-4 w-4" />
                </Button>
                <Button
                  v-if="isWhatsmeowProvider && (campaign.status === 'draft' || campaign.status === 'paused')"
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="viewGroups(campaign)"
                  :aria-label="$t('campaigns.campaignGroups')"
                >
                  <UsersRound class="h-4 w-4" />
                </Button>
                <Button
                  v-if="campaign.status === 'draft'"
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="openEditDialog(campaign)"
                  :aria-label="$t('campaigns.editCampaign')"
                >
                  <Pencil class="h-4 w-4" />
                </Button>
                <Button
                  v-if="
                    campaign.status === 'draft' ||
                    campaign.status === 'scheduled'
                  "
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10 text-primary"
                  @click="startCampaign(campaign)"
                  :aria-label="$t('campaigns.start')"
                >
                  <Play class="h-4 w-4" />
                </Button>
                <Button
                  v-if="
                    campaign.status === 'running' ||
                    campaign.status === 'processing'
                  "
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="pauseCampaign(campaign)"
                  :aria-label="$t('campaigns.pause')"
                >
                  <Pause class="h-4 w-4" />
                </Button>
                <Button
                  v-if="campaign.status === 'paused'"
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10 text-primary"
                  @click="startCampaign(campaign)"
                  :aria-label="$t('campaigns.resume')"
                >
                  <Play class="h-4 w-4" />
                </Button>
                <Button
                  v-if="
                    campaign.failed_count > 0 &&
                    (campaign.status === 'completed' ||
                      campaign.status === 'paused' ||
                      campaign.status === 'failed')
                  "
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10"
                  @click="retryFailed(campaign)"
                  :aria-label="$t('campaigns.retryFailed')"
                >
                  <RefreshCw class="h-4 w-4" />
                </Button>
                <Button
                  v-if="
                    campaign.status === 'running' ||
                    campaign.status === 'paused' ||
                    campaign.status === 'processing' ||
                    campaign.status === 'queued'
                  "
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10 text-destructive"
                  @click="openCancelDialog(campaign)"
                  :aria-label="$t('campaigns.cancelCampaign')"
                >
                  <XCircle class="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-10 w-10 text-destructive"
                  @click="openDeleteDialog(campaign)"
                  :disabled="
                    campaign.status === 'running' ||
                    campaign.status === 'processing'
                  "
                  :aria-label="$t('campaigns.deleteCampaign')"
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </div>
            </template>
            <template #empty-action>
              <Button
                v-if="!searchQuery"
                variant="outline"
                size="sm"
                @click="showCreateDialog = true"
              >
                <Plus class="h-4 w-4 mr-2" />
                {{ $t("campaigns.createCampaign") }}
              </Button>
            </template>
          </DataTable>
        </div>
      </div>
    </ScrollArea>

    <!-- View Recipients Dialog -->
    <Dialog v-model:open="showRecipientsDialog">
      <DialogContent class="sm:max-w-[700px] max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>{{ $t("campaigns.campaignRecipients") }}</DialogTitle>
          <DialogDescription>
            {{ selectedCampaign?.name }} -
            {{ $t("campaigns.recipientCount", { count: recipients.length }) }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4">
          <div
            v-if="isLoadingRecipients"
            class="flex items-center justify-center py-8"
          >
            <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
          <div
            v-else-if="recipients.length === 0"
            class="text-center py-8 text-muted-foreground"
          >
            <Users class="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>{{ $t("campaigns.noRecipientsYet") }}</p>
            <Button
              v-if="selectedCampaign?.status === 'draft'"
              variant="outline"
              size="sm"
              class="mt-4"
              @click="
                showRecipientsDialog = false;
                openAddRecipientsDialog(selectedCampaign as any);
              "
            >
              <UserPlus class="h-4 w-4 mr-2" />
              {{ $t("campaigns.addRecipients") }}
            </Button>
          </div>
          <ScrollArea v-else class="h-[400px]">
              <table class="w-full text-sm">
                  <thead class="sticky top-0 bg-background border-b">
                    <tr>
                      <th scope="col" class="text-left py-2 px-3">
                        {{ $t("campaigns.phoneNumber") }}
                      </th>
                      <th scope="col" class="text-left py-2 px-3">
                        {{ $t("campaigns.name") }}
                      </th>
                      <th scope="col" class="text-left py-2 px-3">
                        {{ $t("campaigns.status") }}
                      </th>
                      <th scope="col" class="text-left py-2 px-3">
                        {{ $t("campaigns.sentAt") }}
                      </th>
                      <th
                        v-if="selectedCampaign?.status === 'draft'"
                        scope="col"
                        class="text-center py-2 px-3 w-14"
                      ></th>
                    </tr>
                  </thead>
              <tbody>
                <tr
                  v-for="recipient in recipients"
                  :key="recipient.id"
                  class="border-b"
                >
                  <td class="py-2.5 px-3 font-mono text-sm">
                    {{ recipient.phone_number }}
                  </td>
                  <td class="py-2.5 px-3 text-sm">
                    {{ recipient.recipient_name || "-" }}
                  </td>
                  <td class="py-2.5 px-3">
                    <div class="flex flex-col gap-1">
                      <Badge
                        variant="outline"
                        :class="getRecipientStatusClass(recipient.status)"
                      >
                        {{ recipient.status }}
                      </Badge>
                      <span
                        v-if="
                          recipient.status === 'failed' &&
                          recipient.error_message
                        "
                        class="text-xs text-destructive max-w-[200px] truncate"
                        :title="recipient.error_message"
                      >
                        {{ recipient.error_message }}
                      </span>
                    </div>
                  </td>
                  <td class="py-2.5 px-3 text-sm text-muted-foreground">
                    {{
                      recipient.sent_at ? formatDate(recipient.sent_at) : "-"
                    }}
                  </td>
                  <td
                    v-if="selectedCampaign?.status === 'draft'"
                    class="py-2.5 px-3 text-center"
                  >
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-9 w-9"
                      @click="deleteRecipient(recipient.id)"
                      :disabled="deletingRecipientId === recipient.id"
                      :aria-label="$t('common.delete')"
                    >
                      <Loader2
                        v-if="deletingRecipientId === recipient.id"
                        class="h-4 w-4 animate-spin"
                      />
                      <Trash2
                        v-else
                        class="h-4 w-4 text-muted-foreground hover:text-destructive"
                      />
                    </Button>
                  </td>
                </tr>
              </tbody>
            </table>
          </ScrollArea>
        </div>
        <DialogFooter>
          <Button
            v-if="selectedCampaign?.status === 'draft'"
            variant="outline"
            size="sm"
            @click="
              showRecipientsDialog = false;
              openAddRecipientsDialog(selectedCampaign as any);
            "
          >
            <UserPlus class="h-4 w-4 mr-2" />
            {{ $t("campaigns.addMore") }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            @click="showRecipientsDialog = false"
            >{{ $t("common.close") }}</Button
          >
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- View Groups Dialog -->
    <Dialog v-model:open="showGroupsDialog">
      <DialogContent class="sm:max-w-[700px] max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>{{ $t("campaigns.campaignGroups") }}</DialogTitle>
          <DialogDescription>
            {{ selectedCampaign?.name }} -
            {{ $t("campaigns.recipientCount", { count: groupRecipients.length }) }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4">
          <div v-if="isLoadingGroups" class="flex items-center justify-center py-8">
            <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
          <div v-else-if="groupRecipients.length === 0" class="text-center py-8 text-muted-foreground">
            <UsersRound class="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>{{ $t("campaigns.noGroupsYet") }}</p>
            <p class="text-sm mt-1">{{ $t("campaigns.noGroupsYetDesc") }}</p>
            <Button
              v-if="selectedCampaign?.status === 'draft'"
              variant="outline"
              size="sm"
              class="mt-4"
              @click="openAddGroupsDialog()"
            >
              <UserPlus class="h-4 w-4 mr-2" />
              {{ $t("campaigns.addGroupTargets") }}
            </Button>
          </div>
          <ScrollArea v-else class="h-[400px]">
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-background border-b">
                <tr>
                  <th scope="col" class="text-left py-2 px-3">{{ $t("campaigns.groupName") }}</th>
                  <th scope="col" class="text-left py-2 px-3">{{ $t("campaigns.groupJID") }}</th>
                  <th scope="col" class="text-left py-2 px-3">{{ $t("campaigns.groupParticipantCount") }}</th>
                  <th scope="col" class="text-left py-2 px-3">{{ $t("campaigns.status") }}</th>
                  <th v-if="selectedCampaign?.status === 'draft'" scope="col" class="text-center py-2 px-3 w-14"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="group in groupRecipients" :key="group.id" class="border-b">
                  <td class="py-2.5 px-3 text-sm">{{ group.group_name || group.recipient_name || "-" }}</td>
                  <td class="py-2.5 px-3 font-mono text-sm">{{ group.group_jid || group.phone_number }}</td>
                  <td class="py-2.5 px-3 text-sm">{{ group.participant_count ?? "-" }}</td>
                  <td class="py-2.5 px-3">
                    <Badge variant="outline" :class="getRecipientStatusClass(group.status)">
                      {{ group.status }}
                    </Badge>
                  </td>
                  <td v-if="selectedCampaign?.status === 'draft'" class="py-2.5 px-3 text-center">
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-destructive"
                      @click="deleteGroupTarget(group.id)"
                    >
                      <Trash2 class="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              </tbody>
            </table>
          </ScrollArea>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showGroupsDialog = false">{{ $t("common.close") }}</Button>
          <Button
            v-if="selectedCampaign?.status === 'draft'"
            @click="openAddGroupsDialog()"
          >
            <UserPlus class="h-4 w-4 mr-2" />
            {{ $t("campaigns.addGroupTargets") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Add Groups Dialog -->
    <Dialog v-model:open="showAddGroupsDialog">
      <DialogContent class="sm:max-w-[700px] max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>{{ $t("campaigns.selectGroupsToAdd") }}</DialogTitle>
          <DialogDescription>{{ $t("campaigns.campaignGroupsDesc") }}</DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-4">
          <div class="flex items-end gap-3">
            <div class="flex-1">
              <Label>{{ $t("campaigns.whatsappInstance") }}</Label>
              <Select v-model="groupSourceInstanceId">
                <SelectTrigger>
                  <SelectValue :placeholder="$t('campaigns.selectInstance')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="inst in instances" :key="inst.id" :value="inst.id">
                    {{ inst.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button
              :disabled="!groupSourceInstanceId || isLoadingAvailableGroups"
              @click="loadAvailableGroups()"
            >
              <Loader2 v-if="isLoadingAvailableGroups" class="h-4 w-4 mr-2 animate-spin" />
              <RefreshCw v-else class="h-4 w-4 mr-2" />
              {{ $t("campaigns.loadAvailableGroups") }}
            </Button>
          </div>

          <div v-if="!groupSourceInstanceId" class="text-center py-4 text-sm text-muted-foreground">
            {{ $t("campaigns.loadAvailableGroupsHint") }}
          </div>
          <div v-else-if="isLoadingAvailableGroups" class="flex items-center justify-center py-8">
            <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
          <div v-else-if="availableGroups.length === 0" class="text-center py-8 text-muted-foreground">
            <UsersRound class="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>{{ $t("campaigns.noAvailableGroups") }}</p>
          </div>
          <template v-else>
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>{{ $t("campaigns.availableGroupsCount", { count: availableGroups.length }) }}</span>
              <div class="flex gap-2">
                <Button variant="ghost" size="sm" @click="selectedAvailableGroupIds = new Set(availableGroups.filter(g => !alreadyAddedGroupJids.has(g.jid)).map(g => g.jid))">
                  {{ $t("common.selectAll") }}
                </Button>
                <Button variant="ghost" size="sm" @click="selectedAvailableGroupIds = new Set()">
                  {{ $t("common.clear") }}
                </Button>
              </div>
            </div>
            <ScrollArea class="h-[350px]">
              <table class="w-full text-sm">
                <thead class="sticky top-0 bg-background border-b">
                  <tr>
                    <th class="py-2 px-3 w-10"></th>
                    <th class="text-left py-2 px-3">{{ $t("campaigns.groupName") }}</th>
                    <th class="text-left py-2 px-3">{{ $t("campaigns.groupJID") }}</th>
                    <th class="text-left py-2 px-3">{{ $t("campaigns.groupParticipantCount") }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="group in availableGroups" :key="group.jid" class="border-b"
                      :class="{ 'opacity-50': alreadyAddedGroupJids.has(group.jid) }">
                    <td class="py-2.5 px-3 text-center">
                      <Checkbox
                        :model-value="selectedAvailableGroupIds.has(group.jid) || alreadyAddedGroupJids.has(group.jid)"
                        :disabled="alreadyAddedGroupJids.has(group.jid)"
                        @update:model-value="(val: boolean) => {
                          if (alreadyAddedGroupJids.has(group.jid)) return;
                          const next = new Set(selectedAvailableGroupIds);
                          val ? next.add(group.jid) : next.delete(group.jid);
                          selectedAvailableGroupIds = next;
                        }"
                      />
                    </td>
                    <td class="py-2.5 px-3">{{ group.name }}</td>
                    <td class="py-2.5 px-3 font-mono text-xs">{{ group.jid }}</td>
                    <td class="py-2.5 px-3">{{ group.participant_count }}</td>
                  </tr>
                </tbody>
              </table>
            </ScrollArea>
          </template>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showAddGroupsDialog = false">{{ $t("common.close") }}</Button>
          <Button
            :disabled="selectedAvailableGroupIds.size === 0 || isLoadingAvailableGroups"
            @click="addGroups()"
          >
            <UserPlus class="h-4 w-4 mr-2" />
            {{ $t("campaigns.addGroupTargetsButton") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Add Recipients Dialog -->
    <Dialog v-model:open="showAddRecipientsDialog">
      <DialogContent class="sm:max-w-[700px] max-h-[85vh]">
        <DialogHeader>
          <DialogTitle>{{ $t("campaigns.addRecipients") }}</DialogTitle>
          <DialogDescription>
            {{
              $t("campaigns.addRecipientsTo", { name: selectedCampaign?.name })
            }}
            <span v-if="templateParamNames.length > 0" class="block mt-1">
              {{
                $t("campaigns.templateRequiresParams", {
                  count: templateParamNames.length,
                })
              }}
            </span>
          </DialogDescription>
        </DialogHeader>

        <!-- Template Preview -->
        <div
          v-if="selectedTemplate?.body_content"
          class="mb-4 p-3 bg-muted/50 rounded-lg border"
        >
          <div class="flex items-center gap-2 mb-2">
            <MessageSquare class="h-4 w-4 text-muted-foreground" />
            <span class="text-sm font-medium">{{
              $t("campaigns.templatePreview")
            }}</span>
          </div>
          <p class="text-sm whitespace-pre-wrap">
            <template
              v-for="(part, idx) in parseTemplateParams(
                selectedTemplate.body_content,
              )"
              :key="idx"
            >
              <span
                v-if="part.isParam"
                class="bg-primary/20 text-primary px-1 rounded font-medium"
                >{{ part.text }}</span
              >
              <template v-else>{{ part.text }}</template>
            </template>
          </p>
        </div>

        <Tabs v-model="addRecipientsTab" class="w-full">
          <TabsList class="grid w-full grid-cols-3">
            <TabsTrigger value="manual">
              <UserPlus class="h-4 w-4 mr-2" />
              {{ $t("campaigns.manualEntry") }}
            </TabsTrigger>
            <TabsTrigger value="contacts">
              <Users class="h-4 w-4 mr-2" />
              {{ $t("campaigns.contactsSource") }}
            </TabsTrigger>
            <TabsTrigger value="csv">
              <FileSpreadsheet class="h-4 w-4 mr-2" />
              {{ $t("campaigns.uploadCsv") }}
            </TabsTrigger>
          </TabsList>

          <!-- Manual Entry Tab -->
          <TabsContent value="manual" class="mt-4">
            <div class="space-y-4">
              <div class="bg-muted p-3 rounded-lg text-sm">
                <p class="font-medium mb-2">
                  {{ $t("campaigns.formatOneLine") }}
                </p>
                <code class="bg-background px-2 py-1 rounded block">{{
                  manualEntryFormat
                }}</code>
                <p
                  v-if="templateParamNames.length > 0"
                  class="text-muted-foreground mt-2 text-xs"
                >
                  {{ $t("campaigns.templateParameters") }}
                  <span v-for="(param, idx) in templateParamNames" :key="param"
                    ><code class="bg-background px-1 rounded">{{
                      formatParamName(param)
                    }}</code
                    ><span v-if="idx < templateParamNames.length - 1"
                      >,
                    </span></span
                  >
                </p>
              </div>
              <div class="space-y-2">
                <Label for="recipients">{{
                  $t("campaigns.recipientsLabel")
                }}</Label>
                <Textarea
                  id="recipients"
                  v-model="recipientsInput"
                  :placeholder="recipientPlaceholder"
                  :rows="8"
                  class="font-mono text-sm"
                  :disabled="isAddingRecipients"
                />
                <!-- Validation status -->
                <div v-if="recipientsInput.trim()" class="space-y-2">
                  <p
                    v-if="manualInputValidation.isValid"
                    class="text-xs text-primary"
                  >
                    {{
                      $t("campaigns.recipientsValid", {
                        count: manualInputValidation.validLines,
                      })
                    }}
                  </p>
                  <div
                    v-else-if="manualInputValidation.invalidLines.length > 0"
                    class="text-xs"
                  >
                    <p class="text-destructive font-medium mb-1">
                      {{
                        $t("campaigns.linesHaveErrors", {
                          invalid: manualInputValidation.invalidLines.length,
                          total: manualInputValidation.totalLines,
                        })
                      }}
                    </p>
                    <ul
                      class="text-destructive space-y-0.5 max-h-20 overflow-y-auto"
                    >
                      <li
                        v-for="err in manualInputValidation.invalidLines.slice(
                          0,
                          5,
                        )"
                        :key="err.lineNumber"
                      >
                        {{
                          $t("campaigns.lineError", {
                            line: err.lineNumber,
                            reason: err.reason,
                          })
                        }}
                      </li>
                      <li
                        v-if="manualInputValidation.invalidLines.length > 5"
                        class="text-muted-foreground"
                      >
                        {{
                          $t("campaigns.andMoreErrors", {
                            count:
                              manualInputValidation.invalidLines.length - 5,
                          })
                        }}
                      </li>
                    </ul>
                  </div>
                  <p v-else class="text-xs text-muted-foreground">
                    {{
                      $t("campaigns.recipientsEntered", {
                        count: manualInputValidation.totalLines,
                      })
                    }}
                  </p>
                </div>
              </div>
              <div class="flex justify-end">
                <Button
                  @click="addRecipients"
                  :disabled="
                    isAddingRecipients || !manualInputValidation.isValid
                  "
                >
                  <Loader2
                    v-if="isAddingRecipients"
                    class="h-4 w-4 mr-2 animate-spin"
                  />
                  <Upload v-else class="h-4 w-4 mr-2" />
                  {{ $t("campaigns.addRecipients") }}
                </Button>
              </div>
            </div>
          </TabsContent>

          <!-- Contacts Tab -->
          <TabsContent value="contacts" class="mt-4">
            <div class="space-y-4">
              <div
                v-if="isWhatsmeowProvider"
                data-testid="campaign-contacts-scope-banner"
                class="rounded-xl border border-primary/20 bg-primary/5 px-4 py-3 light:border-primary/20 light:bg-primary/10"
              >
                <div class="flex items-center justify-between gap-3 flex-wrap">
                  <div class="space-y-1">
                    <p
                      class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                    >
                      {{ $t("campaigns.contactsSourceScope") }}
                    </p>
                    <p class="text-sm font-medium text-foreground">
                      {{
                        selectedCampaignInstanceLabel ||
                        $t("campaigns.contactsMissingInstance")
                      }}
                    </p>
                  </div>
                  <Badge
                    variant="outline"
                    class="border-primary/25 bg-white/70 text-primary light:border-primary/30 light:bg-white"
                  >
                    {{ $t("campaigns.contactsSource") }}
                  </Badge>
                </div>
              </div>

              <div class="flex items-center gap-2">
                <Input
                  v-model="contactsSearchQuery"
                  :placeholder="$t('contacts.searchContacts') + '...'"
                  class="flex-1"
                  :disabled="isAddingRecipients"
                />
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="
                    isLoadingContactsForImport ||
                    filteredContactsForImport.length === 0 ||
                    isAddingRecipients
                  "
                  @click="
                    toggleAllFilteredContacts(!areAllFilteredContactsSelected)
                  "
                >
                  {{
                    areAllFilteredContactsSelected
                      ? $t("common.deselectAll")
                      : $t("common.selectAll")
                  }}
                </Button>
              </div>

              <div class="rounded-xl border bg-muted/30 p-3 light:bg-white">
                <div
                  class="grid grid-cols-1 gap-3"
                  :class="
                    isWhatsmeowProvider ? 'lg:grid-cols-4' : 'sm:grid-cols-3'
                  "
                >
                  <div v-if="isWhatsmeowProvider" class="space-y-1">
                    <Label class="text-xs font-medium text-muted-foreground">
                      {{ $t("campaigns.contactDateType") }}
                    </Label>
                    <Select
                      v-model="contactsImportDateBasis"
                      :disabled="isAddingRecipients"
                    >
                      <SelectTrigger
                        data-testid="campaign-contacts-date-basis-trigger"
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="created">
                          {{ $t("campaigns.createdContactsDateBasis") }}
                        </SelectItem>
                        <SelectItem value="incoming_any">
                          {{ $t("campaigns.contactsThatMessagedUs") }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <p class="text-[11px] text-muted-foreground">
                      {{
                        contactsImportDateBasis === "incoming_any"
                          ? $t("campaigns.contactsInboundDateHint")
                          : $t("campaigns.contactsCreatedDateHint")
                      }}
                    </p>
                  </div>

                  <div class="space-y-1">
                    <Label class="text-xs font-medium text-muted-foreground">
                      {{ $t("analytics.from") }}
                    </Label>
                    <Input
                      v-model="contactsDateFrom"
                      type="date"
                      :disabled="isAddingRecipients"
                    />
                  </div>
                  <div class="space-y-1">
                    <Label class="text-xs font-medium text-muted-foreground">
                      {{ $t("analytics.to") }}
                    </Label>
                    <Input
                      v-model="contactsDateTo"
                      type="date"
                      :disabled="isAddingRecipients"
                    />
                  </div>
                  <div class="flex items-end">
                    <Button
                      variant="outline"
                      class="w-full"
                      :disabled="
                        isAddingRecipients ||
                        (!contactsDateFrom && !contactsDateTo)
                      "
                      @click="
                        contactsDateFrom = '';
                        contactsDateTo = '';
                      "
                    >
                      {{ $t("common.clear") }}
                    </Button>
                  </div>
                </div>
              </div>

              <div
                v-if="isLoadingContactsForImport"
                class="flex items-center justify-center py-8"
              >
                <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
              </div>

              <div
                v-else-if="contactsImportMissingInstance"
                class="rounded-xl border border-dashed border-primary/30 bg-primary/5 py-8 px-4 text-center text-primary light:border-primary/30 light:bg-primary/10"
              >
                <Users class="h-12 w-12 mx-auto mb-2 opacity-60" />
                <p class="font-medium">
                  {{ $t("campaigns.contactsMissingInstance") }}
                </p>
              </div>

              <div
                v-else-if="filteredContactsForImport.length === 0"
                class="text-center py-8 text-muted-foreground"
              >
                <Users class="h-12 w-12 mx-auto mb-2 opacity-50" />
                <p>
                  {{
                    hasContactsImportFilters
                      ? $t("contacts.noMatchingContacts")
                      : $t("contacts.noContactsYet")
                  }}
                </p>
              </div>

              <div v-else class="border rounded-lg overflow-hidden">
                <ScrollArea class="h-[300px]">
                  <table class="w-full text-sm">
                    <thead class="sticky top-0 bg-muted border-b">
                      <tr>
                        <th class="text-left py-2 px-3 w-10"></th>
                        <th class="text-left py-2 px-3">
                          {{ $t("campaigns.name") }}
                        </th>
                        <th class="text-left py-2 px-3">
                          {{ $t("campaigns.phoneNumber") }}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr
                        v-for="contact in filteredContactsForImport"
                        :key="contact.id"
                        class="border-b last:border-0"
                      >
                        <td class="py-2 px-3">
                          <Checkbox
                            :id="`campaign-contact-${contact.id}`"
                            :checked="selectedContactIdSet.has(contact.id)"
                            @update:checked="
                              (checked) =>
                                toggleContactSelection(
                                  contact,
                                  checked === true,
                                )
                            "
                          />
                        </td>
                        <td class="py-2 px-3">
                          {{ getContactRecipientDisplayName(contact) }}
                        </td>
                        <td class="py-2 px-3 font-mono">
                          {{ contact.phone_number }}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </ScrollArea>
              </div>

              <div class="flex items-center justify-between">
                <p class="text-xs text-muted-foreground">
                  {{ contactsImportPage }} / {{ contactsImportTotalPages }} ·
                  {{ contactsImportTotal }} {{ $t("resources.contacts") }}
                </p>
                <div class="flex items-center gap-2">
                  <div class="flex items-center gap-2">
                    <Label class="text-xs text-muted-foreground">{{
                      $t("campaigns.pageSize")
                    }}</Label>
                    <Select
                      :model-value="String(contactsImportPageSize)"
                      :disabled="isAddingRecipients"
                      @update:model-value="updateContactsImportPageSize"
                    >
                      <SelectTrigger class="h-8 w-[90px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem
                          v-for="size in contactsImportPageSizeOptions"
                          :key="size"
                          :value="String(size)"
                        >
                          {{ size }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="
                      isLoadingContactsForImport ||
                      contactsImportPage <= 1 ||
                      isAddingRecipients
                    "
                    @click="goToContactsImportPage(contactsImportPage - 1)"
                  >
                    {{ $t("common.back") }}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="
                      isLoadingContactsForImport ||
                      contactsImportPage >= contactsImportTotalPages ||
                      isAddingRecipients
                    "
                    @click="goToContactsImportPage(contactsImportPage + 1)"
                  >
                    {{ $t("common.next") }}
                  </Button>
                </div>
              </div>

              <div class="flex items-center justify-between">
                <p class="text-xs text-muted-foreground">
                  {{ selectedContactsForImport.length }}
                  {{ $t("common.selected") }}
                </p>
                <Button
                  @click="addRecipientsFromContacts"
                  :disabled="
                    isAddingRecipients || selectedContactsForImport.length === 0
                  "
                >
                  <Loader2
                    v-if="isAddingRecipients"
                    class="h-4 w-4 mr-2 animate-spin"
                  />
                  <Upload v-else class="h-4 w-4 mr-2" />
                  {{ $t("campaigns.addRecipients") }}
                </Button>
              </div>
            </div>
          </TabsContent>

          <!-- CSV Upload Tab -->
          <TabsContent value="csv" class="mt-4">
            <div class="space-y-4">
              <!-- CSV Format Info -->
              <div class="bg-muted p-3 rounded-lg text-sm">
                <p class="font-medium mb-2">
                  {{ $t("campaigns.requiredCsvColumns") }}
                </p>
                <div class="flex flex-wrap gap-2">
                  <code
                    v-for="col in csvColumnsHint"
                    :key="col"
                    class="bg-background px-2 py-1 rounded text-xs"
                    >{{ col }}</code
                  >
                </div>
                <p
                  v-if="templateParamNames.length > 0"
                  class="text-muted-foreground mt-2 text-xs"
                >
                  {{ $t("campaigns.templateParameters") }}
                  <span v-for="(param, idx) in templateParamNames" :key="param"
                    ><code class="bg-background px-1 rounded">{{
                      formatParamName(param)
                    }}</code
                    ><span v-if="idx < templateParamNames.length - 1"
                      >,
                    </span></span
                  >
                </p>
              </div>

              <!-- File Upload -->
              <div class="space-y-2">
                <Label for="csv-file">{{
                  $t("campaigns.selectCsvFile")
                }}</Label>
                <div class="flex items-center gap-2">
                  <Input
                    id="csv-file"
                    type="file"
                    accept=".csv"
                    @change="handleCSVFileSelect"
                    :disabled="isValidatingCSV || isAddingRecipients"
                    class="flex-1"
                  />
                  <Button
                    v-if="csvFile"
                    variant="outline"
                    size="icon"
                    @click="
                      csvFile = null;
                      csvValidation = null;
                    "
                    :disabled="isValidatingCSV || isAddingRecipients"
                  >
                    <XCircle class="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <!-- Validation Results -->
              <div
                v-if="isValidatingCSV"
                class="flex items-center justify-center py-8"
              >
                <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
                <span class="ml-2 text-muted-foreground">{{
                  $t("campaigns.validatingCsv")
                }}</span>
              </div>

              <div v-else-if="csvValidation" class="space-y-4">
                <!-- Global Errors -->
                <div
                  v-if="csvValidation.errors.length > 0"
                  class="bg-destructive/10 border border-destructive/20 rounded-lg p-3"
                >
                  <div
                    class="flex items-center gap-2 text-destructive font-medium mb-2"
                  >
                    <AlertTriangle class="h-4 w-4" />
                    {{ $t("campaigns.validationErrors") }}
                  </div>
                  <ul class="list-disc list-inside text-sm text-destructive">
                    <li v-for="error in csvValidation.errors" :key="error">
                      {{ error }}
                    </li>
                  </ul>
                </div>

                <!-- Warnings -->
                <div
                  v-if="
                    csvValidation.warnings && csvValidation.warnings.length > 0
                  "
                  class="rounded-lg border border-primary/20 bg-primary/10 p-3"
                >
                  <div
                    class="mb-2 flex items-center gap-2 font-medium text-primary"
                  >
                    <AlertTriangle class="h-4 w-4" />
                    {{ $t("campaigns.warnings") }}
                  </div>
                  <ul class="list-disc list-inside text-sm text-primary">
                    <li
                      v-for="warning in csvValidation.warnings"
                      :key="warning"
                    >
                      {{ warning }}
                    </li>
                  </ul>
                </div>

                <!-- Column Mapping Info -->
                <div
                  v-if="
                    csvValidation.columnMapping &&
                    csvValidation.columnMapping.length > 0
                  "
                  class="bg-muted/50 border rounded-lg p-3"
                >
                  <div class="text-sm font-medium mb-2">
                    {{ $t("campaigns.columnMapping") }}
                  </div>
                  <div class="flex flex-wrap gap-2">
                    <div
                      v-for="mapping in csvValidation.columnMapping"
                      :key="mapping.paramName"
                      class="text-xs bg-background border rounded px-2 py-1"
                    >
                      <span class="text-muted-foreground">{{
                        mapping.csvColumn
                      }}</span>
                      <span class="mx-1">→</span>
                      <span class="font-mono text-primary">{{
                        formatParamName(mapping.paramName)
                      }}</span>
                    </div>
                  </div>
                </div>

                <!-- Summary -->
                <div class="flex flex-wrap items-center gap-4 text-sm">
                  <div class="flex items-center gap-1">
                    <Check class="h-4 w-4 text-primary" />
                    <span
                      >{{ csvValidation.rows.filter((r) => r.isValid).length }}
                      {{ $t("campaigns.valid") }}</span
                    >
                  </div>
                  <div
                    v-if="
                      csvValidation.rows.filter((r) => !r.isValid).length > 0
                    "
                    class="flex items-center gap-1"
                  >
                    <AlertTriangle class="h-4 w-4 text-destructive" />
                    <span
                      >{{ csvValidation.rows.filter((r) => !r.isValid).length }}
                      {{ $t("campaigns.invalid") }}</span
                    >
                  </div>
                  <div
                    v-if="
                      csvValidation.rows.filter((r) =>
                        r.errors.some((e) => e.includes('Duplicate')),
                      ).length > 0
                    "
                    class="flex items-center gap-1 text-primary"
                  >
                    <Users class="h-4 w-4" />
                    <span
                      >{{
                        csvValidation.rows.filter((r) =>
                          r.errors.some((e) => e.includes("Duplicate")),
                        ).length
                      }}
                      {{ $t("campaigns.duplicates") }}</span
                    >
                  </div>
                  <div class="text-muted-foreground">
                    {{ $t("campaigns.columns") }}
                    {{ csvValidation.csvColumns.join(", ") }}
                  </div>
                </div>

                <!-- Preview Table -->
                <div
                  v-if="csvValidation.rows.length > 0"
                  class="border rounded-lg overflow-hidden"
                >
                  <ScrollArea class="h-[200px]">
                    <table class="w-full text-sm">
                      <thead class="sticky top-0 bg-muted border-b">
                        <tr>
                          <th class="text-left py-2 px-3 w-8"></th>
                          <th class="text-left py-2 px-3">
                            {{ $t("campaigns.phone") }}
                          </th>
                          <th class="text-left py-2 px-3">
                            {{ $t("campaigns.name") }}
                          </th>
                          <th class="text-left py-2 px-3">
                            {{ $t("campaigns.parameters") }}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr
                          v-for="(row, index) in csvValidation.rows.slice(
                            0,
                            50,
                          )"
                          :key="index"
                          :class="row.isValid ? '' : 'bg-destructive/5'"
                          class="border-b last:border-0"
                        >
                          <td class="py-2 px-3">
                            <Check
                              v-if="row.isValid"
                              class="h-4 w-4 text-primary"
                            />
                            <Tooltip v-else>
                              <TooltipTrigger>
                                <AlertTriangle
                                  class="h-4 w-4 text-destructive"
                                />
                              </TooltipTrigger>
                              <TooltipContent>
                                <ul class="text-xs">
                                  <li v-for="err in row.errors" :key="err">
                                    {{ err }}
                                  </li>
                                </ul>
                              </TooltipContent>
                            </Tooltip>
                          </td>
                          <td class="py-2 px-3 font-mono">
                            {{ row.phone_number || "-" }}
                          </td>
                          <td class="py-2 px-3">{{ row.name || "-" }}</td>
                          <td class="py-2 px-3 text-muted-foreground">
                            {{
                              Object.values(row.params)
                                .filter((p) => p)
                                .join(", ") || "-"
                            }}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </ScrollArea>
                  <div
                    v-if="csvValidation.rows.length > 50"
                    class="text-xs text-muted-foreground text-center py-2 border-t"
                  >
                    {{
                      $t("campaigns.showingFirst", {
                        count: 50,
                        total: csvValidation.rows.length,
                      })
                    }}
                  </div>
                </div>

                <!-- Import Button -->
                <div class="flex justify-end">
                  <Button
                    @click="addRecipientsFromCSV"
                    :disabled="
                      isAddingRecipients ||
                      !csvValidation.isValid ||
                      csvValidation.rows.filter((r) => r.isValid).length === 0
                    "
                  >
                    <Loader2
                      v-if="isAddingRecipients"
                      class="h-4 w-4 mr-2 animate-spin"
                    />
                    <Upload v-else class="h-4 w-4 mr-2" />
                    {{
                      $t("campaigns.importRecipients", {
                        count: csvValidation.rows.filter((r) => r.isValid)
                          .length,
                      })
                    }}
                  </Button>
                </div>
              </div>

              <!-- Empty state -->
              <div v-else class="text-center py-8 text-muted-foreground">
                <FileSpreadsheet class="h-12 w-12 mx-auto mb-2 opacity-50" />
                <p>{{ $t("campaigns.selectCsvToPreview") }}</p>
              </div>
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter class="border-t pt-4 mt-4">
          <Button
            variant="outline"
            size="sm"
            @click="showAddRecipientsDialog = false"
            :disabled="isAddingRecipients"
          >
            {{ $t("common.cancel") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('campaigns.deleteCampaign')"
      :item-name="campaignToDelete?.name"
      @confirm="confirmDeleteCampaign"
    />

    <!-- Cancel Confirmation Dialog -->
    <AlertDialog v-model:open="cancelDialogOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{
            $t("campaigns.cancelConfirmTitle")
          }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{
              $t("campaigns.cancelConfirmDesc", {
                name: campaignToCancel?.name,
              })
            }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{
            $t("campaigns.keepRunning")
          }}</AlertDialogCancel>
          <AlertDialogAction @click="confirmCancelCampaign">{{
            $t("campaigns.cancelCampaign")
          }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Media Preview Dialog -->
    <Dialog v-model:open="showMediaPreviewDialog">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ $t("campaigns.mediaPreview") }}</DialogTitle>
          <DialogDescription>
            {{ previewingCampaign?.header_media_filename }}
          </DialogDescription>
        </DialogHeader>
        <div class="flex items-center justify-center py-4">
          <img
            v-if="
              previewingCampaign?.header_media_mime_type?.startsWith(
                'image/',
              ) && previewingCampaign?.id
            "
            :src="getMediaPreviewUrl(previewingCampaign.id)"
            :alt="previewingCampaign?.header_media_filename"
            class="max-w-full max-h-[60vh] object-contain rounded"
          />
          <video
            v-else-if="
              previewingCampaign?.header_media_mime_type?.startsWith(
                'video/',
              ) && previewingCampaign?.id
            "
            :src="getMediaPreviewUrl(previewingCampaign.id)"
            controls
            class="max-w-full max-h-[60vh] rounded"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showMediaPreviewDialog = false">{{
            $t("common.close")
          }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <ContentPickerModal
      v-model:open="isContentPickerOpen"
      @select="handleContentSelect"
    />
  </div>
</template>
