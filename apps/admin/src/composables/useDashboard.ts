import { useQuery } from '@tanstack/vue-query';
import { api } from '@/lib/api';

export interface DashboardStats {
  total_geckos: number;
  breeding: number;
  available: number;
  waitlist: number;
}

export type DashboardRefKind = 'gecko' | 'waitlist';

export type DashboardItemKind =
  | 'waitlist_stale'
  | 'hold_stale'
  | 'gecko_created'
  | 'waitlist_created'
  | 'media_uploaded';

export interface DashboardItem {
  kind: DashboardItemKind;
  title: string;
  detail: string;
  at: string;
  ref_kind: DashboardRefKind;
  ref_id: number;
}

export interface DashboardData {
  stats: DashboardStats;
  needs_attention: DashboardItem[];
  recent_activity: DashboardItem[];
}

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: async () => {
      const { data } = await api.get<DashboardData>('/api/admin/dashboard');
      return data;
    },
    staleTime: 30_000,
  });
}
