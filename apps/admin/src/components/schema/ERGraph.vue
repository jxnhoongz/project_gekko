<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { Button } from '@/components/ui/button';
import {
  MousePointer2,
  Hand,
  RotateCcw,
  Maximize2,
} from 'lucide-vue-next';
import type { DbTable } from '@/types/schema';

type Positions = Record<string, { cx: number; cy: number }>;

const props = defineProps<{
  tables: DbTable[];
  selected?: string | null;
  /** Render at a larger default size (fullscreen modal). */
  fullscreen?: boolean;
  /** Hide the internal fullscreen button (e.g. inside a modal). */
  hideFullscreen?: boolean;
}>();

const emit = defineEmits<{
  (e: 'select', name: string): void;
  (e: 'fullscreen'): void;
}>();

/** Shared layout positions (v-model:positions). Keyed by table name. */
const positions = defineModel<Positions>('positions', { default: () => ({}) });

/** View dimensions — inline = 640x380, fullscreen = 1200x720. */
const W = computed(() => (props.fullscreen ? 1200 : 640));
const H = computed(() => (props.fullscreen ? 720 : 380));
const NODE_W = 160;
const NODE_H = 54;

type Tool = 'cursor' | 'hand';
const tool = ref<Tool>('cursor');

// Pan offset (in SVG-unit space)
const pan = ref({ x: 0, y: 0 });

// Drag state
const draggingNode = ref<string | null>(null);
let dragStartMouse = { x: 0, y: 0 };
let dragStartNode = { cx: 0, cy: 0 };

let panningStart: { x: number; y: number; panX: number; panY: number } | null = null;

const svgRef = ref<SVGSVGElement | null>(null);

// Filter goose internal tables so the graph stays clean
const displayTables = computed(() =>
  props.tables.filter((t) => !t.name.startsWith('goose_')),
);

// Compute default circular layout
function defaultLayout(): Positions {
  const list = displayTables.value;
  const n = list.length;
  const out: Positions = {};
  if (n === 0) return out;
  const cx = W.value / 2;
  const cy = H.value / 2;
  if (n === 1) {
    out[list[0].name] = { cx, cy };
    return out;
  }
  const r = Math.min(W.value, H.value) / 2 - 90;
  for (let i = 0; i < n; i++) {
    const angle = (i / n) * Math.PI * 2 - Math.PI / 2;
    out[list[i].name] = { cx: cx + Math.cos(angle) * r, cy: cy + Math.sin(angle) * r };
  }
  return out;
}

// Initialize / backfill positions when tables load (preserve user-dragged ones).
watch(
  displayTables,
  (tables) => {
    const current = positions.value ?? {};
    const next: Positions = { ...current };
    let changed = false;
    const defaults = defaultLayout();
    for (const t of tables) {
      if (!next[t.name]) {
        next[t.name] = defaults[t.name];
        changed = true;
      }
    }
    // Drop entries for tables that no longer exist
    for (const k of Object.keys(next)) {
      if (!tables.find((t) => t.name === k)) {
        delete next[k];
        changed = true;
      }
    }
    if (changed || Object.keys(current).length === 0) {
      positions.value = next;
    }
  },
  { immediate: true },
);

interface ResolvedNode {
  table: DbTable;
  cx: number;
  cy: number;
  x: number;
  y: number;
}

const nodes = computed<ResolvedNode[]>(() =>
  displayTables.value.map((t) => {
    const p = positions.value[t.name] ?? { cx: W.value / 2, cy: H.value / 2 };
    return {
      table: t,
      cx: p.cx,
      cy: p.cy,
      x: p.cx - NODE_W / 2,
      y: p.cy - NODE_H / 2,
    };
  }),
);

const nodeByName = computed(() => {
  const m = new Map<string, ResolvedNode>();
  for (const n of nodes.value) m.set(n.table.name, n);
  return m;
});

interface Edge {
  from: ResolvedNode;
  to: ResolvedNode;
}

const edges = computed<Edge[]>(() => {
  const out: Edge[] = [];
  for (const n of nodes.value) {
    for (const fk of n.table.foreign_keys ?? []) {
      const to = nodeByName.value.get(fk.ref_table);
      if (!to) continue;
      out.push({ from: n, to });
    }
  }
  return out;
});

