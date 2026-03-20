import axios from "axios";

function resolveApiBaseURL(): string {
  const origin = import.meta.env.VITE_API_ORIGIN?.trim() ?? "";
  const path = import.meta.env.VITE_API_BASE?.trim() || "/api/v1";
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  if (!origin) return normalizedPath;
  return `${origin.replace(/\/$/, "")}${normalizedPath}`;
}

const baseURL = resolveApiBaseURL();

export const api = axios.create({ baseURL, headers: { "Content-Type": "application/json" } });
api.interceptors.response.use((r) => r, (e) => { if (e.response?.data?.error) e.appError = e.response.data.error; return Promise.reject(e); });
