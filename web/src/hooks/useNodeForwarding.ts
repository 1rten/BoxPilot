import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getNodeForwarding,
  updateNodeForwarding,
  restartNodeForwarding,
  type UpdateNodeForwardingBody,
} from "../api/nodes";
import { useToast } from "../components/common/ToastContext";

export function useNodeForwarding(nodeId?: string | null) {
  return useQuery({
    queryKey: ["node-forwarding", nodeId],
    queryFn: () => getNodeForwarding(nodeId as string),
    enabled: !!nodeId,
  });
}

export function useUpdateNodeForwarding() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateNodeForwardingBody) => updateNodeForwarding(body),
    onSuccess: (_data, vars) => {
      q.invalidateQueries({ queryKey: ["node-forwarding", vars.node_id] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", "Forwarding override saved");
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        "Unknown error";
      addToast("error", message);
    },
  });
}

export function useRestartNodeForwarding() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (nodeId: string) => restartNodeForwarding(nodeId),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", "Node proxy restarted");
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        "Unknown error";
      addToast("error", message);
    },
  });
}
