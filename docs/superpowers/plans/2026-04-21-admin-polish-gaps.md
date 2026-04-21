# Admin Polish Gaps — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close four small UX gaps in the admin — sire/dam searchable picker, set-as-cover on photos, inline caption edit, and upload success toast — with one new generic media PATCH endpoint plus one specialized set-cover endpoint.

**Architecture:** Backend adds `PATCH /api/media/{id}` (caption + display_order) and `POST /api/media/{id}/set-cover` (atomic re-sequence). Frontend adds a reusable `GeckoPicker.vue` combobox component, two TanStack Query mutations (`useUpdateMedia`, `useSetCoverMedia`), and wires sire/dam + ⭐ cover + caption-ribbon into the existing edit drawer.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, sqlc@v1.27, goose; Vue 3.5, TypeScript, TanStack Vue Query, Vitest, vue-tsc.

**Spec:** `docs/superpowers/specs/2026-04-21-admin-polish-gaps-design.md`

---

## File Structure

**Backend (modified):**
- `backend/internal/queries/media.sql` — add `UpdateMedia`, `UpdateMediaDisplayOrder`, `ListMediaForGeckoRaw` queries (the last for the set-cover re-sequence loop).
- `backend/internal/http/media.go` — append `patch` + `setCover` handlers and register their routes in `MountMedia`.
- `backend/internal/http/media_test.go` — **new file**; covers both handlers.
- `backend/internal/db/media.sql.go` — **generated** by `sqlc generate`, do not hand-edit.

**Frontend (new):**
- `apps/admin/src/components/GeckoPicker.vue` — reusable combobox for sire/dam.

**Frontend (modified):**
- `apps/admin/src/composables/useGeckos.ts` — add `useUpdateMedia` + `useSetCoverMedia` mutations.
- `apps/admin/src/components/GeckoFormSheet.vue` — add sire/dam inputs, ⭐ set-cover button, caption ribbon + inline edit, upload success toast.

---

## Task 1: Backend — add `UpdateMedia` + set-cover-friendly queries

**Files:**
- Modify: `backend/internal/queries/media.sql`

- [ ] **Step 1: Append new queries**

Open `backend/internal/queries/media.sql` and append at the end (after the existing queries, preserving them):

```sql
-- name: UpdateMediaCaption :one
UPDATE media
SET caption = $2
WHERE id = $1
RETURNING id, gecko_id, url, type, caption, display_order, uploaded_at;

-- name: UpdateMediaDisplayOrder :one
UPDATE media
SET display_order = $2
WHERE id = $1
RETURNING id, gecko_id, url, type, caption, display_order, uploaded_at;

-- name: UpdateMediaCaptionAndOrder :one
UPDATE media
SET caption = $2, display_order = $3
WHERE id = $1
RETURNING id, gecko_id, url, type, caption, display_order, uploaded_at;

-- name: ListMediaIDsForGeckoOrdered :many
-- Returns the media ids for a gecko in the same (display_order, uploaded_at)
-- order the client sees. Used by set-cover to reassign display_order.
SELECT id
FROM media
WHERE gecko_id = $1
ORDER BY display_order, uploaded_at;
```

- [ ] **Step 2: Generate Go code**

Run: `cd /home/zen/dev/project_gekko/backend && /home/zen/go/bin/sqlc generate`
Expected: silent exit 0; `backend/internal/db/media.sql.go` regenerated with 4 new methods.

- [ ] **Step 3: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/queries/media.sql backend/internal/db/media.sql.go backend/internal/db/querier.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): sqlc queries for media caption + display_order updates

Three UPDATE variants (caption only, display_order only, both) keep
the handler simple — it picks the right one based on which fields the
client sent. Plus ListMediaIDsForGeckoOrdered for the set-cover
re-sequence pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Backend — `PATCH /api/media/{id}` handler

**Files:**
- Modify: `backend/internal/http/media.go`

- [ ] **Step 1: Register the new route**

Open `backend/internal/http/media.go`. Find the `MountMedia` function and add the `PATCH` route inside the authenticated group:

```go
func MountMedia(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner, cfg *config.Config) {
	d := &mediaDeps{pool: pool, q: db.New(pool), cfg: cfg}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Post("/api/geckos/{id}/media", d.upload)
		pr.Patch("/api/media/{id}", d.patch)
		pr.Delete("/api/media/{id}", d.delete)
	})
}
```

- [ ] **Step 2: Add the handler at the bottom of the file**

Append this function near the bottom of `backend/internal/http/media.go` (after `delete`, before the helpers):

