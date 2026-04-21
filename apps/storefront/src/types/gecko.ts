export interface PublicGecko {
  code: string;
  name: string;
  species_code: string;
  species_name: string;
  morph: string;
  sex: 'M' | 'F' | 'U';
  hatch_date: string | null;
  list_price_usd: string | null;
  cover_photo_url: string | null;
}

export interface PublicGeckoDetail extends PublicGecko {
  photos: { url: string; caption: string; display_order: number }[];
}

export interface PublicGeckoListResponse {
  geckos: PublicGecko[];
  total: number;
}
