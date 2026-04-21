<script setup lang="ts">
import { computed, nextTick, onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
} from 'reka-ui';
import {
  LayoutDashboard,
  Turtle,
  ClipboardList,
  DollarSign,
  Image,
  Database,
  Settings,
  Search,
} from 'lucide-vue-next';
import { useGeckos } from '@/composables/useGeckos';
import { api } from '@/lib/api';
import type { SchemaResponse, DbTable } from '@/types/schema';

interface Item {
  key: string;
  label: string;
  hint: string;
  group: string;
  icon: typeof LayoutDashboard;
  go: () => void;
}

const router = useRouter();
const open = ref(false);
const query = ref('');
const highlighted = ref(0);
const inputRef = ref<HTMLInputElement | null>(null);
const tables = ref<DbTable[]>([]);
let loadedTables = false;

const { data: geckosData } = useGeckos();
const geckos = computed(() => geckosData.value?.geckos ?? []);

async function loadTables() {
  if (loadedTables) return;
  try {
    const res = await api.get<SchemaResponse>('/api/admin/schema');
    tables.value = res.data.tables;
  } catch {
    /* silent */
  } finally {
    loadedTables = true;
  }
}

function iconFor(name: string) {
  const map: Record<string, typeof LayoutDashboard> = {
    dashboard: LayoutDashboard,
    geckos: Turtle,
    waitlist: ClipboardList,
    sales: DollarSign,
    photos: Image,
    schema: Database,
    settings: Settings,
  };
  return map[name] ?? Database;
}

const baseItems = computed<Item[]>(() => {
  const out: Item[] = [];

  const navs = [
    { name: 'dashboard', label: 'Dashboard', hint: 'Go to dashboard' },
    { name: 'geckos',    label: 'Geckos',    hint: 'Browse the collection' },
    { name: 'waitlist',  label: 'Waitlist',  hint: 'Open waitlist' },
    { name: 'sales',     label: 'Sales',     hint: 'Open sales' },
    { name: 'photos',    label: 'Photos',    hint: 'Open photos' },
    { name: 'schema',    label: 'Schema',    hint: 'Open database viewer' },
    { name: 'settings',  label: 'Settings',  hint: 'Open settings' },
  ];
  for (const n of navs) {
    out.push({
      key: `nav:${n.name}`,
      label: n.label,
      hint: n.hint,
      group: 'Navigate',
      icon: iconFor(n.name),
      go: () => router.push({ name: n.name }),
    });
  }

  for (const g of geckos.value) {
    const morph = g.traits.map((t) => t.trait_name).join(' ') || 'Normal';
    out.push({
      key: `gecko:${g.id}`,
      label: g.name || g.code,
      hint: `${g.code} · ${g.species_name} · ${morph}`,
      group: 'Geckos',
      icon: Turtle,
      go: () => router.push({ name: 'gecko-detail', params: { id: g.id } }),
    });
  }

  for (const t of tables.value) {
    if (t.name.startsWith('goose_')) continue;
    out.push({
      key: `table:${t.name}`,
      label: t.name,
      hint: `${t.columns.length} cols · ${t.row_count} rows`,
      group: 'Tables',
      icon: Database,
      go: () => router.push({ name: 'schema', query: { t: t.name } }),
    });
  }

  return out;
});

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return baseItems.value;
  return baseItems.value.filter(
    (it) =>
      it.label.toLowerCase().includes(q) ||
      it.hint.toLowerCase().includes(q) ||
      it.group.toLowerCase().includes(q),
  );
});

const groups = computed(() => {
  const g = new Map<string, Item[]>();
  for (const it of filtered.value) {
    if (!g.has(it.group)) g.set(it.group, []);
    g.get(it.group)!.push(it);
  }
  return Array.from(g.entries()).map(([name, items]) => ({ name, items }));
});

// Flat index for keyboard navigation
const flat = computed(() => filtered.value);

watch(filtered, () => {
  highlighted.value = 0;
});

