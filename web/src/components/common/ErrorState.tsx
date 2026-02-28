import type { ReactNode } from "react";
import { Alert, Button } from "antd";
import { useI18n } from "../../i18n/context";

interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps): ReactNode {
  const { tr } = useI18n();
  return (
    <Alert
      type="error"
      message={message}
      showIcon
      action={
        onRetry && (
          <Button size="small" type="primary" onClick={onRetry}>
            {tr("common.retry", "Retry")}
          </Button>
        )
      }
      style={{ marginBottom: 16 }}
    />
  );
}
