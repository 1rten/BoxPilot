import { api } from "./client";
import type { ForwardingRuntimeStatus, ProxySettingsData, ProxyType, ProxyConfig } from "./types";

export async function getProxySettings(): Promise<ProxySettingsData> {
  const { data } = await api.get<{ data: ProxySettingsData }>("/settings/proxy");
  return data.data;
}

export interface UpdateProxySettingsBody {
  proxy_type: ProxyType;
  enabled: boolean;
  listen_address: "127.0.0.1" | "0.0.0.0";
  port: number;
  auth_mode: "none" | "basic";
  username?: string;
  password?: string;
}

export async function updateProxySettings(
  body: UpdateProxySettingsBody
): Promise<ProxySettingsData> {
  const { data } = await api.post<{ data: ProxySettingsData }>("/settings/proxy/update", body);
  return data.data;
}

export interface ApplyProxyResult {
  config_version: number;
  config_hash: string;
  restart_output: string;
  reloaded_at: string;
}

export async function applyProxySettings(): Promise<ApplyProxyResult> {
  const { data } = await api.post<{ data: ApplyProxyResult }>("/settings/proxy/apply", {});
  return data.data;
}

export async function getForwardingRuntimeStatus(): Promise<ForwardingRuntimeStatus> {
  const { data } = await api.get<{ data: ForwardingRuntimeStatus }>("/settings/forwarding/status");
  return data.data;
}

export async function startForwardingRuntime(): Promise<ForwardingRuntimeStatus> {
  const { data } = await api.post<{ data: ForwardingRuntimeStatus }>("/settings/forwarding/start", {});
  return data.data;
}

export async function stopForwardingRuntime(): Promise<ForwardingRuntimeStatus> {
  const { data } = await api.post<{ data: ForwardingRuntimeStatus }>("/settings/forwarding/stop", {});
  return data.data;
}

export function buildProxyUrl(cfg: ProxyConfig): string {
  const scheme = cfg.proxy_type === "http" ? "http" : "socks5";
  const host = cfg.listen_address;
  const auth =
    cfg.auth_mode === "basic" && cfg.username && cfg.password
      ? `${encodeURIComponent(cfg.username)}:${encodeURIComponent(cfg.password)}@`
      : "";
  return `${scheme}://${auth}${host}:${cfg.port}`;
}
