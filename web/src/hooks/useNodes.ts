import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Node } from "../api/types";

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
  return useMutation({
    mutationFn: async (body: { id: string; name?: string; enabled?: boolean }) => {
      const { data } = await api.post<{ data: Node }>("/nodes/update", body);
      return data.data;
    },
    onSuccess: () => q.invalidateQueries({ queryKey: ["nodes"] }),
  });
}
