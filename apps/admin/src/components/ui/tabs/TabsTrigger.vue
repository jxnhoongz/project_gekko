<script setup lang="ts">
import { TabsTrigger, type TabsTriggerProps } from 'reka-ui';
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<TabsTriggerProps & { class?: HTMLAttributes['class'] }>();
const delegated = computed(() => {
  const { class: _c, ...rest } = props;
  return rest;
});
</script>

<template>
  <TabsTrigger
    v-bind="delegated"
    :class="cn(
      'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-all',
      'hover:text-foreground',
      'data-[state=active]:bg-brand-gold-600 data-[state=active]:text-white data-[state=active]:shadow-sm',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
      'disabled:pointer-events-none disabled:opacity-50',
      props.class,
    )"
  >
    <slot />
  </TabsTrigger>
</template>
