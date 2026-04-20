import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { api, storeToken, clearToken, readToken } from '@/lib/api';

export interface Admin {
  id: number;
  email: string;
  name: string;
}

export const useAuthStore = defineStore('auth', () => {
  const admin = ref<Admin | null>(null);
  const loading = ref(false);

  const isAuthenticated = computed(() => admin.value !== null);

  async function login(email: string, password: string): Promise<boolean> {
    loading.value = true;
    try {
      const res = await api.post<{ token: string; admin: Admin }>('/api/auth/login', {
        email,
        password,
      });
      storeToken(res.data.token);
      admin.value = res.data.admin;
      return true;
    } catch {
      return false;
    } finally {
      loading.value = false;
    }
  }

  function logout() {
    clearToken();
    admin.value = null;
  }

  async function restore() {
    if (!readToken()) return;
    try {
      const res = await api.get<Admin>('/api/auth/me');
      admin.value = res.data;
    } catch {
      clearToken();
      admin.value = null;
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('gekko:unauthorized', () => logout());
  }

  return { admin, loading, isAuthenticated, login, logout, restore };
});
