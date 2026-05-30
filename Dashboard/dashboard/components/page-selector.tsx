"use client";

import { useState, useRef, useEffect } from "react";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { Globe, ChevronDown, CheckCircle, Instagram, Facebook } from "lucide-react";
import { getSafeDisplayUrl } from "@/lib/security";

function PlatformIcon({ platform }: { platform: string }) {
  if (platform === "instagram") return <Instagram className="h-3 w-3 text-pink-400" />;
  return <Facebook className="h-3 w-3 text-blue-400" />;
}

export function PageSelector() {
  const { pages, selectedPage, setSelectedPageId, isLoading } = usePageContext();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 mb-3 rounded-lg border border-border bg-accent/30 px-3 py-2 animate-pulse">
        <div className="h-4 w-24 rounded bg-border" />
      </div>
    );
  }

  if (pages.length === 0) {
    return (
      <div className="mx-3 mb-3 rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted-foreground text-center">
        لا توجد صفحات
      </div>
    );
  }

  const selectedAvatarUrl = getSafeDisplayUrl(selectedPage?.avatar_url);

  return (
    <div ref={ref} className="relative mx-3 mb-3">
      <button
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "w-full flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-all",
          "border-border bg-accent/40 hover:bg-accent/70 hover:border-primary/40",
          open && "border-primary/40 bg-accent/70"
        )}
      >
        {selectedPage ? (
          <>
            {selectedAvatarUrl ? (
              <img src={selectedAvatarUrl} alt="" referrerPolicy="no-referrer" className="h-5 w-5 rounded-full object-cover" />
            ) : (
              <div className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/20">
                <Globe className="h-3 w-3 text-primary" />
              </div>
            )}
            <span className="flex-1 text-right text-foreground font-medium truncate max-w-[110px]">
              {selectedPage.name}
            </span>
            <PlatformIcon platform={selectedPage.platform} />
          </>
        ) : (
          <>
            <Globe className="h-4 w-4 text-muted-foreground" />
            <span className="flex-1 text-right text-muted-foreground">اختر صفحة</span>
          </>
        )}
        <ChevronDown className={cn("h-3 w-3 text-muted-foreground transition-transform", open && "rotate-180")} />
      </button>

      {open && (
        <div className="absolute top-full left-0 right-0 z-50 mt-1 rounded-lg border border-border bg-card shadow-xl overflow-hidden">
          <div className="p-1 max-h-56 overflow-y-auto">
            {pages.map((page) => {
              const avatarUrl = getSafeDisplayUrl(page.avatar_url);
              return (
                <button
                  key={page.id}
                  onClick={() => { setSelectedPageId(page.id); setOpen(false); }}
                  className={cn(
                    "w-full flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors text-right",
                    page.id === selectedPage?.id
                      ? "bg-primary/15 text-primary"
                      : "text-foreground hover:bg-accent"
                  )}
                >
                  {avatarUrl ? (
                    <img src={avatarUrl} alt="" referrerPolicy="no-referrer" className="h-6 w-6 rounded-full object-cover flex-shrink-0" />
                  ) : (
                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/20 flex-shrink-0">
                      <Globe className="h-3 w-3 text-primary" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0 text-right">
                    <p className="font-medium truncate">{page.name}</p>
                    <div className="flex items-center gap-1 justify-end">
                      <PlatformIcon platform={page.platform} />
                      <span className="text-xs text-muted-foreground capitalize">{page.platform}</span>
                      {!page.is_active && (
                        <span className="text-xs text-red-400">(غير نشط)</span>
                      )}
                    </div>
                  </div>
                  {page.id === selectedPage?.id && (
                    <CheckCircle className="h-4 w-4 text-primary flex-shrink-0" />
                  )}
                </button>
              );
            })}
          </div>
          <div className="border-t border-border px-3 py-2">
            <p className="text-[10px] text-muted-foreground/60 text-center">{pages.length} صفحة مسجلة</p>
          </div>
        </div>
      )}
    </div>
  );
}
