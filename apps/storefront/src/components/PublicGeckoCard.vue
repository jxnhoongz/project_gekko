<script setup lang="ts">
import { Card } from '@/components/ui/card';
import { Mars, Venus, HelpCircle } from 'lucide-vue-next';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import type { PublicGecko } from '@/types/gecko';
import { ageFromBirth } from '@/lib/format';
import { computed } from 'vue';

const props = defineProps<{ gecko: PublicGecko }>();

const sexIcon = computed(() => {
  const m = { M: Mars, F: Venus, U: HelpCircle } as const;
  return m[props.gecko.sex];
});

const sexColor = computed(() => {
  const m = {
    M: 'text-sky-700 bg-sky-100',
    F: 'text-rose-700 bg-rose-100',
    U: 'text-brand-dark-600 bg-brand-cream-200',
  } as const;
  return m[props.gecko.sex];
});
</script>

<template>
  <RouterLink
    :to="{ name: 'gecko-detail', params: { code: gecko.code } }"
    class="block"
  >
    <Card
      class="group overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 cursor-pointer transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
    >
      <div
        class="relative h-48 bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center overflow-hidden"
      >
        <img
          v-if="gecko.cover_photo_url"
          :src="gecko.cover_photo_url"
          :alt="gecko.name"
          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
        />
        <LowPolyGecko v-else :size="160" />
        <div
          class="absolute top-3 right-3 flex size-7 items-center justify-center rounded-full shadow-sm"
          :class="sexColor"
        >
          <component :is="sexIcon" class="size-4" stroke-width="2" />
        </div>
      </div>
      <div class="p-5 flex flex-col gap-2">
        <div class="flex items-start justify-between gap-3">
          <div class="flex flex-col">
            <span class="text-[11px] uppercase tracking-wider text-brand-dark-600 font-semibold">
              {{ gecko.code }}
            </span>
            <h3 class="font-serif text-xl text-brand-dark-950 leading-tight">
              {{ gecko.name || 'Unnamed' }}
            </h3>
          </div>
          <div v-if="gecko.list_price_usd" class="text-right shrink-0">
            <div class="font-semibold text-brand-gold-700">${{ gecko.list_price_usd }}</div>
          </div>
        </div>
        <div class="text-xs text-brand-dark-600">
          <span>{{ gecko.species_name }}</span>
          <span class="mx-2 size-1 inline-block rounded-full bg-brand-cream-400 align-middle" />
          <span class="text-brand-dark-700 font-medium">{{ gecko.morph }}</span>
        </div>
        <div v-if="gecko.hatch_date" class="text-xs text-brand-dark-600">
          {{ ageFromBirth(gecko.hatch_date) }} old
        </div>
      </div>
    </Card>
  </RouterLink>
</template>
