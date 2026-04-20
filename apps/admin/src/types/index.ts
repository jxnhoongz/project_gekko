export type Species = 'Leopard Gecko' | 'Crested Gecko' | 'African Fat-Tail';
export type Sex = 'Male' | 'Female' | 'Unsexed';
export type GeckoStatus = 'Available' | 'Hold' | 'Breeding' | 'Personal' | 'Sold';

export interface Gecko {
  id: string;
  code: string;
  name: string;
  species: Species;
  morph: string;
  sex: Sex;
  bornAt: string;
  status: GeckoStatus;
  weightG: number;
  priceUsd?: number;
  photoUrl?: string;
  notes?: string;
}

export interface Feeding {
  id: string;
  geckoId: string;
  at: string;
  prey: string;
  quantity: number;
  note?: string;
}

export interface WeightEntry {
  id: string;
  geckoId: string;
  at: string;
  grams: number;
}

export type ShedCompleteness = 'Complete' | 'Partial' | 'Stuck';

export interface Shed {
  id: string;
  geckoId: string;
  at: string;
  completeness: ShedCompleteness;
  note?: string;
}

export interface HealthLog {
  id: string;
  geckoId: string;
  at: string;
  title: string;
  severity: 'Note' | 'Watch' | 'Alert';
  detail: string;
}

export type ActivityKind = 'feeding' | 'weight' | 'shed' | 'waitlist' | 'sale' | 'health';

export interface ActivityItem {
  id: string;
  kind: ActivityKind;
  title: string;
  detail: string;
  at: string;
  geckoId?: string;
}

export type UpcomingKind = 'feeding' | 'weigh' | 'shed-check' | 'pairing';

export interface UpcomingAction {
  id: string;
  kind: UpcomingKind;
  title: string;
  detail: string;
  dueAt: string;
  geckoId?: string;
  overdue?: boolean;
}