```go
type patchMediaReq struct {
	Caption      *string `json:"caption"`
	DisplayOrder *int32  `json:"display_order"`
}

func (d *mediaDeps) patch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req patchMediaReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Caption == nil && req.DisplayOrder == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		return
	}

	if req.Caption != nil && len(*req.Caption) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "caption too long (max 500)"})
		return
	}
	if req.DisplayOrder != nil && (*req.DisplayOrder < 0 || *req.DisplayOrder > 10000) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_order out of range"})
		return
	}

	ctx := r.Context()
	var (
		row db.Medium
	)
	switch {
	case req.Caption != nil && req.DisplayOrder != nil:
		row, err = d.q.UpdateMediaCaptionAndOrder(ctx, db.UpdateMediaCaptionAndOrderParams{
			ID:           int32(id64),
			Caption:      pgText(*req.Caption),
			DisplayOrder: *req.DisplayOrder,
		})
	case req.Caption != nil:
		row, err = d.q.UpdateMediaCaption(ctx, db.UpdateMediaCaptionParams{
			ID:      int32(id64),
			Caption: pgText(*req.Caption),
		})
	case req.DisplayOrder != nil:
		row, err = d.q.UpdateMediaDisplayOrder(ctx, db.UpdateMediaDisplayOrderParams{
			ID:           int32(id64),
			DisplayOrder: *req.DisplayOrder,
		})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mediaDTO{
		ID:           row.ID,
		Url:          row.Url,
		Type:         string(row.Type),
		Caption:      textOrEmpty(row.Caption),
		DisplayOrder: row.DisplayOrder,
	})
}
```

If the file doesn't already import `"errors"` at the top, add it to the import block. Check existing imports and add only if missing.

- [ ] **Step 3: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Smoke-test with curl**

With the backend running on :8420 (air or `go run ./cmd/gekko`), run:
```bash
TOKEN=$(curl -sS -X POST http://localhost:8420/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"zen@zeneticgekkos.com","password":"gekko-dev-2026"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

# Get any existing media id to test against:
MID=$(curl -sS http://localhost:8420/api/geckos/1 -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json; d = json.load(sys.stdin)
print(d['photos'][0]['id'] if d.get('photos') else '')")
echo "media id: $MID"

# PATCH with empty body — should 400
curl -sS -X PATCH "http://localhost:8420/api/media/$MID" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{}' -w "\nstatus=%{http_code}\n"

# PATCH caption — should 200
curl -sS -X PATCH "http://localhost:8420/api/media/$MID" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"caption":"smoke-test"}' | python3 -m json.tool
```
Expected: first PATCH returns `status=400` with the "no fields to update" error; second PATCH returns the media DTO with the new caption.

If no geckos have photos, skip the live smoke — the unit tests in Task 4 cover the logic.

- [ ] **Step 5: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/media.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): PATCH /api/media/{id} for caption + display_order

Picks the right of three UPDATE queries based on which optional fields
the client sent. Validates caption length (<=500) and display_order
range (0..10000). Returns 400 on empty body, 404 on unknown id.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Backend — `POST /api/media/{id}/set-cover` handler

**Files:**
- Modify: `backend/internal/http/media.go`

- [ ] **Step 1: Register the new route**

In `MountMedia`, add the new POST route inside the authenticated group:

```go
func MountMedia(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner, cfg *config.Config) {
	d := &mediaDeps{pool: pool, q: db.New(pool), cfg: cfg}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Post("/api/geckos/{id}/media", d.upload)
		pr.Patch("/api/media/{id}", d.patch)
		pr.Post("/api/media/{id}/set-cover", d.setCover)
		pr.Delete("/api/media/{id}", d.delete)
	})
}
```

- [ ] **Step 2: Add the handler**

Append to `backend/internal/http/media.go` (near `patch`, before helpers):

```go
func (d *mediaDeps) setCover(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	targetID64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	targetID := int32(targetID64)

	ctx := r.Context()
	target, err := d.q.GetMediaByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}
	if !target.GeckoID.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "media is not attached to a gecko"})
		return
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	ids, err := qtx.ListMediaIDsForGeckoOrdered(ctx, target.GeckoID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed: " + err.Error()})
		return
	}

	// Build the desired sequence: target first, then everyone else in their
	// existing order.
	desired := make([]int32, 0, len(ids))
	desired = append(desired, targetID)
	for _, id := range ids {
		if id == targetID {
			continue
		}
		desired = append(desired, id)
	}

	for i, id := range desired {
		if _, err := qtx.UpdateMediaDisplayOrder(ctx, db.UpdateMediaDisplayOrderParams{
			ID:           id,
			DisplayOrder: int32(i),
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reorder failed: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/media.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): POST /api/media/{id}/set-cover atomic re-sequence

Moves the target photo to display_order=0 and shifts the rest
down by one in a single transaction, preserving relative order.
Clean integer sequence per-gecko, no drift / negatives.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Backend — tests for PATCH + set-cover

**Files:**
- Create: `backend/internal/http/media_test.go`

- [ ] **Step 1: Write the test file**

Create `backend/internal/http/media_test.go`:

```go
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func mediaSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool, *config.Config) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := "media+" + time.Now().Format("150405.000000000") + "@example.com"
	hash, err := auth.HashPassword("test-password-123")
	require.NoError(t, err)
	q := db.New(pool)
	admin, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email: email, PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", admin.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	tok, err := signer.Issue(int64(admin.ID), admin.Email)
	require.NoError(t, err)

	cfg := &config.Config{UploadDir: filepath.Join(t.TempDir(), "uploads")}
	r := chi.NewRouter()
	MountMedia(r, pool, signer, cfg)
	return r, tok, pool, cfg
}

