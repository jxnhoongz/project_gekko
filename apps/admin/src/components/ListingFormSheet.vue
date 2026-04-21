<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { toast } from 'vue-sonner';
import {
  DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogClose,
} from 'reka-ui';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { X, Plus, Trash2, Info, Turtle, Boxes, Package } from 'lucide-vue-next';
import {
  useCreateListing, useUpdateListing, useListing,
} from '@/composables/useListings';
import type { ListingStatus, ListingType, ListingWritePayload } from '@/types/listing';
import type { Listing } from '@/types/listing';
import { LISTING_STATUS_LABEL, LISTING_TYPE_LABEL } from '@/types/listing';
import { useGeckos } from '@/composables/useGeckos';
import { useListings } from '@/composables/useListings';

const props = defineProps<{ listing?: Listing | null }>();
const open = defineModel<boolean>('open', { default: false });

const isEdit = computed(() => !!props.listing);

const editId = computed(() => (props.listing ? props.listing.id : null));
const { data: fullListing } = useListing(editId);

// ---- form state ----
const type = ref<ListingType>('GECKO');
const sku = ref('');
const title = ref('');
const description = ref('');
const priceUsd = ref('');
const depositUsd = ref('');
const status = ref<ListingStatus>('DRAFT');
const coverPhotoUrl = ref('');

const geckoIds = ref<number[]>([]);
const components = ref<{ component_listing_id: number; quantity: number }[]>([]);

function reset(l: Listing | null | undefined) {
  if (l) {
    type.value = l.type;
    sku.value = l.sku ?? '';
    title.value = l.title;
    description.value = l.description ?? '';
    priceUsd.value = l.price_usd;
    depositUsd.value = l.deposit_usd ?? '';
    status.value = l.status;
    coverPhotoUrl.value = l.cover_photo_url ?? '';
    geckoIds.value = (l.geckos ?? []).map((g) => g.gecko_id);
    components.value = (l.components ?? []).map((c) => ({
      component_listing_id: c.component_listing_id,
      quantity: c.quantity,
    }));
  } else {
    type.value = 'GECKO';
    sku.value = '';
    title.value = '';
    description.value = '';
    priceUsd.value = '';
    depositUsd.value = '';
    status.value = 'DRAFT';
    coverPhotoUrl.value = '';
    geckoIds.value = [];
    components.value = [];
  }
}

watch(open, (v) => {
  if (v) reset(props.listing ?? null);
});

watch(fullListing, (l) => {
  if (!l || !open.value) return;
  geckoIds.value = (l.geckos ?? []).map((g) => g.gecko_id);
  components.value = (l.components ?? []).map((c) => ({
    component_listing_id: c.component_listing_id,
    quantity: c.quantity,
  }));
});

// ---- picker data ----
const { data: geckosData } = useGeckos();
const allGeckos = computed(() => geckosData.value?.geckos ?? []);

const { data: listingsData } = useListings();
const candidateComponents = computed(() => {
  const list = listingsData.value?.listings ?? [];
  return list.filter((l) => l.type !== 'PACKAGE' && (!props.listing || l.id !== props.listing.id));
});

function addGecko(id: number) {
  if (!geckoIds.value.includes(id)) geckoIds.value.push(id);
}
function removeGecko(id: number) {
  geckoIds.value = geckoIds.value.filter((g) => g !== id);
}

function addComponent(id: number) {
  if (!components.value.find((c) => c.component_listing_id === id)) {
    components.value.push({ component_listing_id: id, quantity: 1 });
  }
}
function removeComponent(id: number) {
  components.value = components.value.filter((c) => c.component_listing_id !== id);
}

const componentTotal = computed(() => {
  return components.value.reduce((sum, row) => {
    const c = candidateComponents.value.find((x) => x.id === row.component_listing_id);
    if (!c) return sum;
    return sum + Number(c.price_usd) * row.quantity;
  }, 0);
});

const createMut = useCreateListing();
const updateMut = useUpdateListing();
const saving = computed(() => createMut.isPending.value || updateMut.isPending.value);

async function submit() {
  if (!title.value.trim()) {
    toast.error('Title is required.');
    return;
  }
  if (!priceUsd.value.trim()) {
    toast.error('Price is required.');
    return;
  }
  if (type.value === 'SUPPLY' && !sku.value.trim()) {
    toast.error('SKU is required for supply listings.');
    return;
  }
  if (type.value === 'GECKO' && geckoIds.value.length === 0) {
    toast.error('Add at least one gecko.');
    return;
  }
  if (type.value === 'PACKAGE' && components.value.length === 0) {
    toast.error('Add at least one component.');
    return;
  }

  const payload: ListingWritePayload = {
    sku: sku.value.trim(),
    type: type.value,
    title: title.value.trim(),
    description: description.value.trim(),
    price_usd: priceUsd.value.trim(),
    deposit_usd: depositUsd.value.trim(),
    status: status.value,
    cover_photo_url: coverPhotoUrl.value.trim(),
    geckos: geckoIds.value.map((id) => ({ gecko_id: id })),
    components: components.value.slice(),
  };

  try {
    const result = props.listing
      ? await updateMut.mutateAsync({ id: props.listing.id, payload })
      : await createMut.mutateAsync(payload);
    toast.success(props.listing ? 'Listing updated.' : `Created "${result.title}".`);
    open.value = false;
  } catch (e: unknown) {
    const msg =
      (e as any)?.response?.data?.error ??
      (e as Error).message ??
      'Save failed';
    toast.error(String(msg));
  }
}

