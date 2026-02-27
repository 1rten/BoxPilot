import type { ReactNode } from "react";
import { createContext, useContext, useState, useCallback } from "react";

type ToastType = "success" | "error";

interface Toast {
  id: number;
  type: ToastType;
  message: string;
}

interface ToastContextValue {
  toasts: Toast[];
  addToast: (type: ToastType, message: string) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((type: ToastType, message: string) => {
    setToasts((prev) => {
      const id = (prev[prev.length - 1]?.id ?? 0) + 1;
      const next = [...prev, { id, type, message }];
      // 自动移除
      setTimeout(() => {
        setToasts((current) => current.filter((t) => t.id !== id));
      }, 3000);
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
        top: 16,
        right: 16,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        zIndex: 100,
      }}
    >
      {ctx.toasts.map((t) => (
        <div
          key={t.id}
          className="bp-card"
          style={{
            minWidth: 240,
            borderLeft:
              t.type === "success" ? "4px solid #16A34A" : "4px solid #DC2626",
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

