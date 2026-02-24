import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { RuntimeStatusData } from "../api/types";
import { useToast } from "../components/common/ToastContext";

export function useRuntimeStatus() {
  return useQuery({
    queryKey: ["runtime-status"],
    queryFn: async () => {
      const { data } = await api.get<{ data: RuntimeStatusData }>(
        "/runtime/status"
      );
      return data.data;
    },
  });
}

export function useRuntimeReload() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async () => {
      const { data } = await api.post("/runtime/reload", {});
      return data;
    },
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      addToast("success", "Runtime reloaded");
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        "Unknown error";
      addToast("error", `Runtime reload failed: ${message}`);
    },
  });
}
