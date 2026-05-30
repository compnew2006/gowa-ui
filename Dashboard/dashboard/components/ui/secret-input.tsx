"use client";

import { useState } from "react";

interface SecretInputProps {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  saved?: boolean;
  className?: string;
}

export function SecretInput({ id, value, onChange, placeholder, saved = false, className = "" }: SecretInputProps) {
  const [editing, setEditing] = useState(!saved);

  if (saved && !editing) {
    return (
      <button
        id={id}
        type="button"
        onClick={() => {
          onChange("");
          setEditing(true);
        }}
        className={className || "rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent"}
      >
        مخفي، اضغط للاستبدال
      </button>
    );
  }

  return (
    <input
      id={id}
      type="password"
      className={className || "rounded-lg border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      autoComplete="new-password"
    />
  );
}
