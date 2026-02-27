import { useEffect, useState } from "react";
import { Button, Card, Form, Input, InputNumber, Select, Switch, Tag } from "antd";
import type { ProxyConfig, ProxyType, RoutingSettingsData } from "../api/types";
import { buildProxyUrl, resolveProxyClientHost } from "../api/settings";
import {
  useProxySettings,
  useUpdateProxySettings,
  useApplyProxySettings,
  useRoutingSettings,
  useUpdateRoutingSettings,
} from "../hooks/useProxySettings";
import { useToast } from "../components/common/ToastContext";

interface ProxyCardProps {
  title: string;
  proxyType: ProxyType;
  data?: ProxyConfig;
}

export default function Settings() {
  const { data, isLoading } = useProxySettings();
  const { data: routingData, isLoading: routingLoading } = useRoutingSettings();
  return (
    <div className="bp-page">
      <div className="bp-page-header">
        <div>
          <h1 className="bp-page-title">Settings</h1>
          <p className="bp-page-subtitle">
            Configure global HTTP and SOCKS5 forwarding behavior.
          </p>
        </div>
      </div>
      <div className="bp-settings-grid">
        <ProxySettingsCard title="HTTP Proxy" proxyType="http" data={data?.http} />
        <ProxySettingsCard title="SOCKS5 Proxy" proxyType="socks" data={data?.socks} />
      </div>
      <div style={{ marginTop: 16 }}>
        <RoutingSettingsCard data={routingData} />
      </div>
      {(isLoading || routingLoading) && (
        <p className="bp-muted" style={{ marginTop: 12 }}>
          Loading settings...
        </p>
      )}
    </div>
  );
}

function ProxySettingsCard({ title, proxyType, data }: ProxyCardProps) {
  const [form] = Form.useForm();
  const { addToast } = useToast();
  const update = useUpdateProxySettings();
  const apply = useApplyProxySettings();
  const authMode = Form.useWatch("auth_mode", form);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!data) return;
    form.setFieldsValue({
      enabled: data.enabled,
      listen_address: data.listen_address,
      port: data.port,
      auth_mode: data.auth_mode,
      username: data.username || "",
      password: data.password || "",
    });
  }, [data, form]);

  const onSave = async () => {
    const values = await form.validateFields();
    update.mutate({
      proxy_type: proxyType,
      enabled: values.enabled,
      listen_address: values.listen_address,
      port: values.port,
      auth_mode: values.auth_mode,
      username: values.username,
      password: values.password,
    });
  };

  const onCopy = async () => {
    if (!data) return;
    const preferredHost = window.location.hostname || undefined;
    const clientHost = resolveProxyClientHost(data.listen_address, preferredHost);
    const url = buildProxyUrl(data, preferredHost);
    try {
      await navigator.clipboard.writeText(url);
      addToast("success", `Connection string copied (${clientHost}:${data.port})`);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      addToast("error", "Copy failed");
    }
  };

  const statusTag = data?.status ? (
    <Tag color={data.status === "running" ? "success" : data.status === "error" ? "error" : "default"}>
      {data.status === "running" ? "Running" : data.status === "error" ? "Error" : "Stopped"}
    </Tag>
  ) : null;

  return (
    <Card className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">Global Forwarding</p>
          <h2 className="bp-card-title">{title}</h2>
        </div>
        {statusTag}
      </div>
      <div className="bp-settings-status-row">
        <span className="bp-muted">Current binding</span>
        <span className="bp-table-mono">
          {data?.listen_address ?? "0.0.0.0"}:{data?.port ?? "-"}
        </span>
      </div>
      <div className="bp-settings-status-row">
        <span className="bp-muted">Copy URL host</span>
        <span className="bp-table-mono">
          {resolveProxyClientHost(
            data?.listen_address ?? "0.0.0.0",
            window.location.hostname || undefined
          )}
        </span>
      </div>
      {data?.error_message && (
        <p className="bp-text-danger" style={{ marginBottom: 12 }}>
          {data.error_message}
        </p>
      )}
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          enabled: true,
          listen_address: "0.0.0.0",
          port: proxyType === "http" ? 7890 : 7891,
          auth_mode: "none",
          username: "",
          password: "",
        }}
      >
        <Form.Item name="enabled" label="Enabled" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item name="listen_address" label="Listen Address">
          <Select
            options={[
              { value: "127.0.0.1", label: "127.0.0.1 (Localhost)" },
              { value: "0.0.0.0", label: "0.0.0.0 (All Interfaces)" },
            ]}
          />
        </Form.Item>
        <Form.Item
          name="port"
          label="Port"
          rules={[
            { required: true, message: "Port is required" },
            { type: "number", min: 1, max: 65535, message: "Port must be 1-65535" },
          ]}
        >
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="auth_mode" label="Auth Mode">
          <Select
            options={[
              { value: "none", label: "None" },
              { value: "basic", label: "Basic" },
            ]}
          />
        </Form.Item>
        {authMode === "basic" && (
          <>
            <Form.Item
              name="username"
              label="Username"
              rules={[
                {
                  required: true,
                  message: "Username is required for Basic auth",
                },
              ]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="password"
              label="Password"
              rules={[
                {
                  required: true,
                  message: "Password is required for Basic auth",
                },
              ]}
            >
              <Input.Password />
            </Form.Item>
          </>
        )}
      </Form>
      <div className="bp-page-actions bp-settings-actions">
        <Button onClick={onSave} type="primary" loading={update.isPending}>
          Save
        </Button>
        <Button onClick={() => apply.mutate()} loading={apply.isPending}>
          Apply / Restart
        </Button>
        <Button onClick={onCopy} disabled={!data}>
          {copied ? "Copied" : "Copy URL"}
        </Button>
      </div>
    </Card>
  );
}

