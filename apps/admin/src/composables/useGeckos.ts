import { useQuery } from '@tanstack/vue-query';
import { api } from '@/lib/api';
import type { Gecko, Trait, Species } from '@/types/gecko';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';

export function useGeckos() {
  return useQuery({
    queryKey: ['geckos'],
    queryFn: async () => {
      const { data } = await api.get<{ geckos: Gecko[]; total: number }>('/api/geckos');
      return data;
    },
    staleTime: 30_000,
  });
}

export function useGecko(id: MaybeRef<number | string | null>) {
  return useQuery({
    queryKey: ['geckos', id],
    queryFn: async () => {
      const v = unref(id);
      if (v === null || v === undefined) throw new Error('no id');
      const { data } = await api.get<Gecko>(`/api/geckos/${v}`);
      return data;
    },
    enabled: () => unref(id) !== null && unref(id) !== undefined && unref(id) !== '',
    staleTime: 30_000,
  });
}

export function useSpecies() {
  return useQuery({
    queryKey: ['species'],
    queryFn: async () => {
      const { data } = await api.get<{ species: Species[] }>('/api/species');
      return data.species;
    },
    staleTime: 5 * 60_000,
  });
}

export function useTraits(speciesId?: MaybeRef<number | null>) {
  return useQuery({
    queryKey: ['traits', speciesId],
    queryFn: async () => {
      const sp = unref(speciesId);
      const params = sp ? { species_id: sp } : undefined;
      const { data } = await api.get<{ traits: Trait[] }>('/api/traits', { params });
      return data.traits;
    },
    staleTime: 5 * 60_000,
  });
}
