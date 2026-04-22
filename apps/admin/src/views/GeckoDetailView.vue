<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { toast } from 'vue-sonner';
import { Card } from '@/components/ui/card';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import {
  ArrowLeft,
  Mars,
  Venus,
  HelpCircle,
  Scale,
  Sparkles,
  Tag,
  Image as ImageIcon,
  Plus,
  AlertTriangle,
  Pencil,
  Trash2,
} from 'lucide-vue-next';
import EmptyState from '@/components/layout/EmptyState.vue';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import WeightSparkline from '@/components/WeightSparkline.vue';
import GeckoFormSheet from '@/components/GeckoFormSheet.vue';
import ListingFormSheet from '@/components/ListingFormSheet.vue';
import { useGecko, useDeleteGecko } from '@/composables/useGeckos';
import { useListings } from '@/composables/useListings';
import type { GeckoStatus, Sex, Zygosity } from '@/types/gecko';
import { STATUS_LABEL, SEX_LABEL } from '@/types/gecko';
import { LISTING_STATUS_LABEL, LISTING_TYPE_LABEL, type Listing } from '@/types/listing';
import { formatDate, ageFromBirth } from '@/lib/format';

const props = defineProps<{ id: string }>();
const router = useRouter();

const idNum = computed(() => Number(props.id));
const { data: gecko, isLoading, isError, error } = useGecko(idNum);

const editOpen = ref(false);
const deleteMut = useDeleteGecko();

const { data: listingsData } = useListings();
const attachedListings = computed((): Listing[] => {
  if (!gecko.value || !listingsData.value) return [];
  const g = gecko.value;
  return listingsData.value.listings.filter(
    (l) => l.type === 'GECKO' && l.title === (g.name || g.code),
  );
});

const listingOpen = ref(false);
const listingDraft = ref<Listing | null>(null);

function openCreateListingForGecko() {
  listingDraft.value = null;
  listingOpen.value = true;
}

async function onDelete() {
  if (!gecko.value) return;
  const label = gecko.value.name || gecko.value.code;
  if (!window.confirm(`Delete ${label}? This permanently removes its genes and photos too.`)) {
    return;
  }
  try {
    await deleteMut.mutateAsync(gecko.value.id);
    toast.success(`Deleted ${label}.`);
    router.push({ name: 'geckos' });
  } catch (e: unknown) {
    toast.error((e as Error).message ?? 'Delete failed');
  }
}

const statusVariant: Record<GeckoStatus, BadgeVariants['variant']> = {
  BREEDING:  'soft',
  AVAILABLE: 'success',
  HOLD:      'warn',
  PERSONAL:  'muted',
  SOLD:      'outline',
  DECEASED:  'outline',
};

const sexIcon: Record<Sex, typeof Mars> = { M: Mars, F: Venus, U: HelpCircle };

const zygBadge: Record<Zygosity, BadgeVariants['variant']> = {
  HOM:      'soft',
  HET:      'muted',
  POSS_HET: 'outline',
};

function back() {
  router.push({ name: 'geckos' });
}
</script>

