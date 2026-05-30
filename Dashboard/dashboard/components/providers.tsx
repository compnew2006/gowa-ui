"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { Toaster } from "sonner";
import { ThemeProvider, useTheme } from "@/lib/theme-context";
import { LanguageProvider } from "@/lib/language-context";

function ToasterWithTheme() {
  const { theme } = useTheme();
  return <Toaster position="bottom-right" richColors theme={theme} />;
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 1,
            refetchOnWindowFocus: false,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <LanguageProvider>
          {children}
          <ToasterWithTheme />
        </LanguageProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
