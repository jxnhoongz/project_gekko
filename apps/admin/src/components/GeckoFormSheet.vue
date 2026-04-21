<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { toast } from 'vue-sonner';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogClose,
} from 'reka-ui';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import GeckoPicker from '@/components/GeckoPicker.vue';
import { X, Plus, Trash2, Info, Upload, ImageOff, Star, Pencil } from 'lucide-vue-next';
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
import type { Gecko, Zygosity, Sex, GeckoStatus } from '@/types/gecko';

const props = defineProps<{
  /** If supplied, form runs in edit mode. */
  gecko?: Gecko | null;
}>();

const open = defineModel<boolean>('open', { default: false });
const emit = defineEmits<{ (e: 'saved', g: Gecko): void }>();

const isEdit = computed(() => !!props.gecko);

// ---- form state ----
const name = ref('');
const speciesId = ref<number | null>(null);
const sex = ref<Sex>('U');
const hatchDate = ref('');
const acquiredDate = ref('');
const status = ref<GeckoStatus>('AVAILABLE');
const notes = ref('');

interface TraitRow {
  trait_id: number;
  zygosity: Zygosity;
}
const traits = ref<TraitRow[]>([]);

const sireId = ref<number | null>(null);
const damId  = ref<number | null>(null);

function reset() {
  const g = props.gecko;
  if (g) {
    name.value = g.name ?? '';
    speciesId.value = g.species_id;
    sex.value = g.sex;
    hatchDate.value = g.hatch_date ?? '';
    acquiredDate.value = g.acquired_date ?? '';
    status.value = g.status;
    notes.value = g.notes ?? '';
    traits.value = g.traits.map((t) => ({
      trait_id: t.trait_id,
      zygosity: t.zygosity,
    }));
    sireId.value = g.sire_id;
    damId.value = g.dam_id;
  } else {
    name.value = '';
    speciesId.value = null;
    sex.value = 'U';
    hatchDate.value = '';
    acquiredDate.value = '';
    status.value = 'AVAILABLE';
    notes.value = '';
    traits.value = [];
    sireId.value = null;
    damId.value = null;
  }
}

watch(open, (v) => {
  if (v) reset();
});

const { data: speciesList } = useSpecies();
const { data: allTraits } = useTraits();

const traitsForSpecies = computed(() => {
  if (!speciesId.value || !allTraits.value) return [];
  return allTraits.value.filter((t) => t.species_id === speciesId.value);
});

const traitById = computed(() => {
  const m = new Map<number, { trait_name: string; trait_code: string }>();
  for (const t of allTraits.value ?? []) {
    m.set(t.id, { trait_name: t.trait_name, trait_code: t.trait_code });
  }
  return m;
});

// When species changes, drop trait rows that don't belong to the new species
watch(speciesId, (sp) => {
  if (!sp) return;
  const valid = new Set((allTraits.value ?? []).filter((t) => t.species_id === sp).map((t) => t.id));
  traits.value = traits.value.filter((row) => valid.has(row.trait_id));
  sireId.value = null;
  damId.value = null;
});

function addTrait() {
  const available = traitsForSpecies.value.filter(
    (t) => !traits.value.find((row) => row.trait_id === t.id),
  );
  if (!available.length) return;
  traits.value.push({ trait_id: available[0].id, zygosity: 'HOM' });
}

function removeTrait(idx: number) {
  traits.value.splice(idx, 1);
}

const createMut = useCreateGecko();
const updateMut = useUpdateGecko();
const uploadMut = useUploadGeckoMedia();
const deleteMediaMut = useDeleteMedia();
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

const saving = computed(() => createMut.isPending.value || updateMut.isPending.value);

// Local photos state starts from the gecko's photos when the sheet opens
// and is updated on upload/delete so the sheet reflects changes immediately.
const photos = ref<{ id: number; url: string; caption: string; display_order: number }[]>([]);
const fileInput = ref<HTMLInputElement | null>(null);

watch(open, (v) => {
  if (v && props.gecko) {
    photos.value = (props.gecko.photos ?? []).map((p) => ({
      id: p.id,
      url: p.url,
      caption: p.caption,
      display_order: p.display_order,
    }));
  }
});

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

async function removePhoto(mediaId: number) {
  if (!props.gecko) return;
  if (!window.confirm('Delete this photo?')) return;
  try {
    await deleteMediaMut.mutateAsync({ mediaId, geckoId: props.gecko.id });
    photos.value = photos.value.filter((p) => p.id !== mediaId);
  } catch (e: unknown) {
    toast.error((e as Error).message ?? 'Delete failed');
  }
}

