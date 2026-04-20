import type { Feeding, WeightEntry, Shed, HealthLog, ActivityItem, UpcomingAction } from '@/types';

function daysAgo(n: number, hour = 19, minute = 0): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  d.setHours(hour, minute, 0, 0);
  return d.toISOString();
}

function hoursFromNow(n: number): string {
  const d = new Date();
  d.setHours(d.getHours() + n, 0, 0, 0);
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

export const activity: ActivityItem[] = [
  { id: 'ac_01', kind: 'waitlist', title: 'New waitlist entry',   detail: 'Kosal Ly — Crested / harlequin',  at: daysAgo(0, 9, 15) },
  { id: 'ac_02', kind: 'feeding',  title: 'Fed Apsara',           detail: '4× Dubia roach',                   at: daysAgo(1, 19, 30), geckoId: 'gk_01' },
  { id: 'ac_03', kind: 'weight',   title: 'Weight: Apsara',       detail: '72 g (+2)',                        at: daysAgo(1, 20, 0),  geckoId: 'gk_01' },
  { id: 'ac_04', kind: 'feeding',  title: 'Fed Chandra',          detail: 'Pangea CGD, 3 ml — finished',      at: daysAgo(1, 20, 45), geckoId: 'gk_03' },
  { id: 'ac_05', kind: 'feeding',  title: 'Fed Khmer',            detail: '4× Dubia roach',                   at: daysAgo(1, 21, 0),  geckoId: 'gk_05' },
  { id: 'ac_06', kind: 'feeding',  title: 'Fed Rithy',            detail: '5× Dubia roach',                   at: daysAgo(2, 19, 45), geckoId: 'gk_02' },
  { id: 'ac_07', kind: 'health',   title: 'Health note: Chandra', detail: 'Stuck shed cleared',               at: daysAgo(6, 22, 10), geckoId: 'gk_03' },
  { id: 'ac_08', kind: 'shed',     title: 'Shed: Apsara',         detail: 'Complete shed',                     at: daysAgo(9, 8, 0),   geckoId: 'gk_01' },
];

export const upcoming: UpcomingAction[] = [
  { id: 'up_01', kind: 'feeding',     title: 'Feed Apsara',        detail: '3 days since last feed',    dueAt: hoursFromNow(2),  geckoId: 'gk_01' },
  { id: 'up_02', kind: 'feeding',     title: 'Feed Rithy',         detail: '2 days since last feed',    dueAt: hoursFromNow(4),  geckoId: 'gk_02' },
  { id: 'up_03', kind: 'weigh',       title: 'Weigh Suri',         detail: 'Weekly weigh-in due',       dueAt: hoursFromNow(20), geckoId: 'gk_04' },
  { id: 'up_04', kind: 'shed-check',  title: 'Shed check: Khmer',  detail: 'Last shed 20 days ago',     dueAt: hoursFromNow(-6), geckoId: 'gk_05', overdue: true },
  { id: 'up_05', kind: 'pairing',     title: 'Pair Apsara × Rithy', detail: 'Introduce for 48h cycle',  dueAt: hoursFromNow(48), geckoId: 'gk_01' },
];
