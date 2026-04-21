# Admin Polish Gaps

> **Goal:** Close four small UX/UI gaps in the admin without expanding scope: enable sire/dam parentage input with a searchable picker, let the operator choose which photo is the cover, let them edit photo captions inline, and give positive feedback when a photo upload succeeds.

## Scope

**In scope**

- Backend: `PATCH /api/admin/media/{id}` accepting `{caption?, display_order?}` (generic media update).
- Backend: `POST /api/admin/media/{id}/set-cover` — atomic re-sequence so the chosen photo becomes `display_order = 0` and all other photos for that gecko shift down by one (relative order preserved).
- Frontend: `GeckoPicker.vue` — a searchable combobox used for the sire and dam slots inside `GeckoFormSheet.vue`. Filters by species + sex + excludes self.
- Frontend: sire + dam inputs wired into the edit drawer (create-mode included); payload already supports `sire_id` and `dam_id`.
- Frontend: hover-only ⭐ button on each photo tile in the edit drawer that calls `set-cover`.
- Frontend: inline caption ribbon per photo tile + pencil-icon edit → inline text input → save on Enter / blur, cancel on Esc.
- Frontend: `toast.success("Uploaded.")` (or pluralised for batch) in the photo upload flow.

**Out of scope**

- Bulk select on the geckos list — defer; deserves its own brainstorm.
- Drag-to-reorder photos — the set-as-cover button handles the real need.
- Search-as-you-type debouncing for very large gecko lists; current collection is small enough that client-side filter of the cached `useGeckos()` data is fine.
- Editing acquired/hatch dates via calendar widget (current `<input type="date">` is adequate).

## Architecture overview

Two new backend endpoints, three frontend changes. Existing `media` table + `GeckoFormSheet` are the touchpoints. No new tables, no migrations.

### Backend endpoints

Both behind `RequireAuth`.

#### `PATCH /api/admin/media/{id}`

**Request body:**
```json
{
  "caption": "optional new caption (empty string clears)",
  "display_order": 3
}
```

Both fields optional — only the provided fields are updated. If neither is provided, returns 400.

**Response:** the updated `mediaDTO` (same shape returned by the upload endpoint).

**Validation:**
- `id` must be an integer.
- `display_order` must be a non-negative integer (0–10_000).
- `caption` length ≤ 500 chars.

**Status codes:** 200 on success, 400 on bad input, 404 if the media id doesn't exist, 500 on DB error.

#### `POST /api/admin/media/{id}/set-cover`

**Request body:** empty (id in path is sufficient).

**Behavior:**
1. Look up the target media row; if not found or `gecko_id IS NULL`, return 404.
2. In a single transaction:
   - Fetch all media rows for that `gecko_id`, ordered by `display_order, uploaded_at`.
   - Move the chosen media to position 0; all others keep their relative order and become 1, 2, 3…
   - `UPDATE media SET display_order = $position WHERE id = $mediaId` for each row.

**Response:** 204 No Content.

**Why not client-side min−1 math:** race conditions if the operator runs the admin on two devices; also keeps `display_order` values tidy (no negatives, no drift).

### Frontend changes

#### `GeckoPicker.vue` (new)

Location: `apps/admin/src/components/GeckoPicker.vue`.

Props:
```ts
{
  modelValue: number | null          // currently-selected gecko id
  speciesId: number | null           // filter candidates to this species
  sex: 'M' | 'F'                     // filter by sex
  excludeId?: number                 // don't allow self-selection
  placeholder?: string               // e.g. 'Sire…' / 'Dam…'
}
```

Emits `update:modelValue` with `number | null`.

