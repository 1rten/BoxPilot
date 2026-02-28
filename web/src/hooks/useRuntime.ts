import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type {
  RuntimeConnectionsData,
  RuntimeLogsData,
  RuntimeStatusData,
  RuntimeTrafficData,
} from "../api/types";
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
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 8_000,
    refetchIntervalInBackground: true,
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
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
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

export function useRuntimeTraffic() {
  return useQuery({
    queryKey: ["runtime-traffic"],
    queryFn: async () => {
      const { data } = await api.get<{ data: RuntimeTrafficData }>("/runtime/traffic");
      return data.data;
    },
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 4000,
    refetchIntervalInBackground: true,
  });
}

export function useRuntimeConnections(query?: string) {
  return useQuery({
    queryKey: ["runtime-connections", query ?? ""],
    queryFn: async () => {
      const { data } = await api.get<{ data: RuntimeConnectionsData }>("/runtime/connections", {
        params: query && query.trim() ? { q: query.trim() } : undefined,
      });
      return data.data;
    },
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 3000,
    refetchIntervalInBackground: true,
  });
}

export function useRuntimeLogs(params?: {
  level?: string;
  q?: string;
  limit?: number;
  enabled?: boolean;
  refetchIntervalMs?: number;
}) {
  const level = (params?.level || "all").trim();
  const q = (params?.q || "").trim();
  const limit = params?.limit ?? 80;
  const enabled = params?.enabled ?? true;
  const refetchIntervalMs = params?.refetchIntervalMs ?? 5000;

  return useQuery({
    queryKey: ["runtime-logs", level, q, limit],
    queryFn: async () => {
      const { data } = await api.get<{ data: RuntimeLogsData }>("/runtime/logs", {
        params: {
          level,
          q: q || undefined,
          limit,
        },
      });
      return data.data;
    },
    enabled,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: refetchIntervalMs,
    refetchIntervalInBackground: true,
  });
}
