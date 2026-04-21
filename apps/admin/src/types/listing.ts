export type ListingType = 'GECKO' | 'PACKAGE' | 'SUPPLY';
export type ListingStatus =
  | 'DRAFT'
  | 'LISTED'
  | 'RESERVED'
  | 'SOLD'
  | 'ARCHIVED';

/** Reference to a gecko attached to a GECKO-type listing.
 *  Backend `listingGeckoRefDTO` — `name` is "" when the gecko has no name
 *  (textOrEmpty on the server side). */
export interface ListingGeckoRef {
  gecko_id: number;
  code: string;
  name: string;
  species_code: string;
}

/** Reference to a component listing inside a PACKAGE-type listing.
 *  Backend `listingComponentRefDTO` resolves title/type/price via join so
 *  the UI can render a package breakdown without a second fetch. */
export interface ListingComponentRef {
  component_listing_id: number;
  title: string;
  type: ListingType;
  price_usd: string;
  quantity: number;
}

/** /api/listings list+detail DTO. Mirrors backend `listingDTO`.
 *  Note: `sku`, `description`, `cover_photo_url` are always strings —
 *  the backend sends "" for NULL via textOrEmpty(). `deposit_usd` is the
 *  only nullable string (pointer on the Go side).
 *  `geckos` / `components` are present on GET detail and absent on list
 *  (handler uses `omitempty`). */
export interface Listing {
  id: number;
  sku: string;
  type: ListingType;
  title: string;
  description: string;
  price_usd: string;
  deposit_usd: string | null;
  status: ListingStatus;
  cover_photo_url: string;
  listed_at: string | null;
  sold_at: string | null;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
  gecko_count: number;
  component_count: number;
  geckos?: ListingGeckoRef[];
  components?: ListingComponentRef[];
}

/** POST/PATCH body. Matches backend `createListingReq`. The backend trims
 *  strings and treats "" as NULL for sku/description/cover_photo_url/
 *  deposit_usd, so plain strings are fine here. */
export interface ListingWritePayload {
  sku: string;
  type: ListingType;
  title: string;
  description: string;
  price_usd: string;
  deposit_usd: string;
  status: ListingStatus | '';
  cover_photo_url: string;
  geckos?: { gecko_id: number }[];
  components?: { component_listing_id: number; quantity: number }[];
}

export interface ListingsListResponse {
  listings: Listing[];
  total: number;
}

export const LISTING_TYPE_LABEL: Record<ListingType, string> = {
  GECKO: 'Gecko',
  PACKAGE: 'Package',
  SUPPLY: 'Supply',
};

export const LISTING_STATUS_LABEL: Record<ListingStatus, string> = {
  DRAFT: 'Draft',
  LISTED: 'Listed',
  RESERVED: 'Reserved',
  SOLD: 'Sold',
  ARCHIVED: 'Archived',
};
