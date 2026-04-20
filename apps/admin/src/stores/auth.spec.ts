import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';

// Mock the api module BEFORE the store import so the mock is used everywhere.
let mockToken: string | null = null;
vi.mock('@/lib/api', () => ({
  api: {
    post: vi.fn(),
    get: vi.fn(),
  },
  storeToken: vi.fn((t: string) => {
    mockToken = t;
  }),
  clearToken: vi.fn(() => {
    mockToken = null;
  }),
  readToken: vi.fn(() => mockToken),
}));

import { api, storeToken, clearToken, readToken } from '@/lib/api';
import { useAuthStore } from './auth';

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    mockToken = null;
    (storeToken as any).mockClear();
    (clearToken as any).mockClear();
    (api.post as any).mockReset();
    (api.get as any).mockReset();
  });

  afterEach(() => {
    mockToken = null;
  });

  it('starts unauthenticated', () => {
    const store = useAuthStore();
    expect(store.isAuthenticated).toBe(false);
    expect(store.admin).toBeNull();
  });

  it('login success stores token and admin', async () => {
    (api.post as any).mockResolvedValue({
      data: { token: 'tok-123', admin: { id: 1, email: 'a@b.c', name: 'A' } },
    });

    const store = useAuthStore();
    const ok = await store.login('a@b.c', 'pw-long-enough');

    expect(ok).toBe(true);
    expect(store.isAuthenticated).toBe(true);
    expect(store.admin).toEqual({ id: 1, email: 'a@b.c', name: 'A' });
    expect(storeToken).toHaveBeenCalledWith('tok-123');
  });

  it('login failure returns false and leaves state unchanged', async () => {
    (api.post as any).mockRejectedValue({ response: { status: 401 } });
    const store = useAuthStore();

    const ok = await store.login('a@b.c', 'wrong');
    expect(ok).toBe(false);
    expect(store.isAuthenticated).toBe(false);
    expect(storeToken).not.toHaveBeenCalled();
  });

  it('logout clears token and admin', async () => {
    (api.post as any).mockResolvedValue({
      data: { token: 'tok', admin: { id: 1, email: 'a@b.c', name: 'A' } },
    });
    const store = useAuthStore();
    await store.login('a@b.c', 'pw-long-enough');

    store.logout();
    expect(store.isAuthenticated).toBe(false);
    expect(store.admin).toBeNull();
    expect(clearToken).toHaveBeenCalled();
  });

  it('restore() fetches /me when a token exists', async () => {
    mockToken = 'existing-tok';
    (api.get as any).mockResolvedValue({
      data: { id: 1, email: 'a@b.c', name: 'A' },
    });

    const store = useAuthStore();
    await store.restore();

    expect(api.get).toHaveBeenCalledWith('/api/auth/me');
    expect(store.isAuthenticated).toBe(true);
    expect(store.admin?.email).toBe('a@b.c');
  });

  it('restore() clears token if /me fails', async () => {
    mockToken = 'bad-tok';
    (api.get as any).mockRejectedValue({ response: { status: 401 } });

    const store = useAuthStore();
    await store.restore();

    expect(store.isAuthenticated).toBe(false);
    expect(clearToken).toHaveBeenCalled();
  });

  it('restore() is a no-op without a token', async () => {
    mockToken = null;
    const store = useAuthStore();
    await store.restore();
    expect(api.get).not.toHaveBeenCalled();
    expect(store.isAuthenticated).toBe(false);
  });

  // readToken is exported; ensure it's referenced so TS `noUnusedLocals` is happy.
  void readToken;
});
