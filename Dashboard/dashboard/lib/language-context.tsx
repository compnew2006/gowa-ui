"use client";

import { createContext, useContext, useEffect, useState } from "react";

type Language = "ar" | "en";
type Direction = "rtl" | "ltr";

interface Translations {
  // Sidebar groups
  navHome: string;
  navConversations: string;
  navCustomers: string;
  navAutomation: string;
  navTeam: string;
  navSettings: string;
  // Nav items
  dashboard: string;
  analytics: string;
  conversations: string;
  shadowMode: string;
  escalations: string;
  customers: string;
  campaigns: string;
  automationRules: string;
  knowledgeBase: string;
  team: string;
  integrations: string;
  compliance: string;
  auditLogs: string;
  pages: string;
  scheduledPosts: string;
  tokens: string;
  systemSettings: string;
  branding: string;
  // UI
  activePage: string;
  logout: string;
  admin: string;
  reviewer: string;
  analyst: string;
  loading: string;
  save: string;
  cancel: string;
  delete: string;
  edit: string;
  confirm: string;
  search: string;
  noData: string;
}

const ar: Translations = {
  navHome: "الرئيسية",
  navConversations: "المحادثات",
  navCustomers: "العملاء",
  navAutomation: "الأتمتة",
  navTeam: "الفريق والتكاملات",
  navSettings: "الإعدادات",
  dashboard: "لوحة التحكم",
  analytics: "التحليلات المتقدمة",
  conversations: "المحادثات",
  shadowMode: "وضع الظل",
  escalations: "التصعيدات",
  customers: "العملاء",
  campaigns: "الحملات التسويقية",
  automationRules: "قواعد الأتمتة",
  knowledgeBase: "قاعدة المعرفة",
  team: "الفريق",
  integrations: "التكاملات",
  compliance: "الامتثال والأمان",
  auditLogs: "سجل التدقيق",
  pages: "الصفحات",
  scheduledPosts: "جدولة المنشورات",
  tokens: "الرموز والرموز المميزة",
  systemSettings: "إعدادات النظام",
  branding: "العلامة التجارية والهوية",
  activePage: "الصفحة النشطة",
  logout: "تسجيل الخروج",
  admin: "مدير النظام",
  reviewer: "مراجع",
  analyst: "محلل",
  loading: "جاري التحميل...",
  save: "حفظ",
  cancel: "إلغاء",
  delete: "حذف",
  edit: "تعديل",
  confirm: "تأكيد",
  search: "بحث",
  noData: "لا توجد بيانات",
};

const en: Translations = {
  navHome: "Home",
  navConversations: "Conversations",
  navCustomers: "Customers",
  navAutomation: "Automation",
  navTeam: "Team & Integrations",
  navSettings: "Settings",
  dashboard: "Dashboard",
  analytics: "Advanced Analytics",
  conversations: "Conversations",
  shadowMode: "Shadow Mode",
  escalations: "Escalations",
  customers: "Customers",
  campaigns: "Marketing Campaigns",
  automationRules: "Automation Rules",
  knowledgeBase: "Knowledge Base",
  team: "Team",
  integrations: "Integrations",
  compliance: "Compliance & Security",
  auditLogs: "Audit Logs",
  pages: "Pages",
  scheduledPosts: "Scheduled Posts",
  tokens: "Tokens",
  systemSettings: "System Settings",
  branding: "Branding & Identity",
  activePage: "Active Page",
  logout: "Logout",
  admin: "System Admin",
  reviewer: "Reviewer",
  analyst: "Analyst",
  loading: "Loading...",
  save: "Save",
  cancel: "Cancel",
  delete: "Delete",
  edit: "Edit",
  confirm: "Confirm",
  search: "Search",
  noData: "No data available",
};

interface LanguageContextValue {
  lang: Language;
  dir: Direction;
  t: Translations;
  toggleLanguage: () => void;
}

const LanguageContext = createContext<LanguageContextValue>({
  lang: "ar",
  dir: "rtl",
  t: ar,
  toggleLanguage: () => {},
});

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLang] = useState<Language>("ar");

  useEffect(() => {
    const stored = localStorage.getItem("language") as Language | null;
    if (stored === "ar" || stored === "en") {
      setLang(stored);
      applyLanguage(stored);
    }
  }, []);

  const applyLanguage = (l: Language) => {
    const html = document.documentElement;
    html.setAttribute("lang", l);
    html.setAttribute("dir", l === "ar" ? "rtl" : "ltr");
  };

  const toggleLanguage = () => {
    const next = lang === "ar" ? "en" : "ar";
    setLang(next);
    applyLanguage(next);
    localStorage.setItem("language", next);
  };

  const dir: Direction = lang === "ar" ? "rtl" : "ltr";
  const t = lang === "ar" ? ar : en;

  return (
    <LanguageContext.Provider value={{ lang, dir, t, toggleLanguage }}>
      {children}
    </LanguageContext.Provider>
  );
}

export const useLanguage = () => useContext(LanguageContext);
