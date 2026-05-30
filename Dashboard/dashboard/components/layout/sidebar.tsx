"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import {
  LayoutDashboard, MessageSquare, AlertTriangle, Users,
  BookOpen, Globe, Key, Settings, Bot, ChevronLeft,
  BarChart3, Users2, Zap, Eye, Plug, ShieldCheck, Megaphone, Clock, Palette,
  Share2, LogOut, Sun, Moon, Languages,
} from "lucide-react";
import { PageSelector } from "@/components/page-selector";
import { clearSession, getStoredUser, type StoredUser } from "@/lib/session";
import { can, type PermissionKey } from "@/lib/permissions";
import { useTheme } from "@/lib/theme-context";
import { useLanguage } from "@/lib/language-context";

export function Sidebar({ onNavigate, className = "" }: { onNavigate?: () => void; className?: string }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<StoredUser | null>(null);
  const [mounted, setMounted] = useState(false);
  const { theme, toggleTheme } = useTheme();
  const { lang, t, toggleLanguage } = useLanguage();

  const { data: profile } = useQuery({
    queryKey: ["agency-profile"],
    queryFn: () => api.getAgencyProfile(),
  });

  useEffect(() => {
    const u = getStoredUser();
    setUser(u);
    setMounted(true);
  }, []);

  const handleLogout = async () => {
    await api.logout().catch(() => undefined);
    clearSession();
    toast.success(lang === "ar" ? "تم تسجيل الخروج بنجاح" : "Logged out successfully");
    router.push("/login");
  };

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);
  const hasPermission = (permission?: PermissionKey) => {
    if (!mounted) return !permission;
    return !permission || user?.role === "admin" || can(permission, user?.permissions);
  };

  const navGroups = [
    {
      label: t.navHome,
      items: [
        { href: "/", label: t.dashboard, icon: LayoutDashboard },
        { href: "/analytics", label: t.analytics, icon: BarChart3 },
      ],
    },
    {
      label: t.navConversations,
      items: [
        { href: "/conversations", label: t.conversations, icon: MessageSquare },
        { href: "/shadow-mode", label: t.shadowMode, icon: Eye },
        { href: "/escalations", label: t.escalations, icon: AlertTriangle },
      ],
    },
    {
      label: t.navCustomers,
      items: [
        { href: "/customers", label: t.customers, icon: Users },
        { href: "/campaigns", label: t.campaigns, icon: Megaphone, permission: "can_manage_campaigns" as PermissionKey },
      ],
    },
    {
      label: t.navAutomation,
      items: [
        { href: "/rules", label: t.automationRules, icon: Zap, permission: "can_manage_settings" as PermissionKey },
        { href: "/knowledge-base", label: t.knowledgeBase, icon: BookOpen, permission: "can_manage_settings" as PermissionKey },
      ],
    },
    {
      label: t.navTeam,
      items: [
        { href: "/team", label: t.team, icon: Users2, permission: "can_manage_team" as PermissionKey },
        { href: "/integrations", label: t.integrations, icon: Plug, permission: "can_manage_settings" as PermissionKey },
        { href: "/compliance", label: t.compliance, icon: ShieldCheck, permission: "can_export" as PermissionKey },
        { href: "/audit-logs", label: t.auditLogs, icon: Clock, permission: "can_export" as PermissionKey },
      ],
    },
    {
      label: t.navSettings,
      items: [
        { href: "/pages", label: t.pages, icon: Globe, permission: "can_manage_settings" as PermissionKey },
        { href: "/posts", label: t.scheduledPosts, icon: Share2, permission: "can_manage_campaigns" as PermissionKey },
        { href: "/tokens", label: t.tokens, icon: Key, permission: "can_manage_settings" as PermissionKey },
        { href: "/settings", label: t.systemSettings, icon: Settings, permission: "can_manage_settings" as PermissionKey },
        { href: "/settings/branding", label: t.branding, icon: Palette, permission: "can_manage_settings" as PermissionKey },
      ],
    },
  ];

  const roleLabel = user?.role === "admin" ? t.admin : user?.role === "reviewer" ? t.reviewer : t.analyst;

  const isRTL = lang === "ar";

  return (
    <aside className={cn(
      "flex h-full w-64 flex-col border-border bg-card",
      isRTL ? "border-l" : "border-r",
      className
    )}>
      {/* Logo */}
      <div className="flex h-16 items-center gap-3 border-b border-border px-6 shrink-0">
        <div
          className="flex h-9 w-9 items-center justify-center rounded-lg text-white font-bold"
          style={{ backgroundColor: profile?.primary_color || "#3b82f6" }}
        >
          {profile?.agency_name ? profile.agency_name[0].toUpperCase() : <Bot className="h-5 w-5" />}
        </div>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-foreground truncate">{profile?.agency_name || "AI Automation"}</p>
          <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{profile?.dashboard_title || "Agency Dashboard"}</p>
        </div>
      </div>

      {/* Page Selector */}
      <div className="border-b border-border py-3">
        <p className="mb-2 px-6 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
          {t.activePage}
        </p>
        <PageSelector />
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto p-3 space-y-4">
        {navGroups.map((group) => {
          const visibleItems = group.items.filter(
            (item) => hasPermission(item.permission as PermissionKey | undefined)
          );
          if (visibleItems.length === 0) return null;

          return (
            <div key={group.label}>
              <p className="mb-1 px-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                {group.label}
              </p>
              <ul className="space-y-0.5">
                {visibleItems.map((item) => {
                  const active = isActive(item.href);
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        onClick={onNavigate}
                        className={cn(
                          "group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all",
                          active
                            ? "bg-primary/15 text-primary"
                            : "text-muted-foreground hover:bg-accent hover:text-foreground"
                        )}
                      >
                        <item.icon className={cn("h-4 w-4 shrink-0", active ? "text-primary" : "")} />
                        <span className="flex-1">{item.label}</span>
                        {active && <ChevronLeft className={cn("h-3 w-3 opacity-60", !isRTL && "rotate-180")} />}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="border-t border-border p-4 shrink-0 space-y-3">
        {/* Theme & Language toggles */}
        <div className="flex items-center gap-2">
          {/* Theme toggle */}
          <button
            onClick={toggleTheme}
            title={theme === "dark" ? (lang === "ar" ? "الوضع النهاري" : "Light Mode") : (lang === "ar" ? "الوضع الليلي" : "Dark Mode")}
            className="flex-1 flex items-center justify-center gap-2 rounded-lg border border-border py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          >
            {theme === "dark" ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
            <span>{theme === "dark" ? (lang === "ar" ? "نهاري" : "Light") : (lang === "ar" ? "ليلي" : "Dark")}</span>
          </button>

          {/* Language toggle */}
          <button
            onClick={toggleLanguage}
            title={lang === "ar" ? "Switch to English" : "التبديل للعربية"}
            className="flex-1 flex items-center justify-center gap-2 rounded-lg border border-border py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          >
            <Languages className="h-3.5 w-3.5" />
            <span>{lang === "ar" ? "English" : "العربية"}</span>
          </button>
        </div>

        {user && (
          <div className="flex items-center justify-between gap-2 px-1">
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-violet-600/20 text-violet-400 font-bold text-xs uppercase shrink-0">
                {user.name ? user.name[0] : "U"}
              </div>
              <div className="min-w-0">
                <p className="text-xs font-semibold text-foreground truncate">{user.name || (lang === "ar" ? "مستخدم" : "User")}</p>
                <p className="text-[10px] text-muted-foreground truncate">{roleLabel}</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="rounded-lg p-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-400 transition-colors"
              title={t.logout}
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        )}
        <div className="flex items-center gap-2 rounded-lg bg-accent/50 px-3 py-2">
          <div className="h-2 w-2 rounded-full bg-green-400 animate-pulse" />
          <span className="text-xs text-muted-foreground">FastAPI + LangGraph v3</span>
        </div>
      </div>
    </aside>
  );
}
