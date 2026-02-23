import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Subscription } from "../api/types";
export function useSubscriptions() {
  return useQuery({ queryKey: ["subscriptions"], queryFn: async () => { const { data } = await api.get<{ data: Subscription[] }>("/subscriptions"); return data.data; } });
}
export function useCreateSubscription() {
  const q = useQueryClient();
  return useMutation({ mutationFn: async (body: { url: string; name?: string }) => { const { data } = await api.post<{ data: Subscription }>("/subscriptions/create", body); return data.data; }, onSuccess: () => q.invalidateQueries({ queryKey: ["subscriptions"] }) });
}
