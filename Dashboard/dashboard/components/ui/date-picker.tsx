"use client";

import { useState, useRef, useEffect } from "react";
import { Calendar as CalendarIcon, ChevronLeft, ChevronRight, X } from "lucide-react";
import { cn } from "@/lib/utils";

interface DatePickerProps {
  value?: string;
  onChange: (date: string | null) => void;
  placeholder?: string;
  className?: string;
}

export function DatePicker({ value, onChange, placeholder = "اختر التاريخ...", className }: DatePickerProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [currentDate, setCurrentDate] = useState<Date>(() => (value ? new Date(value) : new Date()));
  const containerRef = useRef<HTMLDivElement>(null);

  // Close when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const selectedDate = value ? new Date(value) : null;

  // Months in Arabic
  const arabicMonths = [
    "يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو",
    "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"
  ];

  const daysOfWeek = ["أح", "إث", "ثلا", "أر", "خم", "جم", "سب"];

  // Helper calculations for calendar grid
  const getDaysInMonth = (year: number, month: number) => new Date(year, month + 1, 0).getDate();
  const getFirstDayOfMonth = (year: number, month: number) => new Date(year, month, 1).getDay();

  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();

  const daysInMonth = getDaysInMonth(year, month);
  const firstDayIndex = getFirstDayOfMonth(year, month);

  // Previous month padding
  const prevMonthIndex = month === 0 ? 11 : month - 1;
  const prevYear = month === 0 ? year - 1 : year;
  const daysInPrevMonth = getDaysInMonth(prevYear, prevMonthIndex);

  const prevMonthDays = Array.from(
    { length: firstDayIndex },
    (_, i) => daysInPrevMonth - firstDayIndex + 1 + i
  );

  // Current month days
  const currentMonthDays = Array.from({ length: daysInMonth }, (_, i) => i + 1);

  // Next month padding (to complete a grid of 42 days)
  const totalDaysDisplayed = prevMonthDays.length + currentMonthDays.length;
  const nextMonthDaysCount = totalDaysDisplayed > 35 ? 42 - totalDaysDisplayed : 35 - totalDaysDisplayed;
  const nextMonthDays = Array.from({ length: nextMonthDaysCount }, (_, i) => i + 1);

  const handlePrevMonth = () => {
    setCurrentDate(new Date(year, month - 1, 1));
  };

  const handleNextMonth = () => {
    setCurrentDate(new Date(year, month + 1, 1));
  };

  const handleDateSelect = (day: number, isCurrentMonth: "prev" | "current" | "next") => {
    let targetYear = year;
    let targetMonth = month;

    if (isCurrentMonth === "prev") {
      targetMonth = month === 0 ? 11 : month - 1;
      targetYear = month === 0 ? year - 1 : year;
    } else if (isCurrentMonth === "next") {
      targetMonth = month === 11 ? 0 : month + 1;
      targetYear = month === 11 ? year + 1 : year;
    }

    // Set time to end of day (23:59:59) so it is valid until the end of that day
    const selected = new Date(targetYear, targetMonth, day, 23, 59, 59);
    onChange(selected.toISOString());
    setIsOpen(false);
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(null);
  };

  const formatDateDisplay = (date: Date) => {
    const d = date.getDate();
    const m = arabicMonths[date.getMonth()];
    const y = date.getFullYear();
    return `${d} ${m} ${y}`;
  };

  return (
    <div className="relative inline-block w-full" ref={containerRef} dir="rtl">
      {/* Input Trigger Button */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "flex w-full items-center justify-between rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground transition-all focus:outline-none focus:ring-2 focus:ring-primary/50 text-right min-h-[38px] hover:border-border-hover",
          className
        )}
      >
        <div className="flex items-center gap-2 truncate">
          <CalendarIcon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
          {selectedDate ? (
            <span className="font-semibold text-foreground">{formatDateDisplay(selectedDate)}</span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
        </div>
        {selectedDate && (
          <span
            onClick={handleClear}
            className="rounded p-0.5 hover:bg-accent text-muted-foreground hover:text-foreground"
          >
            <X className="h-3.5 w-3.5" />
          </span>
        )}
      </button>

      {/* Calendar Dropdown Popover */}
      {isOpen && (
        <div className="absolute z-50 mt-1 w-72 rounded-xl border border-border bg-card/95 p-4 shadow-2xl backdrop-blur-md animate-in fade-in zoom-in-95 duration-150 right-0 sm:right-auto sm:left-0">
          {/* Header */}
          <div className="flex items-center justify-between pb-3 border-b border-border/60 mb-3">
            <button
              type="button"
              onClick={handlePrevMonth}
              className="rounded-lg p-1.5 hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
            >
              <ChevronRight className="h-4 w-4" /> {/* Swapped arrows for RTL */}
            </button>
            <span className="text-sm font-bold text-foreground">
              {arabicMonths[month]} {year}
            </span>
            <button
              type="button"
              onClick={handleNextMonth}
              className="rounded-lg p-1.5 hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
            >
              <ChevronLeft className="h-4 w-4" /> {/* Swapped arrows for RTL */}
            </button>
          </div>

          {/* Days of week */}
          <div className="grid grid-cols-7 gap-1 text-center text-xs font-semibold text-muted-foreground mb-2">
            {daysOfWeek.map((day) => (
              <div key={day} className="py-1">
                {day}
              </div>
            ))}
          </div>

          {/* Days Grid */}
          <div className="grid grid-cols-7 gap-1 text-center text-sm">
            {/* Previous Month Days */}
            {prevMonthDays.map((day) => (
              <button
                key={`prev-${day}`}
                type="button"
                onClick={() => handleDateSelect(day, "prev")}
                className="py-1.5 rounded-lg text-muted-foreground/30 hover:bg-accent hover:text-foreground transition-all text-xs"
              >
                {day}
              </button>
            ))}

            {/* Current Month Days */}
            {currentMonthDays.map((day) => {
              const isSelected =
                selectedDate &&
                selectedDate.getDate() === day &&
                selectedDate.getMonth() === month &&
                selectedDate.getFullYear() === year;

              const isToday =
                new Date().getDate() === day &&
                new Date().getMonth() === month &&
                new Date().getFullYear() === year;

              return (
                <button
                  key={`curr-${day}`}
                  type="button"
                  onClick={() => handleDateSelect(day, "current")}
                  className={cn(
                    "py-1.5 rounded-lg transition-all font-medium text-xs relative",
                    isSelected
                      ? "bg-primary text-primary-foreground shadow-md shadow-primary/20"
                      : "hover:bg-accent text-foreground",
                    isToday && !isSelected && "ring-1 ring-primary/50 text-primary font-bold"
                  )}
                >
                  {day}
                </button>
              );
            })}

            {/* Next Month Days */}
            {nextMonthDays.map((day) => (
              <button
                key={`next-${day}`}
                type="button"
                onClick={() => handleDateSelect(day, "next")}
                className="py-1.5 rounded-lg text-muted-foreground/30 hover:bg-accent hover:text-foreground transition-all text-xs"
              >
                {day}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
