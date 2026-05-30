"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn, formatDate } from "@/lib/utils";
import { Users, Search, Pencil, Trash2, Save, X, Info } from "lucide-react";
import Link from "next/link";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

const INTENT_COLOR: Record<string, string> = {
  High: "bg-green-500/20 text-green-400 border-green-500/30",
  Medium: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  Low: "bg-gray-500/20 text-gray-400 border-gray-500/30",
};

const STATUS_COLOR: Record<string, string> = {
  prospect: "bg-blue-500/20 text-blue-400",
  qualified: "bg-purple-500/20 text-purple-400",
  converted: "bg-green-500/20 text-green-400",
  churned: "bg-red-500/20 text-red-400",
};

const CHURN_COLOR: Record<string, string> = {
  low: "bg-green-500/15 text-green-400",
  medium: "bg-yellow-500/15 text-yellow-400",
  high: "bg-red-500/15 text-red-400",
};

const PLATFORM_BADGE: Record<string, { label: string; style: string; icon: string }> = {
  facebook: { label: "فيسبوك", style: "bg-blue-500/10 text-blue-400 border-blue-500/20", icon: "🌐" },
  instagram: { label: "إنستغرام", style: "bg-pink-500/10 text-pink-400 border-pink-500/20", icon: "📸" },
  whatsapp: { label: "واتساب", style: "bg-green-500/10 text-green-400 border-green-500/20", icon: "💬" },
};