// Helper: insert a media row directly via sqlc (avoids testing multipart in these unit tests).
func insertTestMedia(t *testing.T, pool *pgxpool.Pool, geckoID int32, url string, order int32) int32 {
	t.Helper()
	q := db.New(pool)
	m, err := q.CreateMedia(context.Background(), db.CreateMediaParams{
		GeckoID: pgtype.Int4{Int32: geckoID, Valid: true},
		Url:     url,
		Column3: db.NullMediaType{MediaType: db.MediaTypeGALLERY, Valid: true},
		Caption: pgtype.Text{},
		Column5: order,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM media WHERE id = $1", m.ID)
	})
	return m.ID
}

// Helper: needs a gecko row (media.gecko_id FK); creates one and cleans up.
func createTestGecko(t *testing.T, pool *pgxpool.Pool) int32 {
	t.Helper()
	q := db.New(pool)
	// Pick any existing species so the FK is satisfied.
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT id FROM species ORDER BY id LIMIT 1").Scan(&speciesID))

	code := "TEST-" + time.Now().Format("150405.000000000")
	g, err := q.CreateGecko(context.Background(), db.CreateGeckoParams{
		Code:      code,
		SpeciesID: speciesID,
		Sex:       db.SexU,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM geckos WHERE id = $1", g.ID)
	})
	return g.ID
}

func TestPatchMedia_empty(t *testing.T) {
	router, tok, _, _ := mediaSetup(t)

	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/999999", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "no fields to update")
}

