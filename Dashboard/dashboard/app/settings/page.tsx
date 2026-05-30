"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, type AppSettings } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { useState, useEffect } from "react";
import { Save, Bot, MessageSquare, Bell, Shield, Settings, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { SecretInput } from "@/components/ui/secret-input";
import { RequirePermission } from "@/components/auth/permission-guard";
import AIModelsSettings from "@/components/settings/ai-models";

function Section({ title, icon: Icon, children }: { title: string; icon: any; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
      <div className="flex items-center gap-2 pb-2 border-b border-border">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <h2 className="font-semibold text-foreground">{title}</h2>
      </div>
      {children}
    </div>
  );
}

function Field({ label, id, children }: { label: string; id?: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <label htmlFor={id} className="text-sm text-muted-foreground cursor-pointer select-none">{label}</label>
      <div className="flex-shrink-0">{children}</div>
    </div>
  );
}

export default function SettingsPage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const { data: settings, isLoading } = useQuery({
    queryKey: ["settings", selectedPageId],
    queryFn: () => api.getSettings(selectedPageId),
  });

  const [activeTab, setActiveTab] = useState<"general" | "custom_models" | "brand_kit">("brand_kit");
  const [form, setForm] = useState<Partial<AppSettings>>({});
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (settings) { setForm(settings); setDirty(false); }
  }, [settings]);

  const update = (key: keyof AppSettings, value: any) => {
    setForm(f => ({ ...f, [key]: value }));
    setDirty(true);
  };

  const saveMutation = useMutation({
    mutationFn: () => api.updateSettings(form, selectedPageId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["settings"] }); setDirty(false); toast.success("تم حفظ الإعدادات"); },
    onError: () => toast.error("فشل الحفظ"),
  });

  if (isLoading) return <div className="flex items-center justify-center h-64 text-muted-foreground">جاري التحميل...</div>;

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-6" dir="rtl">
      {/* Settings Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">الإعدادات</h1>
          <p className="text-sm text-muted-foreground mt-1">تكوين إعدادات النظام وقنوات الاتصال والذكاء الاصطناعي</p>
        </div>
        {dirty && (activeTab === "general" || activeTab === "brand_kit") && (
          <button
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-lg hover:opacity-90 disabled:opacity-50 transition-all active:scale-95 animate-in fade-in duration-150"
          >
            <Save className="h-4.5 w-4.5" />
            حفظ التغييرات
          </button>
        )}
      </div>

      {/* Modern Glassmorphic Tab Switcher */}
      <div className="flex gap-2 border-b border-border/80 pb-px">
        <button
          onClick={() => setActiveTab("brand_kit")}
          className={cn(
            "flex items-center gap-2 pb-3 text-sm font-semibold border-b-2 px-4 transition-all duration-200 outline-none",
            activeTab === "brand_kit"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
        >
          <Bot className="h-4 w-4" />
          هوية العلامة التجارية (Brand Kit)
        </button>
        <button
          onClick={() => setActiveTab("general")}
          className={cn(
            "flex items-center gap-2 pb-3 text-sm font-semibold border-b-2 px-4 transition-all duration-200 outline-none",
            activeTab === "general"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
        >
          <Settings className="h-4 w-4" />
          الإعدادات العامة والرسائل
        </button>
        <button
          onClick={() => setActiveTab("custom_models")}
          className={cn(
            "flex items-center gap-2 pb-3 text-sm font-semibold border-b-2 px-4 transition-all duration-200 outline-none",
            activeTab === "custom_models"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
        >
          <Sparkles className="h-4 w-4" />
          نماذج الذكاء الاصطناعي المخصصة
        </button>
      </div>

      {/* Tabs Content */}
      {activeTab === "general" && (
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
          <Section title="محددات استجابة الذكاء الاصطناعي" icon={Bot}>
            <Field label="حد الثقة للرد" id="confidence-threshold">
              <div className="flex items-center gap-2">
                <input
                  id="confidence-threshold"
                  type="range" min={0} max={1} step={0.01}
                  value={form.confidence_threshold ?? 0.85}
                  onChange={e => update("confidence_threshold", parseFloat(e.target.value))}
                  className="w-32 accent-primary cursor-pointer"
                />
                <span className="text-sm font-mono text-primary w-12 text-left">{((form.confidence_threshold ?? 0.85) * 100).toFixed(0)}%</span>
              </div>
            </Field>
            <Field label="أقصى عدد من المحاولات" id="max-retries">
              <input
                id="max-retries"
                type="number" min={1} max={10}
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-20 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.max_retries ?? 3}
                onChange={e => update("max_retries", parseInt(e.target.value))}
              />
            </Field>
          </Section>

          <Section title="إعدادات الرسائل والردود" icon={MessageSquare}>
            <Field label="اللغة الافتراضية" id="default-language">
              <select
                id="default-language"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                value={form.default_language ?? "ar"}
                onChange={e => update("default_language", e.target.value)}
              >
                <option value="ar">العربية (Arabic)</option>
                <option value="en">الإنجليزية (English)</option>
              </select>
            </Field>
            <Field label="تصعيد تلقائي للتعليقات الغاضبة">
              <button
                type="button"
                role="switch"
                aria-checked={Boolean(form.auto_escalate_angry)}
                onClick={() => update("auto_escalate_angry", !form.auto_escalate_angry)}
                className={cn("relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50", form.auto_escalate_angry ? "bg-primary" : "bg-muted")}
              >
                <span className={cn("inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200", form.auto_escalate_angry ? "translate-x-6" : "translate-x-1")} />
              </button>
            </Field>
            <Field label="وضع الإحماء والتدريب">
              <button
                type="button"
                role="switch"
                aria-checked={Boolean(form.warmup_mode)}
                onClick={() => update("warmup_mode", !form.warmup_mode)}
                className={cn("relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50", form.warmup_mode ? "bg-primary" : "bg-muted")}
              >
                <span className={cn("inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200", form.warmup_mode ? "translate-x-6" : "translate-x-1")} />
              </button>
            </Field>
            <Field label="تفعيل الردود الخاصة في الرسائل (DM)">
              <button
                type="button"
                role="switch"
                aria-checked={Boolean(form.enable_private_replies)}
                onClick={() => update("enable_private_replies", !form.enable_private_replies)}
                className={cn("relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50", form.enable_private_replies ? "bg-primary" : "bg-muted")}
              >
                <span className={cn("inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200", form.enable_private_replies ? "translate-x-6" : "translate-x-1")} />
              </button>
            </Field>
          </Section>

          <Section title="إشعارات Telegram الفورية" icon={Bell}>
            <Field label="رمز اتصال البوت (Bot Token)" id="telegram-bot-token">
              <SecretInput
                id="telegram-bot-token"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.telegram_bot_token ?? ""}
                onChange={value => update("telegram_bot_token", value)}
                placeholder="123456:ABC..."
                saved={Boolean(settings?.telegram_bot_token)}
              />
            </Field>
            <Field label="معرف دردشة المسؤول (Chat ID)" id="telegram-chat-id">
              <input
                id="telegram-chat-id"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.telegram_chat_id ?? ""}
                onChange={e => update("telegram_chat_id", e.target.value)}
                placeholder="-100123456789"
              />
            </Field>
            <p className="text-xs text-muted-foreground mt-1">ستتلقى إشعارات فورية عبر Telegram عند وجود تصعيدات أو أخطاء.</p>
          </Section>

          <Section title="إشعارات WhatsApp" icon={MessageSquare}>
            <Field label="رقم الهاتف المستلم" id="whatsapp-phone">
              <input
                id="whatsapp-phone"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.whatsapp_notification_phone ?? ""}
                onChange={e => update("whatsapp_notification_phone", e.target.value)}
                placeholder="34600000000"
              />
            </Field>
            <Field label="مفتاح الـ API (من CallMeBot)" id="whatsapp-api-key">
              <SecretInput
                id="whatsapp-api-key"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.whatsapp_notification_api_key ?? ""}
                onChange={value => update("whatsapp_notification_api_key", value)}
                placeholder="123456"
                saved={Boolean(settings?.whatsapp_notification_api_key)}
              />
            </Field>
            <p className="text-xs text-muted-foreground mt-1">استخدم مزود الخدمة CallMeBot لتلقي الإشعارات مباشرة على هاتفك المحمول.</p>
          </Section>

          <Section title="إعدادات WhatsApp Cloud API الرسمية" icon={MessageSquare}>
            <Field label="معرف رقم الهاتف Business" id="whatsapp-business-phone-id">
              <input
                id="whatsapp-business-phone-id"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.whatsapp_business_phone_number_id ?? ""}
                onChange={e => update("whatsapp_business_phone_number_id", e.target.value)}
                placeholder="1234567890..."
              />
            </Field>
            <Field label="رمز اتصال Cloud API" id="whatsapp-cloud-api-token">
              <SecretInput
                id="whatsapp-cloud-api-token"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                value={form.whatsapp_cloud_api_token ?? ""}
                onChange={value => update("whatsapp_cloud_api_token", value)}
                placeholder="EAAG..."
                saved={Boolean(settings?.whatsapp_cloud_api_token)}
              />
            </Field>
            <p className="text-xs text-muted-foreground mt-1">تكوين بوابة WhatsApp الرسمية من Meta للاتصال السحابي المباشر.</p>
          </Section>

          <Section title="بوابة Webhook للاتصال" icon={Shield}>
            <Field label="رمز التحقق (Verify Token)" id="webhook-verify-token">
              <input
                id="webhook-verify-token"
                className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-48 font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
                value={form.webhook_verify_token ?? ""}
                onChange={e => update("webhook_verify_token", e.target.value)}
              />
            </Field>
            <p className="text-xs text-muted-foreground">رابط الاستقبال: <code className="text-primary font-mono bg-background px-1.5 py-0.5 rounded border">/api/webhook/meta</code></p>
            <Field label="نسبة تنبيه حد المعدل" id="rate-limit-threshold">
              <div className="flex items-center gap-2">
                <input
                  id="rate-limit-threshold"
                  type="number" min={1} max={100}
                  className="rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground w-20 focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                  value={form.rate_limit_warning_threshold ?? 80}
                  onChange={e => update("rate_limit_warning_threshold", parseInt(e.target.value))}
                />
                <span className="text-sm text-muted-foreground">%</span>
              </div>
            </Field>
          </Section>

          <Section title="ردود الثقة المنخفضة والبديلة" icon={MessageSquare}>
            <Field label="الرسالة العربية البديلة" id="safe-reply-ar">
              <textarea
                id="safe-reply-ar"
                className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 h-20 resize-none animate-in fade-in"
                value={form.safe_reply_ar ?? ""}
                onChange={e => update("safe_reply_ar", e.target.value)}
                placeholder="شكراً لتواصلك معنا..."
              />
            </Field>
            <Field label="الرسالة الإنجليزية البديلة" id="safe-reply-en">
              <textarea
                id="safe-reply-en"
                className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 h-20 resize-none font-mono animate-in fade-in"
                value={form.safe_reply_en ?? ""}
                onChange={e => update("safe_reply_en", e.target.value)}
                placeholder="Thank you for reaching out..."
              />
            </Field>
            <p className="text-xs text-muted-foreground">تُسخدم هذه الرسائل تلقائياً عند انخفاض نسبة ثقة النموذج أو عدم توفر إجابة ملائمة.</p>
          </Section>

          <Section title="الردود العامة الافتراضية على التعليقات" icon={MessageSquare}>
            <div className="space-y-3">
              <p className="text-xs text-muted-foreground">
                تظهر هذه الرسالة كرد عام على التعليقات العامة، بينما يتم إرسال الرد الذكي المخصص عبر الرسائل الخاصة (DM).
              </p>
              <Field label="الرسالة العربية" id="public-reply-ar">
                <textarea
                  id="public-reply-ar"
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 h-16 resize-none animate-in fade-in"
                  value={form.public_reply_message_ar ?? ""}
                  onChange={e => update("public_reply_message_ar", e.target.value)}
                  placeholder="تم التواصل معك على الخاص"
                />
              </Field>
              <Field label="الرسالة الإنجليزية" id="public-reply-en">
                <textarea
                  id="public-reply-en"
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 h-16 resize-none font-mono animate-in fade-in"
                  value={form.public_reply_message_en ?? ""}
                  onChange={e => update("public_reply_message_en", e.target.value)}
                  placeholder="We've contacted you privately"
                />
              </Field>
            </div>
          </Section>

        </div>
      )}

      {activeTab === "brand_kit" && (
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
          <Section title="الملف العام للهوية" icon={Bot}>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-1.5 block">مجال العمل (Industry)</label>
                <input
                  type="text"
                  placeholder="مثال: مطاعم، تقنية معلومات، أزياء..."
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50"
                  value={form.brand_industry ?? ""}
                  onChange={e => update("brand_industry", e.target.value)}
                />
              </div>
              <div>
                <label className="text-sm font-medium mb-1.5 block">نبرة الصوت (Tone of Voice)</label>
                <select
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
                  value={form.brand_tone_of_voice ?? "Professional"}
                  onChange={e => update("brand_tone_of_voice", e.target.value)}
                >
                  <option value="Professional">مهني / احترافي (Professional)</option>
                  <option value="Friendly">ودود / لطيف (Friendly)</option>
                  <option value="Bold">جريء / قوي (Bold)</option>
                  <option value="Humorous">مرح / فكاهي (Humorous)</option>
                  <option value="Empathetic">متعاطف / داعم (Empathetic)</option>
                  <option value="Educational">تعليمي / معرفي (Educational)</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium mb-1.5 block">الجمهور المستهدف (Target Audience)</label>
                <textarea
                  rows={3}
                  placeholder="مثال: الشباب من عمر 18-30 المهتمين بالرياضة وتناول الغذاء الصحي..."
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none"
                  value={form.brand_target_audience ?? ""}
                  onChange={e => update("brand_target_audience", e.target.value)}
                />
              </div>
            </div>
          </Section>

          <Section title="تفاصيل العلامة التجارية والمنشورات" icon={Sparkles}>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-1.5 block">وصف العلامة التجارية (Description)</label>
                <textarea
                  rows={4}
                  placeholder="اكتب وصفاً تفصيلياً لعلامتك التجارية ورسالتك والقيم الأساسية التي تقدمها..."
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none"
                  value={form.brand_description ?? ""}
                  onChange={e => update("brand_description", e.target.value)}
                />
              </div>
              <div>
                <label className="text-sm font-medium mb-1.5 block">منشورات سابقة ناجحة (Reference Posts)</label>
                <textarea
                  rows={4}
                  placeholder="أدخل أمثلة على منشورات سابقة ناجحة لتوجيه نموذج الذكاء الاصطناعي في الأسلوب والصياغة..."
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none font-mono text-xs"
                  value={form.brand_sample_posts ?? ""}
                  onChange={e => update("brand_sample_posts", e.target.value)}
                />
              </div>
            </div>
          </Section>

          <Section title="الكلمات والمحددات اللغوية" icon={Shield}>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-1.5 block">الهاشتاقات المفضلة (Preferred Hashtags)</label>
                <input
                  type="text"
                  placeholder="مثال: #رياضة, #تغذية, #صحة"
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono"
                  value={form.brand_preferred_hashtags ?? ""}
                  onChange={e => update("brand_preferred_hashtags", e.target.value)}
                />
                <span className="text-xs text-muted-foreground mt-1 block font-sans">مفصولة بفاصلة. سيتم إدراجها تلقائياً بالمنشورات المحدثة.</span>
              </div>
              <div>
                <label className="text-sm font-medium mb-1.5 block">الكلمات المحظورة (Restricted Words)</label>
                <input
                  type="text"
                  placeholder="مثال: خصم، مجاني، أفضل، أرخص..."
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground w-full focus:outline-none focus:ring-2 focus:ring-primary/50"
                  value={form.brand_restricted_words ?? ""}
                  onChange={e => update("brand_restricted_words", e.target.value)}
                />
                <span className="text-xs text-muted-foreground mt-1 block font-sans">الكلمات التي سيحرص الذكاء الاصطناعي على تجنبها في منشوراتك تماماً.</span>
              </div>
            </div>
          </Section>
        </div>
      )}

      {activeTab === "custom_models" && <AIModelsSettings />}
    </div>
    </RequirePermission>
  );
}
