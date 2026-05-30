"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Menu, X } from "lucide-react";
import { Sidebar } from "@/components/layout/sidebar";
import { api } from "@/lib/api";
import { clearSession, setSession } from "@/lib/session";
import { useLanguage } from "@/lib/language-context";
import { cn } from "@/lib/utils";

export function AppShell({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);
  const { dir } = useLanguage();

  const isRTL = dir === "rtl";
  const isPublicRoute = pathname === "/login" || pathname === "/register";

  useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  useEffect(() => {
    let cancelled = false;
    if (isPublicRoute) {
      setIsAuthenticated(true);
      return;
    }

    api.getMe()
      .then((user) => {
        if (cancelled) return;
        setSession(user?.user ?? user);
        setIsAuthenticated(true);
      })
      .catch(() => {
        if (cancelled) return;
        clearSession();
        setIsAuthenticated(false);
        router.push("/login");
      });

    return () => {
      cancelled = true;
    };
  }, [pathname, isPublicRoute, router]);

  // Enforce blank layout for login & register pages
  if (isPublicRoute) {
    return <>{children}</>;
  }

  // Show a gorgeous, premium Arabic glassmorphic loading spinner while checking credentials
  if (isAuthenticated === null) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-[#030014] text-foreground">
        <div className="flex flex-col items-center gap-4">
          <div className="relative flex h-16 w-16 items-center justify-center">
            <div className="absolute h-full w-full animate-spin rounded-full border-4 border-violet-500/20 border-t-violet-500" />
            <div className="absolute h-10 w-10 animate-ping rounded-full border-2 border-indigo-500/10" />
          </div>
          <p className="text-sm text-indigo-200/60 animate-pulse font-medium">جاري التحقق من الهوية...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Desktop sidebar - RTL: right side | LTR: left side */}
      <div className={cn(
        "hidden lg:fixed lg:inset-y-0 lg:z-40 lg:block",
        isRTL ? "lg:right-0" : "lg:left-0"
      )}>
        <Sidebar />
      </div>

      {/* Mobile header */}
      <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-border bg-card/95 px-4 backdrop-blur lg:hidden">
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="rounded-lg border border-border p-2 text-muted-foreground hover:bg-accent hover:text-foreground"
          aria-label="فتح القائمة"
        >
          <Menu className="h-5 w-5" />
        </button>
        <div className="text-sm font-semibold text-foreground">AI Automation</div>
      </header>

      {/* Mobile drawer */}
      {open && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button
            type="button"
            className="absolute inset-0 bg-background/80 backdrop-blur-sm"
            aria-label="إغلاق القائمة"
            onClick={() => setOpen(false)}
          />
          {/* Drawer slides from right (RTL) or left (LTR) */}
          <div className={cn(
            "absolute inset-y-0 max-w-[85vw]",
            isRTL ? "right-0" : "left-0"
          )}>
            <Sidebar onNavigate={() => setOpen(false)} />
            <button
              type="button"
              onClick={() => setOpen(false)}
              className={cn(
                "absolute top-3 rounded-lg border border-border bg-card p-2 text-muted-foreground hover:bg-accent hover:text-foreground",
                isRTL ? "left-3" : "right-3"
              )}
              aria-label="إغلاق القائمة"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Main content - margin offset based on direction */}
      <main className={cn(
        "min-h-screen",
        isRTL ? "lg:mr-64" : "lg:ml-64"
      )}>
        {children}
      </main>
    </div>
  );
}
