<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import {
  Key,
  Link as LinkIcon,
  ArrowLeftRight,
  Columns as ColumnsIcon,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
} from 'lucide-vue-next';
import type { DbTable, RowsResponse } from '@/types/schema';
import { api } from '@/lib/api';

const props = defineProps<{
  table: DbTable;
  allTables: DbTable[];
}>();

const emit = defineEmits<{ (e: 'select', name: string): void }>();

const rowsData = ref<RowsResponse | null>(null);
const loadingRows = ref(false);
const rowError = ref<string | null>(null);
const offset = ref(0);
const limit = 25;

const inboundFks = computed(() => {
  const out: { fromTable: string; fromColumn: string; toColumn: string }[] = [];
  for (const t of props.allTables) {
    for (const fk of t.foreign_keys ?? []) {
      if (fk.ref_table === props.table.name) {
        out.push({ fromTable: t.name, fromColumn: fk.column, toColumn: fk.ref_column });
      }
    }
  }
  return out;
});

async function loadRows() {
  loadingRows.value = true;
  rowError.value = null;
  try {
    const res = await api.get<RowsResponse>(`/api/admin/table/${props.table.name}`, {
      params: { limit, offset: offset.value },
    });
    rowsData.value = res.data;
  } catch (e: unknown) {
    rowError.value = (e as Error).message ?? 'Failed to load rows';
  } finally {
    loadingRows.value = false;
  }
}

// Reload when selected table changes or offset changes
watch(
  () => props.table.name,
  () => {
    offset.value = 0;
    loadRows();
  },
);
watch(offset, loadRows);

onMounted(loadRows);

function prevPage() {
  offset.value = Math.max(0, offset.value - limit);
}
function nextPage() {
  if (rowsData.value && offset.value + limit < rowsData.value.total) {
    offset.value = offset.value + limit;
  }
}
</script>

