import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getNodeForwarding,
  updateNodeForwarding,
  restartNodeForwarding,
  type UpdateNodeForwardingBody,
} from "../api/nodes";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

export function useNodeForwarding(nodeId?: string | null) {
  return useQuery({
    queryKey: ["node-forwarding", nodeId],
    queryFn: () => getNodeForwarding(nodeId as string),
    enabled: !!nodeId,
  });
}

export function useUpdateNodeForwarding() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateNodeForwardingBody) => updateNodeForwarding(body),
    onSuccess: (_data, vars) => {
      q.invalidateQueries({ queryKey: ["node-forwarding", vars.node_id] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.forwarding.override_saved", "Forwarding override saved"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", message);
    },
  });
}

export function useRestartNodeForwarding() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (nodeId: string) => restartNodeForwarding(nodeId),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", tr("toast.forwarding.node_restarted", "Node proxy restarted"));
    },
    onError: (error: unknown) => {
      const anyErr = error as any;
      const message =
        anyErr?.appError?.message ||
        anyErr?.response?.data?.error?.message ||
        anyErr?.message ||
        tr("toast.unknown", "Unknown error");
      addToast("error", message);
    },
  });
}
