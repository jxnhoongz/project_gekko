package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
)

// MountSchema registers admin-only schema introspection + row browser.
func MountSchema(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &schemaDeps{pool: pool}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/admin/schema", d.schema)
		pr.Get("/api/admin/table/{name}", d.rows)
	})
}

type schemaDeps struct {
	pool *pgxpool.Pool
}

type columnDTO struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	IsPK     bool    `json:"is_pk"`
	Default  *string `json:"default"`
}

type foreignKeyDTO struct {
	Column     string `json:"column"`
	RefTable   string `json:"ref_table"`
	RefColumn  string `json:"ref_column"`
	Constraint string `json:"constraint"`
}

type indexDTO struct {
	Name    string   `json:"name"`
	Unique  bool     `json:"unique"`
	Primary bool     `json:"primary"`
	Columns []string `json:"columns"`
}

type tableDTO struct {
	Name        string          `json:"name"`
	RowCount    int64           `json:"row_count"`
	Columns     []columnDTO     `json:"columns"`
	ForeignKeys []foreignKeyDTO `json:"foreign_keys"`
	Indexes     []indexDTO      `json:"indexes"`
}

type schemaResp struct {
	Schema string     `json:"schema"`
	Tables []tableDTO `json:"tables"`
}

// schema lists every user table in the `public` schema.
func (d *schemaDeps) schema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tables, err := listTables(ctx, d.pool)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "schema list failed: " + err.Error()})
		return
	}

	cols, err := listColumns(ctx, d.pool)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "column list failed: " + err.Error()})
		return
	}

	fks, err := listForeignKeys(ctx, d.pool)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fk list failed: " + err.Error()})
		return
	}

	idxs, err := listIndexes(ctx, d.pool)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "index list failed: " + err.Error()})
		return
	}

	out := make([]tableDTO, 0, len(tables))
	for _, t := range tables {
		count, err := rowCount(ctx, d.pool, t)
		if err != nil {
			count = -1
		}
		out = append(out, tableDTO{
			Name:        t,
			RowCount:    count,
			Columns:     cols[t],
			ForeignKeys: fks[t],
			Indexes:     idxs[t],
		})
	}

	writeJSON(w, http.StatusOK, schemaResp{Schema: "public", Tables: out})
}

// Row-browser. Validates `name` against known tables, then runs
// SELECT * FROM <tbl> ORDER BY <pk> LIMIT $1 OFFSET $2.
type rowsResp struct {
	Table   string       `json:"table"`
	Columns []string     `json:"columns"`
	Types   []string     `json:"types"`
	Rows    [][]*string  `json:"rows"`
	Total   int64        `json:"total"`
	Limit   int32        `json:"limit"`
	Offset  int32        `json:"offset"`
}

var nameSafe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (d *schemaDeps) rows(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !nameSafe.MatchString(name) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid table name"})
		return
	}

	ctx := r.Context()
	tables, err := listTables(ctx, d.pool)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "schema list failed: " + err.Error()})
		return
	}
	ok := false
	for _, t := range tables {
		if t == name {
			ok = true
			break
		}
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
		return
	}

	limit := parseInt32(r.URL.Query().Get("limit"), 50, 1, 200)
	offset := parseInt32(r.URL.Query().Get("offset"), 0, 0, 1_000_000)

	total, err := rowCount(ctx, d.pool, name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count failed: " + err.Error()})
		return
	}

	pk := primaryKeyColumns(ctx, d.pool, name)
	order := "1"
	if len(pk) > 0 && nameSafe.MatchString(pk[0]) {
		order = quoteIdent(pk[0])
	}

	// Safe: `name` and `order` have been validated against regex + schema allowlist.
	q := `SELECT * FROM ` + quoteIdent(name) +
		` ORDER BY ` + order +
		` LIMIT $1 OFFSET $2`

	rowsRes, err := d.pool.Query(ctx, q, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed: " + err.Error()})
		return
	}
	defer rowsRes.Close()

	fd := rowsRes.FieldDescriptions()
	colNames := make([]string, len(fd))
	colTypes := make([]string, len(fd))
	for i, f := range fd {
		colNames[i] = string(f.Name)
		colTypes[i] = strconv.FormatUint(uint64(f.DataTypeOID), 10)
	}

	out := make([][]*string, 0, limit)
	for rowsRes.Next() {
		vals, err := rowsRes.Values()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "row read failed: " + err.Error()})
			return
		}
		row := make([]*string, len(vals))
		for i, v := range vals {
			if v == nil {
				row[i] = nil
				continue
			}
			s := stringifyValue(v)
			row[i] = &s
		}
		out = append(out, row)
	}
	if err := rowsRes.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "iteration failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, rowsResp{
		Table:   name,
		Columns: colNames,
		Types:   colTypes,
		Rows:    out,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// --- helpers (DB-side queries) ---

