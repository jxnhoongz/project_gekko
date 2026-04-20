<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';
const props = defineProps<{ class?: HTMLAttributes['class'] }>();
const classes = computed(() => cn('w-full caption-bottom text-sm', props.class));
</script>

<template>
  <div class="relative w-full overflow-x-auto">
    <table :class="classes"><slot /></table>
  </div>
</template>
