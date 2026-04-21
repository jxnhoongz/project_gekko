import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { api } from '@/lib/api';
import type { Gecko, Trait, Species } from '@/types/gecko';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';

export interface GeckoWritePayload {
  name: string;
  species_id: number;
  sex: 'M' | 'F' | 'U';
  hatch_date: string;
  acquired_date: string;
  status: string;
  sire_id: number | null;
  dam_id: number | null;
  list_price_usd: string;
  notes: string;
  traits: { trait_id: number; zygosity: 'HOM' | 'HET' | 'POSS_HET' }[];
}

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

function invalidateGeckos(qc: ReturnType<typeof useQueryClient>, id?: number) {
  qc.invalidateQueries({ queryKey: ['geckos'] });
  if (id !== undefined) qc.invalidateQueries({ queryKey: ['geckos', id] });
}

export function useCreateGecko() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: GeckoWritePayload) => {
      const { data } = await api.post<Gecko>('/api/geckos', payload);
      return data;
    },
    onSuccess: (g) => invalidateGeckos(qc, g.id),
  });
}

export function useUpdateGecko() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, payload }: { id: number; payload: GeckoWritePayload }) => {
      const { data } = await api.patch<Gecko>(`/api/geckos/${id}`, payload);
      return data;
    },
    onSuccess: (g) => invalidateGeckos(qc, g.id),
  });
}

export function useDeleteGecko() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/api/geckos/${id}`);
      return id;
    },
    onSuccess: (id) => invalidateGeckos(qc, id),
  });
}
