import type { InheritanceType } from './morph';
export type { InheritanceType };

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
  inheritance_type: InheritanceType;
  super_form_name: string;
  example_photo_url: string;
  notes: string;
}

export interface GeckoTrait {
  trait_id: number;
  trait_name: string;
  trait_code: string;
  zygosity: Zygosity;
  is_dominant: boolean;
  inheritance_type: InheritanceType;
  super_form_name: string;
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
  notes: string;
  morph_label: string;
  created_at: string;
  traits: GeckoTrait[];
  cover_photo_url: string | null;
  photos?: GeckoPhoto[];
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
