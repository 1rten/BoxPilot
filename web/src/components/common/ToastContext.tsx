import type { ReactNode } from "react";
import { createContext, useContext, useState, useCallback } from "react";

type ToastType = "success" | "error";

interface Toast {
  id: number;
  type: ToastType;
  message: string;
  createdAt: number;
}

interface ToastContextValue {
  toasts: Toast[];
  addToast: (type: ToastType, message: string) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);
const TOAST_TTL_MS = 3000;
const TOAST_DUP_WINDOW_MS = 1200;
const TOAST_MAX_STACK = 4;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((type: ToastType, message: string) => {
    setToasts((prev) => {
      const now = Date.now();
      const duplicated = prev.some(
        (item) =>
          item.type === type &&
          item.message === message &&
          now - item.createdAt <= TOAST_DUP_WINDOW_MS
      );
      if (duplicated) {
        return prev;
      }

      const id = (prev[prev.length - 1]?.id ?? 0) + 1;
      const next = [...prev, { id, type, message, createdAt: now }].slice(-TOAST_MAX_STACK);
      // 自动移除
      setTimeout(() => {
        setToasts((current) => current.filter((t) => t.id !== id));
      }, TOAST_TTL_MS);
      return next;
    });
  }, []);

  return (
    <ToastContext.Provider value={{ toasts, addToast }}>
      {children}
      <ToastList />
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}

function ToastList() {
  const ctx = useContext(ToastContext);
  if (!ctx || ctx.toasts.length === 0) return null;

  return (
    <div
      style={{
        position: "fixed",
        bottom: "max(16px, env(safe-area-inset-bottom))",
        right: 16,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        zIndex: 110,
        maxHeight: "40vh",
        overflowY: "auto",
        pointerEvents: "none",
      }}
    >
      {ctx.toasts.map((t) => (
        <div
          key={t.id}
          className="bp-card"
          style={{
            minWidth: 240,
            maxWidth: "min(420px, calc(100vw - 24px))",
            borderLeft:
              t.type === "success" ? "4px solid #16A34A" : "4px solid #DC2626",
            pointerEvents: "auto",
          }}
        >
          <p
            style={{
              margin: 0,
              fontSize: 14,
              color: t.type === "success" ? "#166534" : "#B91C1C",
            }}
          >
            {t.message}
          </p>
        </div>
      ))}
    </div>
  );
}
