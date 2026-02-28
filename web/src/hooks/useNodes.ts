import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Node } from "../api/types";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

export function useNodes(params?: { enabled?: number; sub_id?: string }) {
  return useQuery({
    queryKey: ["nodes", params],
    queryFn: async () => {
      const { data } = await api.get<{ data: Node[] }>("/nodes", { params });
      return data.data;
    },
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 15_000,
    refetchIntervalInBackground: true,
  });
}

export function useUpdateNode() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body: { id: string; name?: string; enabled?: boolean; forwarding_enabled?: boolean }) => {
      const { data } = await api.post<{ data: Node }>("/nodes/update", body);
      return data.data;
    },
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.node.updated", "Node updated"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.node.update_failed", "Update node failed: {message}", { message }));
    },
  });
}

export function useTestNodes() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body: { node_ids: string[]; mode?: "http" | "ping" }) => {
      const { data } = await api.post<{ data: Array<{ node_id: string; status: string; latency_ms?: number | null; error?: string | null }> }>(
        "/nodes/test",
        body
      );
      return data.data;
    },
    onSuccess: (rows) => {
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      const ok = rows.filter((r) => r.status === "ok").length;
      const fail = rows.length - ok;
      if (fail === 0) {
        addToast("success", tr("toast.node.test_ok", "Tested {count} node(s)", { count: ok }));
      } else {
        addToast("error", tr("toast.node.test_partial", "Test finished: {ok} ok, {fail} failed", { ok, fail }));
      }
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.node.test_failed", "Node test failed: {message}", { message }));
    },
  });
}

export function useBatchForwarding() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body: { node_ids: string[]; forwarding_enabled: boolean }) => {
      const { data } = await api.post<{ data: { updated: number } }>("/nodes/forwarding/batch", body);
      return data.data;
    },
    onSuccess: (result, variables) => {
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      const action = variables.forwarding_enabled
        ? tr("nodes.forwarding.enable", "Enable forwarding")
        : tr("nodes.forwarding.disable", "Disable forwarding");
      addToast("success", tr("toast.forwarding.batch", "{action}: {count} node(s)", { action, count: result.updated }));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", tr("toast.forwarding.batch_failed", "Batch forwarding update failed: {message}", { message }));
    },
  });
}
