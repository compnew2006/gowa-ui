"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "@/lib/api";
import type { Campaign, CampaignPreview } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn, formatDateTime, formatRelativeTime } from "@/lib/utils";
import {
  Megaphone, Plus, Pencil, Trash2, Play, Pause,
  Users, Clock, Image as ImageIcon, Video, X, Upload,
  CheckCircle2, AlertCircle, Timer, FileText,
} from "lucide-react";
import { useState, useRef, useCallback } from "react";
import { toast } from "sonner";
import { getSafeDisplayUrl, validateCampaignMediaFile } from "@/lib/security";
import { RequirePermission } from "@/components/auth/permission-guard";

type View = "list" | "create" | "edit";

const STATUS_COLOR: Record<string, string> = {
  draft: "bg-gray-500/20 text-gray-400 border-gray-500/30",
  scheduled: "bg-blue-500/20 text-blue-400 border-blue-500/30",
  active: "bg-green-500/20 text-green-400 border-green-500/30",
  paused: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  completed: "bg-purple-500/20 text-purple-400 border-purple-500/30",
};

const STATUS_LABEL: Record<string, string> = {
  draft: "مسودة",
  scheduled: "مجدولة",
  active: "نشطة",
  paused: "موقوفة",
  completed: "مكتملة",
};

const EMPTY_FORM = {
  name: "",
  description: "",
  message_ar: "",
  message_en: "",
  media_urls: [] as string[],
  media_type: undefined as string | undefined,
  send_at: "",
  interval_hours: "" as string | number,
  max_sends: "" as string | number,
  target_filter: {
    purchase_intent: "",
    conversion_status: "",
    churn_risk: "",
  },
  customer_ids: [] as string[],
};

function StatBadge({ icon: Icon, label, value, color }: { icon: any; label: string; value: any; color?: string }) {
  return (
    <div className="flex items-center gap-2">
      <Icon className={cn("h-4 w-4", color ?? "text-muted-foreground")} />
      <span className="text-xs text-muted-foreground">{label}:</span>
      <span className="text-xs font-medium text-foreground">{value}</span>
    </div>
  );
}

