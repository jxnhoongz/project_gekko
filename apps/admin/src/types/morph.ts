export type InheritanceType = 'RECESSIVE' | 'CO_DOMINANT' | 'DOMINANT' | 'POLYGENIC';

export interface MorphComboTrait {
  trait_id: number;
  trait_name: string;
  trait_code: string;
  required_zygosity: 'HOM' | 'HET' | 'POSS_HET';
}

export interface MorphCombo {
  id: number;
  species_id: number;
  name: string;
  code: string;
  description: string;
  notes: string;
  example_photo_url: string;
  requirements: MorphComboTrait[];
}

export interface MorphCombosListResponse {
  combos: MorphCombo[];
  total: number;
}

export interface MorphComboTraitInput {
  trait_id: number;
  required_zygosity: 'HOM' | 'HET' | 'POSS_HET';
}

export interface MorphComboWritePayload {
  species_id: number;
  name: string;
  code: string;
  description: string;
  notes: string;
  example_photo_url: string;
  requirements: MorphComboTraitInput[];
}

export const INHERITANCE_TYPE_LABEL: Record<InheritanceType, string> = {
  RECESSIVE: 'Recessive',
  CO_DOMINANT: 'Co-Dominant',
  DOMINANT: 'Dominant',
  POLYGENIC: 'Polygenic',
};
