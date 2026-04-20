<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';
const props = defineProps<{ class?: HTMLAttributes['class'] }>();
const classes = computed(() => cn('[&_tr]:border-b', props.class));
</script>

<template>
  <thead :class="classes"><slot /></thead>
</template>