function CampaignCard({
  campaign,
  onEdit,
  onDelete,
  onActivate,
  onPause,
}: {
  campaign: Campaign;
  onEdit: () => void;
  onDelete: () => void;
  onActivate: () => void;
  onPause: () => void;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-5 space-y-4 hover:border-primary/40 transition-colors">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="font-semibold text-foreground truncate">{campaign.name}</h3>
            <span className={cn("text-[11px] px-2 py-0.5 rounded-full border font-medium", STATUS_COLOR[campaign.status])}>
              {STATUS_LABEL[campaign.status] ?? campaign.status}
            </span>
          </div>
          {campaign.description && (
            <p className="text-sm text-muted-foreground mt-0.5 line-clamp-1">{campaign.description}</p>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {campaign.status === "draft" || campaign.status === "paused" ? (
            <button
              onClick={onActivate}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-green-500/15 text-green-400 hover:bg-green-500/25 text-xs transition-colors"
              title="تفعيل"
            >
              <Play className="h-3.5 w-3.5" />
              تفعيل
            </button>
          ) : campaign.status === "active" ? (
            <button
              onClick={onPause}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-yellow-500/15 text-yellow-400 hover:bg-yellow-500/25 text-xs transition-colors"
              title="إيقاف"
            >
              <Pause className="h-3.5 w-3.5" />
              إيقاف
            </button>
          ) : null}
          <button onClick={onEdit} className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground transition-colors">
            <Pencil className="h-4 w-4" />
          </button>
          <button onClick={onDelete} className="p-1.5 rounded-lg hover:bg-red-500/15 text-muted-foreground hover:text-red-400 transition-colors">
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Message preview */}
      {(campaign.message_ar || campaign.message_en) && (
        <div className="rounded-lg bg-accent/30 px-3 py-2 text-sm text-foreground/80 line-clamp-2">
          {campaign.message_ar || campaign.message_en}
        </div>
      )}

      {/* Media preview */}
      {campaign.media_urls.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {campaign.media_urls.slice(0, 3).map((url, i) => {
            const safeUrl = getSafeDisplayUrl(url);
            return (
              <div key={i} className="h-14 w-14 rounded-lg bg-accent/50 border border-border flex items-center justify-center overflow-hidden">
                {campaign.media_type === "image" && safeUrl ? (
                  <img src={safeUrl} alt="" referrerPolicy="no-referrer" className="h-full w-full object-cover" onError={(e) => { (e.target as any).style.display = "none"; }} />
                ) : (
                  <Video className="h-5 w-5 text-muted-foreground" />
                )}
              </div>
            );
          })}
          {campaign.media_urls.length > 3 && (
            <div className="h-14 w-14 rounded-lg bg-accent/50 border border-border flex items-center justify-center text-xs text-muted-foreground">
              +{campaign.media_urls.length - 3}
            </div>
          )}
        </div>
      )}

      {/* Stats row */}
      <div className="flex flex-wrap gap-4 pt-1 border-t border-border">
        <StatBadge icon={Users} label="المستلمون" value={campaign.total_recipients} />
        {campaign.sent_count > 0 && <StatBadge icon={CheckCircle2} label="أُرسل" value={campaign.sent_count} color="text-green-400" />}
        {campaign.failed_count > 0 && <StatBadge icon={AlertCircle} label="فشل" value={campaign.failed_count} color="text-red-400" />}
        {campaign.send_at && <StatBadge icon={Timer} label="موعد الإرسال" value={formatDateTime(campaign.send_at)} />}
        {campaign.interval_hours && <StatBadge icon={Clock} label="التكرار" value={`كل ${campaign.interval_hours}ساعة`} />}
        <span className="text-xs text-muted-foreground mr-auto">{formatRelativeTime(campaign.created_at)}</span>
      </div>
    </div>
  );
}

