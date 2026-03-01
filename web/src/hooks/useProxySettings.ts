import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getProxySettings,
  updateProxySettings,
  applyProxySettings,
  getForwardingRuntimeStatus,
  getForwardingSummary,
  getForwardingPolicy,
  startForwardingRuntime,
  stopForwardingRuntime,
  getRoutingSettings,
  getRoutingSummary,
  updateRoutingSettings,
  updateForwardingPolicy,
  type UpdateProxySettingsBody,
  type UpdateRoutingSettingsBody,
  type UpdateForwardingPolicyBody,
} from "../api/settings";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

export function useProxySettings() {
  return useQuery({
    queryKey: ["proxy-settings"],
    queryFn: getProxySettings,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });
}

export function useUpdateProxySettings() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateProxySettingsBody) => updateProxySettings(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["proxy-settings"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", tr("toast.proxy.saved", "Proxy settings saved"));
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

export function useApplyProxySettings() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: applyProxySettings,
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["proxy-settings"] });
      q.invalidateQueries({ queryKey: ["forwarding-runtime-status"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", tr("toast.proxy.applied", "Proxy settings applied"));
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

export function useForwardingRuntimeStatus() {
  return useQuery({
    queryKey: ["forwarding-runtime-status"],
    queryFn: getForwardingRuntimeStatus,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 5_000,
    refetchIntervalInBackground: true,
  });
}

export function useForwardingSummary() {
  return useQuery({
    queryKey: ["forwarding-summary"],
    queryFn: getForwardingSummary,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 5_000,
    refetchIntervalInBackground: true,
  });
}

export function useStartForwardingRuntime() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: startForwardingRuntime,
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["forwarding-runtime-status"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["proxy-settings"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", tr("toast.forwarding.started", "Forwarding started"));
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

export function useStopForwardingRuntime() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: stopForwardingRuntime,
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["forwarding-runtime-status"] });
      q.invalidateQueries({ queryKey: ["forwarding-summary"] });
      q.invalidateQueries({ queryKey: ["proxy-settings"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      q.invalidateQueries({ queryKey: ["runtime-traffic"] });
      addToast("success", tr("toast.forwarding.stopped", "Forwarding stopped"));
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

export function useRoutingSettings() {
  return useQuery({
    queryKey: ["routing-settings"],
    queryFn: getRoutingSettings,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });
}

export function useUpdateRoutingSettings() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateRoutingSettingsBody) => updateRoutingSettings(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["routing-settings"] });
      q.invalidateQueries({ queryKey: ["routing-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.routing.saved", "Routing bypass settings saved"));
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

export function useRoutingSummary() {
  return useQuery({
    queryKey: ["routing-summary"],
    queryFn: getRoutingSummary,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 10_000,
    refetchIntervalInBackground: true,
  });
}

export function useForwardingPolicy() {
  return useQuery({
    queryKey: ["forwarding-policy"],
    queryFn: getForwardingPolicy,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });
}

export function useUpdateForwardingPolicy() {
  const { tr } = useI18n();
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateForwardingPolicyBody) => updateForwardingPolicy(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["forwarding-policy"] });
      q.invalidateQueries({ queryKey: ["runtime-status"] });
      q.invalidateQueries({ queryKey: ["runtime-connections"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", tr("toast.forwarding.policy_saved", "Forwarding policy saved"));
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
