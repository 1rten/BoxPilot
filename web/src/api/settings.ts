import { api } from "./client";
import type {
  ForwardingRuntimeStatus,
  ForwardingSummaryData,
  ProxySettingsData,
  ProxyType,
  ProxyConfig,
  RoutingSettingsData,
  RoutingSummaryData,
} from "./types";

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

export async function getForwardingSummary(): Promise<ForwardingSummaryData> {
  const { data } = await api.get<{ data: ForwardingSummaryData }>("/settings/forwarding/summary");
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

export async function getRoutingSettings(): Promise<RoutingSettingsData> {
  const { data } = await api.get<{ data: RoutingSettingsData }>("/settings/routing");
  return data.data;
}

export interface UpdateRoutingSettingsBody {
  bypass_private_enabled: boolean;
  bypass_domains: string[];
  bypass_cidrs: string[];
}

export async function updateRoutingSettings(
  body: UpdateRoutingSettingsBody
): Promise<RoutingSettingsData> {
  const { data } = await api.post<{ data: RoutingSettingsData }>("/settings/routing/update", body);
  return data.data;
}

export async function getRoutingSummary(): Promise<RoutingSummaryData> {
  const { data } = await api.get<{ data: RoutingSummaryData }>("/settings/routing/summary");
  return data.data;
}

export function resolveProxyClientHost(
  listenAddress: string,
  preferredHost?: string
): string {
  if (preferredHost && preferredHost.trim() !== "") {
    return preferredHost.trim();
  }
  if (listenAddress === "0.0.0.0") {
    return "127.0.0.1";
  }
  return listenAddress;
}

export function buildProxyUrl(cfg: ProxyConfig, preferredHost?: string): string {
  const scheme = cfg.proxy_type === "http" ? "http" : "socks5";
  const host = resolveProxyClientHost(cfg.listen_address, preferredHost);
  const auth =
    cfg.auth_mode === "basic" && cfg.username && cfg.password
      ? `${encodeURIComponent(cfg.username)}:${encodeURIComponent(cfg.password)}@`
      : "";
  return `${scheme}://${auth}${host}:${cfg.port}`;
}
