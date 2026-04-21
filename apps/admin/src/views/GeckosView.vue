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
import GeckoCard from '@/components/GeckoCard.vue';
import { useGeckos } from '@/composables/useGeckos';
import type { GeckoStatus } from '@/types/gecko';
import { STATUS_LABEL } from '@/types/gecko';

const { data, isLoading, isError, error, refetch } = useGeckos();

const search = ref('');
const statusFilter = ref<GeckoStatus | 'ALL'>('ALL');
const speciesFilter = ref<string>('ALL'); // 'ALL' | species_code ('LP'/'CR'/'AF')

const statuses: (GeckoStatus | 'ALL')[] = [
  'ALL', 'BREEDING', 'AVAILABLE', 'HOLD', 'PERSONAL', 'SOLD',
];
const speciesList = [
  { code: 'ALL', label: 'All' },
  { code: 'LP',  label: 'Leopard' },
  { code: 'CR',  label: 'Crested' },
  { code: 'AF',  label: 'Fat-tail' },
];

const statusBadge: Record<GeckoStatus | 'ALL', BadgeVariants['variant']> = {
  ALL:       'outline',
  BREEDING:  'soft',
  AVAILABLE: 'success',
  HOLD:      'warn',
  PERSONAL:  'muted',
  SOLD:      'outline',
  DECEASED:  'outline',
};

const filtered = computed(() => {
  const geckos = data.value?.geckos ?? [];
  const q = search.value.trim().toLowerCase();
  return geckos.filter((g) => {
    if (statusFilter.value !== 'ALL' && g.status !== statusFilter.value) return false;
    if (speciesFilter.value !== 'ALL' && g.species_code !== speciesFilter.value) return false;
    if (!q) return true;
    const morphStr = g.traits.map((t) => t.trait_name).join(' ').toLowerCase();
    return (
      g.name?.toLowerCase().includes(q) ||
      g.code.toLowerCase().includes(q) ||
      g.species_name.toLowerCase().includes(q) ||
      morphStr.includes(q)
    );
  });
});

function clearFilters() {
  search.value = '';
  statusFilter.value = 'ALL';
  speciesFilter.value = 'ALL';
}

function statusLabel(s: GeckoStatus | 'ALL') {
  if (s === 'ALL') return 'All';
  return STATUS_LABEL[s];
}
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader eyebrow="Collection" title="Geckos" subtitle="Everyone currently in the rack.">
      <template #actions>
        <Button variant="default" size="sm">
          <Plus class="size-4" />
          Add gecko
        </Button>
      </template>
    </PageHeader>

    <!-- Filter bar -->
    <div
      class="flex flex-col lg:flex-row gap-3 lg:items-center lg:justify-between rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-4"
    >
      <div class="relative flex-1 lg:max-w-sm">
        <Search
          class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-brand-dark-500 pointer-events-none"
        />
        <Input v-model="search" placeholder="Search name, code, morph…" class="pl-9 bg-white" />
      </div>
      <div class="flex flex-col sm:flex-row gap-3 lg:items-center">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="flex items-center gap-1 text-xs text-brand-dark-600 mr-1">
            <Filter class="size-3" /> Status
          </span>
          <button
            v-for="s in statuses"
            :key="s"
            type="button"
            class="focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
            @click="statusFilter = s"
          >
            <Badge
              :variant="statusFilter === s ? statusBadge[s] : 'outline'"
              :class="
                statusFilter === s
                  ? 'ring-2 ring-brand-gold-400/40'
                  : 'hover:bg-brand-cream-100 cursor-pointer'
              "
            >
              {{ statusLabel(s) }}
            </Badge>
          </button>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-xs text-brand-dark-600 mr-1">Species</span>
          <button
            v-for="sp in speciesList"
            :key="sp.code"
            type="button"
            class="focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
            @click="speciesFilter = sp.code"
          >
            <Badge
              :variant="speciesFilter === sp.code ? 'default' : 'outline'"
              :class="
                speciesFilter === sp.code
                  ? ''
                  : 'hover:bg-brand-cream-100 cursor-pointer'
              "
            >
              {{ sp.label }}
            </Badge>
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
        <div class="text-sm font-semibold text-red-900">Couldn't load geckos.</div>
        <div class="text-xs text-red-800 break-all">{{ (error as Error)?.message }}</div>
      </div>
      <Button variant="outline" size="sm" @click="refetch()">Retry</Button>
    </Card>

    <!-- Loading -->
    <div
      v-else-if="isLoading"
      class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"
    >
      <Skeleton v-for="n in 6" :key="n" class="h-72 rounded-xl" />
    </div>

    <!-- Grid -->
    <div
      v-else-if="filtered.length"
      class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3"
    >
      <GeckoCard v-for="g in filtered" :key="g.id" :gecko="g" />
    </div>

    <EmptyState
      v-else-if="(data?.geckos?.length ?? 0) === 0"
      title="No geckos yet."
      description="Your collection is empty. Add your first gecko when you're ready."
    >
      <template #actions>
        <Button variant="default" size="sm"><Plus class="size-4" /> Add gecko</Button>
      </template>
    </EmptyState>

    <EmptyState
      v-else
      title="No geckos match that filter."
      description="Try clearing your filters to see the full rack."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
      </template>
    </EmptyState>
  </div>
</template>