Behavior:
- Fetches `useGeckos()` from the cache (already preloaded by list/dashboard).
- Filters candidates to: `species_id === speciesId && sex === props.sex && id !== excludeId`.
- Renders a `<input>` with type-to-filter; dropdown shows matching `ZGCODE · Name` rows.
- Arrow keys + Enter / Tab select; Esc closes; clicking an item outside closes.
- Empty placeholder when nothing selected; a "Clear" button when something is selected.
- If `speciesId` is null (the drawer hasn't picked a species yet), the input is disabled with hint "Pick species first."

#### `GeckoFormSheet.vue` (modified)

Immediately below the existing "Species + Sex" grid, add two more input groups in a two-column layout:

- **Sire** — `<GeckoPicker v-model="sireId" :species-id="speciesId" sex="M" :exclude-id="gecko?.id">`.
- **Dam** — `<GeckoPicker v-model="damId" :species-id="speciesId" sex="F" :exclude-id="gecko?.id">`.

Two new refs in script:
```ts
const sireId = ref<number | null>(null);
const damId = ref<number | null>(null);
```

Reset logic in `reset()` pre-fills from the gecko (edit mode) or nulls (create mode).

Submit payload: replace the hard-coded `sire_id: null, dam_id: null` with `sire_id: sireId.value, dam_id: damId.value`.

Watcher on `speciesId`: when species changes, clear both sire and dam (the new species has a different trait/sex pool and the existing ids may not match).

#### Photo tile updates in `GeckoFormSheet.vue`

Current photo tile has a single hover-only delete button in the top-right. Expand to:

- Top-left: ⭐ button (filled if this is the cover / `display_order === 0`, outline otherwise). Clicking calls `useSetCoverMedia` mutation.
- Top-right: existing 🗑 delete button (unchanged).
- Bottom: a caption ribbon (`absolute bottom-0 left-0 right-0`) with black/50 background and white text. Hidden when there's no caption and not hovered.
  - On hover with no caption: show a muted "Add caption…" ribbon with pencil icon.
  - Clicking the ribbon (or pencil) swaps the ribbon to an `<input>` autofocused, value=current caption.
  - Enter or blur saves via `useUpdateMedia` mutation.
  - Esc restores the previous value and closes.

State per tile is local:
```ts
const editingId = ref<number | null>(null);
const draftCaption = ref('');
```

Only one tile in edit mode at a time (setting `editingId` to another value commits the previous draft).

#### Upload toast

In the existing `onFilesPicked` handler inside `GeckoFormSheet.vue`, after the loop collects successes + failures, call:
```ts
if (successes.length === 1) toast.success('Uploaded.');
else if (successes.length > 1) toast.success(`Uploaded ${successes.length} photos.`);
```

Error toasts per-file already exist; don't change that path.

### Composables

Two new mutations in `apps/admin/src/composables/useGeckos.ts`:

```ts
export function useUpdateMedia() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      mediaId, geckoId, patch,
    }: {
      mediaId: number;
      geckoId: number;
      patch: { caption?: string; display_order?: number };
    }) => {
      const { data } = await api.patch(`/api/admin/media/${mediaId}`, patch);
      return { geckoId, media: data };
    },
    onSuccess: ({ geckoId }) => invalidateGeckos(qc, geckoId),
  });
}

export function useSetCoverMedia() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ mediaId, geckoId }: { mediaId: number; geckoId: number }) => {
      await api.post(`/api/admin/media/${mediaId}/set-cover`);
      return geckoId;
    },
    onSuccess: (geckoId) => invalidateGeckos(qc, geckoId),
  });
}
```

## Data flow

```
User edits caption or clicks ⭐
         │
         ▼
GeckoFormSheet.vue — useUpdateMedia / useSetCoverMedia mutation
         │ PATCH|POST
         ▼
backend/internal/http/media.go — update / setCover handler
         │ sqlc
         ▼
backend/internal/queries/media.sql — UpdateMedia / SetCoverForMedia
         │
         ▼
Postgres media table
```

## Error handling

- **Update with no fields:** backend returns 400 `{"error":"no fields to update"}`.
- **display_order out of range:** 400.
- **Caption too long:** 400.
- **Media not found:** 404.
- **Transaction failure during set-cover:** rolled back; 500 with error message.
- **Frontend upload success with no data returned:** defensive — toast only fires when `uploadMut.mutateAsync()` resolves.
- **Network error:** existing axios interceptor + mutation `isError` pattern — toast an error, no state corruption.

## Testing

**Backend (go test)**
- `TestSetCover_reseqencesAllMedia` — create gecko, upload 3 photos, call set-cover on id=3, assert order is now [3→0, 1→1, 2→2].
- `TestSetCover_notFound` — unknown id → 404.
- `TestPatchMedia_captionOnly` — PATCH with `{caption}` updates only caption, display_order unchanged.
- `TestPatchMedia_displayOrderOnly` — PATCH with `{display_order}` updates only display_order.
- `TestPatchMedia_empty` — PATCH with `{}` → 400.

**Admin (vitest)**
- No new vitest coverage. All changes are UI interactions against mocked mutations; the existing 7 tests stay green.

## Rollout

- Commit groups:
  1. Backend queries + handlers + tests.
  2. Frontend `GeckoPicker.vue` + `useUpdateMedia` / `useSetCoverMedia` composables.
  3. `GeckoFormSheet.vue` updates (sire/dam picker wiring, ⭐ button, caption ribbon, upload toast).
- Pushed together so the frontend never runs against a stale backend.
