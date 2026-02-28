import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getProxySettings,
  updateProxySettings,
  applyProxySettings,
  getForwardingRuntimeStatus,
  getForwardingSummary,
  startForwardingRuntime,
  stopForwardingRuntime,
  getRoutingSettings,
  getRoutingSummary,
  updateRoutingSettings,
  type UpdateProxySettingsBody,
  type UpdateRoutingSettingsBody,
} from "../api/settings";
import { useToast } from "../components/common/ToastContext";

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
      addToast("success", "Proxy settings saved");
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

export function useApplyProxySettings() {
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
      addToast("success", "Proxy settings applied");
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
      addToast("success", "Forwarding started");
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

export function useStopForwardingRuntime() {
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
      addToast("success", "Forwarding stopped");
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
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateRoutingSettingsBody) => updateRoutingSettings(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["routing-settings"] });
      q.invalidateQueries({ queryKey: ["routing-summary"] });
      q.invalidateQueries({ queryKey: ["runtime-logs"] });
      addToast("success", "Routing bypass settings saved");
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