func TestPatchMedia_captionOnly(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	mediaID := insertTestMedia(t, pool, geckoID, "/uploads/test/a.png", 0)

	body := bytes.NewReader([]byte(`{"caption":"a new caption"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/"+itoa(mediaID), body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got mediaDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "a new caption", got.Caption)
	assert.Equal(t, int32(0), got.DisplayOrder)
}

func TestPatchMedia_displayOrderOnly(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	mediaID := insertTestMedia(t, pool, geckoID, "/uploads/test/b.png", 5)

	body := bytes.NewReader([]byte(`{"display_order":9}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/"+itoa(mediaID), body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got mediaDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, int32(9), got.DisplayOrder)
}

func TestSetCover_reseqencesAllMedia(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	id1 := insertTestMedia(t, pool, geckoID, "/uploads/test/1.png", 0)
	id2 := insertTestMedia(t, pool, geckoID, "/uploads/test/2.png", 1)
	id3 := insertTestMedia(t, pool, geckoID, "/uploads/test/3.png", 2)

	req := httptest.NewRequest(http.MethodPost, "/api/media/"+itoa(id3)+"/set-cover", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code, "body=%s", rr.Body.String())

	// After set-cover on id3: id3 → 0, id1 → 1, id2 → 2.
	q := db.New(pool)
	m1, _ := q.GetMediaByID(context.Background(), id1)
	m2, _ := q.GetMediaByID(context.Background(), id2)
	m3, _ := q.GetMediaByID(context.Background(), id3)
	assert.Equal(t, int32(0), m3.DisplayOrder, "target becomes display_order=0")
	assert.Equal(t, int32(1), m1.DisplayOrder, "former first shifts to 1")
	assert.Equal(t, int32(2), m2.DisplayOrder, "former second shifts to 2")
}

func TestSetCover_notFound(t *testing.T) {
	router, tok, _, _ := mediaSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/media/999999/set-cover", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// itoa keeps the test imports light.
func itoa(n int32) string {
	return strconvItoa(int(n))
}
```

Also add one small helper file so the tests link. Create `backend/internal/http/testhelpers_test.go`:

```go
package http

import "strconv"

// strconvItoa is a tiny indirection because the test file avoids pulling
// the strconv import into the rest of the package. Kept in a separate
// *_test.go file so it doesn't ship in non-test builds.
func strconvItoa(i int) string { return strconv.Itoa(i) }
```

- [ ] **Step 2: Run the new tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./internal/http/... -run "PatchMedia|SetCover" -v`
Expected: 5 tests PASS — `TestPatchMedia_empty`, `TestPatchMedia_captionOnly`, `TestPatchMedia_displayOrderOnly`, `TestSetCover_reseqencesAllMedia`, `TestSetCover_notFound`.

- [ ] **Step 3: Run the full backend test suite**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages report `ok`.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/media_test.go backend/internal/http/testhelpers_test.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "test(backend): PATCH media + set-cover integration tests

Five tests covering empty patch (400), caption-only, display_order-only,
full set-cover re-sequence assertion, and set-cover on unknown id (404).
Uses real DB via DB_URL + a per-test gecko and media rows with cleanup.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Frontend — `useUpdateMedia` + `useSetCoverMedia` mutations

**Files:**
- Modify: `apps/admin/src/composables/useGeckos.ts`

- [ ] **Step 1: Append the two mutations**

Open `apps/admin/src/composables/useGeckos.ts`. At the end of the file, append:

```ts
export function useUpdateMedia() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      mediaId,
      geckoId,
      patch,
    }: {
      mediaId: number;
      geckoId: number;
      patch: { caption?: string; display_order?: number };
    }) => {
      const { data } = await api.patch(`/api/media/${mediaId}`, patch);
      return { geckoId, media: data };
    },
    onSuccess: ({ geckoId }) => invalidateGeckos(qc, geckoId),
  });
}

export function useSetCoverMedia() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      mediaId,
      geckoId,
    }: {
      mediaId: number;
      geckoId: number;
    }) => {
      await api.post(`/api/media/${mediaId}/set-cover`);
      return geckoId;
    },
    onSuccess: (geckoId) => invalidateGeckos(qc, geckoId),
  });
}
```

- [ ] **Step 2: Type-check**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds, no TS errors.

- [ ] **Step 3: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/composables/useGeckos.ts
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): useUpdateMedia + useSetCoverMedia mutations

Wraps PATCH /api/media/{id} and POST /api/media/{id}/set-cover for
the upcoming photo-tile UI. Each invalidates the gecko cache on
success so the detail view reflects new cover / caption without a
manual refetch.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Frontend — `GeckoPicker.vue` reusable combobox

**Files:**
- Create: `apps/admin/src/components/GeckoPicker.vue`

- [ ] **Step 1: Create the component**

Write `apps/admin/src/components/GeckoPicker.vue`:

```vue
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { X, ChevronDown } from 'lucide-vue-next';
import { useGeckos } from '@/composables/useGeckos';
import type { Sex } from '@/types/gecko';

const props = defineProps<{
  modelValue: number | null;
  speciesId: number | null;
  sex: Sex;           // 'M' | 'F' (sire / dam). 'U' accepted for completeness.
  excludeId?: number; // don't allow self-selection
  placeholder?: string;
}>();

const emit = defineEmits<{ (e: 'update:modelValue', v: number | null): void }>();

const { data: geckosData } = useGeckos();
const allGeckos = computed(() => geckosData.value?.geckos ?? []);

const open = ref(false);
const query = ref('');
const inputRef = ref<HTMLInputElement | null>(null);

const candidates = computed(() => {
  if (props.speciesId === null) return [];
  return allGeckos.value.filter(
    (g) =>
      g.species_id === props.speciesId &&
      g.sex === props.sex &&
      g.id !== props.excludeId,
  );
});

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return candidates.value;
  return candidates.value.filter(
    (g) =>
      g.code.toLowerCase().includes(q) ||
      (g.name ?? '').toLowerCase().includes(q),
  );
});

const selected = computed(() =>
  allGeckos.value.find((g) => g.id === props.modelValue) ?? null,
);

const display = computed(() => {
  if (!selected.value) return '';
  return `${selected.value.code}${selected.value.name ? ' · ' + selected.value.name : ''}`;
});

function pick(id: number) {
  emit('update:modelValue', id);
  open.value = false;
  query.value = '';
}

function clear(e: Event) {
  e.stopPropagation();
  emit('update:modelValue', null);
  query.value = '';
}

function openPicker() {
  if (props.speciesId === null) return;
  open.value = true;
  setTimeout(() => inputRef.value?.focus(), 0);
}

// Close on outside click
const wrapperRef = ref<HTMLDivElement | null>(null);
function onDocClick(e: MouseEvent) {
  if (!wrapperRef.value) return;
  if (!wrapperRef.value.contains(e.target as Node)) open.value = false;
}

watch(open, (v) => {
  if (v) document.addEventListener('mousedown', onDocClick);
  else document.removeEventListener('mousedown', onDocClick);
});
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <!-- Closed state: shows selected pill or placeholder -->
    <button
      v-if="!open"
      type="button"
      class="w-full h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-left text-sm flex items-center gap-2 transition-colors disabled:bg-brand-cream-100 disabled:text-brand-dark-400 disabled:cursor-not-allowed"
      :disabled="speciesId === null"
      @click="openPicker"
    >
      <span v-if="speciesId === null" class="text-brand-dark-500">Pick species first</span>
      <span v-else-if="selected" class="flex-1 truncate text-brand-dark-950">{{ display }}</span>
      <span v-else class="flex-1 truncate text-brand-dark-500">{{ placeholder ?? 'Select…' }}</span>
      <button
        v-if="selected"
        type="button"
        class="size-5 rounded hover:bg-brand-cream-200 flex items-center justify-center"
        aria-label="Clear"
        @click="clear"
      >
        <X class="size-3" />
      </button>
      <ChevronDown v-else class="size-4 text-brand-dark-500 shrink-0" />
    </button>

    <!-- Open state: search input + dropdown -->
    <div
      v-else
      class="absolute inset-x-0 top-0 z-30 rounded-md border border-brand-cream-300 bg-white shadow-lg"
    >
      <input
        ref="inputRef"
        v-model="query"
        type="text"
        :placeholder="placeholder ?? 'Type to search…'"
        class="w-full h-9 px-3 text-sm border-b border-brand-cream-200 outline-none"
        @keydown.esc="open = false"
      />
      <ul
        v-if="filtered.length"
        class="max-h-60 overflow-y-auto py-1"
        role="listbox"
      >
        <li
          v-for="g in filtered"
          :key="g.id"
          class="px-3 py-1.5 text-sm cursor-pointer hover:bg-brand-cream-100 flex items-center gap-2"
          @click="pick(g.id)"
        >
          <span class="font-mono text-brand-dark-700">{{ g.code }}</span>
          <span v-if="g.name" class="text-brand-dark-950">· {{ g.name }}</span>
        </li>
      </ul>
      <div v-else class="px-3 py-4 text-xs text-brand-dark-500 text-center">
        No matching geckos.
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Type-check**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/components/GeckoPicker.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): GeckoPicker combobox for sire/dam selection

Reusable component that filters the cached gecko list by species + sex
+ exclude-id, with type-to-filter. Disabled until species is picked.
Not yet wired into any view — that comes with the sire/dam inputs in
GeckoFormSheet.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Frontend — wire sire/dam inputs into `GeckoFormSheet.vue`

**Files:**
- Modify: `apps/admin/src/components/GeckoFormSheet.vue`

- [ ] **Step 1: Add imports and refs**

Open `apps/admin/src/components/GeckoFormSheet.vue`. In the `<script setup>` block:

Find the existing import line for icons and composables, and add:

```ts
import GeckoPicker from '@/components/GeckoPicker.vue';
```

Find the existing form-state ref block (the one with `name`, `speciesId`, `sex`, `hatchDate`, etc.) and add below them:

```ts
const sireId = ref<number | null>(null);
const damId  = ref<number | null>(null);
```

Find the `reset()` function and update the if/else blocks so both branches also reset sire/dam. The edit branch:

```ts
if (g) {
  name.value = g.name ?? '';
  speciesId.value = g.species_id;
  sex.value = g.sex;
  hatchDate.value = g.hatch_date ?? '';
  acquiredDate.value = g.acquired_date ?? '';
  status.value = g.status;
  priceUsd.value = g.list_price_usd ?? '';
  notes.value = g.notes ?? '';
  traits.value = g.traits.map((t) => ({
    trait_id: t.trait_id,
    zygosity: t.zygosity,
  }));
  sireId.value = g.sire_id;
  damId.value = g.dam_id;
}
```

The else branch (create mode):

```ts
} else {
  name.value = '';
  speciesId.value = null;
  sex.value = 'U';
  hatchDate.value = '';
  acquiredDate.value = '';
  status.value = 'AVAILABLE';
  priceUsd.value = '';
  notes.value = '';
  traits.value = [];
  sireId.value = null;
  damId.value = null;
}
```

Still in the script block, find the existing `watch(speciesId, ...)` — append sire+dam clearing in the same block (the existing one drops invalid traits; now also drop stale parentage):

```ts
watch(speciesId, (sp) => {
  if (!sp) return;
  const valid = new Set((allTraits.value ?? []).filter((t) => t.species_id === sp).map((t) => t.id));
  traits.value = traits.value.filter((row) => valid.has(row.trait_id));
  sireId.value = null;
  damId.value = null;
});
```

Find the payload object in `submit()` and replace the hard-coded nulls:

```ts
const payload: GeckoWritePayload = {
  name: name.value.trim(),
  species_id: speciesId.value,
  sex: sex.value,
  hatch_date: hatchDate.value,
  acquired_date: acquiredDate.value,
  status: status.value,
  sire_id: sireId.value,
  dam_id: damId.value,
  list_price_usd: priceUsd.value.trim(),
  notes: notes.value.trim(),
  traits: traits.value,
};
```

- [ ] **Step 2: Add the sire/dam inputs in the template**

Find the existing two-column grid that contains **hatch_date** and **acquired_date**. Immediately **above** it (still inside the same scrollable body area, after the "Status / Price" grid or adjacent to species/sex — wherever feels grouped; the cleanest is right after the "Hatch / Acquired" grid), add a new two-column grid:

```vue
<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
  <div class="flex flex-col gap-2">
    <Label>Sire</Label>
    <GeckoPicker
      v-model="sireId"
      :species-id="speciesId"
      sex="M"
      :exclude-id="gecko?.id"
      placeholder="Search sires…"
    />
  </div>
  <div class="flex flex-col gap-2">
    <Label>Dam</Label>
    <GeckoPicker
      v-model="damId"
      :species-id="speciesId"
      sex="F"
      :exclude-id="gecko?.id"
      placeholder="Search dams…"
    />
  </div>
</div>
```

- [ ] **Step 3: Build + test**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds, no TS errors.

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run test`
Expected: 7 tests PASS.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/components/GeckoFormSheet.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): sire + dam pickers in edit-gecko drawer