<template>
  <div>
  <!-- Loading -->
  <div v-if="isLoading" class="flex flex-col gap-6">
    <Skeleton class="h-6 w-24" />
    <Skeleton class="h-64 w-full rounded-xl" />
    <Skeleton class="h-10 w-80" />
    <Skeleton class="h-64 w-full rounded-xl" />
  </div>

  <!-- Error -->
  <div v-else-if="isError" class="flex flex-col gap-6">
    <Card class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3">
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex-1">
        <div class="text-sm font-semibold text-red-900">Couldn't load gecko.</div>
        <div class="text-xs text-red-800 break-all">{{ (error as Error)?.message }}</div>
      </div>
      <Button variant="outline" size="sm" @click="back">Back</Button>
    </Card>
  </div>

  <!-- Not found -->
  <div v-else-if="!gecko" class="flex flex-col gap-6">
    <EmptyState title="Gecko not found" description="That record doesn't exist in your collection.">
      <template #actions>
        <Button variant="outline" size="sm" @click="back">
          <ArrowLeft class="size-4" /> Back to geckos
        </Button>
      </template>
    </EmptyState>
  </div>

  <!-- Detail -->
  <div v-else class="flex flex-col gap-8">
    <div class="flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" @click="back">
        <ArrowLeft class="size-4" /> Geckos
      </Button>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" @click="editOpen = true">
          <Pencil class="size-4" /> Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="deleteMut.isPending.value"
          class="!text-red-700 hover:!bg-red-50"
          @click="onDelete"
        >
          <Trash2 class="size-4" /> Delete
        </Button>
      </div>
    </div>

    <!-- Hero -->
    <Card class="relative overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
      <div class="grid grid-cols-1 md:grid-cols-[280px_1fr] gap-0">
        <div
          class="relative bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center aspect-square md:aspect-auto md:h-full p-8"
        >
          <img
            v-if="gecko.cover_photo_url"
            :src="gecko.cover_photo_url"
            :alt="gecko.name"
            class="w-full h-full object-cover"
          />
          <LowPolyGecko v-else :size="240" class="animate-float" />
        </div>
        <div class="p-6 md:p-8 flex flex-col gap-4">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="statusVariant[gecko.status]">{{ STATUS_LABEL[gecko.status] }}</Badge>
            <Badge variant="outline">{{ gecko.code }}</Badge>
            <span class="inline-flex items-center gap-1 text-xs text-brand-dark-600">
              <component :is="sexIcon[gecko.sex]" class="size-3.5" stroke-width="2" />
              {{ SEX_LABEL[gecko.sex] }}
            </span>
          </div>
          <div class="flex flex-col gap-1">
            <h1 class="font-serif text-4xl md:text-5xl leading-none">
              {{ gecko.name || 'Unnamed' }}
            </h1>
            <p class="text-brand-dark-600">
              {{ gecko.species_name }} ·
              <span class="text-brand-dark-700 font-medium">{{ gecko.morph_label }}</span>
            </p>
          </div>
          <p v-if="gecko.notes" class="text-sm text-brand-dark-700 max-w-xl">
            {{ gecko.notes }}
          </p>

          <dl class="grid grid-cols-2 sm:grid-cols-3 gap-4 pt-4 border-t border-brand-cream-200 mt-auto">
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Age</dt>
              <dd class="font-serif text-xl text-brand-dark-950">
                {{ gecko.hatch_date ? ageFromBirth(gecko.hatch_date) : '—' }}
              </dd>
            </div>
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Traits</dt>
              <dd class="font-serif text-xl text-brand-dark-950">{{ gecko.traits.length }}</dd>
            </div>
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Born</dt>
              <dd class="font-serif text-xl text-brand-dark-950">
                {{ gecko.hatch_date ? formatDate(gecko.hatch_date) : '—' }}
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </Card>

    <!-- Listings (commerce) -->
    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Tag class="size-4 text-brand-gold-700" />
          <h3 class="font-serif text-xl text-brand-dark-950">Listings</h3>
        </div>
        <Button variant="outline" size="sm" @click="openCreateListingForGecko">
          <Plus class="size-4" /> Create listing
        </Button>
      </div>
      <div v-if="attachedListings.length" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div
          v-for="l in attachedListings"
          :key="l.id"
          class="rounded-lg border border-brand-cream-300 bg-brand-cream-50 p-3 flex items-center gap-3 cursor-pointer hover:bg-brand-cream-100"
          @click="listingDraft = l; listingOpen = true"
        >
          <div class="flex flex-col min-w-0 flex-1">
            <div class="flex items-center gap-2 mb-1">
              <Badge variant="soft">{{ LISTING_TYPE_LABEL[l.type] }}</Badge>
              <Badge :variant="l.status === 'LISTED' ? 'success' : 'muted'">{{ LISTING_STATUS_LABEL[l.status] }}</Badge>
            </div>
            <span class="text-sm font-medium text-brand-dark-950 truncate">{{ l.title }}</span>
          </div>
          <div class="text-brand-gold-700 font-semibold">${{ l.price_usd }}</div>
        </div>
      </div>
      <p v-else class="text-sm text-brand-dark-500">
        No listings for this gecko yet. Click "Create listing" to add one.
      </p>
    </section>

    <!-- Tabs -->
    <Tabs default-value="overview">
      <TabsList class="w-full sm:w-auto">
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="genetics">Genetics</TabsTrigger>
        <TabsTrigger value="lineage">Lineage</TabsTrigger>
        <TabsTrigger value="husbandry">Husbandry</TabsTrigger>
        <TabsTrigger value="photos">Photos</TabsTrigger>
      </TabsList>

      <!-- Overview -->
      <TabsContent value="overview">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
            <div class="px-6 py-5 flex items-center gap-2">
              <Sparkles class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">Traits</h3>
              <span class="ml-auto text-xs text-brand-dark-600">{{ gecko.traits.length }}</span>
            </div>
            <Separator />
            <ul v-if="gecko.traits.length" class="divide-y divide-brand-cream-200">
              <li
                v-for="t in gecko.traits"
                :key="t.trait_id"
                class="flex items-center justify-between px-6 py-3"
              >
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-brand-dark-950">{{ t.trait_name }}</span>
                  <span
                    v-if="t.trait_code"
                    class="font-mono text-[10px] text-brand-dark-500"
                    >{{ t.trait_code }}</span
                  >
                </div>
                <Badge :variant="zygBadge[t.zygosity]">{{ t.zygosity }}</Badge>
              </li>
            </ul>
            <div v-else class="px-6 py-6 text-sm text-brand-dark-600">
              No genetics assigned yet.
            </div>
          </Card>

          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
            <div class="px-6 py-5 flex items-center gap-2">
              <Scale class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">Weight trend</h3>
            </div>
            <Separator />
            <div class="p-6">
              <p class="text-sm text-brand-dark-600">
                Weight logging comes in Phase 7. The chart will light up once husbandry
                logging lands.
              </p>
              <WeightSparkline :points="[]" class="mt-4 opacity-40" />
            </div>
          </Card>
        </div>
      </TabsContent>

      <!-- Genetics (same shape as overview's Traits card but full width) -->
      <TabsContent value="genetics">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
          <div class="px-6 py-5 flex items-center justify-between">
            <h3 class="font-serif text-lg">Genetic profile</h3>
            <Button size="sm" disabled><Plus class="size-4" /> Assign trait</Button>
          </div>
          <Separator />
          <ul v-if="gecko.traits.length" class="divide-y divide-brand-cream-200">
            <li
              v-for="t in gecko.traits"
              :key="t.trait_id"
              class="flex items-center justify-between px-6 py-4"
            >
              <div class="flex flex-col gap-0.5">
                <span class="font-medium text-brand-dark-950">{{ t.trait_name }}</span>
                <span class="text-xs text-brand-dark-500">
                  {{ t.trait_code || '—' }}
                  <span v-if="t.is_dominant" class="ml-2 text-brand-gold-700 font-medium">dominant</span>
                </span>
              </div>
              <Badge :variant="zygBadge[t.zygosity]">{{ t.zygosity }}</Badge>
            </li>
          </ul>
          <div v-else class="p-6">
            <EmptyState
              title="No traits assigned."
              description="Use 'Assign trait' to record this gecko's genetic profile once the Add/Edit flow lands."
            />
          </div>
        </Card>
      </TabsContent>

      <!-- Lineage -->
      <TabsContent value="lineage">
        <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
          <div class="flex flex-col gap-4">
            <h3 class="font-serif text-lg">Parentage</h3>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="rounded-lg border border-brand-cream-300 p-4 bg-white">
                <div class="text-xs uppercase tracking-wider text-brand-dark-600">Sire</div>
                <div class="font-serif text-xl mt-1">
                  {{ gecko.sire_id ? '#' + gecko.sire_id : '—' }}
                </div>
              </div>
              <div class="rounded-lg border border-brand-cream-300 p-4 bg-white">
                <div class="text-xs uppercase tracking-wider text-brand-dark-600">Dam</div>
                <div class="font-serif text-xl mt-1">
                  {{ gecko.dam_id ? '#' + gecko.dam_id : '—' }}
                </div>
              </div>
            </div>
            <p class="text-xs text-brand-dark-500">
              Self-referencing parentage is wired up in the schema.
              Full pedigree walking arrives with the breeding phase.
            </p>
          </div>
        </Card>
      </TabsContent>

      <!-- Husbandry -->
      <TabsContent value="husbandry">
        <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
          <EmptyState
            title="Husbandry logging lands in Phase 7."
            description="Feedings, weights, sheds and health notes will show up here as daily logging becomes a habit."
          />
        </Card>
      </TabsContent>

      <!-- Photos -->
      <TabsContent value="photos">
        <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
          <div v-if="gecko.photos && gecko.photos.length" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            <img
              v-for="p in gecko.photos"
              :key="p.id"
              :src="p.url"
              :alt="p.caption"
              class="w-full aspect-square object-cover rounded-lg"
            />
          </div>
          <EmptyState
            v-else
            title="No photos yet."
            description="Upload photos to build a growth gallery for this gecko."
          >
            <template #actions>
              <Button variant="default" size="sm" disabled>
                <ImageIcon class="size-4" /> Upload (coming soon)
              </Button>
            </template>
          </EmptyState>
        </Card>
      </TabsContent>
    </Tabs>

    <!-- Edit drawer -->
    <GeckoFormSheet v-model:open="editOpen" :gecko="gecko" />
    <ListingFormSheet v-model:open="listingOpen" :listing="listingDraft" />
  </div>
  </div>
</template>

<style scoped>
.animate-float {
  animation: gecko-float 6s ease-in-out infinite;
}
@keyframes gecko-float {
  0%, 100% { transform: translateY(0); }
  50%      { transform: translateY(-6px); }
}
</style>
