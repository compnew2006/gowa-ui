"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, type CustomAIModel } from "@/lib/api";
import { useState, useEffect } from "react";
import { Plus, Trash2, Edit2, Check, AlertCircle, Bot, Sparkles, Key, Globe, Eye, EyeOff, Activity } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

export default function AIModelsSettings() {
  const qc = useQueryClient();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<CustomAIModel | null>(null);
  const [showKey, setShowKey] = useState(false);

  // Form states
  const [name, setName] = useState("");
  const [provider, setProvider] = useState("openai");
  const [modelName, setModelName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [apiBase, setApiBase] = useState("");
  const [isActive, setIsActive] = useState(true);

  // Testing states
  const [testingModelId, setTestingModelId] = useState<string | null>(null);

  // Modal accessibility Escape-key handler
  useEffect(() => {
    if (!isModalOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        closeModal();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isModalOpen]);

  // Fetch AI models
  const { data: models = [], isLoading } = useQuery({
    queryKey: ["custom-ai-models"],
    queryFn: () => api.getAIModels(),
  });

  // Create Model Mutation
  const createMutation = useMutation({
    mutationFn: (data: { name: string; provider: string; model_name: string; api_key: string; api_base?: string; is_active: boolean }) =>
      api.createAIModel(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custom-ai-models"] });
      toast.success("تم إضافة نموذج الذكاء الاصطناعي بنجاح!");
      closeModal();
    },
    onError: (err: any) => {
      toast.error(err.message || "فشل إضافة النموذج");
    },
  });

  // Update Model Mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CustomAIModel> & { api_key?: string } }) =>
      api.updateAIModel(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custom-ai-models"] });
      toast.success("تم تحديث النموذج بنجاح!");
      closeModal();
    },
    onError: (err: any) => {
      toast.error(err.message || "فشل تحديث النموذج");
    },
  });

  // Delete Model Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteAIModel(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custom-ai-models"] });
      toast.success("تم حذف النموذج بنجاح");
    },
    onError: (err: any) => {
      toast.error(err.message || "فشل حذف النموذج");
    },
  });

  // Toggle Active State Mutation
  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      api.updateAIModel(id, { is_active }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custom-ai-models"] });
      toast.success("تم تغيير حالة تفعيل النموذج!");
    },
    onError: (err: any) => {
      toast.error(err.message || "فشل تفعيل النموذج");
    },
  });

  // Test Connection Mutation
  const testConnectionMutation = useMutation({
    mutationFn: (id: string) => {
      setTestingModelId(id);
      return api.testAIModel(id);
    },
    onSuccess: (res) => {
      setTestingModelId(null);
      if (res.status === "success") {
        toast.success(`تم الاتصال بالنموذج بنجاح! استجابة التجربة: "${res.response}"`);
      } else {
        toast.error(`فشل الاتصال: ${res.message || "خطأ غير معروف"}`);
      }
    },
    onError: (err: any) => {
      setTestingModelId(null);
      toast.error(err.message || "فشل اختبار الاتصال بالنموذج");
    },
  });

  const openAddModal = () => {
    setEditingModel(null);
    setName("");
    setProvider("openai");
    setModelName("");
    setApiKey("");
    setApiBase("");
    setIsActive(true);
    setShowKey(false);
    setIsModalOpen(true);
  };

  const openEditModal = (model: CustomAIModel) => {
    setEditingModel(model);
    setName(model.name);
    setProvider(model.provider);
    setModelName(model.model_name);
    setApiKey(""); // Keep API key empty to prevent overwrite unless updated
    setApiBase(model.api_base || "");
    setIsActive(model.is_active);
    setShowKey(false);
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingModel(null);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !modelName.trim()) {
      toast.error("يرجى ملء جميع الحقول المطلوبة");
      return;
    }

    if (editingModel) {
      const data: Partial<CustomAIModel> & { api_key?: string } = {
        name,
        provider,
        model_name: modelName,
        api_base: apiBase.trim() || undefined,
        is_active: isActive,
      };
      if (apiKey.trim()) {
        data.api_key = apiKey;
      }
      updateMutation.mutate({ id: editingModel.id, data });
    } else {
      if (!apiKey.trim()) {
        toast.error("يرجى إدخال مفتاح API الخاص بالنموذج");
        return;
      }
      createMutation.mutate({
        name,
        provider,
        model_name: modelName,
        api_key: apiKey,
        api_base: apiBase.trim() || undefined,
        is_active: isActive,
      });
    }
  };

  const activeModel = models.find((m) => m.is_active);

  return (
    <div className="space-y-6">
      {/* Dynamic Model Header Banner */}
      <div className="relative overflow-hidden rounded-2xl border border-indigo-500/20 bg-gradient-to-r from-indigo-500/10 via-purple-500/10 to-pink-500/10 p-6 text-foreground shadow-2xl backdrop-blur-md">
        <div className="absolute right-0 top-0 -mr-6 -mt-6 h-24 w-24 rounded-full bg-indigo-500/10 blur-xl" />
        <div className="absolute left-0 bottom-0 -ml-6 -mb-6 h-24 w-24 rounded-full bg-purple-500/10 blur-xl" />
        
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-500/20 text-indigo-300">
              <Bot className="h-6 w-6 animate-pulse" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-foreground">النماذج المخصصة للذكاء الاصطناعي</h2>
              <p className="text-sm text-muted-foreground mt-1">أضف نماذج الذكاء الاصطناعي الخاصة بك من OpenAI أو Anthropic أو غيرها وتحكّم بها بالكامل.</p>
            </div>
          </div>
          <button
            onClick={openAddModal}
            className="flex items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-600/30 transition-all hover:bg-indigo-500 hover:shadow-indigo-500/40 active:scale-95"
          >
            <Plus className="h-4 w-4" />
            إضافة نموذج جديد
          </button>
        </div>

        {activeModel ? (
          <div className="mt-5 flex items-center gap-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-3.5 text-emerald-200">
            <Check className="h-5 w-5 text-emerald-400 flex-shrink-0" />
            <div className="text-sm">
              النموذج المفعّل حالياً هو <strong className="text-emerald-300 font-semibold">{activeModel.name}</strong> ({activeModel.model_name}).
              جميع تعليقات العملاء والردود الذكية تدار وتتم عبره الآن.
            </div>
          </div>
        ) : (
          <div className="mt-5 flex items-center gap-3 rounded-lg bg-amber-500/10 border border-amber-500/20 p-3.5 text-amber-200">
            <AlertCircle className="h-5 w-5 text-amber-400 flex-shrink-0" />
            <div className="text-sm">
              لم يتم تفعيل أي نموذج مخصص حالياً. سيقوم النظام باستخدام النموذج الافتراضي من ملف الإعدادات البيئية للمخدم.
            </div>
          </div>
        )}
      </div>

      {/* Grid List of Custom AI Models */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center h-64 space-y-4">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent" />
          <p className="text-sm text-muted-foreground">جاري تحميل النماذج الذكية...</p>
        </div>
      ) : models.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/40 p-12 text-center backdrop-blur-sm">
          <Sparkles className="h-12 w-12 text-muted-foreground/60 mb-4" />
          <h3 className="text-lg font-semibold text-foreground">لا توجد نماذج مخصصة</h3>
          <p className="text-sm text-muted-foreground mt-2 max-w-sm">
            يمكنك إدخال مفتاح API الخاص بك لـ OpenAI أو Anthropic هنا لإعطاء لوحة التحكم القوة الكاملة مباشرة من المتصفح.
          </p>
          <button
            onClick={openAddModal}
            className="mt-6 flex items-center gap-2 rounded-xl bg-indigo-600/10 border border-indigo-500/20 px-4 py-2 text-sm font-semibold text-indigo-300 transition-all hover:bg-indigo-600 hover:text-white"
          >
            <Plus className="h-4 w-4" />
            أضف نموذجك الأول الآن
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 lg:grid-cols-3">
          {models.map((model) => (
            <div
              key={model.id}
              className={cn(
                "group relative overflow-hidden rounded-2xl border bg-card/60 p-5 shadow-lg backdrop-blur-md transition-all duration-300 hover:-translate-y-1 hover:shadow-xl",
                model.is_active
                  ? "border-emerald-500/30 ring-1 ring-emerald-500/20"
                  : "border-border/80 hover:border-border-hover"
              )}
            >
              {/* Active Badge glowing background */}
              {model.is_active && (
                <div className="absolute left-0 top-0 -ml-16 -mt-16 h-32 w-32 rounded-full bg-emerald-500/5 blur-xl" />
              )}

              <div className="flex items-start justify-between gap-2">
                <div className="space-y-1">
                  <h3 className="font-bold text-foreground group-hover:text-primary transition-colors">{model.name}</h3>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="inline-flex items-center rounded-md bg-indigo-500/10 px-2 py-0.5 text-xs font-medium text-indigo-300 border border-indigo-500/20">
                      {model.provider.toUpperCase()}
                    </span>
                    <span className="text-xs text-muted-foreground font-mono">{model.model_name}</span>
                  </div>
                </div>

                {/* Switch for instant activation */}
                <button
                  onClick={() => toggleActiveMutation.mutate({ id: model.id, is_active: !model.is_active })}
                  disabled={toggleActiveMutation.isPending}
                  role="switch"
                  aria-checked={model.is_active}
                  aria-label={`تفعيل نموذج ${model.name}`}
                  className={cn(
                    "relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50",
                    model.is_active ? "bg-emerald-600" : "bg-muted"
                  )}
                >
                  <span
                    className={cn(
                      "inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200",
                      model.is_active ? "translate-x-6" : "translate-x-1"
                    )}
                  />
                </button>
              </div>

              {/* API details */}
              <div className="mt-6 space-y-2 border-t border-border/60 pt-4 text-xs">
                <div className="flex items-center justify-between text-muted-foreground">
                  <span className="flex items-center gap-1"><Key className="h-3.5 w-3.5" /> مفتاح API:</span>
                  <span className="font-mono bg-background/50 px-2 py-0.5 rounded border border-border text-foreground">
                    {model.api_key_masked}
                  </span>
                </div>
                {model.api_base && (
                  <div className="flex items-center justify-between text-muted-foreground">
                    <span className="flex items-center gap-1"><Globe className="h-3.5 w-3.5" /> عنوان API:</span>
                    <span className="font-mono bg-background/50 px-2 py-0.5 rounded border border-border text-foreground truncate max-w-[150px]" title={model.api_base}>
                      {model.api_base}
                    </span>
                  </div>
                )}
              </div>

              {/* Actions footer */}
              <div className="mt-5 flex items-center justify-end gap-2 border-t border-border/40 pt-3">
                <button
                  onClick={() => testConnectionMutation.mutate(model.id)}
                  disabled={testConnectionMutation.isPending && testingModelId === model.id}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-indigo-500/20 bg-background px-3 py-2 text-xs font-semibold text-indigo-400 transition-all hover:bg-indigo-500/10 hover:text-indigo-300 disabled:opacity-50 min-h-[38px]"
                >
                  {testConnectionMutation.isPending && testingModelId === model.id ? (
                    <div className="h-3 w-3 animate-spin rounded-full border border-indigo-400 border-t-transparent" />
                  ) : (
                    <Activity className="h-3.5 w-3.5" />
                  )}
                  اختبار الاتصال
                </button>
                <button
                  onClick={() => openEditModal(model)}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-muted-foreground transition-all hover:bg-accent hover:text-foreground min-h-[38px]"
                >
                  <Edit2 className="h-3.5 w-3.5" />
                  تعديل
                </button>
                <button
                  onClick={() => {
                    if (confirm("هل أنت متأكد من رغبتك في حذف هذا النموذج نهائياً؟")) {
                      deleteMutation.mutate(model.id);
                    }
                  }}
                  disabled={deleteMutation.isPending}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-red-500/20 bg-background px-3 py-2 text-xs font-semibold text-red-400 transition-all hover:bg-red-500/10 hover:text-red-300 min-h-[38px]"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  حذف
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add / Edit Glassmorphism Modal Overlay */}
      {isModalOpen && (
        <div 
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200"
          onClick={closeModal}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
            className="w-full max-w-lg overflow-hidden rounded-2xl border border-border/80 bg-card/90 shadow-2xl backdrop-blur-md animate-in zoom-in-95 duration-200"
            dir="rtl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="border-b border-border bg-muted/30 p-5">
              <h3 id="modal-title" className="text-lg font-bold text-foreground flex items-center gap-2">
                <Bot className="h-5 w-5 text-indigo-400" />
                {editingModel ? "تعديل نموذج ذكاء اصطناعي" : "إضافة نموذج ذكاء اصطناعي جديد"}
              </h3>
              <p className="text-xs text-muted-foreground mt-1">
                تكوين معلمات الاتصال بالكامل لحسابك السحابي. سيتم تشفير المفاتيح وتأمينها تلقائياً.
              </p>
            </div>

            <form onSubmit={handleSubmit} className="p-5 space-y-4">
              {/* Custom Model Name */}
              <div className="space-y-1.5">
                <label htmlFor="model-custom-name" className="text-sm font-semibold text-foreground">اسم النموذج المخصص (للوحة التحكم)</label>
                <input
                  id="model-custom-name"
                  type="text"
                  required
                  placeholder="مثال: نموذج العملاء، نموذج الدعم الفني..."
                  className="w-full rounded-xl border border-border bg-background px-4 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>

              {/* Provider dropdown & Model name */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <label htmlFor="model-provider" className="text-sm font-semibold text-foreground">مزود الخدمة</label>
                  <select
                    id="model-provider"
                    className="w-full rounded-xl border border-border bg-background px-4 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                    value={provider}
                    onChange={(e) => setProvider(e.target.value)}
                  >
                    <option value="openai">OpenAI</option>
                    <option value="anthropic">Anthropic (Claude)</option>
                    <option value="zhipuai">ZhipuAI (GLM)</option>
                    <option value="litellm">LiteLLM Proxy</option>
                    <option value="custom">عنوان مخصص (OpenAI-compatible)</option>
                  </select>
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="model-technical-name" className="text-sm font-semibold text-foreground">اسم المعرف التقني للنموذج</label>
                  <input
                    id="model-technical-name"
                    type="text"
                    required
                    placeholder="مثال: gpt-4o-mini, claude-3-5-sonnet"
                    className="w-full rounded-xl border border-border bg-background px-4 py-2 text-sm font-mono text-foreground focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                    value={modelName}
                    onChange={(e) => setModelName(e.target.value)}
                  />
                </div>
              </div>

              {/* API Key */}
              <div className="space-y-1.5">
                <label htmlFor="model-api-key" className="text-sm font-semibold text-foreground">
                  مفتاح API الخاص بك {editingModel && <span className="text-xs text-amber-400">(اتركه فارغاً لعدم التعديل)</span>}
                </label>
                <div className="relative">
                  <input
                    id="model-api-key"
                    type={showKey ? "text" : "password"}
                    placeholder={editingModel ? "مخفي ومؤمن تلقائياً..." : "sk-..."}
                    required={!editingModel}
                    className="w-full rounded-xl border border-border bg-background pl-12 pr-4 py-2 text-sm font-mono text-foreground focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                  />
                  <button
                    type="button"
                    onClick={() => setShowKey(!showKey)}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </div>

              {/* API Base URL */}
              <div className="space-y-1.5">
                <label htmlFor="model-api-base" className="text-sm font-semibold text-foreground flex items-center gap-1">
                  عنوان API الأساسي <span className="text-xs text-muted-foreground font-normal">(اختياري للبروكسي أو المخدم المخصص)</span>
                </label>
                <input
                  id="model-api-base"
                  type="url"
                  placeholder="https://api.openai.com/v1"
                  className="w-full rounded-xl border border-border bg-background px-4 py-2 text-sm font-mono text-foreground focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                  value={apiBase}
                  onChange={(e) => setApiBase(e.target.value)}
                />
              </div>

              {/* Toggle to activate right away */}
              <div className="flex items-center justify-between rounded-xl bg-muted/40 p-4 border border-border/60">
                <div>
                  <h4 className="text-sm font-semibold text-foreground">تفعيل هذا النموذج فوراً</h4>
                  <p className="text-xs text-muted-foreground mt-0.5">سيتم تبديل الردود إليه فوراً وإلغاء تفعيل أي نموذج آخر.</p>
                </div>
                <button
                  type="button"
                  onClick={() => setIsActive(!isActive)}
                  role="switch"
                  aria-checked={isActive}
                  aria-label="تفعيل هذا النموذج فوراً"
                  className={cn(
                    "relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/50",
                    isActive ? "bg-indigo-600" : "bg-muted"
                  )}
                >
                  <span
                    className={cn(
                      "inline-block h-4 w-4 transform rounded-full bg-white transition-transform duration-200",
                      isActive ? "translate-x-6" : "translate-x-1"
                    )}
                  />
                </button>
              </div>

              {/* Actions buttons */}
              <div className="flex justify-end gap-2 border-t border-border/60 pt-4 mt-6">
                <button
                  type="button"
                  onClick={closeModal}
                  className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold text-muted-foreground transition-all hover:bg-accent hover:text-foreground"
                >
                  إلغاء
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="flex items-center gap-2 rounded-xl bg-indigo-600 px-6 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-600/20 transition-all hover:bg-indigo-500 hover:shadow-indigo-500/30"
                >
                  {(createMutation.isPending || updateMutation.isPending) && (
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  )}
                  {editingModel ? "حفظ التغييرات" : "إضافة النموذج وتفعيله"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
