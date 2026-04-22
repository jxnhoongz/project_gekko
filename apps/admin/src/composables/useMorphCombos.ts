import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';
import { api } from '@/lib/api';
import type {
  MorphCombo,
  MorphCombosListResponse,
  MorphComboWritePayload,
} from '@/types/morph';

export const morphComboKeys = {
  all: ['morph-combos'] as const,
  list: (speciesCode?: string) =>
    [...morphComboKeys.all, 'list', speciesCode ?? ''] as const,
  detail: (id: number | string) =>
    [...morphComboKeys.all, 'detail', id] as const,
};

export function useMorphCombos(speciesCode?: MaybeRef<string>) {
  return useQuery({
    queryKey: () => morphComboKeys.list(unref(speciesCode)),
    queryFn: async () => {
      const params = unref(speciesCode)
        ? { species_code: unref(speciesCode) }
        : undefined;
      const { data } = await api.get<MorphCombosListResponse>(
        '/api/morph-combos',
        { params },
      );
      return data;
    },
    staleTime: 60_000,
  });
}

export function useMorphCombo(id: MaybeRef<number | string | null>) {
  return useQuery({
    queryKey: morphComboKeys.detail(unref(id) ?? 0),
    queryFn: async () => {
      const v = unref(id);
      if (!v) throw new Error('no id');
      const { data } = await api.get<MorphCombo>(`/api/morph-combos/${v}`);
      return data;
    },
    enabled: () => !!unref(id),
    staleTime: 60_000,
  });
}

function invalidateMorphCombos(
  qc: ReturnType<typeof useQueryClient>,
  id?: number,
) {
  qc.invalidateQueries({ queryKey: morphComboKeys.all });
  if (id !== undefined) {
    qc.invalidateQueries({ queryKey: morphComboKeys.detail(id) });
  }
}

export function useCreateMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: MorphComboWritePayload) => {
      const { data } = await api.post<MorphCombo>('/api/morph-combos', payload);
      return data;
    },
    onSuccess: (mc) => invalidateMorphCombos(qc, mc.id),
  });
}

export function useUpdateMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      payload,
    }: {
      id: number;
      payload: MorphComboWritePayload;
    }) => {
      const { data } = await api.patch<MorphCombo>(
        `/api/morph-combos/${id}`,
        payload,
      );
      return data;
    },
    onSuccess: (mc) => invalidateMorphCombos(qc, mc.id),
  });
}

export function useDeleteMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/api/morph-combos/${id}`);
      return id;
    },
    onSuccess: (id) => invalidateMorphCombos(qc, id),
  });
}
