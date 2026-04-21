import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';
import { api } from '@/lib/api';
import type {
  Listing,
  ListingWritePayload,
  ListingsListResponse,
} from '@/types/listing';

/** Query key factory — keep all listing-related cache keys in one place
 *  so invalidation stays consistent with TanStack's partial-match semantics
 *  (invalidating `['listings']` fans out to list + every detail). */
export const listingKeys = {
  all: ['listings'] as const,
  list: () => [...listingKeys.all, 'list'] as const,
  detail: (id: number | string) => [...listingKeys.all, 'detail', id] as const,
};

export function useListings() {
  return useQuery({
    queryKey: listingKeys.list(),
    queryFn: async () => {
      const { data } = await api.get<ListingsListResponse>('/api/listings');
      return data;
    },
    staleTime: 30_000,
  });
}

export function useListing(id: MaybeRef<number | string | null>) {
  return useQuery({
    queryKey: listingKeys.detail(unref(id) ?? 0),
    queryFn: async () => {
      const v = unref(id);
      if (v === null || v === undefined || v === '') throw new Error('no id');
      const { data } = await api.get<Listing>(`/api/listings/${v}`);
      return data;
    },
    enabled: () => {
      const v = unref(id);
      return v !== null && v !== undefined && v !== '';
    },
    staleTime: 30_000,
  });
}

function invalidateListings(
  qc: ReturnType<typeof useQueryClient>,
  id?: number,
) {
  qc.invalidateQueries({ queryKey: listingKeys.list() });
  if (id !== undefined) {
    qc.invalidateQueries({ queryKey: listingKeys.detail(id) });
  }
}

export function useCreateListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: ListingWritePayload) => {
      const { data } = await api.post<Listing>('/api/listings', payload);
      return data;
    },
    onSuccess: (l) => invalidateListings(qc, l.id),
  });
}

export function useUpdateListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      payload,
    }: {
      id: number;
      payload: ListingWritePayload;
    }) => {
      const { data } = await api.patch<Listing>(`/api/listings/${id}`, payload);
      return data;
    },
    onSuccess: (l) => invalidateListings(qc, l.id),
  });
}

export function useDeleteListing() {
  const qc = useQueryClient();
  return useMutation({
    // Backend returns 204 No Content — don't read response.data, axios
    // leaves it undefined and JSON parsing would throw.
    mutationFn: async (id: number) => {
      await api.delete(`/api/listings/${id}`);
      return id;
    },
    onSuccess: (id) => invalidateListings(qc, id),
  });
}
