import axios from "axios";
export const api = axios.create({ baseURL: "/api/v1", headers: { "Content-Type": "application/json" } });
api.interceptors.response.use((r) => r, (e) => { if (e.response?.data?.error) e.appError = e.response.data.error; return Promise.reject(e); });
