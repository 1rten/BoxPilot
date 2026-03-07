import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type {
  RuntimeConnectionsData,
  RuntimeGroupSelectData,
  RuntimeGroupSummaryData,
  RuntimeLogsData,
  RuntimeProxyCheckData,
  RuntimeStatusData,
  RuntimeTrafficData,
} from "../api/types";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

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
  const { tr } = useI18n();
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
      addToast("success", tr("toast.runtime.reloaded", "Runtime reloaded"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.runtime.reload_failed", "Runtime reload failed: {message}", { message }));
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

export function useRuntimeProxyCheck() {
  const { tr } = useI18n();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body?: { target_url?: string; timeout_ms?: number }) => {
      const { data } = await api.post<{ data: RuntimeProxyCheckData }>("/runtime/proxy/check", body ?? {});
      return data.data;
    },
    onSuccess: () => {
      addToast("success", tr("toast.runtime.proxy_check_ok", "Proxy chain check completed"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.runtime.proxy_check_failed", "Proxy chain check failed: {message}", { message }));
    },
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

export function useRuntimeGroups() {
  return useQuery({
    queryKey: ["runtime-groups"],
    queryFn: async () => {
      const { data } = await api.get<{ data: RuntimeGroupSummaryData }>("/runtime/groups");
      return data.data;
    },
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 10_000,
    refetchIntervalInBackground: true,
  });
}

export function useSelectRuntimeGroup() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body: { group_tag: string; selected_outbound: string }) => {
      const { data } = await api.post<{ data: RuntimeGroupSelectData }>(
        `/runtime/groups/${encodeURIComponent(body.group_tag)}/select`,
        { selected_outbound: body.selected_outbound }
      );
      return data.data;
    },
    onSuccess: (data) => {
      q.invalidateQueries({ queryKey: ["runtime-groups"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      const isAutoChoice = data.selected_is_auto;
      if (isAutoChoice) {
        if (data.auto_probe_error) {
          if (isSoftAutoProbeError(data.auto_probe_error)) {
            addToast(
              "success",
              tr(
                "toast.runtime.group_selected_auto_probe_deferred",
                "Auto mode enabled. Runtime probe is temporarily unavailable; periodic checks will continue."
              )
            );
            return;
          }
          addToast(
            "info",
            tr(
              "toast.runtime.group_selected_auto_probe_failed",
              "Auto mode enabled, but immediate probe failed ({message}). Runtime will continue periodic checks.",
              { message: data.auto_probe_error }
            )
          );
          return;
        }
        if (data.runtime_effective_outbound && data.runtime_effective_outbound !== data.selected_outbound) {
          addToast(
            "success",
            tr("toast.runtime.group_selected_auto", "Auto tested candidates. Current best node: {outbound}", {
              outbound: data.runtime_effective_outbound,
            })
          );
          return;
        }
        addToast(
          "success",
          tr(
            "toast.runtime.group_selected_auto_pending",
            "Auto mode enabled. Candidate test was triggered; best node will update shortly."
          )
        );
        return;
      }
      addToast("success", tr("toast.runtime.group_selected", "Routing group selection applied"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.runtime.group_select_failed", "Failed to update routing group: {message}", { message }));
    },
  });
}

function isSoftAutoProbeError(message?: string): boolean {
  const lower = (message || "").toLowerCase();
  if (!lower) return false;
  return (
    lower.includes("clash api unavailable") ||
    lower.includes("connection refused") ||
    lower.includes("context deadline exceeded") ||
    lower.includes("i/o timeout")
  );
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
