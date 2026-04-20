import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';

const TOKEN_STORAGE_KEY = 'gekko.admin.token';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8420',
  withCredentials: false,
});

api.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
  const tok = localStorage.getItem(TOKEN_STORAGE_KEY);
  if (tok) {
    cfg.headers.set('Authorization', `Bearer ${tok}`);
  }
  return cfg;
});

api.interceptors.response.use(
  (res) => res,
  (err: AxiosError) => {
    if (err.response?.status === 401) {
      window.dispatchEvent(new CustomEvent('gekko:unauthorized'));
    }
    return Promise.reject(err);
  },
);

export function storeToken(tok: string) {
  localStorage.setItem(TOKEN_STORAGE_KEY, tok);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}

export function readToken(): string | null {
  return localStorage.getItem(TOKEN_STORAGE_KEY);
}
