import { useRuntimeStatus, useRuntimeReload } from "../hooks/useRuntime";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { useNodes } from "../hooks/useNodes";
import { ErrorState } from "../components/common/ErrorState";
import { Button, Card } from "antd";
import { Link } from "react-router-dom";

export default function Dashboard() {
  const {
    data: runtime,
    isLoading: runtimeLoading,
    error: runtimeError,
  } = useRuntimeStatus();
  const reload = useRuntimeReload();
  const { data: subs } = useSubscriptions();
  const { data: nodes } = useNodes({});

  return (
    <div>
      <h1 className="bp-page-title">Dashboard</h1>
      <div
        style={{
          display: "grid",
          gap: 16,
          gridTemplateColumns: "repeat(auto-fit, minmax(260px, 320px))",
          justifyContent: "center",
        }}
      >
        <Card title="Runtime" hoverable>
          {runtimeLoading && !runtime && (
            <p style={{ color: "#64748B", fontSize: 14 }}>Loading runtime status...</p>
          )}
          {runtimeError && (
            <ErrorState
              message={`Failed to load runtime status: ${(runtimeError as Error).message}`}
            />
          )}
          {runtime && !runtimeError && (
            <>
              <div style={{ fontSize: 14, color: "#334155" }}>
                <p>
                  <strong>Config version:</strong> {runtime.config_version}{" "}
                  <span style={{ marginLeft: 8 }}>
                    <strong>Hash:</strong> {runtime.config_hash.slice(0, 8)}
                  </span>
                </p>
                <p>
                  <strong>Mode:</strong> {runtime.runtime_mode}{" "}
                  <span style={{ marginLeft: 8 }}>
                    <strong>Container:</strong> {runtime.singbox_container}
                  </span>
                </p>
              </div>
              <div style={{ marginTop: 12 }}>
                <Button
                  type="primary"
                  onClick={() => reload.mutate()}
                  loading={reload.isPending}
                >
                  Reload
                </Button>
              </div>
            </>
          )}
        </Card>

        <Card
          title="Subscriptions"
          hoverable
          extra={
            <Link to="/subscriptions" style={{ fontSize: 12 }}>
              View All &rsaquo;
            </Link>
          }
        >
          <p style={{ fontSize: 14, color: "#334155", marginBottom: 0 }}>
            Total: <strong>{subs?.length ?? 0}</strong>
          </p>
        </Card>

        <Card
          title="Nodes"
          hoverable
          extra={
            <Link to="/nodes" style={{ fontSize: 12 }}>
              View All &rsaquo;
            </Link>
          }
        >
          <p style={{ fontSize: 14, color: "#334155", marginBottom: 0 }}>
            Total: <strong>{nodes?.length ?? 0}</strong>
          </p>
        </Card>
      </div>
    </div>
  );
}