interface RoutingCardProps {
  data?: RoutingSettingsData;
}

function RoutingSettingsCard({ data }: RoutingCardProps) {
  const [form] = Form.useForm();
  const update = useUpdateRoutingSettings();
  const apply = useApplyProxySettings();

  useEffect(() => {
    if (!data) return;
    form.setFieldsValue({
      bypass_private_enabled: data.bypass_private_enabled,
      bypass_domains_text: (data.bypass_domains || []).join("\n"),
      bypass_cidrs_text: (data.bypass_cidrs || []).join("\n"),
    });
  }, [data, form]);

  const onSave = async () => {
    const values = await form.validateFields();
    update.mutate({
      bypass_private_enabled: values.bypass_private_enabled,
      bypass_domains: splitLines(values.bypass_domains_text),
      bypass_cidrs: splitLines(values.bypass_cidrs_text),
    });
  };

  return (
    <Card className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">Route Rules</p>
          <h2 className="bp-card-title">Routing Bypass</h2>
        </div>
        {data?.updated_at ? <span className="bp-muted">Updated {data.updated_at}</span> : null}
      </div>
      <p className="bp-muted" style={{ marginTop: 0 }}>
        Matched domains and CIDRs will go direct instead of proxy.
      </p>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          bypass_private_enabled: true,
          bypass_domains_text: "localhost\nlocal",
          bypass_cidrs_text:
            "127.0.0.0/8\n10.0.0.0/8\n172.16.0.0/12\n192.168.0.0/16\n169.254.0.0/16\n::1/128\nfc00::/7\nfe80::/10",
        }}
      >
        <Form.Item name="bypass_private_enabled" label="Enable bypass rules" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item name="bypass_domains_text" label="Bypass domains (one per line)">
          <Input.TextArea rows={4} placeholder="localhost&#10;local" />
        </Form.Item>
        <Form.Item name="bypass_cidrs_text" label="Bypass CIDRs (one per line)">
          <Input.TextArea rows={6} placeholder="192.168.0.0/16&#10;10.0.0.0/8" />
        </Form.Item>
      </Form>
      <div className="bp-page-actions bp-settings-actions">
        <Button onClick={onSave} type="primary" loading={update.isPending}>
          Save
        </Button>
        <Button onClick={() => apply.mutate()} loading={apply.isPending}>
          Apply / Restart
        </Button>
      </div>
    </Card>
  );
}

function splitLines(raw: string): string[] {
  return raw
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}
