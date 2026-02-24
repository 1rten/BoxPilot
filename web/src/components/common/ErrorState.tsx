import type { ReactNode } from "react";
import { Alert, Button } from "antd";

interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps): ReactNode {
  return (
    <Alert
      type="error"
      message={message}
      showIcon
      action={
        onRetry && (
          <Button size="small" type="primary" onClick={onRetry}>
            Retry
          </Button>
        )
      }
      style={{ marginBottom: 16 }}
    />
  );
}

