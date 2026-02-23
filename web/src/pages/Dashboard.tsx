import { useRuntimeStatus, useRuntimeReload } from "../hooks/useRuntime";

export default function Dashboard() {
  const { data: status, isLoading, error } = useRuntimeStatus();
  const reload = useRuntimeReload();

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p>Error: {(error as Error).message}</p>;

  return (
    <div>
      <h1>Dashboard</h1>
      {status && (
        <p>Config version: {status.config_version} | Hash: {status.config_hash}</p>
      )}
      <button onClick={() => reload.mutate()} disabled={reload.isPending}>
        {reload.isPending ? "Reloading..." : "Reload"}
      </button>
    </div>
  );
}