Two GeckoPicker instances (sire: sex=M, dam: sex=F) filtered by the
current gecko's species, with self-exclude. Clears both when species
changes. Payload now sends real sire_id / dam_id instead of nulls.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Frontend — set-as-cover button + caption edit + upload toast

**Files:**
- Modify: `apps/admin/src/components/GeckoFormSheet.vue`

- [ ] **Step 1: Add icon + composable imports**

In the existing import blocks of `GeckoFormSheet.vue`, update as follows.

Extend the lucide-vue-next import:
```ts
import { X, Plus, Trash2, Info, Upload, ImageOff, Star, Pencil } from 'lucide-vue-next';
```

Extend the composable import to include the two new mutations:
```ts
import {
  useSpecies,
  useTraits,
  useCreateGecko,
  useUpdateGecko,
  useUploadGeckoMedia,
  useDeleteMedia,
  useUpdateMedia,
  useSetCoverMedia,
  type GeckoWritePayload,
} from '@/composables/useGeckos';
```

- [ ] **Step 2: Add script state + handlers for cover/caption**

In the script, near the existing `uploadMut` / `deleteMediaMut` declarations, add:

```ts
const setCoverMut = useSetCoverMedia();
const updateMediaMut = useUpdateMedia();

const editingCaptionId = ref<number | null>(null);
const draftCaption = ref('');

function startEditCaption(mediaId: number, current: string) {
  commitCaptionIfEditing(); // commit any other open edit first
  editingCaptionId.value = mediaId;
  draftCaption.value = current;
}

function cancelEditCaption() {
  editingCaptionId.value = null;
  draftCaption.value = '';
}

async function commitCaptionIfEditing() {
  if (editingCaptionId.value === null) return;
  if (!props.gecko) {
    cancelEditCaption();
    return;
  }
  const mediaId = editingCaptionId.value;
  const next = draftCaption.value.trim();
  const existing = photos.value.find((p) => p.id === mediaId);
  if (!existing || existing.caption === next) {
    cancelEditCaption();
    return;
  }
  try {
    const { media } = await updateMediaMut.mutateAsync({
      mediaId,
      geckoId: props.gecko.id,
      patch: { caption: next },
    });
    const idx = photos.value.findIndex((p) => p.id === mediaId);
    if (idx !== -1) {
      photos.value[idx] = {
        id: media.id,
        url: media.url,
        caption: media.caption,
        display_order: media.display_order,
      };
    }
  } catch (e: unknown) {
    const msg = (e as any)?.response?.data?.error ?? (e as Error).message ?? 'Caption save failed';
    toast.error(String(msg));
  } finally {
    cancelEditCaption();
  }
}

async function setAsCover(mediaId: number) {
  if (!props.gecko) return;
  try {
    await setCoverMut.mutateAsync({ mediaId, geckoId: props.gecko.id });
    // Re-sequence local state: move target to index 0, rest preserve order.
    const target = photos.value.find((p) => p.id === mediaId);
    if (!target) return;
    const rest = photos.value.filter((p) => p.id !== mediaId);
    photos.value = [
      { ...target, display_order: 0 },
      ...rest.map((p, i) => ({ ...p, display_order: i + 1 })),
    ];
    toast.success('Cover updated.');
  } catch (e: unknown) {
    const msg = (e as any)?.response?.data?.error ?? (e as Error).message ?? 'Set cover failed';
    toast.error(String(msg));
  }
}
```

