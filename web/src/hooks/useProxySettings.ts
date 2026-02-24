import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getProxySettings,
  updateProxySettings,
  applyProxySettings,
  type UpdateProxySettingsBody,
} from "../api/settings";
import { useToast } from "../components/common/ToastContext";

export function useProxySettings() {
  return useQuery({
    queryKey: ["proxy-settings"],
    queryFn: getProxySettings,
  });
}

export function useUpdateProxySettings() {
  const q = useQueryClient();
  const { addToast } = useToast();
  return useMutation({
    mutationFn: (body: UpdateProxySettingsBody) => updateProxySettings(body),
    onSuccess: () => {
      q.invalidateQueries({ queryKey: ["proxy-settings"] });
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
