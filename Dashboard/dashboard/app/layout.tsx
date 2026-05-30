import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Providers } from "@/components/providers";
import { PageProvider } from "@/lib/page-context";
import { AppShell } from "@/components/layout/app-shell";

const inter = Inter({ subsets: ["latin"] });

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "AI Automation Dashboard",
  description: "AI-powered Facebook & Instagram Comment Automation",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ar" dir="rtl" className="dark">
      <head>
        {/* Inline script to apply saved theme BEFORE first paint to avoid flash */}
        <script
          dangerouslySetInnerHTML={{
            __html: `
              try {
                var theme = localStorage.getItem('theme') || 'dark';
                var lang = localStorage.getItem('language') || 'ar';
                document.documentElement.className = theme;
                document.documentElement.setAttribute('lang', lang);
                document.documentElement.setAttribute('dir', lang === 'en' ? 'ltr' : 'rtl');
              } catch(e) {}
            `,
          }}
        />
      </head>
      <body className={inter.className}>
        <Providers>
          <PageProvider>
            <AppShell>{children}</AppShell>
          </PageProvider>
        </Providers>
      </body>
    </html>
  );
}