function CampaignForm({
  initial,
  onSave,
  onCancel,
  saving,
  pageId,
}: {
  initial?: Campaign;
  onSave: (data: any) => void;
  onCancel: () => void;
  saving: boolean;
  pageId?: string | null;
}) {
  const [form, setForm] = useState(() => {
    if (!initial) return EMPTY_FORM;
    return {
      name: initial.name,
      description: initial.description ?? "",
      message_ar: initial.message_ar,
      message_en: initial.message_en,
      media_urls: initial.media_urls,
      media_type: initial.media_type,
      send_at: initial.send_at ? initial.send_at.slice(0, 16) : "",
      interval_hours: initial.interval_hours ?? "",
      max_sends: initial.max_sends ?? "",
      target_filter: {
        purchase_intent: (initial.target_filter?.purchase_intent as string) ?? "",
        conversion_status: (initial.target_filter?.conversion_status as string) ?? "",
        churn_risk: (initial.target_filter?.churn_risk as string) ?? "",
      },
      customer_ids: initial.customer_ids ?? [],
    };
  });

  const [uploading, setUploading] = useState(false);
  const [preview, setPreview] = useState<CampaignPreview | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const set = (key: string, val: any) => setForm(f => ({ ...f, [key]: val }));
  const setFilter = (key: string, val: string) =>
    setForm(f => ({ ...f, target_filter: { ...f.target_filter, [key]: val } }));

  const handleUpload = useCallback(async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setUploading(true);
    const urls: string[] = [];
    let mtype = form.media_type;
    for (const file of Array.from(files)) {
      const error = validateCampaignMediaFile(file);
      if (error) {
        toast.error(`${file.name}: ${error}`);
        continue;
      }
      try {
        const res = await api.uploadCampaignMedia(file);
        urls.push(res.url);
        mtype = res.media_type;
      } catch (e: any) {
        toast.error(`فشل رفع ${file.name}: ${e.message}`);
      }
    }
    set("media_urls", [...form.media_urls, ...urls]);
    if (mtype) set("media_type", mtype);
    setUploading(false);
  }, [form.media_urls, form.media_type]);

  const removeMedia = (idx: number) => {
    const updated = form.media_urls.filter((_, i) => i !== idx);
    set("media_urls", updated);
    if (updated.length === 0) set("media_type", undefined);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim()) { toast.error("اسم الحملة مطلوب"); return; }
    if (!form.message_ar.trim() && !form.message_en.trim()) { toast.error("نص الرسالة مطلوب"); return; }

    const cleanFilter: Record<string, string> = {};
    if (form.target_filter.purchase_intent) cleanFilter.purchase_intent = form.target_filter.purchase_intent;
    if (form.target_filter.conversion_status) cleanFilter.conversion_status = form.target_filter.conversion_status;
    if (form.target_filter.churn_risk) cleanFilter.churn_risk = form.target_filter.churn_risk;

    onSave({
      name: form.name.trim(),
      description: form.description.trim() || undefined,
      message_ar: form.message_ar,
      message_en: form.message_en,
      media_urls: form.media_urls,
      media_type: form.media_type,
      send_at: form.send_at ? new Date(form.send_at).toISOString() : undefined,
      interval_hours: form.interval_hours ? Number(form.interval_hours) : undefined,
      max_sends: form.max_sends ? Number(form.max_sends) : undefined,
      target_filter: cleanFilter,
      customer_ids: form.customer_ids,
    });
  };

  const previewRecipients = async () => {
    const cleanFilter: Record<string, string> = {};
    if (form.target_filter.purchase_intent) cleanFilter.purchase_intent = form.target_filter.purchase_intent;
    if (form.target_filter.conversion_status) cleanFilter.conversion_status = form.target_filter.conversion_status;
    if (form.target_filter.churn_risk) cleanFilter.churn_risk = form.target_filter.churn_risk;

    setPreviewing(true);
    try {
      const result = initial
        ? await api.previewCampaignRecipients(initial.id)
        : await api.previewCampaignAudience({ target_filter: cleanFilter, customer_ids: form.customer_ids, page_id: pageId });
      setPreview(result);
    } catch (error) {
      if (error instanceof ApiError && [404, 501].includes(error.status)) {
        toast.error("معاينة المستلمين غير متاحة من الخادم حالياً");
      } else {
        toast.error("فشل تحميل معاينة المستلمين");
      }
    } finally {
      setPreviewing(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-3xl">
      {/* Basic info */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="font-semibold text-foreground flex items-center gap-2">
          <FileText className="h-4 w-4 text-primary" />
          معلومات الحملة
        </h3>
        <div className="grid gap-4">
          <div>
            <label className="block text-sm text-muted-foreground mb-1.5">اسم الحملة <span className="text-red-400">*</span></label>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="مثال: حملة العملاء ذوي النية العالية"
              value={form.name}
              onChange={e => set("name", e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-sm text-muted-foreground mb-1.5">الوصف (اختياري)</label>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="وصف مختصر للحملة..."
              value={form.description}
              onChange={e => set("description", e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* Targeting */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="font-semibold text-foreground flex items-center gap-2">
          <Users className="h-4 w-4 text-primary" />
          استهداف العملاء
        </h3>
        <p className="text-xs text-muted-foreground">اختر فلاتر لاستهداف مجموعة تلقائية، أو اتركها فارغة لاستهداف الجميع.</p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">نية الشراء</label>
            <select
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              value={form.target_filter.purchase_intent}
              onChange={e => setFilter("purchase_intent", e.target.value)}
            >
              <option value="">الكل</option>
              <option value="High">High</option>
              <option value="Medium">Medium</option>
              <option value="Low">Low</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">حالة التحويل</label>
            <select
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              value={form.target_filter.conversion_status}
              onChange={e => setFilter("conversion_status", e.target.value)}
            >
              <option value="">الكل</option>
              <option value="prospect">Prospect</option>
              <option value="qualified">Qualified</option>
              <option value="converted">Converted</option>
              <option value="churned">Churned</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">خطر المغادرة</label>
            <select
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              value={form.target_filter.churn_risk}
              onChange={e => setFilter("churn_risk", e.target.value)}
            >
              <option value="">الكل</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>
        </div>
        <div className="rounded-lg border border-border bg-accent/20 p-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-sm font-medium text-foreground">معاينة المستلمين</p>
              <p className="text-xs text-muted-foreground">تحقق من حجم الجمهور قبل الحفظ أو التفعيل.</p>
            </div>
            <button
              type="button"
              onClick={previewRecipients}
              disabled={previewing}
              className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-accent disabled:opacity-50"
            >
              {previewing ? "جاري..." : "معاينة"}
            </button>
          </div>
          {preview && (
            <div className="mt-3 text-xs text-muted-foreground">
              <span className="font-medium text-foreground">{preview.count}</span> مستلم محتمل
              {preview.sample.length > 0 && (
                <span> · عينة: {preview.sample.map(customer => customer.name).join("، ")}</span>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Message content */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="font-semibold text-foreground flex items-center gap-2">
          <Megaphone className="h-4 w-4 text-primary" />
          محتوى الرسالة
        </h3>
        <div className="grid gap-4">
          <div>
            <label className="block text-sm text-muted-foreground mb-1.5">
              الرسالة بالعربية <span className="text-red-400">*</span>
            </label>
            <textarea
              rows={4}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none"
              placeholder="اكتب رسالتك بالعربية..."
              value={form.message_ar}
              onChange={e => set("message_ar", e.target.value)}
              dir="rtl"
            />
          </div>
          <div>
            <label className="block text-sm text-muted-foreground mb-1.5">الرسالة بالإنجليزية</label>
            <textarea
              rows={4}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none"
              placeholder="Write your message in English..."
              value={form.message_en}
              onChange={e => set("message_en", e.target.value)}
              dir="ltr"
            />
          </div>
        </div>
      </div>

      {/* Media upload */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="font-semibold text-foreground flex items-center gap-2">
          <ImageIcon className="h-4 w-4 text-primary" />
          وسائط الحملة (صور / فيديو)
        </h3>

        {/* Existing media previews */}
        {form.media_urls.length > 0 && (
          <div className="flex flex-wrap gap-3">
            {form.media_urls.map((url, i) => {
              const safeUrl = getSafeDisplayUrl(url);
              return (
                <div key={i} className="relative h-20 w-20 rounded-lg border border-border bg-accent/40 overflow-hidden group">
                  {form.media_type === "image" && safeUrl ? (
                    <img src={safeUrl} alt="" referrerPolicy="no-referrer" className="h-full w-full object-cover" />
                  ) : (
                    <div className="h-full w-full flex items-center justify-center">
                      <Video className="h-8 w-8 text-muted-foreground" />
                    </div>
                  )}
                  <button
                    type="button"
                    onClick={() => removeMedia(i)}
                    className="absolute top-1 right-1 p-0.5 rounded-full bg-red-500 text-white opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              );
            })}
          </div>
        )}

        {/* Upload zone */}
        <button
          type="button"
          onClick={() => fileRef.current?.click()}
          disabled={uploading}
          className={cn(
            "w-full rounded-lg border-2 border-dashed border-border py-8 flex flex-col items-center gap-2 text-muted-foreground hover:border-primary/50 hover:text-foreground transition-colors",
            uploading && "opacity-60 cursor-not-allowed"
          )}
        >
          <Upload className="h-6 w-6" />
          <span className="text-sm">{uploading ? "جاري الرفع..." : "اضغط لرفع صور أو فيديو"}</span>
          <span className="text-xs opacity-60">JPEG, PNG, GIF, WebP, MP4, MOV, WebM</span>
        </button>
        <input
          ref={fileRef}
          type="file"
          multiple
          accept="image/*,video/*"
          className="hidden"
          onChange={e => handleUpload(e.target.files)}
        />
      </div>

      {/* Scheduling */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="font-semibold text-foreground flex items-center gap-2">
          <Clock className="h-4 w-4 text-primary" />
          جدولة الإرسال
        </h3>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">موعد الإرسال</label>
            <input
              type="datetime-local"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              value={form.send_at}
              onChange={e => set("send_at", e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">التكرار (ساعات)</label>
            <input
              type="number"
              min={1}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="مثال: 24"
              value={form.interval_hours}
              onChange={e => set("interval_hours", e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1.5">الحد الأقصى للإرسال</label>
            <input
              type="number"
              min={1}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="مثال: 3"
              value={form.max_sends}
              onChange={e => set("max_sends", e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-3">
        <button
          type="submit"
          disabled={saving || uploading}
          className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground disabled:opacity-50 hover:bg-primary/90 transition-colors"
        >
          {saving ? "جاري الحفظ..." : initial ? "حفظ التعديلات" : "إنشاء الحملة"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent transition-colors"
        >
          إلغاء
        </button>
      </div>
    </form>
  );
}

export default function CampaignsPage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const [view, setView] = useState<View>("list");
  const [editingCampaign, setEditingCampaign] = useState<Campaign | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const campaignParams: Record<string, string> = {};
  if (statusFilter) campaignParams.status = statusFilter;
  if (selectedPageId) campaignParams.page_id = selectedPageId;

  const { data, isLoading } = useQuery({
    queryKey: ["campaigns", selectedPageId, statusFilter],
    queryFn: () => api.getCampaigns(Object.keys(campaignParams).length ? campaignParams : undefined),
  });

  const createMutation = useMutation({
    mutationFn: (d: any) => api.createCampaign({ ...d, page_id: selectedPageId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["campaigns"] });
      setView("list");
      toast.success("تم إنشاء الحملة");
    },
    onError: (e: any) => toast.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => api.updateCampaign(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["campaigns"] });
      setView("list");
      setEditingCampaign(null);
      toast.success("تم تحديث الحملة");
    },
    onError: (e: any) => toast.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteCampaign(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["campaigns"] });
      setConfirmDelete(null);
      toast.success("تم حذف الحملة");
    },
    onError: (e: any) => toast.error(e.message),
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) => api.activateCampaign(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["campaigns"] }); toast.success("تم تفعيل الحملة"); },
    onError: (e: any) => toast.error(e.message),
  });

  const pauseMutation = useMutation({
    mutationFn: (id: string) => api.pauseCampaign(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["campaigns"] }); toast.success("تم إيقاف الحملة"); },
    onError: (e: any) => toast.error(e.message),
  });

  if (view === "create") {
    return (
      <RequirePermission permission="can_manage_campaigns">
      <div className="p-6 space-y-6">
        <div className="flex items-center gap-3">
          <button onClick={() => setView("list")} className="p-2 rounded-lg hover:bg-accent text-muted-foreground">
            <X className="h-4 w-4" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">حملة جديدة</h1>
            <p className="text-sm text-muted-foreground">إنشاء حملة تسويقية للعملاء</p>
          </div>
        </div>
        <CampaignForm
          onSave={(d) => createMutation.mutate(d)}
          onCancel={() => setView("list")}
          saving={createMutation.isPending}
          pageId={selectedPageId}
        />
      </div>
      </RequirePermission>
    );
  }

  if (view === "edit" && editingCampaign) {
    return (
      <RequirePermission permission="can_manage_campaigns">
      <div className="p-6 space-y-6">
        <div className="flex items-center gap-3">
          <button onClick={() => { setView("list"); setEditingCampaign(null); }} className="p-2 rounded-lg hover:bg-accent text-muted-foreground">
            <X className="h-4 w-4" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">تعديل الحملة</h1>
            <p className="text-sm text-muted-foreground">{editingCampaign.name}</p>
          </div>
        </div>
        <CampaignForm
          initial={editingCampaign}
          onSave={(d) => updateMutation.mutate({ id: editingCampaign.id, data: d })}
          onCancel={() => { setView("list"); setEditingCampaign(null); }}
          saving={updateMutation.isPending}
          pageId={selectedPageId}
        />
      </div>
      </RequirePermission>
    );
  }

  return (
    <RequirePermission permission="can_manage_campaigns">
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground flex items-center gap-2">
            <Megaphone className="h-6 w-6 text-primary" />
            الحملات التسويقية
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">أرسل رسائل مباشرة مجمّعة لعملائك المستهدفين</p>
        </div>
        <button
          onClick={() => setView("create")}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          حملة جديدة
        </button>
      </div>

      {/* Summary stats */}
      {data && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            { label: "الإجمالي", value: data.total, color: "text-foreground" },
            { label: "نشطة", value: data.data.filter(c => c.status === "active").length, color: "text-green-400" },
            { label: "مجدولة", value: data.data.filter(c => c.status === "scheduled").length, color: "text-blue-400" },
            { label: "مسودات", value: data.data.filter(c => c.status === "draft").length, color: "text-gray-400" },
          ].map(({ label, value, color }) => (
            <div key={label} className="rounded-xl border border-border bg-card p-4 text-center">
              <p className={cn("text-2xl font-bold", color)}>{value}</p>
              <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
            </div>
          ))}
        </div>
      )}

      {/* Filter bar */}
      <div className="flex items-center gap-2 flex-wrap">
        {["", "draft", "scheduled", "active", "paused", "completed"].map(s => (
          <button
            key={s}
            onClick={() => setStatusFilter(s)}
            className={cn(
              "px-3 py-1.5 rounded-lg text-xs font-medium transition-colors border",
              statusFilter === s
                ? "bg-primary/20 text-primary border-primary/40"
                : "border-border text-muted-foreground hover:bg-accent hover:text-foreground"
            )}
          >
            {s === "" ? "الكل" : STATUS_LABEL[s]}
          </button>
        ))}
      </div>

      {/* List */}
      {isLoading ? (
        <div className="flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
      ) : data?.data.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 gap-4 rounded-xl border border-dashed border-border bg-card/50 text-muted-foreground">
          <Megaphone className="h-12 w-12 opacity-20" />
          <div className="text-center">
            <p className="text-sm font-medium">لا توجد حملات</p>
            <p className="text-xs mt-1 opacity-70">اضغط على "حملة جديدة" لإنشاء أولى حملاتك</p>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {data?.data.map(campaign => (
            <CampaignCard
              key={campaign.id}
              campaign={campaign}
              onEdit={() => { setEditingCampaign(campaign); setView("edit"); }}
              onDelete={() => setConfirmDelete(campaign.id)}
              onActivate={() => activateMutation.mutate(campaign.id)}
              onPause={() => pauseMutation.mutate(campaign.id)}
            />
          ))}
        </div>
      )}

      {/* Delete confirm modal */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="rounded-xl border border-border bg-card p-6 w-full max-w-sm space-y-4 shadow-2xl">
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-500/15 p-2">
                <Trash2 className="h-5 w-5 text-red-400" />
              </div>
              <div>
                <p className="font-semibold text-foreground">حذف الحملة</p>
                <p className="text-sm text-muted-foreground">هذا الإجراء لا يمكن التراجع عنه</p>
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => deleteMutation.mutate(confirmDelete)}
                disabled={deleteMutation.isPending}
                className="flex-1 rounded-lg bg-red-500 py-2 text-sm font-medium text-white disabled:opacity-50 hover:bg-red-600"
              >
                {deleteMutation.isPending ? "جاري الحذف..." : "حذف"}
              </button>
              <button
                onClick={() => setConfirmDelete(null)}
                className="flex-1 rounded-lg border border-border py-2 text-sm font-medium text-muted-foreground hover:bg-accent"
              >
                إلغاء
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
    </RequirePermission>
  );
}
