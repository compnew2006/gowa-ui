"use client";

import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, type Page } from "@/lib/api";

interface PageContextValue {
  pages: Page[];
  selectedPage: Page | null;
  selectedPageId: string | null;
  setSelectedPageId: (id: string | null) => void;
  isLoading: boolean;
}

const PageContext = createContext<PageContextValue>({
  pages: [],
  selectedPage: null,
  selectedPageId: null,
  setSelectedPageId: () => {},
  isLoading: false,
});

const STORAGE_KEY = "selectedPageId";

export function PageProvider({ children }: { children: ReactNode }) {
  const [selectedPageId, setSelectedPageIdState] = useState<string | null>(null);

  const { data: pages = [], isLoading } = useQuery({
    queryKey: ["pages"],
    queryFn: () => api.getPages(),
    staleTime: 60_000,
  });

  // Restore from localStorage once pages are loaded
  useEffect(() => {
    if (pages.length === 0) return;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && pages.find((p) => p.id === stored)) {
      setSelectedPageIdState(stored);
    } else {
      // Default to first active page
      const first = pages.find((p) => p.is_active) ?? pages[0];
      if (first) {
        setSelectedPageIdState(first.id);
        localStorage.setItem(STORAGE_KEY, first.id);
      }
    }
  }, [pages]);

  const setSelectedPageId = (id: string | null) => {
    setSelectedPageIdState(id);
    if (id) localStorage.setItem(STORAGE_KEY, id);
    else localStorage.removeItem(STORAGE_KEY);
  };

  const selectedPage = pages.find((p) => p.id === selectedPageId) ?? null;

  return (
    <PageContext.Provider value={{ pages, selectedPage, selectedPageId, setSelectedPageId, isLoading }}>
      {children}
    </PageContext.Provider>
  );
}

export function usePageContext() {
  return useContext(PageContext);
}