- [ ] **Step 3: Upload success toast**

Find the existing `onFilesPicked` function. Currently it pushes each successful upload to `photos.value` inside a try/catch loop. Replace the function body with a version that tracks successes and emits a single toast at the end:

```ts
async function onFilesPicked(e: Event) {
  if (!props.gecko) return;
  const input = e.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  if (!files.length) return;
  input.value = '';

  let successCount = 0;
  for (const f of files) {
    if (!f.type.startsWith('image/')) {
      toast.error(`${f.name}: not an image`);
      continue;
    }
    if (f.size > 10 * 1024 * 1024) {
      toast.error(`${f.name}: larger than 10 MB`);
      continue;
    }
    try {
      const { media } = await uploadMut.mutateAsync({ geckoId: props.gecko.id, file: f });
      photos.value.push({
        id: media.id,
        url: media.url,
        caption: media.caption,
        display_order: media.display_order,
      });
      successCount++;
    } catch (e: unknown) {
      const msg = (e as any)?.response?.data?.error ?? (e as Error).message ?? 'Upload failed';
      toast.error(`${f.name}: ${msg}`);
    }
  }

  if (successCount === 1) toast.success('Uploaded.');
  else if (successCount > 1) toast.success(`Uploaded ${successCount} photos.`);
}
```