// Edge endpoints clipped to node rectangle edges (looks better than center-to-center)
function clipToRect(fromCx: number, fromCy: number, toCx: number, toCy: number) {
  const dx = toCx - fromCx;
  const dy = toCy - fromCy;
  if (dx === 0 && dy === 0) return { x: fromCx, y: fromCy };
  const halfW = NODE_W / 2;
  const halfH = NODE_H / 2;
  const sx = dx >= 0 ? 1 : -1;
  const sy = dy >= 0 ? 1 : -1;
  const tX = Math.abs(dx) > 0 ? halfW / Math.abs(dx) : Infinity;
  const tY = Math.abs(dy) > 0 ? halfH / Math.abs(dy) : Infinity;
  const t = Math.min(tX, tY);
  return { x: fromCx + dx * t * sx / (sx || 1), y: fromCy + dy * t * sy / (sy || 1) };
}

function edgePath(e: Edge): { x1: number; y1: number; x2: number; y2: number } {
  const a = clipToRect(e.from.cx, e.from.cy, e.to.cx, e.to.cy);
  const b = clipToRect(e.to.cx, e.to.cy, e.from.cx, e.from.cy);
  return { x1: a.x, y1: a.y, x2: b.x, y2: b.y };
}

// Coord transform: client pixels -> SVG viewBox units
function toSvg(e: PointerEvent): { x: number; y: number } {
  const svg = svgRef.value;
  if (!svg) return { x: 0, y: 0 };
  const pt = svg.createSVGPoint();
  pt.x = e.clientX;
  pt.y = e.clientY;
  const ctm = svg.getScreenCTM();
  if (!ctm) return { x: 0, y: 0 };
  const p = pt.matrixTransform(ctm.inverse());
  return { x: p.x, y: p.y };
}

function onNodePointerDown(e: PointerEvent, name: string) {
  if (tool.value !== 'cursor') return;
  e.stopPropagation();
  const p = toSvg(e);
  draggingNode.value = name;
  dragStartMouse = p;
  const cur = positions.value[name];
  dragStartNode = { cx: cur?.cx ?? 0, cy: cur?.cy ?? 0 };
  (e.target as Element).setPointerCapture?.(e.pointerId);
}

function onSvgPointerDown(e: PointerEvent) {
  if (tool.value === 'hand') {
    // Pan — only when clicking background (not a node).
    const p = toSvg(e);
    panningStart = { x: p.x, y: p.y, panX: pan.value.x, panY: pan.value.y };
    (e.currentTarget as Element).setPointerCapture?.(e.pointerId);
  }
}

function onPointerMove(e: PointerEvent) {
  if (draggingNode.value) {
    const p = toSvg(e);
    const dx = p.x - dragStartMouse.x;
    const dy = p.y - dragStartMouse.y;
    const next = { ...positions.value };
    next[draggingNode.value] = {
      cx: dragStartNode.cx + dx,
      cy: dragStartNode.cy + dy,
    };
    positions.value = next;
    return;
  }
  if (panningStart) {
    const p = toSvg(e);
    pan.value = {
      x: panningStart.panX - (p.x - panningStart.x),
      y: panningStart.panY - (p.y - panningStart.y),
    };
  }
}

function onPointerUp(_e: PointerEvent) {
  draggingNode.value = null;
  panningStart = null;
}

function onNodeClick(_e: MouseEvent, name: string) {
  // Click is only considered a selection if we didn't drag.
  if (draggingNode.value) return;
  emit('select', name);
}

function resetLayout() {
  positions.value = defaultLayout();
  pan.value = { x: 0, y: 0 };
}

async function openFullscreen() {
  emit('fullscreen');
  await nextTick();
}

const viewBox = computed(() => `${pan.value.x} ${pan.value.y} ${W.value} ${H.value}`);
const cursorClass = computed(() => {
  if (draggingNode.value) return 'cursor-grabbing';
  if (panningStart) return 'cursor-grabbing';
  return tool.value === 'hand' ? 'cursor-grab' : 'cursor-default';
});
</script>

