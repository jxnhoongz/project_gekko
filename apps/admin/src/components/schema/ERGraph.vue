<script setup lang="ts">
import { computed } from 'vue';
import type { DbTable } from '@/types/schema';

const props = defineProps<{
  tables: DbTable[];
  selected?: string | null;
}>();
const emit = defineEmits<{ (e: 'select', name: string): void }>();

const W = 640;
const H = 380;
const NODE_W = 140;
const NODE_H = 48;

// Filter goose internal tables so the graph stays clean
const displayTables = computed(() =>
  props.tables.filter((t) => !t.name.startsWith('goose_')),
);

interface NodePos {
  table: DbTable;
  x: number;
  y: number;
  cx: number;
  cy: number;
}

const nodes = computed<NodePos[]>(() => {
  const list = displayTables.value;
  const n = list.length;
  if (n === 0) return [];
  const cx = W / 2;
  const cy = H / 2;
  // For small n keep them roughly on a circle. For 1, center it.
  if (n === 1) {
    return [{ table: list[0], x: cx - NODE_W / 2, y: cy - NODE_H / 2, cx, cy }];
  }
  const r = Math.min(W, H) / 2 - 80;
  return list.map((t, i) => {
    const angle = (i / n) * Math.PI * 2 - Math.PI / 2;
    const nx = cx + Math.cos(angle) * r;
    const ny = cy + Math.sin(angle) * r;
    return { table: t, x: nx - NODE_W / 2, y: ny - NODE_H / 2, cx: nx, cy: ny };
  });
});

const nodeByName = computed(() => {
  const m = new Map<string, NodePos>();
  for (const n of nodes.value) m.set(n.table.name, n);
  return m;
});

interface Edge {
  from: NodePos;
  to: NodePos;
  fkLabel: string;
}

const edges = computed<Edge[]>(() => {
  const out: Edge[] = [];
  for (const n of nodes.value) {
    for (const fk of n.table.foreign_keys ?? []) {
      const to = nodeByName.value.get(fk.ref_table);
      if (!to) continue;
      out.push({ from: n, to, fkLabel: `${fk.column} → ${fk.ref_column}` });
    }
  }
  return out;
});
</script>

<template>
  <div class="relative w-full rounded-xl border border-brand-cream-300 bg-brand-cream-50 overflow-hidden">
    <div class="flex items-center justify-between px-4 py-3 border-b border-brand-cream-200">
      <div class="text-xs uppercase tracking-wider text-brand-dark-600 font-semibold">
        Entity relationships
      </div>
      <div class="text-xs text-brand-dark-500">
        {{ displayTables.length }} tables · {{ edges.length }} FKs
      </div>
    </div>
    <svg
      :viewBox="`0 0 ${W} ${H}`"
      class="w-full h-[320px] sm:h-[380px]"
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <marker
          id="arrow"
          viewBox="0 0 10 10"
          refX="10"
          refY="5"
          markerWidth="7"
          markerHeight="7"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#b06c12" />
        </marker>
      </defs>

      <!-- Edges -->
      <g>
        <line
          v-for="(e, i) in edges"
          :key="`edge-${i}`"
          :x1="e.from.cx"
          :y1="e.from.cy"
          :x2="e.to.cx"
          :y2="e.to.cy"
          stroke="#b06c12"
          stroke-width="1.5"
          stroke-opacity="0.55"
          stroke-dasharray="4 3"
          marker-end="url(#arrow)"
        />
      </g>

      <!-- Nodes -->
      <g>
        <g
          v-for="n in nodes"
          :key="n.table.name"
          class="cursor-pointer"
          @click="emit('select', n.table.name)"
        >
          <rect
            :x="n.x"
            :y="n.y"
            :width="NODE_W"
            :height="NODE_H"
            rx="8"
            :fill="selected === n.table.name ? '#fbefd0' : '#faf8f3'"
            :stroke="selected === n.table.name ? '#b06c12' : '#d9cdac'"
            stroke-width="1.5"
          />
          <text
            :x="n.cx"
            :y="n.cy - 3"
            text-anchor="middle"
            font-family="DM Serif Display, serif"
            font-size="15"
            fill="#110e0b"
          >
            {{ n.table.name }}
          </text>
          <text
            :x="n.cx"
            :y="n.cy + 14"
            text-anchor="middle"
            font-family="Inter, sans-serif"
            font-size="10"
            fill="#4e463f"
          >
            {{ n.table.columns.length }} cols · {{ n.table.row_count }} rows
          </text>
        </g>
      </g>

      <text
        v-if="nodes.length === 0"
        :x="W / 2"
        :y="H / 2"
        text-anchor="middle"
        font-family="Inter, sans-serif"
        fill="#4e463f"
      >
        No tables.
      </text>
    </svg>
  </div>
</template>
