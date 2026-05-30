"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { toast } from "sonner";
import { Lock, Mail, User, Loader2, Sparkles } from "lucide-react";
import Link from "next/link";

const registrationEnabled = process.env.NEXT_PUBLIC_ENABLE_REGISTRATION === "true";

export default function RegisterPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !email || !password) {
      toast.error("يرجى ملء جميع الحقول المطلوبة");
      return;
    }

    setLoading(true);
    try {
      await api.register({ name, email, password });
      toast.success("تم إنشاء حساب المشرف بنجاح! يمكنك الآن تسجيل الدخول 🎉");
      router.push("/login");
    } catch (err: any) {
      toast.error(err.message || "فشلت عملية إنشاء الحساب، يرجى المحاولة لاحقاً");
    } finally {
      setLoading(false);
    }
  };

  if (!registrationEnabled) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-tr from-slate-950 via-slate-900 to-indigo-950 p-6">
        <div className="w-full max-w-md bg-slate-900/40 backdrop-blur-xl border border-white/10 rounded-3xl p-8 shadow-2xl space-y-5 text-center">
          <div className="inline-flex p-3 bg-indigo-500/10 border border-indigo-500/20 rounded-2xl text-indigo-400">
            <Sparkles className="w-8 h-8" />
          </div>
          <div className="space-y-2">
            <h1 className="text-2xl font-extrabold text-white tracking-tight">التسجيل مغلق</h1>
            <p className="text-sm leading-6 text-slate-400">
              إنشاء الحسابات العامة غير متاح في بيئة الإنتاج. اطلب دعوة من مسؤول النظام.
            </p>
          </div>
          <Link href="/login" className="inline-flex justify-center rounded-xl bg-indigo-600 px-5 py-3 text-sm font-semibold text-white hover:bg-indigo-500 transition-colors">
            العودة لتسجيل الدخول
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-tr from-slate-950 via-slate-900 to-indigo-950 p-6 relative overflow-hidden">
      {/* Dynamic Background Blurs */}
      <div className="absolute top-1/4 left-1/4 w-80 h-80 bg-indigo-500/10 rounded-full blur-3xl" />
      <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl" />

      {/* Main Glassmorphic Card */}
      <div className="w-full max-w-md bg-slate-900/40 backdrop-blur-xl border border-white/10 rounded-3xl p-8 shadow-2xl relative z-10 space-y-6">
        <div className="text-center space-y-2">
          <div className="inline-flex p-3 bg-indigo-500/10 border border-indigo-500/20 rounded-2xl mb-2 text-indigo-400">
            <Sparkles className="w-8 h-8" />
          </div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-white via-slate-100 to-indigo-300">
            إنشاء حساب جديد
          </h1>
          <p className="text-sm text-slate-400">قم بتهيئة حساب مشرف جديد لإدارة منصتك الذكية</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Name field */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-slate-300 block">الاسم الكامل</label>
            <div className="relative">
              <User className="absolute right-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
              <input
                type="text"
                required
                placeholder="أحمد علي"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-slate-950/40 border border-white/5 focus:border-indigo-500/50 rounded-xl py-3 pr-11 pl-4 text-sm text-white placeholder-slate-600 focus:ring-2 focus:ring-indigo-500/20 outline-none transition-all"
              />
            </div>
          </div>

          {/* Email field */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-slate-300 block">البريد الإلكتروني</label>
            <div className="relative">
              <Mail className="absolute right-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
              <input
                type="email"
                required
                dir="ltr"
                placeholder="email@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-slate-950/40 border border-white/5 focus:border-indigo-500/50 rounded-xl py-3 pr-11 pl-4 text-sm text-white placeholder-slate-600 focus:ring-2 focus:ring-indigo-500/20 outline-none transition-all"
              />
            </div>
          </div>

          {/* Password field */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-slate-300 block">كلمة المرور</label>
            <div className="relative">
              <Lock className="absolute right-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
              <input
                type="password"
                required
                dir="ltr"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-950/40 border border-white/5 focus:border-indigo-500/50 rounded-xl py-3 pr-11 pl-4 text-sm text-white placeholder-slate-600 focus:ring-2 focus:ring-indigo-500/20 outline-none transition-all"
              />
            </div>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-semibold py-3.5 rounded-xl transition-all shadow-lg shadow-indigo-600/35 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {loading ? (
              <>
                <Loader2 className="w-5 h-5 animate-spin" />
                جاري إنشاء الحساب...
              </>
            ) : (
              "إنشاء الحساب"
            )}
          </button>
        </form>

        {/* Footer Link */}
        <div className="text-center pt-2">
          <p className="text-sm text-slate-400">
            لديك حساب بالفعل؟{" "}
            <Link href="/login" className="text-indigo-400 hover:text-indigo-300 font-semibold transition-colors">
              تسجيل الدخول
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