function onKeydown(ev: KeyboardEvent) {
  const mod = ev.metaKey || ev.ctrlKey;
  if (mod && ev.key.toLowerCase() === 'k') {
    ev.preventDefault();
    toggle();
    return;
  }
  if (!open.value) return;
  if (ev.key === 'ArrowDown') {
    ev.preventDefault();
    highlighted.value = Math.min(flat.value.length - 1, highlighted.value + 1);
  } else if (ev.key === 'ArrowUp') {
    ev.preventDefault();
    highlighted.value = Math.max(0, highlighted.value - 1);
  } else if (ev.key === 'Enter') {
    const it = flat.value[highlighted.value];
    if (it) pick(it);
  } else if (ev.key === 'Escape') {
    open.value = false;
  }
}

function toggle() {
  open.value = !open.value;
}

function pick(it: Item) {
  it.go();
  open.value = false;
  query.value = '';
}

watch(open, (v) => {
  if (v) {
    loadTables();
    nextTick(() => inputRef.value?.focus());
  }
});

onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));

defineExpose({ open: () => (open.value = true) });
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-brand-dark-950/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
      />
      <DialogContent
        class="fixed left-1/2 top-[18%] z-50 w-[min(640px,92vw)] -translate-x-1/2 rounded-xl border border-brand-cream-300 bg-brand-cream-50 shadow-2xl overflow-hidden data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"
        aria-describedby=""
      >
        <div class="flex items-center gap-2 px-4 py-3 border-b border-brand-cream-200">
          <Search class="size-4 text-brand-dark-500" />
          <input
            ref="inputRef"
            v-model="query"
            type="text"
            placeholder="Search geckos, tables, pages…"
            class="flex-1 bg-transparent outline-none text-brand-dark-950 placeholder:text-brand-dark-500"
          />
          <kbd
            class="hidden sm:inline-flex items-center gap-1 rounded-md border border-brand-cream-300 bg-white px-1.5 py-0.5 text-[10px] text-brand-dark-600"
          >
            ESC
          </kbd>
        </div>

        <div class="max-h-[60vh] overflow-y-auto">
          <div v-if="!flat.length" class="px-6 py-10 text-center text-sm text-brand-dark-500">
            No matches.
          </div>

          <div v-for="g in groups" :key="g.name" class="py-2">
            <div
              class="px-4 py-1 text-[10px] uppercase tracking-wider text-brand-dark-500 font-semibold"
            >
              {{ g.name }}
            </div>
            <ul>
              <li v-for="it in g.items" :key="it.key">
                <button
                  type="button"
                  class="w-full flex items-center gap-3 px-4 py-2 text-left text-sm hover:bg-brand-cream-100 transition-colors"
                  :class="
                    flat[highlighted]?.key === it.key
                      ? 'bg-brand-gold-100/80 text-brand-dark-950'
                      : 'text-brand-dark-800'
                  "
                  @click="pick(it)"
                  @mouseenter="highlighted = flat.findIndex((x) => x.key === it.key)"
                >
                  <component :is="it.icon" class="size-4 text-brand-gold-700 shrink-0" />
                  <span class="font-medium">{{ it.label }}</span>
                  <span class="text-xs text-brand-dark-500 truncate ml-auto">{{ it.hint }}</span>
                </button>
              </li>
            </ul>
          </div>
        </div>

        <div class="flex items-center justify-between px-4 py-2 border-t border-brand-cream-200 text-[11px] text-brand-dark-500">
          <div class="flex items-center gap-2">
            <kbd class="rounded border border-brand-cream-300 bg-white px-1.5 py-0.5">↑</kbd>
            <kbd class="rounded border border-brand-cream-300 bg-white px-1.5 py-0.5">↓</kbd>
            navigate
            <kbd class="rounded border border-brand-cream-300 bg-white px-1.5 py-0.5 ml-2">↵</kbd>
            open
          </div>
          <div class="flex items-center gap-1">
            <kbd class="rounded border border-brand-cream-300 bg-white px-1.5 py-0.5">⌘</kbd>
            <kbd class="rounded border border-brand-cream-300 bg-white px-1.5 py-0.5">K</kbd>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
