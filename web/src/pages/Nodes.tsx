import { useNodes } from "../hooks/useNodes";

export default function Nodes() {
  const { data: list, isLoading } = useNodes({});

  if (isLoading) return <p>Loading...</p>;

  return (
    <div>
      <h1>Nodes</h1>
      <ul>
        {(list || []).map((n) => (
          <li key={n.id}>{n.tag} ({n.type}) {n.enabled ? "on" : "off"}</li>
        ))}
      </ul>
    </div>
  );
}
