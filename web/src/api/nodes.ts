import { api } from "./client";
import type { NodeForwardingData, ProxyType } from "./types";

export async function getNodeForwarding(nodeId: string): Promise<NodeForwardingData> {
  const { data } = await api.get<{ data: NodeForwardingData }>("/nodes/forwarding", {
    params: { node_id: nodeId },
  });
  return data.data;
}

export interface UpdateNodeForwardingBody {
  node_id: string;
  proxy_type: ProxyType;
  use_global?: boolean;
  enabled?: boolean;
  port?: number;
  auth_mode?: "none" | "basic";
  username?: string;
  password?: string;
}

export async function updateNodeForwarding(
  body: UpdateNodeForwardingBody
): Promise<NodeForwardingData> {
  const { data } = await api.post<{ data: NodeForwardingData }>("/nodes/forwarding/update", body);
  return data.data;
}

export async function restartNodeForwarding(nodeId: string): Promise<void> {
  await api.post("/nodes/forwarding/restart", { node_id: nodeId });
}