<template>
  <div class="flex flex-col gap-5">
    <!-- Header -->
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div class="flex flex-col gap-1">
        <span class="text-xs uppercase tracking-[0.16em] text-brand-gold-700 font-semibold">
          Table
        </span>
        <h2 class="font-serif text-3xl text-brand-dark-950 leading-none">{{ table.name }}</h2>
        <div class="flex items-center gap-3 text-sm text-brand-dark-600 mt-1">
          <span>{{ table.row_count.toLocaleString() }} rows</span>
          <span class="size-1 rounded-full bg-brand-cream-400" />
          <span>{{ table.columns.length }} columns</span>
          <span
            v-if="(table.foreign_keys?.length ?? 0) > 0"
            class="flex items-center gap-1"
          >
            <span class="size-1 rounded-full bg-brand-cream-400" />
            {{ table.foreign_keys!.length }} FK
          </span>
        </div>
      </div>
    </div>

    <Tabs default-value="columns">
      <TabsList>
        <TabsTrigger value="columns"><ColumnsIcon class="size-3.5 mr-1" /> Columns</TabsTrigger>
        <TabsTrigger value="relations"><ArrowLeftRight class="size-3.5 mr-1" /> Relations</TabsTrigger>
        <TabsTrigger value="indexes"><Key class="size-3.5 mr-1" /> Indexes</TabsTrigger>
        <TabsTrigger value="rows"><LinkIcon class="size-3.5 mr-1" /> Rows</TabsTrigger>
      </TabsList>

      <!-- Columns -->
      <TabsContent value="columns">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Nullable</TableHead>
                <TableHead>Default</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="c in table.columns" :key="c.name">
                <TableCell>
                  <div class="flex items-center gap-2">
                    <span class="font-mono text-sm text-brand-dark-950">{{ c.name }}</span>
                    <Badge v-if="c.is_pk" variant="soft" class="gap-1">
                      <Key class="size-3" /> PK
                    </Badge>
                  </div>
                </TableCell>
                <TableCell class="font-mono text-xs text-brand-dark-700">{{ c.type }}</TableCell>
                <TableCell>
                  <Badge :variant="c.nullable ? 'muted' : 'outline'">
                    {{ c.nullable ? 'NULL' : 'NOT NULL' }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <span
                    v-if="c.default"
                    class="font-mono text-xs text-brand-dark-600 line-clamp-1"
                  >{{ c.default }}</span>
                  <span v-else class="text-xs text-brand-dark-400">—</span>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </Card>
      </TabsContent>

      <!-- Relations -->
      <TabsContent value="relations">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
            <div class="px-5 py-4 flex items-center gap-2">
              <ArrowLeftRight class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">References out</h3>
              <span class="ml-auto text-xs text-brand-dark-500"
                >{{ table.foreign_keys?.length ?? 0 }}</span
              >
            </div>
            <Separator />
            <ul v-if="table.foreign_keys?.length" class="divide-y divide-brand-cream-200">
              <li
                v-for="fk in table.foreign_keys"
                :key="fk.constraint"
                class="px-5 py-3 flex flex-col gap-1"
              >
                <div class="flex items-center gap-2 text-sm">
                  <span class="font-mono text-brand-dark-950">{{ fk.column }}</span>
                  <span class="text-brand-dark-500">→</span>
                  <button
                    type="button"
                    class="font-mono text-brand-gold-700 hover:underline"
                    @click="emit('select', fk.ref_table)"
                  >
                    {{ fk.ref_table }}.{{ fk.ref_column }}
                  </button>
                </div>
                <span class="text-xs text-brand-dark-500 font-mono">{{ fk.constraint }}</span>
              </li>
            </ul>
            <div v-else class="px-5 py-8 text-sm text-brand-dark-500 text-center">
              No outgoing foreign keys.
            </div>
          </Card>

          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
            <div class="px-5 py-4 flex items-center gap-2">
              <ArrowLeftRight class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">Referenced by</h3>
              <span class="ml-auto text-xs text-brand-dark-500">{{ inboundFks.length }}</span>
            </div>
            <Separator />
            <ul v-if="inboundFks.length" class="divide-y divide-brand-cream-200">
              <li
                v-for="(fk, i) in inboundFks"
                :key="i"
                class="px-5 py-3 flex items-center gap-2 text-sm"
              >
                <button
                  type="button"
                  class="font-mono text-brand-gold-700 hover:underline"
                  @click="emit('select', fk.fromTable)"
                >
                  {{ fk.fromTable }}.{{ fk.fromColumn }}
                </button>
                <span class="text-brand-dark-500">→</span>
                <span class="font-mono text-brand-dark-950">{{ fk.toColumn }}</span>
              </li>
            </ul>
            <div v-else class="px-5 py-8 text-sm text-brand-dark-500 text-center">
              Nothing references this table yet.
            </div>
          </Card>
        </div>
      </TabsContent>

      <!-- Indexes -->
      <TabsContent value="indexes">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Index</TableHead>
                <TableHead>Columns</TableHead>
                <TableHead>Type</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="idx in table.indexes ?? []" :key="idx.name">
                <TableCell>
                  <span class="font-mono text-sm text-brand-dark-950">{{ idx.name }}</span>
                </TableCell>
                <TableCell>
                  <span class="font-mono text-xs text-brand-dark-700">
                    {{ idx.columns.join(', ') }}
                  </span>
                </TableCell>
                <TableCell>
                  <div class="flex gap-1">
                    <Badge v-if="idx.primary" variant="soft">Primary</Badge>
                    <Badge v-if="idx.unique && !idx.primary" variant="accent">Unique</Badge>
                    <Badge v-if="!idx.unique && !idx.primary" variant="muted">Index</Badge>
                  </div>
                </TableCell>
              </TableRow>
              <TableRow v-if="!(table.indexes && table.indexes.length)">
                <TableCell colspan="3" class="text-center text-sm text-brand-dark-500 py-8">
                  No indexes defined.
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </Card>
      </TabsContent>

      <!-- Rows -->
      <TabsContent value="rows">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
          <div class="px-5 py-4 flex items-center justify-between gap-2">
            <h3 class="font-serif text-lg">Rows</h3>
            <div class="flex items-center gap-2">
              <Button variant="outline" size="sm" :disabled="loadingRows" @click="loadRows">
                <RefreshCw class="size-4" :class="loadingRows ? 'animate-spin' : ''" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                :disabled="offset === 0 || loadingRows"
                @click="prevPage"
              >
                <ChevronLeft class="size-4" />
              </Button>
              <span class="text-xs text-brand-dark-600">
                <template v-if="rowsData">
                  {{ offset + 1 }}–{{ Math.min(offset + limit, rowsData.total) }} of
                  {{ rowsData.total }}
                </template>
                <template v-else>…</template>
              </span>
              <Button
                variant="ghost"
                size="sm"
                :disabled="!rowsData || offset + limit >= rowsData.total || loadingRows"
                @click="nextPage"
              >
                <ChevronRight class="size-4" />
              </Button>
            </div>
          </div>
          <Separator />

          <Card
            v-if="rowError"
            class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3 m-4"
          >
            <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
            <div class="flex-1 min-w-0 text-sm text-red-800 break-all">{{ rowError }}</div>
          </Card>

          <div v-else-if="loadingRows && !rowsData" class="p-5 flex flex-col gap-3">
            <Skeleton v-for="n in 6" :key="n" class="h-8 w-full" />
          </div>

          <div v-else-if="rowsData && rowsData.rows.length">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead
                    v-for="c in rowsData.columns"
                    :key="c"
                    class="font-mono text-[10px]"
                  >
                    {{ c }}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="(row, i) in rowsData.rows" :key="i">
                  <TableCell
                    v-for="(val, j) in row"
                    :key="j"
                    class="font-mono text-xs align-top max-w-[280px]"
                  >
                    <span v-if="val === null" class="text-brand-dark-400 italic">NULL</span>
                    <span v-else class="break-all line-clamp-3">{{ val }}</span>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <div v-else class="px-5 py-10 text-center text-sm text-brand-dark-500">
            No rows yet.
          </div>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>
