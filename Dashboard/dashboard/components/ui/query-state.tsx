import { AlertCircle, Inbox } from "lucide-react";

export function LoadingState({ label = "جاري التحميل..." }: { label?: string }) {
  return (
    <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
      {label}
    </div>
  );
}

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border border-border bg-card text-center text-muted-foreground">
      <Inbox className="h-8 w-8 opacity-40" />
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-xs leading-5">{description}</p>}
    </div>
  );
}

export function ErrorState({ title = "تعذر تحميل البيانات", description }: { title?: string; description?: string }) {
  return (
    <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border border-red-500/30 bg-red-500/10 text-center text-red-300">
      <AlertCircle className="h-8 w-8" />
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-xs leading-5 text-red-200/80">{description}</p>}
    </div>
  );
}
