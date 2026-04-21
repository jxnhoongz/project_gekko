import type { Feeding, WeightEntry, Shed, HealthLog } from '@/types';

function daysAgo(n: number, hour = 19, minute = 0): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  d.setHours(hour, minute, 0, 0);
  return d.toISOString();
}

export const feedings: Feeding[] = [
  { id: 'fd_01', geckoId: 'gk_01', at: daysAgo(1), prey: 'Dubia roach', quantity: 4 },
  { id: 'fd_02', geckoId: 'gk_01', at: daysAgo(4), prey: 'Mealworm', quantity: 6 },
  { id: 'fd_03', geckoId: 'gk_01', at: daysAgo(7), prey: 'Dubia roach', quantity: 5 },
  { id: 'fd_04', geckoId: 'gk_02', at: daysAgo(2), prey: 'Dubia roach', quantity: 5 },
  { id: 'fd_05', geckoId: 'gk_02', at: daysAgo(5), prey: 'Cricket', quantity: 6 },
  { id: 'fd_06', geckoId: 'gk_03', at: daysAgo(1), prey: 'Pangea CGD', quantity: 3, note: '3ml, finished' },
  { id: 'fd_07', geckoId: 'gk_03', at: daysAgo(3), prey: 'Pangea CGD', quantity: 3 },
  { id: 'fd_08', geckoId: 'gk_04', at: daysAgo(2), prey: 'Pangea CGD', quantity: 3 },
  { id: 'fd_09', geckoId: 'gk_05', at: daysAgo(1), prey: 'Dubia roach', quantity: 4 },
  { id: 'fd_10', geckoId: 'gk_06', at: daysAgo(2), prey: 'Mealworm', quantity: 5 },
];

export const weights: WeightEntry[] = [
  { id: 'wt_01', geckoId: 'gk_01', at: daysAgo(28), grams: 68 },
  { id: 'wt_02', geckoId: 'gk_01', at: daysAgo(14), grams: 70 },
  { id: 'wt_03', geckoId: 'gk_01', at: daysAgo(1),  grams: 72 },
  { id: 'wt_04', geckoId: 'gk_02', at: daysAgo(28), grams: 65 },
  { id: 'wt_05', geckoId: 'gk_02', at: daysAgo(14), grams: 67 },
  { id: 'wt_06', geckoId: 'gk_02', at: daysAgo(2),  grams: 68 },
  { id: 'wt_07', geckoId: 'gk_03', at: daysAgo(30), grams: 40 },
  { id: 'wt_08', geckoId: 'gk_03', at: daysAgo(7),  grams: 42 },
  { id: 'wt_09', geckoId: 'gk_04', at: daysAgo(30), grams: 34 },
  { id: 'wt_10', geckoId: 'gk_04', at: daysAgo(3),  grams: 38 },
  { id: 'wt_11', geckoId: 'gk_05', at: daysAgo(21), grams: 52 },
  { id: 'wt_12', geckoId: 'gk_05', at: daysAgo(2),  grams: 54 },
  { id: 'wt_13', geckoId: 'gk_06', at: daysAgo(30), grams: 30 },
  { id: 'wt_14', geckoId: 'gk_06', at: daysAgo(5),  grams: 34 },
];

export const sheds: Shed[] = [
  { id: 'sh_01', geckoId: 'gk_01', at: daysAgo(9),  completeness: 'Complete' },
  { id: 'sh_02', geckoId: 'gk_02', at: daysAgo(12), completeness: 'Complete' },
  { id: 'sh_03', geckoId: 'gk_03', at: daysAgo(6),  completeness: 'Partial', note: 'Toe stuck, soaked and cleared' },
  { id: 'sh_04', geckoId: 'gk_04', at: daysAgo(15), completeness: 'Complete' },
  { id: 'sh_05', geckoId: 'gk_05', at: daysAgo(20), completeness: 'Complete' },
  { id: 'sh_06', geckoId: 'gk_06', at: daysAgo(8),  completeness: 'Complete' },
];

export const healthLogs: HealthLog[] = [
  {
    id: 'hl_01',
    geckoId: 'gk_03',
    at: daysAgo(6),
    title: 'Stuck shed on rear toe',
    severity: 'Watch',
    detail: 'Soaked 10 min, cleared with damp q-tip. Re-check next shed.',
  },
  {
    id: 'hl_02',
    geckoId: 'gk_06',
    at: daysAgo(18),
    title: 'Light appetite',
    severity: 'Note',
    detail: 'Ate half portion for 3 days. Temps checked — now back to normal.',
  },
];
