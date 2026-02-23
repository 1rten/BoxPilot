import { useSubscriptions, useCreateSubscription } from "../hooks/useSubscriptions";
import { useState } from "react";

export default function Subscriptions() {
  const { data: list, isLoading } = useSubscriptions();
  const create = useCreateSubscription();
  const [url, setUrl] = useState("");

  const handleCreate = () => {
    if (!url.trim()) return;
    create.mutate({ url: url.trim() }, { onSuccess: () => setUrl("") });
  };

  if (isLoading) return <p>Loading...</p>;

  return (
    <div>
      <h1>Subscriptions</h1>
      <div>
        <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="Subscription URL" />
        <button onClick={handleCreate} disabled={create.isPending}>Add</button>
      </div>
      <ul>
        {(list || []).map((s) => (
          <li key={s.id}>{s.name || s.url} ({s.enabled ? "on" : "off"})</li>
        ))}
      </ul>
    </div>
  );
}
