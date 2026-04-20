<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Search, RefreshCw, AlertTriangle } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import EmptyState from '@/components/layout/EmptyState.vue';
import { api } from '@/lib/api';
import { formatDate, timeAgo } from '@/lib/format';

interface WaitlistEntry {
  id: number;
  email: string;
  telegram: string;
  phone: string;
  interested_in: string;
  source: string;
  notes: string;
  contacted_at: string | null;
  created_at: string;
}

const loading = ref(true);
const error = ref<string | null>(null);
const entries = ref<WaitlistEntry[]>([]);
const total = ref(0);
const search = ref('');

async function load() {
  loading.value = true;
  error.value = null;
  try {
    const res = await api.get<{ entries: WaitlistEntry[]; total: number }>('/api/waitlist');
    entries.value = res.data.entries;
    total.value = res.data.total;
  } catch (e: unknown) {
    error.value = (e as Error).message ?? 'Failed to load waitlist';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return entries.value;
  return entries.value.filter(
    (e) =>
      e.email.toLowerCase().includes(q) ||
      e.telegram.toLowerCase().includes(q) ||
      e.interested_in.toLowerCase().includes(q) ||
      e.notes.toLowerCase().includes(q),
  );
});

const sourceVariant = (s: string) => {
  if (s === 'website')  return 'soft';
  if (s === 'telegram') return 'accent';
  if (s === 'referral') return 'muted';
  return 'outline';
};
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Leads"
      title="Waitlist"
      :subtitle="`${total} interested ${total === 1 ? 'person' : 'people'} waiting for the drop.`"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="loading" @click="load">
          <RefreshCw class="size-4" :class="loading ? 'animate-spin' : ''" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div
      class="flex flex-col sm:flex-row gap-3 sm:items-center sm:justify-between rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-4"
    >
      <div class="relative flex-1 sm:max-w-sm">
        <Search
          class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-brand-dark-500 pointer-events-none"
        />
        <Input v-model="search" placeholder="Search email, interest, notes…" class="pl-9 bg-white" />
      </div>
      <div class="text-xs text-brand-dark-600">
        Showing {{ filtered.length }} of {{ entries.length }}
      </div>
    </div>

    <!-- Error banner -->
    <Card
      v-if="error"
      class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3"
    >
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex flex-col gap-1 flex-1 min-w-0">
        <span class="text-sm font-semibold text-red-900">Couldn't load waitlist.</span>
        <span class="text-xs text-red-800 break-all">{{ error }}</span>
      </div>
      <Button variant="outline" size="sm" @click="load">Retry</Button>
    </Card>

    <!-- Loading skeleton -->
    <Card v-else-if="loading" class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
      <div class="p-6 flex flex-col gap-3">
        <Skeleton v-for="n in 5" :key="n" class="h-12 w-full" />
      </div>
    </Card>

    <!-- Data table -->
    <Card
      v-else-if="filtered.length"
      class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead class="hidden md:table-cell">Telegram</TableHead>
            <TableHead>Interested in</TableHead>
            <TableHead class="hidden sm:table-cell">Source</TableHead>
            <TableHead class="text-right">Added</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="e in filtered" :key="e.id">
            <TableCell>
              <div class="flex flex-col">
                <span class="font-medium text-brand-dark-950">{{ e.email }}</span>
                <span v-if="e.notes" class="text-xs text-brand-dark-600 line-clamp-1">{{ e.notes }}</span>
              </div>
            </TableCell>
            <TableCell class="hidden md:table-cell text-sm text-brand-dark-700">
              {{ e.telegram || '—' }}
            </TableCell>
            <TableCell>
              <Badge variant="outline">{{ e.interested_in || 'Any' }}</Badge>
            </TableCell>
            <TableCell class="hidden sm:table-cell">
              <Badge :variant="sourceVariant(e.source) as any">{{ e.source || 'website' }}</Badge>
            </TableCell>
            <TableCell class="text-right">
              <div class="flex flex-col items-end">
                <span class="text-sm text-brand-dark-950">{{ formatDate(e.created_at) }}</span>
                <span class="text-xs text-brand-dark-500">{{ timeAgo(e.created_at) }}</span>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>

    <!-- Empty -->
    <EmptyState
      v-else
      title="Nothing on the waitlist yet."
      description="When someone signs up on the storefront, they'll show up here."
    />
  </div>
</template>
