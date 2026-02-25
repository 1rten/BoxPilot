import { api } from "./client";
import type { Subscription } from "./types";

export async function getSubscriptions(): Promise<Subscription[]> {
  const { data } = await api.get<{ data: Subscription[] }>("/subscriptions");
  return data.data;
}

export interface CreateSubscriptionBody {
  url: string;
  name?: string;
  type?: string;
  auto_update_enabled?: boolean;
  refresh_interval_sec?: number;
}

export async function createSubscription(body: CreateSubscriptionBody): Promise<Subscription> {
  const { data } = await api.post<{ data: Subscription }>("/subscriptions/create", {
    refresh_interval_sec: 3600,
    ...body,
  });
  return data.data;
}

export interface UpdateSubscriptionBody {
  id: string;
  name?: string;
  url?: string;
  enabled?: boolean;
  auto_update_enabled?: boolean;
  refresh_interval_sec?: number;
}

export async function updateSubscription(body: UpdateSubscriptionBody): Promise<Subscription> {
  const { id, name, url, enabled, auto_update_enabled, refresh_interval_sec } = body;
  const payload: any = { id };
  if (name !== undefined) payload.name = name;
  if (url !== undefined) payload.url = url;
  if (enabled !== undefined) payload.enabled = enabled;
  if (auto_update_enabled !== undefined) payload.auto_update_enabled = auto_update_enabled;
  if (refresh_interval_sec !== undefined) payload.refresh_interval_sec = refresh_interval_sec;
  const { data } = await api.post<{ data: Subscription }>("/subscriptions/update", payload);
  return data.data;
}

export async function deleteSubscription(id: string): Promise<void> {
  await api.post("/subscriptions/delete", { id });
}

export interface RefreshSubscriptionResult {
  sub_id: string;
  not_modified: boolean;
  nodes_total: number;
  nodes_enabled: number;
  fetched_at: string;
}

export async function refreshSubscription(id: string): Promise<RefreshSubscriptionResult> {
  const { data } = await api.post<RefreshSubscriptionResult>("/subscriptions/refresh", { id });
  return data;
}