<template>
  <div class="relative w-full rounded-xl border border-brand-cream-300 bg-brand-cream-50 overflow-hidden">
    <!-- Header / toolbar -->
    <div class="flex items-center justify-between px-4 py-3 border-b border-brand-cream-200 gap-2">
      <div class="text-xs uppercase tracking-wider text-brand-dark-600 font-semibold">
        Entity relationships
      </div>

      <div class="flex items-center gap-2">
        <div class="hidden sm:flex items-center text-xs text-brand-dark-500 mr-2">
          {{ displayTables.length }} tables · {{ edges.length }} FKs
        </div>

        <!-- Tool toggle -->
        <div
          class="inline-flex items-center rounded-md border border-brand-cream-300 bg-white p-0.5"
          role="group"
          aria-label="Canvas tool"
        >
          <button
            type="button"
            aria-label="Cursor (move nodes)"
            title="Cursor — drag nodes to rearrange"
            class="inline-flex items-center justify-center size-7 rounded transition-colors"
            :class="
              tool === 'cursor'
                ? 'bg-brand-gold-100 text-brand-gold-800'
                : 'text-brand-dark-600 hover:bg-brand-cream-100'
            "
            @click="tool = 'cursor'"
          >
            <MousePointer2 class="size-4" />
          </button>
          <button
            type="button"
            aria-label="Hand (pan canvas)"
            title="Hand — drag the canvas"
            class="inline-flex items-center justify-center size-7 rounded transition-colors"
            :class="
              tool === 'hand'
                ? 'bg-brand-gold-100 text-brand-gold-800'
                : 'text-brand-dark-600 hover:bg-brand-cream-100'
            "
            @click="tool = 'hand'"
          >
            <Hand class="size-4" />
          </button>
        </div>

        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="Reset layout"
          title="Reset layout"
          @click="resetLayout"
        >
          <RotateCcw class="size-4" />
        </Button>

        <Button
          v-if="!hideFullscreen"
          variant="ghost"
          size="icon-sm"
          aria-label="Open in full screen"
          title="Full screen"
          @click="openFullscreen"
        >
          <Maximize2 class="size-4" />
        </Button>
      </div>
    </div>

    <svg
      ref="svgRef"
      :viewBox="viewBox"
      :class="[
        'w-full block select-none',
        fullscreen ? 'h-[80vh]' : 'h-[320px] sm:h-[380px]',
        cursorClass,
      ]"
      xmlns="http://www.w3.org/2000/svg"
      @pointerdown="onSvgPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
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
        <pattern id="er-dots" x="0" y="0" width="16" height="16" patternUnits="userSpaceOnUse">
          <circle cx="1" cy="1" r="0.8" fill="#d9cdac" />
        </pattern>
      </defs>

      <!-- Background grid (helps hand-tool pan feel anchored) -->
      <rect
        :x="pan.x - W"
        :y="pan.y - H"
        :width="W * 3"
        :height="H * 3"
        fill="url(#er-dots)"
      />

      <!-- Edges -->
      <g>
        <line
          v-for="(e, i) in edges"
          :key="`edge-${i}-${e.from.table.name}-${e.to.table.name}`"
          :x1="edgePath(e).x1"
          :y1="edgePath(e).y1"
          :x2="edgePath(e).x2"
          :y2="edgePath(e).y2"
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
          :transform="`translate(${n.x} ${n.y})`"
          :class="[
            'select-none',
            tool === 'cursor' ? 'cursor-grab' : 'cursor-default',
            draggingNode === n.table.name ? 'cursor-grabbing' : '',
          ]"
          @pointerdown="onNodePointerDown($event, n.table.name)"
          @click="onNodeClick($event, n.table.name)"
        >
          <rect
            :width="NODE_W"
            :height="NODE_H"
            rx="10"
            :fill="selected === n.table.name ? '#fbefd0' : '#faf8f3'"
            :stroke="selected === n.table.name ? '#b06c12' : '#d9cdac'"
            stroke-width="1.5"
          />
          <text
            :x="NODE_W / 2"
            y="22"
            text-anchor="middle"
            font-family="DM Serif Display, serif"
            font-size="15"
            fill="#110e0b"
            pointer-events="none"
          >
            {{ n.table.name }}
          </text>
          <text
            :x="NODE_W / 2"
            y="40"
            text-anchor="middle"
            font-family="Inter, sans-serif"
            font-size="10"
            fill="#4e463f"
            pointer-events="none"
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