func listTables(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func listColumns(ctx context.Context, pool *pgxpool.Pool) (map[string][]columnDTO, error) {
	const q = `
		SELECT
			c.table_name,
			c.column_name,
			CASE
				WHEN c.data_type = 'character varying' AND c.character_maximum_length IS NOT NULL
				  THEN 'varchar(' || c.character_maximum_length || ')'
				WHEN c.data_type = 'character' AND c.character_maximum_length IS NOT NULL
				  THEN 'char(' || c.character_maximum_length || ')'
				WHEN c.data_type = 'numeric' AND c.numeric_precision IS NOT NULL
				  THEN 'numeric(' || c.numeric_precision || ',' || COALESCE(c.numeric_scale, 0) || ')'
				ELSE c.data_type
			END AS col_type,
			c.is_nullable = 'YES' AS nullable,
			c.column_default,
			COALESCE(pk.is_pk, FALSE) AS is_pk
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT
				tc.table_name,
				kcu.column_name,
				TRUE AS is_pk
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON kcu.constraint_name = tc.constraint_name
			 AND kcu.table_schema   = tc.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = 'public'
		) pk ON pk.table_name = c.table_name AND pk.column_name = c.column_name
		WHERE c.table_schema = 'public'
		ORDER BY c.table_name, c.ordinal_position`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]columnDTO{}
	for rows.Next() {
		var (
			table, name, typ string
			nullable, isPK   bool
			def              *string
		)
		if err := rows.Scan(&table, &name, &typ, &nullable, &def, &isPK); err != nil {
			return nil, err
		}
		out[table] = append(out[table], columnDTO{
			Name:     name,
			Type:     typ,
			Nullable: nullable,
			IsPK:     isPK,
			Default:  def,
		})
	}
	return out, rows.Err()
}

func listForeignKeys(ctx context.Context, pool *pgxpool.Pool) (map[string][]foreignKeyDTO, error) {
	const q = `
		SELECT
			tc.constraint_name,
			tc.table_name,
			kcu.column_name,
			ccu.table_name  AS ref_table,
			ccu.column_name AS ref_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_name = tc.constraint_name
		 AND kcu.table_schema   = tc.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name = tc.constraint_name
		 AND ccu.table_schema    = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = 'public'
		ORDER BY tc.table_name, kcu.ordinal_position`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]foreignKeyDTO{}
	for rows.Next() {
		var cname, table, col, refT, refC string
		if err := rows.Scan(&cname, &table, &col, &refT, &refC); err != nil {
			return nil, err
		}
		out[table] = append(out[table], foreignKeyDTO{
			Constraint: cname, Column: col, RefTable: refT, RefColumn: refC,
		})
	}
	return out, rows.Err()
}

func listIndexes(ctx context.Context, pool *pgxpool.Pool) (map[string][]indexDTO, error) {
	const q = `
		SELECT
			t.relname  AS table_name,
			i.relname  AS index_name,
			ix.indisunique,
			ix.indisprimary,
			array_agg(a.attname ORDER BY k.ord) AS cols
		FROM pg_class t
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_index ix    ON t.oid = ix.indrelid
		JOIN pg_class i     ON i.oid = ix.indexrelid
		JOIN unnest(ix.indkey) WITH ORDINALITY AS k(attnum, ord) ON TRUE
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
		WHERE n.nspname = 'public' AND t.relkind = 'r'
		GROUP BY t.relname, i.relname, ix.indisunique, ix.indisprimary
		ORDER BY t.relname, i.relname`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]indexDTO{}
	for rows.Next() {
		var (
			table, idx string
			uniq, prim bool
			cols       []string
		)
		if err := rows.Scan(&table, &idx, &uniq, &prim, &cols); err != nil {
			return nil, err
		}
		out[table] = append(out[table], indexDTO{
			Name: idx, Unique: uniq, Primary: prim, Columns: cols,
		})
	}
	return out, rows.Err()
}

func primaryKeyColumns(ctx context.Context, pool *pgxpool.Pool, table string) []string {
	const q = `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_name = tc.constraint_name
		 AND kcu.table_schema   = tc.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema    = 'public'
		  AND tc.table_name      = $1
		ORDER BY kcu.ordinal_position`
	rows, err := pool.Query(ctx, q, table)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			out = append(out, s)
		}
	}
	return out
}

func rowCount(ctx context.Context, pool *pgxpool.Pool, table string) (int64, error) {
	if !nameSafe.MatchString(table) {
		return 0, errors.New("invalid table name")
	}
	q := `SELECT COUNT(*) FROM ` + quoteIdent(table)
	var n int64
	if err := pool.QueryRow(ctx, q).Scan(&n); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func quoteIdent(s string) string {
	// Simple identifier quoter. `nameSafe` regex already forbids quotes, so
	// this is just wrapping. Double any embedded quotes defensively.
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '"', '"')
			continue
		}
		out = append(out, s[i])
	}
	out = append(out, '"')
	return string(out)
}

func stringifyValue(v any) string {
	switch x := v.(type) {
	case []byte:
		return string(x)
	case string:
		return x
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		s := string(b)
		// Unwrap JSON-string quoting so the frontend can display plain strings.
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			var unq string
			if err := json.Unmarshal(b, &unq); err == nil {
				return unq
			}
		}
		return s
	}
}
