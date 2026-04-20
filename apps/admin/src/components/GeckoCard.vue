<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { Card } from '@/components/ui/card';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Mars, Venus, HelpCircle } from 'lucide-vue-next';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import type { Gecko, GeckoStatus, Sex } from '@/types';
import { ageFromBirth } from '@/lib/format';

const props = defineProps<{ gecko: Gecko }>();
const router = useRouter();

const statusVariant: Record<GeckoStatus, BadgeVariants['variant']> = {
  Breeding:  'soft',
  Available: 'success',
  Hold:      'warn',
  Personal:  'muted',
  Sold:      'outline',
};

const sexIcon = computed(() => {
  const m: Record<Sex, typeof Mars> = { Male: Mars, Female: Venus, Unsexed: HelpCircle };
  return m[props.gecko.sex];
});

const sexColor = computed(() => {
  const m: Record<Sex, string> = {
    Male:    'text-sky-700 bg-sky-100',
    Female:  'text-rose-700 bg-rose-100',
    Unsexed: 'text-brand-dark-600 bg-brand-cream-200',
  };
  return m[props.gecko.sex];
});

function open() {
  router.push({ name: 'gecko-detail', params: { id: props.gecko.id } });
}
</script>

<template>
  <Card
    class="group relative overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 cursor-pointer transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
    @click="open"
  >
    <div
      class="relative h-40 bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center"
    >
      <LowPolyGecko class="opacity-95 group-hover:scale-105 transition-transform duration-300" :size="150" />
      <div class="absolute top-3 left-3">
        <Badge :variant="statusVariant[gecko.status]">{{ gecko.status }}</Badge>
      </div>
      <div
        class="absolute top-3 right-3 flex size-7 items-center justify-center rounded-full shadow-sm"
        :class="sexColor"
        :title="gecko.sex"
      >
        <component :is="sexIcon" class="size-4" stroke-width="2" />
      </div>
    </div>
    <div class="p-5 flex flex-col gap-3">
      <div class="flex items-start justify-between gap-3">
        <div class="flex flex-col">
          <span class="text-[11px] uppercase tracking-wider text-brand-dark-600 font-semibold">
            {{ gecko.code }}
          </span>
          <h3 class="font-serif text-xl text-brand-dark-950 leading-tight">{{ gecko.name }}</h3>
        </div>
        <div v-if="gecko.priceUsd" class="text-right shrink-0">
          <div class="font-semibold text-brand-gold-700">${{ gecko.priceUsd }}</div>
          <div class="text-[10px] uppercase tracking-wide text-brand-dark-600">USD</div>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-brand-dark-600">
        <span>{{ gecko.species }}</span>
        <span class="size-1 rounded-full bg-brand-cream-400" />
        <span class="text-brand-dark-700 font-medium">{{ gecko.morph }}</span>
      </div>
      <div class="flex items-center justify-between pt-3 border-t border-brand-cream-200 text-xs">
        <div class="flex items-center gap-1.5">
          <span class="text-brand-dark-600">Age</span>
          <span class="font-medium text-brand-dark-950">{{ ageFromBirth(gecko.bornAt) }}</span>
        </div>
        <div class="flex items-center gap-1.5">
          <span class="text-brand-dark-600">Weight</span>
          <span class="font-medium text-brand-dark-950">{{ gecko.weightG }} g</span>
        </div>
      </div>
    </div>
  </Card>
</template>
