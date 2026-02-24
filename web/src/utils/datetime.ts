export function formatDateTime(input?: string | null): string {
  if (!input) return "-";
  const d = new Date(input);
  if (Number.isNaN(d.getTime())) return input;

  const y = d.getFullYear();
  const m = pad(d.getMonth() + 1);
  const day = pad(d.getDate());
  const h = pad(d.getHours());
  const min = pad(d.getMinutes());
  const s = pad(d.getSeconds());
  return `${y}-${m}-${day} ${h}:${min}:${s}`;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

