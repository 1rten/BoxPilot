import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Subscription } from "../api/types";
import {
  getSubscriptions,
  createSubscription,
  updateSubscription,
  deleteSubscription,
  refreshSubscription,
  type RefreshSubscriptionResult,
  type CreateSubscriptionBody,
  type UpdateSubscriptionBody,
} from "../api/subscriptions";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

export function useSubscriptions() {
  return useQuery<Subscription[]>({
    queryKey: ["subscriptions"],
    queryFn: getSubscriptions,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 15_000,
    refetchIntervalInBackground: true,
  });
}

export function useCreateSubscription() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: CreateSubscriptionBody) => createSubscription(body),
    onSuccess: async (created) => {
      let refreshed = false;
      try {
        const result = await refreshSubscription(created.id);
        refreshed = true;
        addToast("success", buildRefreshMessage(result, true, tr));
      } catch (error: unknown) {
        const message = extractErrorMessage(error);
        addToast("error", tr("toast.sub.created_refresh_failed", "Subscription created, but initial refresh failed: {message}", { message }));
      }
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      if (!refreshed) {
        addToast("success", tr("toast.sub.created", "Subscription created"));
      }
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", tr("toast.sub.create_failed", "Create subscription failed: {message}", { message }));
    },
  });
}

export function useUpdateSubscription() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateSubscriptionBody) => updateSubscription(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.sub.updated", "Subscription updated"));
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", tr("toast.sub.update_failed", "Update subscription failed: {message}", { message }));
    },
  });
}

export function useDeleteSubscription() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (id: string) => deleteSubscription(id),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.sub.deleted", "Subscription deleted"));
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", tr("toast.sub.delete_failed", "Delete subscription failed: {message}", { message }));
    },
  });
}

export function useRefreshSubscription() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (id: string) => refreshSubscription(id),
    onSuccess: (result) => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", buildRefreshMessage(result, false, tr));
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", tr("toast.sub.refresh_failed", "Refresh subscription failed: {message}", { message }));
    },
  });
}

function buildRefreshMessage(
  result: RefreshSubscriptionResult,
  isInitial: boolean,
  tr: (key: string, fallback?: string, params?: Record<string, string | number | boolean | null | undefined>) => string
): string {
  if (result.not_modified) {
    return isInitial
      ? tr("toast.sub.created.no_change", "Subscription created. No new nodes from source.")
      : tr("toast.sub.no_change", "No changes detected");
  }
  if (result.nodes_total <= 0) {
    return isInitial
      ? tr("toast.sub.created.no_nodes", "Subscription created. Source returned no usable nodes.")
      : tr("toast.sub.refreshed.no_nodes", "Refreshed: no usable nodes");
  }
  return isInitial
    ? tr("toast.sub.created.synced", "Subscription created and synced {count} node(s)", { count: result.nodes_total })
    : tr("toast.sub.refreshed.synced", "Refreshed {count} node(s)", { count: result.nodes_total });
}

function extractErrorMessage(error: unknown): string {
  const anyErr = error as any;
  if (anyErr?.appError?.message) return anyErr.appError.message as string;
  if (anyErr?.response?.data?.error?.message)
    return anyErr.response.data.error.message as string;
  if (anyErr?.message) return anyErr.message as string;
  return "Unknown error";
}
