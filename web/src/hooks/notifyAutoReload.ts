import type { QueryClient } from "@tanstack/react-query";
import { getForwardingRuntimeStatus } from "../api/settings";

type AddToast = (type: "success" | "error" | "info", message: string) => void;
type Translate = (
  key: string,
  fallback?: string,
  params?: Record<string, string | number | boolean | null | undefined>
) => string;

export async function notifyAutoReloadQueuedIfRunning(
  q: QueryClient,
  addToast: AddToast,
  tr: Translate
) {
  try {
    let running = false;
    const cached = q.getQueryData<any>(["forwarding-runtime-status"]);
    if (typeof cached?.running === "boolean") {
      running = cached.running;
    } else {
      const status = await q.fetchQuery({
        queryKey: ["forwarding-runtime-status"],
        queryFn: getForwardingRuntimeStatus,
        staleTime: 0,
      });
      running = Boolean((status as any)?.running);
    }

    if (!running) return;

    addToast(
      "info",
      tr(
        "toast.runtime.auto_reload_queued",
        "Changes detected. Runtime reload has been queued."
      )
    );
  } catch {
    // Best-effort UX hint only; ignore status fetch failures here.
  }
}