const showGeckoPicker = ref(false);
const showComponentPicker = ref(false);

const statuses: ListingStatus[] = ['DRAFT', 'LISTED', 'RESERVED', 'SOLD', 'ARCHIVED'];
const typeOptions: { value: ListingType; icon: typeof Turtle; label: string }[] = [
  { value: 'GECKO',   icon: Turtle,  label: 'Gecko' },
  { value: 'SUPPLY',  icon: Boxes,   label: 'Supply' },
  { value: 'PACKAGE', icon: Package, label: 'Package' },
];
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-brand-dark-950/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
      />
      <DialogContent
        class="fixed inset-y-0 right-0 z-50 w-[min(600px,100vw)] bg-brand-cream-50 border-l border-brand-cream-300 shadow-2xl overflow-hidden flex flex-col data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right data-[state=closed]:duration-300 data-[state=open]:duration-500"
        aria-describedby=""
      >
        <div class="flex items-start justify-between px-6 py-5 border-b border-brand-cream-200 shrink-0">
          <div class="flex flex-col gap-1">
            <span class="text-xs uppercase tracking-[0.16em] text-brand-gold-700 font-semibold">
              {{ isEdit ? 'Edit' : 'New listing' }}
            </span>
            <h2 class="font-serif text-2xl text-brand-dark-950 leading-tight">
              {{ isEdit ? `Edit "${listing?.title}"` : 'Create a listing' }}
            </h2>
            <div v-if="!isEdit" class="flex items-center gap-1 text-xs text-brand-dark-600 mt-1">
              <Info class="size-3" />
              Pick a type below. Type is locked once created.
            </div>
            <div v-else class="text-xs text-brand-dark-500 font-mono mt-1">
              #{{ listing?.id }} · {{ LISTING_TYPE_LABEL[type] }}
            </div>
          </div>
          <DialogClose
            class="rounded-md p-1 text-brand-dark-600 hover:bg-brand-cream-200 hover:text-brand-dark-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Close"
          >
            <X class="size-5" />
          </DialogClose>
        </div>

        <div class="flex-1 overflow-y-auto px-6 py-5 flex flex-col gap-5">
          <!-- Type selector (create only) -->
          <div v-if="!isEdit" class="flex flex-col gap-2">
            <Label>Type</Label>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="opt in typeOptions"
                :key="opt.value"
                type="button"
                class="rounded-lg border p-3 text-left transition-colors"
                :class="type === opt.value
                  ? 'border-brand-gold-600 bg-brand-gold-100 text-brand-gold-900'
                  : 'border-brand-cream-300 bg-white hover:bg-brand-cream-100'"
                @click="type = opt.value"
              >
                <component :is="opt.icon" class="size-4 mb-1" />
                <div class="text-sm font-medium">{{ opt.label }}</div>
              </button>
            </div>
          </div>

          <!-- Title + SKU -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="lf-title">Title <span class="text-destructive">*</span></Label>
              <Input id="lf-title" v-model="title" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="lf-sku">
                SKU
                <span v-if="type === 'SUPPLY'" class="text-destructive">*</span>
              </Label>
              <Input id="lf-sku" v-model="sku" placeholder="e.g. TANK-20G" class="bg-white" />
            </div>
          </div>

          <!-- Description -->
          <div class="flex flex-col gap-2">
            <Label for="lf-desc">Description</Label>
            <textarea
              id="lf-desc"
              v-model="description"
              rows="3"
              class="rounded-md border border-brand-cream-300 bg-white px-3 py-2 text-sm resize-y"
              placeholder="Customer-facing copy."
            />
          </div>

          <!-- Price + Deposit + Status -->
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="lf-price">Price (USD) <span class="text-destructive">*</span></Label>
              <Input id="lf-price" v-model="priceUsd" type="number" step="0.01" min="0" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="lf-deposit">Deposit (USD)</Label>
              <Input id="lf-deposit" v-model="depositUsd" type="number" step="0.01" min="0" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="lf-status">Status</Label>
              <select
                id="lf-status"
                v-model="status"
                class="h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-sm"
              >
                <option v-for="s in statuses" :key="s" :value="s">{{ LISTING_STATUS_LABEL[s] }}</option>
              </select>
            </div>
          </div>

          <!-- Cover photo url -->
          <div class="flex flex-col gap-2">
            <Label for="lf-cover">Cover photo URL</Label>
            <Input id="lf-cover" v-model="coverPhotoUrl" placeholder="https://… or /uploads/…" class="bg-white" />
            <div class="text-xs text-brand-dark-500">
              For GECKO listings this stays blank — the storefront pulls from the gecko's photos.
            </div>
          </div>

          <Separator />

          <!-- GECKO junction -->
          <div v-if="type === 'GECKO'" class="flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <h3 class="font-serif text-lg">Geckos</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="showGeckoPicker = !showGeckoPicker"
              >
                <Plus class="size-4" /> Add gecko
              </Button>
            </div>

            <div v-if="showGeckoPicker" class="rounded-lg border border-brand-cream-300 bg-white p-2 max-h-60 overflow-y-auto">
              <ul>
                <li
                  v-for="g in allGeckos.filter((x) => !geckoIds.includes(x.id))"
                  :key="g.id"
                  class="px-3 py-1.5 text-sm cursor-pointer hover:bg-brand-cream-100 flex items-center gap-2 rounded"
                  @click="addGecko(g.id); showGeckoPicker = false"
                >
                  <span class="font-mono text-brand-dark-700">{{ g.code }}</span>
                  <span v-if="g.name" class="text-brand-dark-950">· {{ g.name }}</span>
                </li>
                <li
                  v-if="allGeckos.filter((x) => !geckoIds.includes(x.id)).length === 0"
                  class="px-3 py-4 text-xs text-brand-dark-500 text-center"
                >
                  Nothing left to add.
                </li>
              </ul>
            </div>

            <div v-if="geckoIds.length" class="flex flex-wrap gap-2">
              <Badge
                v-for="id in geckoIds"
                :key="id"
                variant="soft"
                class="flex items-center gap-1 pr-1"
              >
                {{ allGeckos.find((g) => g.id === id)?.code ?? '#' + id }}
                <button
                  type="button"
                  class="ml-1 size-4 rounded hover:bg-brand-gold-200 flex items-center justify-center"
                  aria-label="Remove"
                  @click="removeGecko(id)"
                >
                  <X class="size-3" />
                </button>
              </Badge>
            </div>
            <div v-else class="text-xs text-brand-dark-500">Add at least one gecko.</div>
          </div>

          <!-- PACKAGE junction -->
          <div v-if="type === 'PACKAGE'" class="flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="font-serif text-lg">Components</h3>
                <p class="text-xs text-brand-dark-600">
                  Pick supply or gecko listings to include. Packages can't contain other packages.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="showComponentPicker = !showComponentPicker"
              >
                <Plus class="size-4" /> Add component
              </Button>
            </div>

            <div v-if="showComponentPicker" class="rounded-lg border border-brand-cream-300 bg-white p-2 max-h-60 overflow-y-auto">
              <ul>
                <li
                  v-for="c in candidateComponents.filter((x) => !components.find((r) => r.component_listing_id === x.id))"
                  :key="c.id"
                  class="px-3 py-1.5 text-sm cursor-pointer hover:bg-brand-cream-100 flex items-center gap-2 rounded"
                  @click="addComponent(c.id); showComponentPicker = false"
                >
                  <Badge :variant="c.type === 'GECKO' ? 'soft' : 'muted'" class="text-[10px]">
                    {{ LISTING_TYPE_LABEL[c.type] }}
                  </Badge>
                  <span class="flex-1 truncate">{{ c.title }}</span>
                  <span class="text-xs text-brand-dark-500">${{ c.price_usd }}</span>
                </li>
              </ul>
            </div>

            <ul v-if="components.length" class="flex flex-col gap-2">
              <li
                v-for="row in components"
                :key="row.component_listing_id"
                class="flex items-center gap-2 rounded-lg border border-brand-cream-300 bg-white p-2"
              >
                <span class="flex-1 truncate text-sm">
                  {{ candidateComponents.find((c) => c.id === row.component_listing_id)?.title ?? '#' + row.component_listing_id }}
                </span>
                <Input
                  type="number"
                  min="1"
                  v-model.number="row.quantity"
                  class="w-20 h-8 bg-white"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Remove"
                  @click="removeComponent(row.component_listing_id)"
                >
                  <Trash2 class="size-4 text-red-700" />
                </Button>
              </li>
            </ul>
            <div v-else class="text-xs text-brand-dark-500">Add at least one component.</div>

            <div v-if="components.length" class="text-xs text-brand-dark-600">
              Components total: <span class="font-semibold text-brand-dark-950">${{ componentTotal.toFixed(2) }}</span>
              <span v-if="Number(priceUsd || 0) && Math.abs(componentTotal - Number(priceUsd)) > 0.01"> · listing price ${{ priceUsd }} is set independently.</span>
            </div>
          </div>
        </div>

        <div class="shrink-0 border-t border-brand-cream-200 p-4 flex items-center justify-end gap-2 bg-brand-cream-50">
          <Button variant="ghost" :disabled="saving" @click="open = false">Cancel</Button>
          <Button :disabled="saving" @click="submit">
            {{ saving ? 'Saving…' : isEdit ? 'Save changes' : 'Create listing' }}
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
