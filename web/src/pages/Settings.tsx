import { useEffect } from "react";
import { Button, Card, Form, Input, InputNumber, Select, Switch, Tag } from "antd";
import type { ProxyConfig, ProxyType } from "../api/types";
import { buildProxyUrl } from "../api/settings";
import { useProxySettings, useUpdateProxySettings, useApplyProxySettings } from "../hooks/useProxySettings";
import { useToast } from "../components/common/ToastContext";

interface ProxyCardProps {
  title: string;
  proxyType: ProxyType;
  data?: ProxyConfig;
}

export default function Settings() {
  const { data, isLoading } = useProxySettings();
  return (
    <div>
      <div className="bp-page-header">
        <h1 className="bp-page-title">Settings</h1>
      </div>
      <div className="bp-dashboard-grid">
        <ProxySettingsCard title="HTTP Proxy" proxyType="http" data={data?.http} />
        <ProxySettingsCard title="SOCKS5 Proxy" proxyType="socks" data={data?.socks} />
      </div>
      {isLoading && (
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
    const url = buildProxyUrl(data);
    try {
      await navigator.clipboard.writeText(url);
      addToast("success", "Connection string copied");
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
    <Card className="bp-dashboard-card bp-dashboard-card--wide">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">Forwarding</p>
          <h2 className="bp-card-title">{title}</h2>
        </div>
        {statusTag}
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
        <Form.Item
          name="username"
          label="Username"
          rules={[
            {
              required: authMode === "basic",
              message: "Username is required for Basic auth",
            },
          ]}
        >
          <Input placeholder="Optional" />
        </Form.Item>
        <Form.Item
          name="password"
          label="Password"
          rules={[
            {
              required: authMode === "basic",
              message: "Password is required for Basic auth",
            },
          ]}
        >
          <Input.Password placeholder="Optional" />
        </Form.Item>
      </Form>
      <div className="bp-page-actions" style={{ marginTop: 12 }}>
        <Button onClick={onCopy} disabled={!data}>
          Copy URL
        </Button>
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
