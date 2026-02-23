import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { RuntimeStatusData } from "../api/types";
export function useRuntimeStatus() {
  return useQuery({ queryKey: ["runtime-status"], queryFn: async () => { const { data } = await api.get<{ data: RuntimeStatusData }>("/runtime/status"); return data.data; } });
}
export function useRuntimeReload() {
  const q = useQueryClient();
  return useMutation({ mutationFn: async () => { const { data } = await api.post("/runtime/reload", {}); return data; }, onSuccess: () => q.invalidateQueries({ queryKey: ["runtime-status"] }) });
}
