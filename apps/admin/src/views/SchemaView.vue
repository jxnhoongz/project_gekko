<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Database,
  Search,
  RefreshCw,
  Key,
  AlertTriangle,
} from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import ERGraph from '@/components/schema/ERGraph.vue';
import TableDetail from '@/components/schema/TableDetail.vue';
import { api } from '@/lib/api';
import type { DbTable, SchemaResponse } from '@/types/schema';

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const error = ref<string | null>(null);
const tables = ref<DbTable[]>([]);
const selected = ref<string | null>(
  typeof route.query.t === 'string' ? route.query.t : null,
);
const search = ref('');

async function load() {
  loading.value = true;
  error.value = null;
  try {
    const res = await api.get<SchemaResponse>('/api/admin/schema');
    tables.value = res.data.tables;
    // auto-select first non-goose table
    const first = tables.value.find((t) => !t.name.startsWith('goose_'));
    if (first && !selected.value) selected.value = first.name;
  } catch (e: unknown) {
    error.value = (e as Error).message ?? 'Failed to load schema';
  } finally {
    loading.value = false;
  }
}
onMounted(load);

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return tables.value;
  return tables.value.filter((t) => t.name.toLowerCase().includes(q));
});

const selectedTable = computed(() =>
  tables.value.find((t) => t.name === selected.value) ?? null,
);

function selectTable(name: string) {
  selected.value = name;
  router.replace({ query: { ...route.query, t: name } });
}

watch(
  () => route.query.t,
  (v) => {
    if (typeof v === 'string' && v !== selected.value) {
      selected.value = v;
    }
  },
);

const userTableCount = computed(
  () => tables.value.filter((t) => !t.name.startsWith('goose_')).length,
);

const totalRows = computed(() =>
  tables.value
    .filter((t) => !t.name.startsWith('goose_'))
    .reduce((sum, t) => sum + Math.max(0, t.row_count), 0),
);

const totalFks = computed(() =>
  tables.value.reduce((sum, t) => sum + (t.foreign_keys?.length ?? 0), 0),
);
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Data"
      title="Schema"
      subtitle="Browse the Postgres database live — tables, columns, relations, and rows."
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loading" @click="load">
          <RefreshCw class="size-4" :class="loading ? 'animate-spin' : ''" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Stats strip -->
    <section class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <Card class="border-brand-cream-300 bg-brand-cream-50 p-4 flex items-center gap-3">
        <div class="flex size-10 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
          <Database class="size-5" stroke-width="1.75" />
        </div>
        <div class="flex flex-col">
          <span class="text-xs text-brand-dark-600">Tables</span>
          <span class="font-serif text-2xl leading-none">{{ userTableCount }}</span>
        </div>
      </Card>
      <Card class="border-brand-cream-300 bg-brand-cream-50 p-4 flex items-center gap-3">
        <div class="flex size-10 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
          <Key class="size-5" stroke-width="1.75" />
        </div>
        <div class="flex flex-col">
          <span class="text-xs text-brand-dark-600">Foreign keys</span>
          <span class="font-serif text-2xl leading-none">{{ totalFks }}</span>
        </div>
      </Card>
      <Card class="border-brand-cream-300 bg-brand-cream-50 p-4 flex items-center gap-3">
        <div class="flex size-10 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
          <Database class="size-5" stroke-width="1.75" />
        </div>
        <div class="flex flex-col">
          <span class="text-xs text-brand-dark-600">Total rows</span>
          <span class="font-serif text-2xl leading-none">{{ totalRows.toLocaleString() }}</span>
        </div>
      </Card>
      <Card class="border-brand-cream-300 bg-brand-cream-50 p-4 flex items-center gap-3">
        <div class="flex size-10 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
          <Database class="size-5" stroke-width="1.75" />
        </div>
        <div class="flex flex-col">
          <span class="text-xs text-brand-dark-600">Schema</span>
          <span class="font-serif text-2xl leading-none">public</span>
        </div>
      </Card>
    </section>

    <!-- Error -->
    <Card
      v-if="error"
      class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3"
    >
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex-1 min-w-0">
        <div class="text-sm font-semibold text-red-900">Couldn't load schema.</div>
        <div class="text-xs text-red-800 break-all">{{ error }}</div>
      </div>
      <Button variant="outline" size="sm" @click="load">Retry</Button>
    </Card>

    <!-- ER graph overview -->
    <ERGraph
      v-if="!loading && tables.length"
      :tables="tables"
      :selected="selected"
      @select="selectTable"
    />

    <!-- Main: sidebar + detail -->
    <div class="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-6">
      <!-- Sidebar -->
      <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden h-fit">
        <div class="p-3 border-b border-brand-cream-200">
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-brand-dark-500 pointer-events-none"
            />
            <Input v-model="search" placeholder="Search tables…" class="pl-9 bg-white h-9" />
          </div>
        </div>

        <div v-if="loading" class="p-4 flex flex-col gap-2">
          <Skeleton v-for="n in 4" :key="n" class="h-10 w-full" />
        </div>

        <ul v-else class="flex flex-col">
          <li v-for="t in filtered" :key="t.name">
            <button
              type="button"
              class="w-full text-left px-4 py-3 flex items-center gap-3 hover:bg-brand-cream-100 transition-colors border-l-2 border-transparent"
              :class="{
                'bg-brand-gold-100/60 !border-brand-gold-600': selected === t.name,
                'opacity-60': t.name.startsWith('goose_'),
              }"
              @click="selectTable(t.name)"
            >
              <Database class="size-4 shrink-0 text-brand-gold-700" />
              <div class="flex flex-col min-w-0 flex-1">
                <span class="font-mono text-sm text-brand-dark-950 truncate">{{ t.name }}</span>
                <span class="text-[10px] text-brand-dark-500">
                  {{ t.columns.length }} cols · {{ t.row_count }} rows
                </span>
              </div>
              <Badge
                v-if="(t.foreign_keys?.length ?? 0) > 0"
                variant="soft"
                class="text-[10px] px-1.5 py-0"
              >
                FK
              </Badge>
            </button>
          </li>
          <li v-if="!filtered.length" class="p-6 text-center text-sm text-brand-dark-500">
            No tables match.
          </li>
        </ul>
      </Card>

      <!-- Detail -->
      <div v-if="selectedTable" class="min-w-0">
        <TableDetail :table="selectedTable" :all-tables="tables" @select="selectTable" />
      </div>
      <Card v-else class="border-brand-cream-300 bg-brand-cream-50 p-12">
        <div class="text-center text-brand-dark-500">
          {{ loading ? 'Loading…' : 'Select a table on the left.' }}
        </div>
      </Card>
    </div>
  </div>
</template>
