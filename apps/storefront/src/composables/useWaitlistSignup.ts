import { useMutation } from '@tanstack/vue-query';
import { api } from '@/lib/api';

export interface WaitlistPayload {
  email: string;
  telegram?: string;
  phone?: string;
  interested_in?: string;
  notes?: string;
}

export interface WaitlistResult {
  id?: number;
  deduplicated?: boolean;
}

export function useWaitlistSignup() {
  return useMutation({
    mutationFn: async (payload: WaitlistPayload) => {
      const { data } = await api.post<WaitlistResult>('/api/public/waitlist', payload);
      return data;
    },
  });
}
