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

export function useSubscriptions() {
  return useQuery<Subscription[]>({
    queryKey: ["subscriptions"],
    queryFn: getSubscriptions,
  });
}

export function useCreateSubscription() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: CreateSubscriptionBody) => createSubscription(body),
    onSuccess: async (created) => {
      let refreshed = false;
      try {
        const result = await refreshSubscription(created.id);
        refreshed = true;
        addToast("success", buildRefreshMessage(result, true));
      } catch (error: unknown) {
        const message = extractErrorMessage(error);
        addToast("error", `Subscription created, but initial refresh failed: ${message}`);
      }
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      if (!refreshed) {
        addToast("success", "Subscription created");
      }
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", `Create subscription failed: ${message}`);
    },
  });
}

export function useUpdateSubscription() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateSubscriptionBody) => updateSubscription(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      addToast("success", "Subscription updated");
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", `Update subscription failed: ${message}`);
    },
  });
}

export function useDeleteSubscription() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (id: string) => deleteSubscription(id),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      addToast("success", "Subscription deleted");
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", `Delete subscription failed: ${message}`);
    },
  });
}

export function useRefreshSubscription() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (id: string) => refreshSubscription(id),
    onSuccess: (result) => {
      q.invalidateQueries({ queryKey: ["subscriptions"] });
      q.invalidateQueries({ queryKey: ["nodes"] });
      addToast("success", buildRefreshMessage(result, false));
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error);
      addToast("error", `Refresh subscription failed: ${message}`);
    },
  });
}

function buildRefreshMessage(result: RefreshSubscriptionResult, isInitial: boolean): string {
  if (result.not_modified) {
    return isInitial ? "Subscription created. No new nodes from source." : "No changes detected";
  }
  if (result.nodes_total <= 0) {
    return isInitial ? "Subscription created. Source returned no usable nodes." : "Refreshed: no usable nodes";
  }
  const noun = result.nodes_total === 1 ? "node" : "nodes";
  return isInitial
    ? `Subscription created and synced ${result.nodes_total} ${noun}`
    : `Refreshed ${result.nodes_total} ${noun}`;
}

function extractErrorMessage(error: unknown): string {
  const anyErr = error as any;
  if (anyErr?.appError?.message) return anyErr.appError.message as string;
  if (anyErr?.response?.data?.error?.message)
    return anyErr.response.data.error.message as string;
  if (anyErr?.message) return anyErr.message as string;
  return "Unknown error";
}