async function submit() {
  if (!speciesId.value) {
    toast.error('Species is required.');
    return;
  }
  const payload: GeckoWritePayload = {
    name: name.value.trim(),
    species_id: speciesId.value,
    sex: sex.value,
    hatch_date: hatchDate.value,
    acquired_date: acquiredDate.value,
    status: status.value,
    sire_id: sireId.value,
    dam_id: damId.value,
    notes: notes.value.trim(),
    traits: traits.value,
  };

  try {
    const result = props.gecko
      ? await updateMut.mutateAsync({ id: props.gecko.id, payload })
      : await createMut.mutateAsync(payload);

    toast.success(props.gecko ? 'Gecko updated.' : `Created ${result.code}.`);
    emit('saved', result);
    open.value = false;
  } catch (e: unknown) {
    const msg =
      (e as any)?.response?.data?.error ??
      (e as Error).message ??
      'Save failed';
    toast.error(String(msg));
  }
}

const statuses: GeckoStatus[] = [
  'AVAILABLE', 'HOLD', 'BREEDING', 'PERSONAL', 'SOLD', 'DECEASED',
];

const zygosities: Zygosity[] = ['HOM', 'HET', 'POSS_HET'];
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-brand-dark-950/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
      />
      <DialogContent
        class="fixed inset-y-0 right-0 z-50 w-[min(560px,100vw)] bg-brand-cream-50 border-l border-brand-cream-300 shadow-2xl overflow-hidden flex flex-col data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right data-[state=closed]:duration-300 data-[state=open]:duration-500"
        aria-describedby=""
      >
        <!-- Header -->
        <div class="flex items-start justify-between px-6 py-5 border-b border-brand-cream-200 shrink-0">
          <div class="flex flex-col gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-brand-gold-700 font-semibold">
              {{ isEdit ? 'Edit' : 'New gecko' }}
            </span>
            <h2 class="font-serif text-2xl text-brand-dark-950 leading-tight">
              {{ isEdit ? `Edit ${gecko?.name || gecko?.code}` : 'Add a gecko' }}
            </h2>
            <div v-if="!isEdit" class="flex items-center gap-1 text-xs text-brand-dark-600 mt-1">
              <Info class="size-3" />
              Code auto-generated on save (e.g. ZGLP-2026-007)
            </div>
            <div v-else class="text-xs text-brand-dark-500 font-mono mt-1">{{ gecko?.code }}</div>
          </div>
          <DialogClose
            class="rounded-md p-1 text-brand-dark-600 hover:bg-brand-cream-200 hover:text-brand-dark-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Close"
          >
            <X class="size-5" />
          </DialogClose>
        </div>

        <!-- Body (scrolling) -->
        <div class="flex-1 overflow-y-auto px-6 py-5 flex flex-col gap-5">
          <div class="flex flex-col gap-2">
            <Label for="gf-name">Name</Label>
            <Input id="gf-name" v-model="name" placeholder="e.g. Apsara" class="bg-white" />
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="gf-species">Species <span class="text-destructive">*</span></Label>
              <select
                id="gf-species"
                v-model="speciesId"
                class="h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-sm"
              >
                <option :value="null" disabled>Choose species…</option>
                <option v-for="s in speciesList ?? []" :key="s.id" :value="s.id">
                  {{ s.common_name }} ({{ s.code }})
                </option>
              </select>
            </div>
            <div class="flex flex-col gap-2">
              <Label>Sex</Label>
              <div
                class="inline-flex items-center rounded-md border border-brand-cream-300 bg-white p-0.5 w-fit"
              >
                <button
                  v-for="opt in (['M','F','U'] as const)"
                  :key="opt"
                  type="button"
                  class="px-3 h-8 rounded text-xs font-medium transition-colors"
                  :class="
                    sex === opt
                      ? 'bg-brand-gold-100 text-brand-gold-800'
                      : 'text-brand-dark-600 hover:bg-brand-cream-100'
                  "
                  @click="sex = opt"
                >
                  {{ opt === 'M' ? 'Male' : opt === 'F' ? 'Female' : 'Unsexed' }}
                </button>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="gf-hatch">Hatch date</Label>
              <Input
                id="gf-hatch" v-model="hatchDate" type="date" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="gf-acquired">Acquired date</Label>
              <Input
                id="gf-acquired" v-model="acquiredDate" type="date" class="bg-white" />
            </div>
          </div>

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

          <div class="flex flex-col gap-2">
            <Label for="gf-status">Status</Label>
            <select
              id="gf-status"
              v-model="status"
              class="h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-sm"
            >
              <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
            </select>
          </div>

          <div class="flex flex-col gap-2">
            <Label for="gf-notes">Notes</Label>
            <textarea
              id="gf-notes"
              v-model="notes"
              rows="3"
              placeholder="Anything worth remembering — temperament, line, holdback reason…"
              class="rounded-md border border-brand-cream-300 bg-white px-3 py-2 text-sm resize-y"
            />
          </div>

          <Separator />

          <!-- Traits -->
          <div class="flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="font-serif text-lg">Genetics</h3>
                <p class="text-xs text-brand-dark-600">
                  Traits are filtered by species — pick species first.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="!speciesId || traitsForSpecies.length === 0"
                @click="addTrait"
              >
                <Plus class="size-4" /> Add trait
              </Button>
            </div>

            <div
              v-if="!traits.length"
              class="rounded-lg border border-dashed border-brand-cream-400 bg-brand-cream-50 p-4 text-sm text-brand-dark-500 text-center"
            >
              No traits assigned yet.
            </div>

            <ul v-else class="flex flex-col gap-2">
              <li
                v-for="(row, idx) in traits"
                :key="idx"
                class="flex items-center gap-2 rounded-lg border border-brand-cream-300 bg-white p-2"
              >
                <select
                  v-model="row.trait_id"
                  class="flex-1 h-8 rounded border border-brand-cream-300 bg-white px-2 text-sm"
                >
                  <option
                    v-for="t in traitsForSpecies"
                    :key="t.id"
                    :value="t.id"
                    :disabled="
                      traits.some((r, i) => i !== idx && r.trait_id === t.id)
                    "
                  >
                    {{ t.trait_name }}<span v-if="t.trait_code"> ({{ t.trait_code }})</span>
                  </option>
                </select>
                <select
                  v-model="row.zygosity"
                  class="h-8 rounded border border-brand-cream-300 bg-white px-2 text-xs"
                >
                  <option v-for="z in zygosities" :key="z" :value="z">{{ z }}</option>
                </select>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Remove trait"
                  @click="removeTrait(idx)"
                >
                  <Trash2 class="size-4 text-red-700" />
                </Button>
              </li>
            </ul>

            <div v-if="isEdit && traits.length" class="flex flex-wrap gap-1 pt-1">
              <Badge
                v-for="(t, i) in traits"
                :key="i"
                variant="soft"
                class="text-[10px]"
              >
                {{ traitById.get(t.trait_id)?.trait_name ?? t.trait_id }} ({{ t.zygosity }})
              </Badge>
            </div>
          </div>

          <Separator />

          <!-- Photos (edit mode only — gecko must exist to attach media) -->
          <div class="flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="font-serif text-lg">Photos</h3>
                <p v-if="isEdit" class="text-xs text-brand-dark-600">
                  JPG, PNG, WebP or GIF · up to 10 MB each · stored on the server.
                </p>
                <p v-else class="text-xs text-brand-dark-600 flex items-center gap-1">
                  <Info class="size-3" /> Save the gecko first, then edit to add photos.
                </p>
              </div>
              <template v-if="isEdit">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  :disabled="uploadMut.isPending.value"
                  @click="fileInput?.click()"
                >
                  <Upload class="size-4" />
                  {{ uploadMut.isPending.value ? 'Uploading…' : 'Upload' }}
                </Button>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/*"
                  multiple
                  class="hidden"
                  @change="onFilesPicked"
                />
              </template>
            </div>

            <div v-if="isEdit && photos.length" class="grid grid-cols-3 gap-2">
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
            </div>

            <div
              v-else-if="isEdit"
              class="rounded-lg border border-dashed border-brand-cream-400 bg-brand-cream-50 p-8 text-center text-sm text-brand-dark-500 flex flex-col items-center gap-2"
            >
              <ImageOff class="size-6" />
              No photos yet. Click Upload to add some.
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="shrink-0 border-t border-brand-cream-200 p-4 flex items-center justify-end gap-2 bg-brand-cream-50">
          <Button variant="ghost" :disabled="saving" @click="open = false">Cancel</Button>
          <Button :disabled="saving" @click="submit">
            {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Create gecko' }}
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
