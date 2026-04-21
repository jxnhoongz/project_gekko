export type Sex = 'M' | 'F' | 'U';
export type Zygosity = 'HOM' | 'HET' | 'POSS_HET';
export type GeckoStatus =
  | 'AVAILABLE'
  | 'HOLD'
  | 'BREEDING'
  | 'PERSONAL'
  | 'SOLD'
  | 'DECEASED';

export interface Species {
  id: number;
  code: string;
  common_name: string;
  scientific_name: string;
  description: string;
}

export interface Trait {
  id: number;
  species_id: number;
  trait_name: string;
  trait_code: string;
  description: string;
  is_dominant: boolean;
}

export interface GeckoTrait {
  trait_id: number;
  trait_name: string;
  trait_code: string;
  zygosity: Zygosity;
  is_dominant: boolean;
}

export interface GeckoPhoto {
  id: number;
  url: string;
  type: 'PROFILE' | 'GALLERY' | 'HUSBANDRY';
  caption: string;
  display_order: number;
}

export interface Gecko {
  id: number;
  code: string;
  name: string;
  species_id: number;
  species_code: string;
  species_name: string;
  sex: Sex;
  hatch_date: string | null;
  acquired_date: string | null;
  status: GeckoStatus;
  sire_id: number | null;
  dam_id: number | null;
  list_price_usd: string | null;
  notes: string;
  created_at: string;
  traits: GeckoTrait[];
  cover_photo_url: string | null;
  photos?: GeckoPhoto[];
}

/** Derive a human-readable morph string from trait list. */
export function morphFromTraits(traits: GeckoTrait[]): string {
  if (!traits.length) return 'Normal';
  const hom = traits.filter((t) => t.zygosity === 'HOM').map((t) => t.trait_name);
  const het = traits.filter((t) => t.zygosity === 'HET').map((t) => t.trait_name);
  const possHet = traits
    .filter((t) => t.zygosity === 'POSS_HET')
    .map((t) => t.trait_name);

  const parts: string[] = [];
  if (hom.length) parts.push(hom.join(' '));
  if (het.length) parts.push(het.map((n) => `het ${n}`).join(' '));
  if (possHet.length) parts.push(possHet.map((n) => `poss. het ${n}`).join(' '));
  return parts.join(', ') || 'Normal';
}

export const SEX_LABEL: Record<Sex, string> = {
  M: 'Male',
  F: 'Female',
  U: 'Unsexed',
};

export const STATUS_LABEL: Record<GeckoStatus, string> = {
  AVAILABLE: 'Available',
  HOLD: 'Hold',
  BREEDING: 'Breeding',
  PERSONAL: 'Personal',
  SOLD: 'Sold',
  DECEASED: 'Deceased',
};