- [ ] **Step 4: Update the photo tile template**

Find the existing photo grid block:

```vue
<div v-if="isEdit && photos.length" class="grid grid-cols-3 gap-2">
  <div
    v-for="p in photos"
    :key="p.id"
    class="relative group aspect-square rounded-lg overflow-hidden border border-brand-cream-300 bg-white"
  >
    <img :src="p.url" :alt="p.caption || 'gecko photo'" class="w-full h-full object-cover" />
    <button
      type="button"
      class="absolute top-1 right-1 size-6 rounded-md bg-brand-dark-950/70 text-white opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center hover:bg-red-600"
      aria-label="Delete photo"
      :disabled="deleteMediaMut.isPending.value"
      @click="removePhoto(p.id)"
    >
      <Trash2 class="size-3.5" />
    </button>
  </div>
</div>
```

Replace the inner `<div v-for>` block entirely with:

```vue
<div
  v-for="p in photos"
  :key="p.id"
  class="relative group aspect-square rounded-lg overflow-hidden border border-brand-cream-300 bg-white"
>
  <img :src="p.url" :alt="p.caption || 'gecko photo'" class="w-full h-full object-cover" />

  <!-- Set-as-cover star (top-left) -->
  <button
    type="button"
    class="absolute top-1 left-1 size-6 rounded-md bg-brand-dark-950/70 text-white opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center hover:bg-brand-gold-600"
    :class="{ 'opacity-100 !bg-brand-gold-600': p.display_order === 0 }"
    :aria-label="p.display_order === 0 ? 'Current cover' : 'Set as cover'"
    :title="p.display_order === 0 ? 'Current cover' : 'Set as cover'"
    :disabled="setCoverMut.isPending.value || p.display_order === 0"
    @click="setAsCover(p.id)"
  >
    <Star class="size-3.5" :class="p.display_order === 0 ? 'fill-current' : ''" />
  </button>

  <!-- Delete (top-right) -->
  <button
    type="button"
    class="absolute top-1 right-1 size-6 rounded-md bg-brand-dark-950/70 text-white opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center hover:bg-red-600"
    aria-label="Delete photo"
    :disabled="deleteMediaMut.isPending.value"
    @click="removePhoto(p.id)"
  >
    <Trash2 class="size-3.5" />
  </button>

  <!-- Caption ribbon (bottom) -->
  <div
    v-if="editingCaptionId === p.id"
    class="absolute inset-x-0 bottom-0 bg-brand-dark-950/80 p-1.5 flex items-center gap-1"
  >
    <input
      :value="draftCaption"
      type="text"
      maxlength="500"
      class="flex-1 bg-transparent text-xs text-white placeholder:text-brand-dark-300 outline-none"
      autofocus
      placeholder="Caption…"
      @input="(ev) => (draftCaption = (ev.target as HTMLInputElement).value)"
      @keydown.enter.prevent="commitCaptionIfEditing"
      @keydown.esc="cancelEditCaption"
      @blur="commitCaptionIfEditing"
    />
  </div>
  <button
    v-else-if="p.caption"
    type="button"
    class="absolute inset-x-0 bottom-0 bg-brand-dark-950/60 text-white text-xs px-2 py-1 text-left truncate hover:bg-brand-dark-950/80 transition-colors"
    @click="startEditCaption(p.id, p.caption)"
  >
    {{ p.caption }}
  </button>
  <button
    v-else
    type="button"
    class="absolute inset-x-0 bottom-0 bg-brand-dark-950/50 text-white/70 text-xs px-2 py-1 text-left opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1"
    @click="startEditCaption(p.id, '')"
  >
    <Pencil class="size-3" /> Add caption…
  </button>
</div>
```

