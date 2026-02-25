import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Node } from "../api/types";
import { useToast } from "../components/common/ToastContext";

export function useNodes(params?: { enabled?: number; sub_id?: string }) {
  return useQuery({
    queryKey: ["nodes", params],
    queryFn: async () => {
      const { data } = await api.get<{ data: Node[] }>("/nodes", { params });
      return data.data;
    },
  });
}

export function useUpdateNode() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: async (body: { id: string; name?: string; enabled?: boolean; forwarding_enabled?: boolean }) => {
      const { data } = await api.post<{ data: Node }>("/nodes/update", body);
      return data.data;
    },
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["nodes"] });
      addToast("success", "Node updated");
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        "Unknown error";
      addToast("error", `Update node failed: ${message}`);
    },
  });
}

export function useTestNodes() {
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
      const ok = rows.filter((r) => r.status === "ok").length;
      const fail = rows.length - ok;
      if (fail === 0) {
        addToast("success", `Tested ${ok} node${ok === 1 ? "" : "s"}`);
      } else {
        addToast("error", `Test finished: ${ok} ok, ${fail} failed`);
      }
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        "Unknown error";
      addToast("error", `Node test failed: ${message}`);
    },
  });
}
