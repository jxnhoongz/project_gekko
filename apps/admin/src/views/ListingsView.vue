<script setup lang="ts">
import { computed, ref } from 'vue';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Plus, Search, Filter, AlertTriangle } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import EmptyState from '@/components/layout/EmptyState.vue';
import ListingCard from '@/components/ListingCard.vue';
import ListingFormSheet from '@/components/ListingFormSheet.vue';
import { useListings } from '@/composables/useListings';
import type { Listing, ListingStatus, ListingType } from '@/types/listing';
import { LISTING_STATUS_LABEL, LISTING_TYPE_LABEL } from '@/types/listing';

const { data, isLoading, isError, error, refetch } = useListings();

const search = ref('');
const typeFilter = ref<ListingType | 'ALL'>('ALL');
const statusFilter = ref<ListingStatus | 'ALL'>('ALL');

const types: (ListingType | 'ALL')[] = ['ALL', 'GECKO', 'PACKAGE', 'SUPPLY'];
const statuses: (ListingStatus | 'ALL')[] = ['ALL', 'LISTED', 'DRAFT', 'RESERVED', 'SOLD', 'ARCHIVED'];

const editOpen = ref(false);
const editing = ref<Listing | null>(null);

function createNew() {
  editing.value = null;
  editOpen.value = true;
}

function onEdit(l: Listing) {
  editing.value = l;
  editOpen.value = true;
}

const filtered = computed(() => {
  const list = data.value?.listings ?? [];
  const q = search.value.trim().toLowerCase();
  return list.filter((l) => {
    if (typeFilter.value !== 'ALL' && l.type !== typeFilter.value) return false;
    if (statusFilter.value !== 'ALL' && l.status !== statusFilter.value) return false;
    if (!q) return true;
    return (
      l.title.toLowerCase().includes(q) ||
      (l.sku ?? '').toLowerCase().includes(q) ||
      (l.description ?? '').toLowerCase().includes(q)
    );
  });
});

function clearFilters() {
  search.value = '';
  typeFilter.value = 'ALL';
  statusFilter.value = 'ALL';
}

function typeLabel(t: ListingType | 'ALL') {
  return t === 'ALL' ? 'All' : LISTING_TYPE_LABEL[t];
}
function statusLabel(s: ListingStatus | 'ALL') {
  return s === 'ALL' ? 'All' : LISTING_STATUS_LABEL[s];
}

const typeBadgeVariant = (t: ListingType | 'ALL'): BadgeVariants['variant'] =>
  t === 'ALL' ? 'outline' : t === 'GECKO' ? 'soft' : t === 'PACKAGE' ? 'accent' : 'muted';
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Commerce"
      title="Listings"
      subtitle="Individual geckos, supply items, and bundled packages — anything you sell."
    >
      <template #actions>
        <Button variant="default" size="sm" @click="createNew">
          <Plus class="size-4" />
          Create listing
        </Button>
      </template>
    </PageHeader>

    <!-- Filter bar -->
    <div class="flex flex-col lg:flex-row gap-3 lg:items-center lg:justify-between rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-4">
      <div class="relative flex-1 lg:max-w-sm">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-brand-dark-500 pointer-events-none" />
        <Input v-model="search" placeholder="Search title, SKU, description…" class="pl-9 bg-white" />
      </div>
      <div class="flex flex-col sm:flex-row gap-3 lg:items-center">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="flex items-center gap-1 text-xs text-brand-dark-600 mr-1"><Filter class="size-3" /> Type</span>
          <button
            v-for="t in types"
            :key="t"
            type="button"
            class="focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
            @click="typeFilter = t"
          >
            <Badge
              :variant="typeFilter === t ? typeBadgeVariant(t) : 'outline'"
              :class="typeFilter === t ? 'ring-2 ring-brand-gold-400/40' : 'hover:bg-brand-cream-100 cursor-pointer'"
            >{{ typeLabel(t) }}</Badge>
          </button>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-xs text-brand-dark-600 mr-1">Status</span>
          <button
            v-for="s in statuses"
            :key="s"
            type="button"
            class="focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
            @click="statusFilter = s"
          >
            <Badge
              :variant="statusFilter === s ? 'default' : 'outline'"
              :class="statusFilter === s ? '' : 'hover:bg-brand-cream-100 cursor-pointer'"
            >{{ statusLabel(s) }}</Badge>
          </button>
        </div>
      </div>
    </div>

    <!-- Error -->
    <Card
      v-if="isError"
      class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3"
    >
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex-1 min-w-0">
        <div class="text-sm font-semibold text-red-900">Couldn't load listings.</div>
        <div class="text-xs text-red-800 break-all">{{ (error as Error)?.message }}</div>
      </div>
      <Button variant="outline" size="sm" @click="refetch()">Retry</Button>
    </Card>

    <!-- Loading -->
    <div v-else-if="isLoading" class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
      <Skeleton v-for="n in 6" :key="n" class="h-72 rounded-xl" />
    </div>

    <!-- Grid -->
    <div v-else-if="filtered.length" class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
      <ListingCard v-for="l in filtered" :key="l.id" :listing="l" @edit="onEdit" />
    </div>

    <EmptyState
      v-else-if="(data?.listings?.length ?? 0) === 0"
      title="No listings yet."
      description="Create your first listing — a gecko, a supply item, or a starter-kit package."
    >
      <template #actions>
        <Button variant="default" size="sm" @click="createNew"><Plus class="size-4" /> Create listing</Button>
      </template>
    </EmptyState>

    <EmptyState
      v-else
      title="No listings match that filter."
      description="Try clearing your filters."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
      </template>
    </EmptyState>

    <ListingFormSheet v-model:open="editOpen" :listing="editing" />
  </div>
</template>
