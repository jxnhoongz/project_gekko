export interface DbColumn {
  name: string;
  type: string;
  nullable: boolean;
  is_pk: boolean;
  default: string | null;
}

export interface DbForeignKey {
  column: string;
  ref_table: string;
  ref_column: string;
  constraint: string;
}

export interface DbIndex {
  name: string;
  unique: boolean;
  primary: boolean;
  columns: string[];
}

export interface DbTable {
  name: string;
  row_count: number;
  columns: DbColumn[];
  foreign_keys: DbForeignKey[] | null;
  indexes: DbIndex[] | null;
}

export interface SchemaResponse {
  schema: string;
  tables: DbTable[];
}

export interface RowsResponse {
  table: string;
  columns: string[];
  types: string[];
  rows: (string | null)[][];
  total: number;
  limit: number;
  offset: number;
}
