<script setup lang="ts">
import { TabsList, type TabsListProps } from 'reka-ui';
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>();
const delegated = computed(() => {
  const { class: _c, ...rest } = props;
  return rest;
});
</script>

<template>
  <TabsList
    v-bind="delegated"
    :class="cn(
      'inline-flex h-10 items-center justify-start rounded-lg bg-brand-cream-100 p-1 text-brand-dark-600 gap-1 border border-brand-cream-300 overflow-x-auto',
      props.class,
    )"
  >
    <slot />
  </TabsList>
</template>
