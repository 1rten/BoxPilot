import { useEffect, useState } from "react";
import { Form, Input, Modal } from "antd";

export type SubscriptionModalMode = "create" | "edit";

export interface SubscriptionModalValues {
  name: string;
  url?: string;
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
      });
    }
  }, [open, initialValues?.name, initialValues?.url, form]);

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
        {mode === "create" && (
          <Form.Item
            label="URL"
            name="url"
            rules={[{ required: true, message: "Please enter subscription URL" }]}
          >
            <Input placeholder="Subscription URL" />
          </Form.Item>
        )}
        <Form.Item
          label="Name"
          name="name"
          rules={[{ required: true, message: "Please enter name" }]}
        >
          <Input placeholder="Name" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

