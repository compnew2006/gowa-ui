"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Palette, Globe, Mail, Layout, Save, Upload } from "lucide-react";
import { toast } from "sonner";
import { getSafeDisplayUrl, isSafeDisplayUrl, validateImageFile } from "@/lib/security";
import { RequirePermission } from "@/components/auth/permission-guard";

export default function BrandingPage() {
  const qc = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const { data: profile, isLoading } = useQuery({
    queryKey: ["agency-profile"],
    queryFn: () => api.getAgencyProfile(),
  });

  const [form, setForm] = useState({
    agency_name: "",
    dashboard_title: "",
    primary_color: "#3b82f6",
    support_email: "",
    logo_url: "",
  });

  useEffect(() => {
    if (profile && !form.agency_name && !isLoading) {
      setForm({
          agency_name: profile.agency_name || "",
          dashboard_title: profile.dashboard_title || "",
          primary_color: profile.primary_color || "#3b82f6",
          support_email: profile.support_email || "",
          logo_url: profile.logo_url || "",
      });
    }
  }, [form.agency_name, isLoading, profile]);

  const mutation = useMutation({
    mutationFn: (data: any) => api.updateAgencyProfile(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["agency-profile"] });
      toast.success("تم حفظ إعدادات العلامة التجارية");
    },
    onError: () => toast.error("فشل الحفظ"),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => api.uploadAgencyLogo(file),
    onSuccess: (data) => {
      setForm(current => ({ ...current, logo_url: data.url }));
      toast.success("تم رفع الشعار");
    },
    onError: () => toast.error("فشل رفع الشعار"),
  });

  useEffect(() => {
    document.documentElement.style.setProperty("--primary", hexToHsl(form.primary_color));
  }, [form.primary_color]);

  const handleSave = () => {
    if (form.logo_url && !isSafeDisplayUrl(form.logo_url)) {
      toast.error("رابط الشعار يجب أن يكون HTTPS أو رابطاً محلياً صالحاً");
      return;
    }
    mutation.mutate(form);
  };

  const safeLogoUrl = getSafeDisplayUrl(form.logo_url);

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-6 max-w-4xl">
      <div>
        <h1 className="text-2xl font-bold text-foreground">العلامة التجارية الوكالة</h1>
        <p className="text-sm text-muted-foreground mt-1">تخصيص مظهر لوحة التحكم لعملائك</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Form */}
        <div className="md:col-span-2 space-y-4">
          <div className="rounded-xl border border-border bg-card p-6 space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Layout className="h-4 w-4 text-primary" /> اسم الوكالة
              </label>
              <input
                className="w-full rounded-lg border border-border bg-background px-4 py-2 text-sm focus:ring-2 focus:ring-primary/50 outline-none"
                value={form.agency_name}
                onChange={(e) => setForm({ ...form, agency_name: e.target.value })}
                placeholder="Social Media AI Agency"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Globe className="h-4 w-4 text-primary" /> عنوان لوحة التحكم
              </label>
              <input
                className="w-full rounded-lg border border-border bg-background px-4 py-2 text-sm focus:ring-2 focus:ring-primary/50 outline-none"
                value={form.dashboard_title}
                onChange={(e) => setForm({ ...form, dashboard_title: e.target.value })}
                placeholder="لوحة تحكم الوكالة"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Mail className="h-4 w-4 text-primary" /> بريد الدعم الفني
              </label>
              <input
                className="w-full rounded-lg border border-border bg-background px-4 py-2 text-sm focus:ring-2 focus:ring-primary/50 outline-none"
                value={form.support_email}
                onChange={(e) => setForm({ ...form, support_email: e.target.value })}
                placeholder="support@agency.com"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Palette className="h-4 w-4 text-primary" /> اللون الأساسي
              </label>
              <div className="flex gap-3">
                <input
                  type="color"
                  className="h-10 w-20 rounded border border-border bg-background p-1 outline-none"
                  value={form.primary_color}
                  onChange={(e) => setForm({ ...form, primary_color: e.target.value })}
                />
                <input
                  className="flex-1 rounded-lg border border-border bg-background px-4 py-2 text-sm focus:ring-2 focus:ring-primary/50 outline-none"
                  value={form.primary_color}
                  onChange={(e) => setForm({ ...form, primary_color: e.target.value })}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Upload className="h-4 w-4 text-primary" /> شعار الوكالة
              </label>
              <div className="flex gap-3">
                <input
                  type="url"
                  dir="ltr"
                  className="flex-1 rounded-lg border border-border bg-background px-4 py-2 text-sm focus:ring-2 focus:ring-primary/50 outline-none"
                  value={form.logo_url}
                  onChange={(e) => setForm({ ...form, logo_url: e.target.value })}
                  placeholder="https://example.com/logo.png"
                />
                <button
                  type="button"
                  onClick={() => fileRef.current?.click()}
                  disabled={uploadMutation.isPending}
                  className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-accent disabled:opacity-50"
                >
                  رفع
                </button>
                <input
                  ref={fileRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    const error = validateImageFile(file);
                    if (error) {
                      toast.error(error);
                      e.target.value = "";
                      return;
                    }
                    uploadMutation.mutate(file);
                  }}
                />
              </div>
            </div>
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleSave}
              disabled={mutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
              {mutation.isPending ? "جاري الحفظ..." : "حفظ التغييرات"}
            </button>
          </div>
        </div>

        {/* Preview */}
        <div className="space-y-4">
          <div className="rounded-xl border border-border bg-card p-6">
            <h3 className="text-sm font-semibold mb-4 border-b border-border pb-2">معاينة الهوية</h3>
            <div className="space-y-4">
              <div className="flex flex-col items-center gap-3 p-4 rounded-lg bg-accent/20 border border-border">
                <div 
                  className="h-16 w-16 rounded-xl flex items-center justify-center text-white text-xl font-bold"
                  style={{ backgroundColor: form.primary_color }}
                >
                  {safeLogoUrl ? (
                    <img src={safeLogoUrl} alt="" referrerPolicy="no-referrer" className="h-full w-full rounded-xl object-cover" />
                  ) : form.agency_name ? form.agency_name[0].toUpperCase() : "A"}
                </div>
                <div className="text-center">
                  <p className="font-bold text-foreground">{form.agency_name || "اسم الوكالة"}</p>
                  <p className="text-[10px] text-muted-foreground uppercase tracking-widest">{form.dashboard_title || "لوحة التحكم"}</p>
                </div>
              </div>

              <div className="space-y-2">
                <div className="h-2 w-full rounded bg-muted animate-pulse" />
                <div className="h-2 w-2/3 rounded bg-muted animate-pulse" />
              </div>
              
              <button 
                className="w-full rounded-md py-2 text-xs font-bold text-white pointer-events-none"
                style={{ backgroundColor: form.primary_color }}
              >
                زر المعاينة
              </button>
            </div>
          </div>

          <div className="rounded-xl border border-border bg-yellow-500/10 p-4">
            <p className="text-xs text-yellow-500 font-medium leading-relaxed">
              * سيتم تطبيق هذه الإعدادات على جميع صفحات لوحة التحكم وواجهات العملاء (White-labeling).
            </p>
          </div>
        </div>
      </div>
    </div>
    </RequirePermission>
  );
}

function hexToHsl(hex: string): string {
  const normalized = hex.replace("#", "");
  if (!/^[0-9a-f]{6}$/i.test(normalized)) return "221 83% 53%";
  const r = parseInt(normalized.slice(0, 2), 16) / 255;
  const g = parseInt(normalized.slice(2, 4), 16) / 255;
  const b = parseInt(normalized.slice(4, 6), 16) / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;
  const d = max - min;
  if (d !== 0) {
    s = d / (1 - Math.abs(2 * l - 1));
    switch (max) {
      case r:
        h = 60 * (((g - b) / d) % 6);
        break;
      case g:
        h = 60 * ((b - r) / d + 2);
        break;
      default:
        h = 60 * ((r - g) / d + 4);
    }
  }
  return `${Math.round((h + 360) % 360)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`;
}
