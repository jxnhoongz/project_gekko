import { useQuery } from '@tanstack/vue-query';
import { api } from '@/lib/api';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';
import type { PublicGeckoDetail, PublicGeckoListResponse } from '@/types/gecko';

export function usePublicGeckos() {
  return useQuery({
    queryKey: ['public', 'geckos'],
    queryFn: async () => {
      const { data } = await api.get<PublicGeckoListResponse>('/api/public/geckos');
      return data;
    },
    staleTime: 60_000,
  });
}

export function usePublicGecko(code: MaybeRef<string | null>) {
  return useQuery({
    queryKey: ['public', 'geckos', code],
    queryFn: async () => {
      const c = unref(code);
      if (!c) throw new Error('no code');
      const { data } = await api.get<PublicGeckoDetail>(`/api/public/geckos/${c}`);
      return data;
    },
    enabled: () => !!unref(code),
    staleTime: 60_000,
    retry: (failureCount, error: any) => {
      // Don't retry on 404 — gecko isn't available.
      if (error?.response?.status === 404) return false;
      return failureCount < 2;
    },
  });
}

// Convenience: latest N available geckos for the home teaser.
export function useAvailableTeaser(n = 3) {
  return useQuery({
    queryKey: ['public', 'geckos', 'teaser', n],
    queryFn: async () => {
      const { data } = await api.get<PublicGeckoListResponse>('/api/public/geckos');
      return data.geckos.slice(0, n);
    },
    staleTime: 60_000,
  });
}
