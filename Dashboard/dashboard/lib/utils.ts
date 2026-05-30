import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(date: string | Date | null | undefined): string {
  if (!date) return "—";
  return new Date(date).toLocaleDateString("ar-EG", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function formatDateTime(date: string | Date | null | undefined): string {
  if (!date) return "—";
  return new Date(date).toLocaleString("ar-EG", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatRelativeTime(date: string | Date | null | undefined): string {
  if (!date) return "—";
  const d = new Date(date);
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "الآن";
  if (mins < 60) return `منذ ${mins} دقيقة`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `منذ ${hrs} ساعة`;
  const days = Math.floor(hrs / 24);
  return `منذ ${days} يوم`;
}

export function getStatusColor(status: string): string {
  const map: Record<string, string> = {
    pending: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    replied: "bg-green-500/20 text-green-400 border-green-500/30",
    resolved: "bg-blue-500/20 text-blue-400 border-blue-500/30",
    escalated: "bg-red-500/20 text-red-400 border-red-500/30",
    open: "bg-red-500/20 text-red-400 border-red-500/30",
    valid: "bg-green-500/20 text-green-400 border-green-500/30",
    expiring_soon: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    expired: "bg-red-500/20 text-red-400 border-red-500/30",
    error: "bg-red-500/20 text-red-400 border-red-500/30",
  };
  return map[status] ?? "bg-gray-500/20 text-gray-400 border-gray-500/30";
}

export function getPriorityColor(priority: string): string {
  const map: Record<string, string> = {
    critical: "bg-red-600/20 text-red-400 border-red-600/30",
    high: "bg-orange-500/20 text-orange-400 border-orange-500/30",
    medium: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    low: "bg-green-500/20 text-green-400 border-green-500/30",
  };
  return map[priority] ?? "bg-gray-500/20 text-gray-400 border-gray-500/30";
}