- [ ] **Step 5: Build + test**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds.

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run test`
Expected: 7 tests PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/components/GeckoFormSheet.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): set-as-cover, caption edit, and upload toast on photo tiles

- ⭐ button top-left of each photo tile (filled on current cover) calls
  /set-cover, local state re-sequences to match, toast confirms.
- Caption ribbon at the bottom: shows caption if any, or hover-only
  'Add caption…' otherwise. Click to edit inline; Enter/blur saves,
  Esc cancels. Only one edit active at a time.
- Upload now emits a single success toast (\"Uploaded.\" for one file,
  \"Uploaded N photos.\" for batches). Errors already toasted per-file.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: End-to-end smoke + push + ping Zen

**Files:** None — verification + push.

- [ ] **Step 1: Full backend tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages `ok` (including new media tests).

- [ ] **Step 2: Full admin build + tests**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build && bun run test`
Expected: build succeeds; 7 tests PASS.

- [ ] **Step 3: Live smoke via Vite proxy**

Run:
```bash
TOKEN=$(curl -sS -X POST http://localhost:5173/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"zen@zeneticgekkos.com","password":"gekko-dev-2026"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

# PATCH caption (if a gecko has a photo)
MID=$(curl -sS http://localhost:5173/api/geckos/1 -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d['photos'][0]['id'] if d.get('photos') else '')")
if [ -n "$MID" ]; then
  echo "--- PATCH caption on media $MID ---"
  curl -sS -X PATCH "http://localhost:5173/api/media/$MID" \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d '{"caption":"smoke-test"}' | python3 -m json.tool
  echo "--- set-cover on media $MID ---"
  curl -sS -X POST "http://localhost:5173/api/media/$MID/set-cover" \
    -H "Authorization: Bearer $TOKEN" -w "status=%{http_code}\n"
else
  echo "no media to smoke-test against — skipping"
fi
```
Expected: PATCH returns JSON with updated caption; set-cover returns `status=204`.

- [ ] **Step 4: Push**

Run:
```bash
cd /home/zen/dev/project_gekko
git push origin main
```
Expected: push succeeds.

- [ ] **Step 5: Ping Zen**

Report to the user:

> Phase E (admin polish gaps) is in. N commits pushed: sire/dam picker (filtered by species + sex), star button on photos sets the cover, caption ribbon edits inline, photo uploads now toast on success. Open any gecko's edit drawer and try it.

---

## Self-Review

**1. Spec coverage:**
- PATCH /api/media/{id} endpoint → Task 2
- POST /api/media/{id}/set-cover endpoint → Task 3
- Backend tests (set-cover re-sequence, PATCH variants, empty body, not found) → Task 4
- GeckoPicker reusable component → Task 6
- Sire/Dam picker wiring + payload send + clear-on-species-change → Task 7
- ⭐ set-as-cover on tile → Task 8
- Caption ribbon + inline edit → Task 8
- Upload success toast (single + plural) → Task 8
- useUpdateMedia + useSetCoverMedia composables → Task 5
- Rollout (grouped commits, push together) → Task 9

**2. Placeholder scan:** Every step has the real code / commands / expected output. No "implement later" / "TBD" / "handle edge cases". ✅

**3. Type consistency:**
- Frontend `useUpdateMedia` returns `{ geckoId, media }`; Task 8's `commitCaptionIfEditing` destructures exactly that shape. ✅
- `useSetCoverMedia` returns `geckoId` on success; Task 8's `setAsCover` uses it for the invalidate side effect only. ✅
- Backend `patchMediaReq` has `Caption *string` + `DisplayOrder *int32`; handler checks nil on both. Tests exercise both-missing (400), caption-only, display_order-only. ✅
- `MountMedia` signature unchanged — the new routes slot into the existing group. ✅
- `GeckoPicker.vue`'s `sex: Sex` prop type imports `Sex` from `@/types/gecko` which defines it as `'M' | 'F' | 'U'`; templates pass `sex="M"` and `sex="F"` — both valid. ✅
- `pgText` helper is already defined in `backend/internal/http/geckos.go:XXX` (file-internal) and `media.go` lives in the same package, so the handler can call it directly. ✅

**4. Ambiguity check:**
- "What if the gecko already has sire_id/dam_id set from seed data?" — `reset()` in Task 7 pre-fills from `g.sire_id` / `g.dam_id`, so those values show in the pickers on open. ✅
- "What if a sire/dam gecko was later changed to a different species?" — stale ids, but the GeckoPicker hides them from the dropdown (species+sex filter). The `display` computed shows the existing selection even if it would no longer match, so the operator can clear it. ✅
- "What happens if set-cover is called on a photo that's already cover?" — the button is disabled via `:disabled="setCoverMut.isPending.value || p.display_order === 0"`. The backend would still execute correctly (target → 0, rest 1..N), just a no-op write. ✅
