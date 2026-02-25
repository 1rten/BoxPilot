import { useEffect } from "react";
import { Form, Input, InputNumber, Modal, Switch } from "antd";

export type SubscriptionModalMode = "create" | "edit";

export interface SubscriptionModalValues {
  name: string;
  url?: string;
  auto_update_enabled: boolean;
  refresh_interval_sec: number;
}

interface SubscriptionModalProps {
  open: boolean;
  mode: SubscriptionModalMode;
  initialValues?: SubscriptionModalValues;
  submitting: boolean;
  onCancel: () => void;
  onSubmit: (values: SubscriptionModalValues) => void;
}

export function SubscriptionModal({
  open,
  mode,
  initialValues,
  submitting,
  onCancel,
  onSubmit,
}: SubscriptionModalProps) {
  const [form] = Form.useForm<SubscriptionModalValues>();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        name: initialValues?.name ?? "",
        url: initialValues?.url ?? "",
        auto_update_enabled: initialValues?.auto_update_enabled ?? false,
        refresh_interval_sec: initialValues?.refresh_interval_sec ?? 3600,
      });
    }
  }, [open, initialValues?.name, initialValues?.url, initialValues?.auto_update_enabled, initialValues?.refresh_interval_sec, form]);

  if (!open) return null;

  const title = mode === "create" ? "Create Subscription" : "Edit Subscription";

  return (
    <Modal
      title={title}
      open={open}
      onCancel={onCancel}
      confirmLoading={submitting}
      okText="Save"
      cancelText="Cancel"
      onOk={() => {
        form.submit();
      }}
    >
      <Form<SubscriptionModalValues>
        form={form}
        layout="vertical"
        onFinish={(values) => {
          onSubmit(values);
        }}
      >
        <Form.Item
          label="URL"
          name="url"
          rules={[{ required: true, message: "Please enter subscription URL" }]}
          extra={mode === "edit" ? "Changing URL will trigger node refresh." : undefined}
        >
          <Input placeholder="Subscription URL" />
        </Form.Item>
        <Form.Item
          label="Name"
          name="name"
          rules={[{ required: true, message: "Please enter name" }]}
        >
          <Input placeholder="Name" />
        </Form.Item>
        <Form.Item
          label="Auto Update"
          name="auto_update_enabled"
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item
          noStyle
          shouldUpdate={(prev, curr) => prev.auto_update_enabled !== curr.auto_update_enabled}
        >
          {({ getFieldValue }) => (
            <Form.Item
              label="Update Interval (seconds)"
              name="refresh_interval_sec"
              rules={[
                { required: true, message: "Please enter update interval" },
                {
                  validator: async (_, value) => {
                    const autoEnabled = !!getFieldValue("auto_update_enabled");
                    if (!autoEnabled) return;
                    if (typeof value !== "number" || value < 60) {
                      throw new Error("When auto update is enabled, interval must be >= 60 seconds");
                    }
                  },
                },
              ]}
            >
              <InputNumber min={60} step={60} precision={0} style={{ width: "100%" }} />
            </Form.Item>
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
}
