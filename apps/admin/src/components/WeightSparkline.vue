<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ points: { at: string; grams: number }[]; width?: number; height?: number }>();

const W = computed(() => props.width ?? 360);
const H = computed(() => props.height ?? 120);

const prep = computed(() => {
  const pts = [...props.points].sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime());
  if (!pts.length) return { path: '', area: '', min: 0, max: 0, labels: [] as string[] };
  const xs = pts.map((p) => new Date(p.at).getTime());
  const ys = pts.map((p) => p.grams);
  const xMin = Math.min(...xs);
  const xMax = Math.max(...xs);
  const yMin = Math.min(...ys) - 2;
  const yMax = Math.max(...ys) + 2;
  const padX = 16, padY = 14;
  const toX = (v: number) =>
    padX + (xMax === xMin ? 0 : ((v - xMin) / (xMax - xMin)) * (W.value - padX * 2));
  const toY = (v: number) =>
    H.value - padY - (yMax === yMin ? 0 : ((v - yMin) / (yMax - yMin)) * (H.value - padY * 2));
  const path = pts
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${toX(new Date(p.at).getTime())} ${toY(p.grams)}`)
    .join(' ');
  const area =
    `M ${toX(xs[0])} ${H.value - padY} ` +
    pts.map((p) => `L ${toX(new Date(p.at).getTime())} ${toY(p.grams)}`).join(' ') +
    ` L ${toX(xs[xs.length - 1])} ${H.value - padY} Z`;
  return {
    path,
    area,
    min: Math.min(...ys),
    max: Math.max(...ys),
    labels: pts.map((p) => new Date(p.at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })),
  };
});
</script>

<template>
  <svg :width="W" :height="H" :viewBox="`0 0 ${W} ${H}`" class="w-full h-auto">
    <defs>
      <linearGradient id="spark-grad" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0%" stop-color="#efc262" stop-opacity="0.6" />
        <stop offset="100%" stop-color="#efc262" stop-opacity="0" />
      </linearGradient>
    </defs>
    <path :d="prep.area" fill="url(#spark-grad)" />
    <path :d="prep.path" fill="none" stroke="#b06c12" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
</template>
