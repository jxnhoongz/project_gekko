<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cva, type VariantProps } from 'class-variance-authority';
import {
  DialogContent,
  DialogOverlay,
  DialogPortal,
  type DialogContentProps,
  type DialogContentEmits,
  useForwardPropsEmits,
} from 'reka-ui';
import { X } from 'lucide-vue-next';
import { cn } from '@/lib/utils';

const sheetVariants = cva(
  'fixed z-50 gap-4 bg-brand-cream-50 p-6 shadow-lg transition ease-in-out data-[state=closed]:duration-300 data-[state=open]:duration-500 data-[state=open]:animate-in data-[state=closed]:animate-out',
  {
    variants: {
      side: {
        top:    'inset-x-0 top-0 border-b border-brand-cream-300 data-[state=closed]:slide-out-to-top data-[state=open]:slide-in-from-top',
        bottom: 'inset-x-0 bottom-0 border-t border-brand-cream-300 data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom',
        left:   'inset-y-0 left-0 h-full w-72 border-r border-brand-cream-300 data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left',
        right:  'inset-y-0 right-0 h-full w-72 border-l border-brand-cream-300 data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right',
      },
    },
    defaultVariants: { side: 'right' },
  },
);

type SheetVariants = VariantProps<typeof sheetVariants>;

interface Props extends DialogContentProps {
  side?: SheetVariants['side'];
  class?: HTMLAttributes['class'];
  hideClose?: boolean;
}
const props = withDefaults(defineProps<Props>(), { side: 'right' });
const emits = defineEmits<DialogContentEmits>();

const delegated = computed(() => {
  const { class: _c, side: _s, hideClose: _h, ...rest } = props;
  return rest;
});
const forwarded = useForwardPropsEmits(delegated, emits);
</script>

<template>
  <DialogPortal>
    <DialogOverlay
      class="fixed inset-0 z-50 bg-brand-dark-950/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
    />
    <DialogContent v-bind="forwarded" :class="cn(sheetVariants({ side }), props.class)">
      <slot />
      <DialogClose
        v-if="!hideClose"
        class="absolute right-4 top-4 rounded-md p-1 text-brand-dark-600 hover:bg-brand-cream-200 hover:text-brand-dark-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <X class="size-4" />
        <span class="sr-only">Close</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
