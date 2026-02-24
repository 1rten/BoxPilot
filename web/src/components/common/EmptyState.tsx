import type { ReactNode } from "react";
import { Button, Empty } from "antd";

interface EmptyStateProps {
  title: string;
  description?: string;
  actionLabel?: string;
  onActionClick?: () => void;
}

export function EmptyState({
  title,
  description,
  actionLabel,
  onActionClick,
}: EmptyStateProps): ReactNode {
  return (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={
        <div>
          <div style={{ fontSize: 16, fontWeight: 500 }}>{title}</div>
          {description && (
            <p style={{ fontSize: 14, color: "#64748B", margin: 0 }}>
              {description}
            </p>
          )}
        </div>
      }
    >
      {actionLabel && onActionClick && (
        <Button type="primary" onClick={onActionClick}>
          {actionLabel}
        </Button>
      )}
    </Empty>
  );
}