export default function CustomersPage() {
  const { selectedPageId } = usePageContext();
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [purchaseIntent, setPurchaseIntent] = useState("");
  const [churnRisk, setChurnRisk] = useState("");
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [bulkTag, setBulkTag] = useState("");

  // Modals & Dialogs States
  const [editCustomer, setEditCustomer] = useState<any>(null);
  const [deleteCustomer, setDeleteCustomer] = useState<any>(null);
  const [showBulkEdit, setShowBulkEdit] = useState(false);
  const [showBulkDelete, setShowBulkDelete] = useState(false);

  // Form States
  const [editForm, setEditForm] = useState({
    full_name: "",
    purchase_intent: "Low",
    conversion_status: "prospect",
    churn_risk: "low",
    lead_score: 0,
    tags: [] as string[],
  });

  const [bulkField, setBulkField] = useState("purchase_intent");
  const [bulkValue, setBulkValue] = useState("Low");

  const params: Record<string, string> = { page: String(page), limit: "20" };
  if (search) params.search = search;
  if (purchaseIntent) params.purchase_intent = purchaseIntent;
  if (churnRisk) params.churn_risk = churnRisk;
  if (selectedPageId) params.page_id = selectedPageId;

  const { data, isLoading } = useQuery({
    queryKey: ["customers", params],
    queryFn: () => api.getCustomers(params),
  });

  // MUTATIONS
  const tagMutation = useMutation({
    mutationFn: () => api.bulkTagCustomers(selectedIds, bulkTag),
    onSuccess: () => { setSelectedIds([]); setBulkTag(""); qc.invalidateQueries({ queryKey: ["customers"] }); toast.success("تم وسم العملاء"); },
    onError: () => toast.error("فشل وسم العملاء"),
  });

  const reEngageMutation = useMutation({
    mutationFn: () => api.bulkReEngage(params),
    onSuccess: () => toast.success("تم إرسال طلب إعادة التفاعل"),
    onError: () => toast.error("فشل طلب إعادة التفاعل"),
  });

  const singleUpdateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => api.updateCustomer(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
      setEditCustomer(null);
      toast.success("تم تحديث العميل بنجاح");
    },
    onError: () => toast.error("فشل تحديث العميل"),
  });

  const singleDeleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteCustomer(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
      setSelectedIds(ids => ids.filter(item => item !== deleteCustomer?.id));
      setDeleteCustomer(null);
      toast.success("تم حذف العميل بنجاح");
    },
    onError: () => toast.error("فشل حذف العميل"),
  });

  const bulkUpdateMutation = useMutation({
    mutationFn: ({ ids, update }: { ids: string[]; update: any }) => api.bulkUpdateCustomers(ids, update),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
      setSelectedIds([]);
      setShowBulkEdit(false);
      toast.success("تم تحديث العملاء بنجاح");
    },
    onError: () => toast.error("فشل التحديث الجماعي للعملاء"),
  });

  const bulkDeleteMutation = useMutation({
    mutationFn: (ids: string[]) => api.bulkDeleteCustomers(ids),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
      setSelectedIds([]);
      setShowBulkDelete(false);
      toast.success("تم حذف العملاء بنجاح");
    },
    onError: () => toast.error("فشل الحذف الجماعي للعملاء"),
  });

  const toggleSelected = (id: string) => setSelectedIds(ids => ids.includes(id) ? ids.filter(item => item !== id) : [...ids, id]);

  const handleAllSelectedToggle = () => {
    const pageIds = (data?.data ?? []).map(c => c.id);
    const allOnPageSelected = pageIds.every(id => selectedIds.includes(id));
    if (allOnPageSelected) {
      setSelectedIds(ids => ids.filter(id => !pageIds.includes(id)));
    } else {
      setSelectedIds(ids => Array.from(new Set([...ids, ...pageIds])));
    }
  };

  const startEdit = (c: any) => {
    setEditCustomer(c);
    setEditForm({
      full_name: c.full_name ?? "",
      purchase_intent: c.purchase_intent ?? "Low",
      conversion_status: c.conversion_status ?? "prospect",
      churn_risk: c.churn_risk ?? "low",
      lead_score: c.lead_score ?? 0,
      tags: c.tags ?? [],
    });
  };

  const handleSingleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editCustomer) return;
    singleUpdateMutation.mutate({ id: editCustomer.id, data: editForm });
  };

  const handleBulkEditSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (selectedIds.length === 0) return;
    bulkUpdateMutation.mutate({
      ids: selectedIds,
      update: { [bulkField]: bulkField === "lead_score" ? Number(bulkValue) : bulkValue }
    });
  };

  const onBulkFieldChange = (field: string) => {
    setBulkField(field);
    if (field === "purchase_intent") setBulkValue("Low");
    else if (field === "conversion_status") setBulkValue("prospect");
    else if (field === "churn_risk") setBulkValue("low");
    else if (field === "lead_score") setBulkValue("0");
  };

  const isAllOnPageSelected = data?.data && data.data.length > 0 && data.data.every(c => selectedIds.includes(c.id));

  return (
    <div className="p-6 space-y-5" dir="rtl">
      <div>
        <h1 className="text-2xl font-bold text-foreground">العملاء</h1>
        <p className="text-sm text-muted-foreground mt-1">{data?.total ?? 0} عميل</p>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            className="w-full rounded-lg border border-border bg-card pr-9 pl-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-right"
            placeholder="البحث بالاسم..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          />
        </div>
        <select
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none cursor-pointer"
          value={purchaseIntent}
          onChange={(e) => { setPurchaseIntent(e.target.value); setPage(1); }}
        >
          <option value="">كل نوايا الشراء</option>
          <option value="High">عالية</option>
          <option value="Medium">متوسطة</option>
          <option value="Low">منخفضة</option>
        </select>
        <select
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none cursor-pointer"
          value={churnRisk}
          onChange={(e) => { setChurnRisk(e.target.value); setPage(1); }}
        >
          <option value="">كل مستويات المغادرة</option>
          <option value="low">منخفض</option>
          <option value="medium">متوسط</option>
          <option value="high">مرتفع ⚠</option>
        </select>
      </div>

      {/* Bulk Operations Panel */}
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-indigo-500/20 bg-indigo-500/5 px-3 py-2 animate-in fade-in duration-200">
        <span className="text-sm text-indigo-300 font-semibold">{selectedIds.length} عميل محدد</span>
        
        <div className="h-4 w-px bg-border/60 mx-1 hidden sm:block" />

        <input
          value={bulkTag}
          onChange={e => setBulkTag(e.target.value)}
          placeholder="وسم جماعي..."
          className="rounded-lg border border-border bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <button
          onClick={() => tagMutation.mutate()}
          disabled={selectedIds.length === 0 || !bulkTag || tagMutation.isPending}
          className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent disabled:opacity-50 transition-colors font-medium"
        >
          إضافة وسم
        </button>

        <div className="h-4 w-px bg-border/60 mx-1" />

        <button
          onClick={() => setShowBulkEdit(true)}
          disabled={selectedIds.length === 0}
          className="rounded-lg border border-indigo-500/30 bg-indigo-500/10 text-indigo-300 px-3 py-1.5 text-xs hover:bg-indigo-500/20 disabled:opacity-50 transition-colors font-medium"
        >
          تعديل مجمع
        </button>

        <button
          onClick={() => setShowBulkDelete(true)}
          disabled={selectedIds.length === 0}
          className="rounded-lg border border-red-500/30 bg-red-500/10 text-red-300 px-3 py-1.5 text-xs hover:bg-red-500/20 disabled:opacity-50 transition-colors font-medium"
        >
          حذف مجمع
        </button>

        <div className="mr-auto">
          <button
            onClick={() => reEngageMutation.mutate()}
            disabled={reEngageMutation.isPending}
            className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent disabled:opacity-50 transition-colors font-medium"
          >
            إعادة تفاعل حسب الفلاتر
          </button>
        </div>
      </div>

      {/* Main Customers Table */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-right border-collapse">
                <thead>
                  <tr className="border-b border-border bg-muted/20">
                    <th className="p-4 font-medium text-muted-foreground w-12 text-center">
                      <input
                        type="checkbox"
                        checked={isAllOnPageSelected}
                        onChange={handleAllSelectedToggle}
                        className="rounded border-border cursor-pointer"
                        aria-label="تحديد جميع العملاء في الصفحة"
                      />
                    </th>
                    <th className="p-4 font-medium text-muted-foreground text-right">العميل</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">المصدر / الصفحة</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">نية الشراء</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">الحالة</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">النقاط</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">التفاعلات</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">خطر المغادرة</th>
                    <th className="p-4 font-medium text-muted-foreground text-right">آخر تفاعل</th>
                    <th className="p-4 font-medium text-muted-foreground text-center w-24">الإجراءات</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {(data?.data ?? []).map((c) => (
                    <tr key={c.id} className="hover:bg-accent/10 transition-colors">
                      <td className="p-4 text-center">
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(c.id)}
                          onChange={() => toggleSelected(c.id)}
                          className="rounded border-border cursor-pointer"
                          aria-label={`تحديد العميل ${c.full_name ?? c.username ?? c.id}`}
                        />
                      </td>
                      <td className="p-4">
                        <Link href={`/customers/${c.id}`} className="flex items-center gap-3 group">
                          <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center text-xs font-bold text-primary flex-shrink-0">
                            {(c.full_name ?? c.username ?? "?")[0].toUpperCase()}
                          </div>
                          <div className="truncate max-w-[200px]">
                            <p className="font-semibold text-foreground group-hover:text-primary transition-colors truncate">{c.full_name ?? c.username ?? "—"}</p>
                            <div className="flex flex-col gap-0.5 mt-0.5 items-start">
                              {c.username && c.full_name && c.username !== c.full_name && (
                                <p className="text-xs text-muted-foreground truncate">@{c.username}</p>
                              )}
                              {c.facebook_id && (
                                <p className="text-[9px] font-mono text-muted-foreground bg-blue-500/10 border border-blue-500/20 rounded px-1 py-0.5 select-all">FB: {c.facebook_id}</p>
                              )}
                              {c.whatsapp_id && (
                                <p className="text-[9px] font-mono text-muted-foreground bg-green-500/10 border border-green-500/20 rounded px-1 py-0.5 select-all">WA: {c.whatsapp_id}</p>
                              )}
                              {c.instagram_id && (
                                <p className="text-[9px] font-mono text-muted-foreground bg-pink-500/10 border border-pink-500/20 rounded px-1 py-0.5 select-all">IG: {c.instagram_id}</p>
                              )}
                            </div>
                          </div>
                        </Link>
                      </td>
                      <td className="p-4">
                        {c.platform ? (
                          <div className="flex items-center gap-2">
                            <span className={cn(
                              "text-[10px] px-2 py-0.5 rounded-full border flex items-center gap-1 font-medium",
                              PLATFORM_BADGE[c.platform]?.style ?? "bg-gray-500/10 text-gray-400 border-gray-500/20"
                            )}>
                              <span>{PLATFORM_BADGE[c.platform]?.icon}</span>
                              <span>{PLATFORM_BADGE[c.platform]?.label}</span>
                            </span>
                            {c.page_name && (
                              <span className="text-xs font-semibold text-foreground/75 truncate max-w-[120px]" title={c.page_name}>
                                {c.page_name}
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </td>
                      <td className="p-4">
                        <span className={cn("text-xs px-2 py-0.5 rounded-full border", INTENT_COLOR[c.purchase_intent] ?? "bg-gray-500/20 text-gray-400 border-gray-500/30")}>
                          {c.purchase_intent}
                        </span>
                      </td>
                      <td className="p-4">
                        <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium border border-transparent", STATUS_COLOR[c.conversion_status] ?? "bg-gray-500/20 text-gray-400")}>
                          {c.conversion_status}
                        </span>
                      </td>
                      <td className="p-4 font-mono text-primary font-bold">{c.lead_score}</td>
                      <td className="p-4 text-muted-foreground font-mono">{c.interaction_count}</td>
                      <td className="p-4">
                        <span className={cn("text-xs px-2 py-0.5 rounded-full font-semibold", CHURN_COLOR[c.churn_risk ?? "low"] ?? "bg-gray-500/15 text-gray-400")}>
                          {c.churn_risk === "high" ? "⚠ " : ""}{c.churn_risk ?? "low"}
                        </span>
                      </td>
                      <td className="p-4 text-muted-foreground font-mono text-xs">{formatDate(c.last_interaction)}</td>
                      <td className="p-4 text-center">
                        <div className="flex items-center justify-center gap-1">
                          <button
                            onClick={() => startEdit(c)}
                            className="p-1.5 rounded-lg hover:bg-indigo-500/20 text-muted-foreground hover:text-indigo-400 transition-colors"
                            title="تعديل العميل"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </button>
                          <button
                            onClick={() => setDeleteCustomer(c)}
                            className="p-1.5 rounded-lg hover:bg-red-500/20 text-muted-foreground hover:text-red-400 transition-colors"
                            title="حذف العميل"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {data?.data.length === 0 && (
              <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground bg-card">
                <Users className="h-8 w-8 opacity-40 animate-pulse" />
                <p className="text-sm">لا يوجد عملاء مطبق عليهم الفلاتر الحالية</p>
              </div>
            )}
          </>
        )}
      </div>

      {/* Pagination */}
      {data && data.total > 20 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">صفحة {page} من {Math.ceil(data.total / 20)}</p>
          <div className="flex gap-2">
            <button disabled={page === 1} onClick={() => setPage(p => p - 1)} className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent transition-colors font-medium">السابق</button>
            <button disabled={page * 20 >= data.total} onClick={() => setPage(p => p + 1)} className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent transition-colors font-medium">التالي</button>
          </div>
        </div>
      )}

      {/* Single Edit Modal */}
      {editCustomer && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-in zoom-in-95 duration-150" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-border bg-muted/40 p-4">
              <div className="flex items-center gap-2">
                <Pencil className="h-4.5 w-4.5 text-indigo-400" />
                <h3 className="font-bold text-foreground">تعديل بيانات العميل</h3>
              </div>
              <button onClick={() => setEditCustomer(null)} className="rounded p-1 hover:bg-accent text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleSingleSave} className="p-5 space-y-4">
              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground block mb-1">الاسم الكامل</label>
                <input
                  type="text"
                  required
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                  value={editForm.full_name}
                  onChange={e => setEditForm(f => ({ ...f, full_name: e.target.value }))}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">نية الشراء</label>
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={editForm.purchase_intent}
                    onChange={e => setEditForm(f => ({ ...f, purchase_intent: e.target.value }))}
                  >
                    <option value="High">عالية (High)</option>
                    <option value="Medium">متوسطة (Medium)</option>
                    <option value="Low">منخفضة (Low)</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">حالة التحويل</label>
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={editForm.conversion_status}
                    onChange={e => setEditForm(f => ({ ...f, conversion_status: e.target.value }))}
                  >
                    <option value="prospect">محتمل (Prospect)</option>
                    <option value="qualified">مؤهل (Qualified)</option>
                    <option value="converted">متحول لشراء (Converted)</option>
                    <option value="churned">مغادر (Churned)</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">خطر المغادرة</label>
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={editForm.churn_risk}
                    onChange={e => setEditForm(f => ({ ...f, churn_risk: e.target.value }))}
                  >
                    <option value="low">منخفض (Low)</option>
                    <option value="medium">متوسط (Medium)</option>
                    <option value="high">مرتفع (High)</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground block mb-1">النقاط (Lead Score)</label>
                  <input
                    type="number"
                    min="0"
                    max="100"
                    required
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
                    value={editForm.lead_score}
                    onChange={e => setEditForm(f => ({ ...f, lead_score: Number(e.target.value) }))}
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2 border-t border-border pt-4 mt-6">
                <button type="button" onClick={() => setEditCustomer(null)} className="rounded-xl border border-border px-4 py-2 text-sm font-semibold text-muted-foreground hover:bg-accent transition-all">إلغاء</button>
                <button type="submit" disabled={singleUpdateMutation.isPending} className="flex items-center gap-1.5 rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-lg hover:opacity-90 disabled:opacity-50 transition-all active:scale-95">
                  <Save className="h-4 w-4" />
                  حفظ البيانات
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Bulk Edit Modal */}
      {showBulkEdit && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-in zoom-in-95 duration-150" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-border bg-muted/40 p-4">
              <div className="flex items-center gap-2">
                <Pencil className="h-4.5 w-4.5 text-indigo-400" />
                <h3 className="font-bold text-foreground">تعديل جماعي للعملاء المحددين</h3>
              </div>
              <button onClick={() => setShowBulkEdit(false)} className="rounded p-1 hover:bg-accent text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="bg-indigo-500/10 border-b border-indigo-500/20 p-3 flex items-start gap-2.5 text-indigo-200 text-xs">
              <Info className="h-4.5 w-4.5 text-indigo-400 flex-shrink-0 mt-0.5" />
              <p>سوف يتم تطبيق التعديل على عدد **{selectedIds.length}** من العملاء المحددين حالياً دفعة واحدة.</p>
            </div>

            <form onSubmit={handleBulkEditSubmit} className="p-5 space-y-4">
              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground block mb-1">اختر الحقل المراد تعديله</label>
                <select
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                  value={bulkField}
                  onChange={e => onBulkFieldChange(e.target.value)}
                >
                  <option value="purchase_intent">نية الشراء</option>
                  <option value="conversion_status">حالة التحويل</option>
                  <option value="churn_risk">خطر المغادرة</option>
                  <option value="lead_score">النقاط (Lead Score)</option>
                </select>
              </div>

              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground block mb-1">القيمة الجديدة للحقل</label>
                
                {bulkField === "purchase_intent" && (
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={bulkValue}
                    onChange={e => setBulkValue(e.target.value)}
                  >
                    <option value="High">عالية (High)</option>
                    <option value="Medium">متوسطة (Medium)</option>
                    <option value="Low">منخفضة (Low)</option>
                  </select>
                )}

                {bulkField === "conversion_status" && (
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={bulkValue}
                    onChange={e => setBulkValue(e.target.value)}
                  >
                    <option value="prospect">محتمل (Prospect)</option>
                    <option value="qualified">مؤهل (Qualified)</option>
                    <option value="converted">متحول لشراء (Converted)</option>
                    <option value="churned">مغادر (Churned)</option>
                  </select>
                )}

                {bulkField === "churn_risk" && (
                  <select
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                    value={bulkValue}
                    onChange={e => setBulkValue(e.target.value)}
                  >
                    <option value="low">منخفض (Low)</option>
                    <option value="medium">متوسط (Medium)</option>
                    <option value="high">مرتفع (High)</option>
                  </select>
                )}

                {bulkField === "lead_score" && (
                  <input
                    type="number"
                    min="0"
                    max="100"
                    required
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
                    value={bulkValue}
                    onChange={e => setBulkValue(e.target.value)}
                  />
                )}
              </div>

              <div className="flex justify-end gap-2 border-t border-border pt-4 mt-6">
                <button type="button" onClick={() => setShowBulkEdit(false)} className="rounded-xl border border-border px-4 py-2 text-sm font-semibold text-muted-foreground hover:bg-accent transition-all">إلغاء</button>
                <button type="submit" disabled={bulkUpdateMutation.isPending} className="flex items-center gap-1.5 rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-lg hover:opacity-90 disabled:opacity-50 transition-all active:scale-95">
                  تطبيق جماعي
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Confirmation Dialog for Single Delete */}
      <ConfirmDialog
        open={Boolean(deleteCustomer)}
        title="حذف العميل نهائياً"
        description={`هل أنت متأكد من رغبتك في حذف العميل "${deleteCustomer?.full_name ?? deleteCustomer?.username ?? 'هذا العميل'}" نهائياً من النظام وقاعدة البيانات؟ لا يمكن التراجع عن هذا الإجراء.`}
        destructive
        confirmLabel="حذف"
        pending={singleDeleteMutation.isPending}
        onCancel={() => setDeleteCustomer(null)}
        onConfirm={() => deleteCustomer && singleDeleteMutation.mutate(deleteCustomer.id)}
      />

      {/* Confirmation Dialog for Bulk Delete */}
      <ConfirmDialog
        open={showBulkDelete}
        title="حذف العملاء المحددين جماعياً"
        description={`هل أنت متأكد من رغبتك في حذف عدد (${selectedIds.length}) عملاء محددين نهائياً من قاعدة البيانات؟ سيتم مسح بياناتهم بالكامل ولا يمكن استرجاعها.`}
        destructive
        confirmLabel="حذف الكل"
        pending={bulkDeleteMutation.isPending}
        onCancel={() => setShowBulkDelete(false)}
        onConfirm={() => bulkDeleteMutation.mutate(selectedIds)}
      />
    </div>
  );
}
