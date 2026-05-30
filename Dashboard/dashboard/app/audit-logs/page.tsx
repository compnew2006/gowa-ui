"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { Clock, Filter, ChevronLeft, ChevronRight } from "lucide-react";

export default function AuditLogsPage() {
  const { selectedPageId } = usePageContext();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [adminFilter, setAdminFilter] = useState("");

  const params: Record<string, string> = {
    page: String(page),
    limit: "50",
  };
  if (selectedPageId) params.page_id = selectedPageId;
  if (search) params.search = search;
  if (actionFilter) params.action = actionFilter;
  if (adminFilter) params.admin_id = adminFilter;

  const { data: logs, isLoading } = useQuery({
    queryKey: ["audit-logs", page, selectedPageId, search, actionFilter, adminFilter],
    queryFn: () => api.getAuditLogs(params),
  });

  const { data: stats } = useQuery({
    queryKey: ["audit-stats", selectedPageId],
    queryFn: () => {
      const sp: Record<string, string> = {};
      if (selectedPageId) sp.page_id = selectedPageId;
      return api.getAuditStats(sp);
    },
  });

  const actionLabel: Record<string, string> = {
    approve: "✅ موافقة",
    reject: "❌ رفض",
    correct: "🔧 تصحيح",
    undo: "↶ رجوع",
    create: "✨ إنشاء",
    update: "📝 تحديث",
    delete: "🗑️ حذف",
  };

  const entityTypeLabel: Record<string, string> = {
    conversation: "محادثة",
    knowledge_base: "قاعدة معرفة",
    settings: "إعدادات",
    rule: "قاعدة",
    customer: "عميل",
    page: "صفحة",
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-3 mb-8">
          <div className="p-3 rounded-lg bg-blue-500/20 border border-blue-500/30">
            <Clock className="w-6 h-6 text-blue-400" />
          </div>
          <div>
            <h1 className="text-3xl font-bold text-white">سجل التدقيق</h1>
            <p className="text-slate-400">تتبع جميع إجراءات الإدارة والتعديلات</p>
          </div>
        </div>

        {/* Stats Cards */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            {Object.entries(stats.by_action || {}).map(([action, count]) => (
              <div
                key={action}
                className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4"
              >
                <div className="text-sm text-slate-400 mb-1">
                  {actionLabel[action] || action}
                </div>
                <div className="text-2xl font-bold text-white">{String(count)}</div>
              </div>
            ))}
          </div>
        )}

        {/* Filters */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 mb-6">
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2 text-slate-400">
              <Filter className="w-5 h-5" />
              <span>التصفية</span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm text-slate-400 mb-2">
                  البحث
                </label>
                <input
                  type="text"
                  placeholder="بحث..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white placeholder-slate-500"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-2">
                  نوع الإجراء
                </label>
                <select
                  value={actionFilter}
                  onChange={(e) => {
                    setActionFilter(e.target.value);
                    setPage(1);
                  }}
                  className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white"
                >
                  <option value="">الكل</option>
                  <option value="approve">موافقة</option>
                  <option value="reject">رفض</option>
                  <option value="correct">تصحيح</option>
                  <option value="undo">رجوع</option>
                  <option value="create">إنشاء</option>
                  <option value="update">تحديث</option>
                  <option value="delete">حذف</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-2">
                  الإدارة / المستخدم
                </label>
                <input
                  type="text"
                  placeholder="معرف الإدارة..."
                  value={adminFilter}
                  onChange={(e) => {
                    setAdminFilter(e.target.value);
                    setPage(1);
                  }}
                  className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white placeholder-slate-500"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Table */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
          {isLoading ? (
            <div className="flex justify-center items-center h-64">
              <div className="text-slate-400">جاري التحميل...</div>
            </div>
          ) : !logs?.data?.length ? (
            <div className="flex justify-center items-center h-64">
              <div className="text-slate-400">لا توجد سجلات</div>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-slate-900/50 border-b border-slate-700/50">
                    <tr>
                      <th className="px-4 py-3 text-right text-slate-400 font-semibold">
                        الوقت
                      </th>
                      <th className="px-4 py-3 text-right text-slate-400 font-semibold">
                        الإجراء
                      </th>
                      <th className="px-4 py-3 text-right text-slate-400 font-semibold">
                        نوع الكيان
                      </th>
                      <th className="px-4 py-3 text-right text-slate-400 font-semibold">
                        الإدارة
                      </th>
                      <th className="px-4 py-3 text-right text-slate-400 font-semibold">
                        التفاصيل
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-700/30">
                    {logs.data.map((log: any) => (
                      <tr
                        key={log.id}
                        className="hover:bg-slate-700/20 transition-colors"
                      >
                        <td className="px-4 py-3 text-slate-300 text-xs">
                          {new Date(log.created_at).toLocaleString("ar-EG")}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={cn(
                              "px-2 py-1 rounded text-xs font-semibold",
                              {
                                "bg-green-500/20 text-green-400":
                                  log.action === "approve",
                                "bg-red-500/20 text-red-400":
                                  log.action === "reject",
                                "bg-yellow-500/20 text-yellow-400":
                                  log.action === "correct",
                                "bg-blue-500/20 text-blue-400":
                                  log.action === "undo",
                                "bg-purple-500/20 text-purple-400":
                                  log.action === "create",
                                "bg-orange-500/20 text-orange-400":
                                  log.action === "update",
                                "bg-pink-500/20 text-pink-400":
                                  log.action === "delete",
                              }
                            )}
                          >
                            {actionLabel[log.action] || log.action}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          {entityTypeLabel[log.entity_type] || log.entity_type}
                        </td>
                        <td className="px-4 py-3 text-slate-300 text-sm">
                          {log.admin_name || log.admin_id || "نظام"}
                        </td>
                        <td className="px-4 py-3 text-slate-400 text-xs max-w-xs truncate" title={log.details ? JSON.stringify(log.details) : log.reason || ""}>
                          {log.details 
                            ? (typeof log.details === 'object' 
                                ? Object.entries(log.details).map(([k, v]) => `${k}: ${typeof v === 'object' ? JSON.stringify(v) : String(v)}`).join(', ') 
                                : String(log.details)) 
                            : log.reason || "-"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {logs && logs.total > 0 && (
                <div className="flex items-center justify-between px-6 py-4 border-t border-slate-700/50">
                  <div className="text-sm text-slate-400">
                    {Math.min((page - 1) * 50 + 1, logs.total)} -{" "}
                    {Math.min(page * 50, logs.total)} من {logs.total}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setPage(Math.max(1, page - 1))}
                      disabled={page === 1}
                      className="p-2 rounded border border-slate-700 hover:bg-slate-700/50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="w-4 h-4 text-slate-400" />
                    </button>
                    <span className="flex items-center px-3 text-sm text-slate-400">
                      {page}
                    </span>
                    <button
                      onClick={() =>
                        setPage(Math.min(Math.ceil(logs.total / 50), page + 1))
                      }
                      disabled={page * 50 >= logs.total}
                      className="p-2 rounded border border-slate-700 hover:bg-slate-700/50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronRight className="w-4 h-4 text-slate-400" />
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
