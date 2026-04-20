<script setup lang="ts">
import { computed, ref } from 'vue';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Plus, Search, Filter } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import EmptyState from '@/components/layout/EmptyState.vue';
import GeckoCard from '@/components/GeckoCard.vue';
import { geckos } from '@/mock';
import type { GeckoStatus, Species } from '@/types';

const search = ref('');
const statusFilter = ref<GeckoStatus | 'All'>('All');
const speciesFilter = ref<Species | 'All'>('All');

const statuses: (GeckoStatus | 'All')[] = ['All', 'Breeding', 'Available', 'Hold', 'Personal', 'Sold'];
const speciesList: (Species | 'All')[] = ['All', 'Leopard Gecko', 'Crested Gecko', 'African Fat-Tail'];

const statusBadge: Record<GeckoStatus | 'All', BadgeVariants['variant']> = {
  All:       'outline',
  Breeding:  'soft',
  Available: 'success',
  Hold:      'warn',
  Personal:  'muted',
  Sold:      'outline',
};

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  return geckos.filter((g) => {
    if (statusFilter.value !== 'All' && g.status !== statusFilter.value) return false;
    if (speciesFilter.value !== 'All' && g.species !== speciesFilter.value) return false;
    if (!q) return true;
    return (
      g.name.toLowerCase().includes(q) ||
      g.code.toLowerCase().includes(q) ||
      g.morph.toLowerCase().includes(q) ||
      g.species.toLowerCase().includes(q)
    );
  });
});

function clearFilters() {
  search.value = '';
  statusFilter.value = 'All';
  speciesFilter.value = 'All';
}
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Collection"
      title="Geckos"
      subtitle="Everyone currently in the rack."
    >
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
        <Input
          v-model="search"
          placeholder="Search name, code, morph…"
          class="pl-9 bg-white"
        />
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
              {{ s }}
            </Badge>
          </button>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-xs text-brand-dark-600 mr-1">Species</span>
          <button
            v-for="sp in speciesList"
            :key="sp"
            type="button"
            class="focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
            @click="speciesFilter = sp"
          >
            <Badge
              :variant="speciesFilter === sp ? 'default' : 'outline'"
              :class="
                speciesFilter === sp
                  ? ''
                  : 'hover:bg-brand-cream-100 cursor-pointer'
              "
            >
              {{ sp }}
            </Badge>
          </button>
        </div>
      </div>
    </div>

    <!-- Grid -->
    <div
      v-if="filtered.length"
      class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3"
    >
      <GeckoCard v-for="g in filtered" :key="g.id" :gecko="g" />
    </div>

    <EmptyState
      v-else
      title="No geckos match that filter."
      description="Try clearing your filters — or add the first gecko to this view."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
      </template>
    </EmptyState>
  </div>
</template>
