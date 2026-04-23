<script setup lang="ts">
import { computed } from 'vue';
import { Card } from '@/components/ui/card';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Package, Boxes } from 'lucide-vue-next';
import GeckoIcon from '@/components/icons/GeckoIcon.vue';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import type { Listing, ListingStatus, ListingType } from '@/types/listing';
import { LISTING_STATUS_LABEL, LISTING_TYPE_LABEL } from '@/types/listing';

const props = defineProps<{ listing: Listing }>();
const emit = defineEmits<{ (e: 'edit', l: Listing): void }>();

const typeIcon = computed(() => {
  const map = { GECKO: GeckoIcon, PACKAGE: Package, SUPPLY: Boxes } as const;
  return map[props.listing.type];
});

const typeBadge: Record<ListingType, BadgeVariants['variant']> = {
  GECKO: 'soft',
  PACKAGE: 'accent',
  SUPPLY: 'muted',
};

const statusBadge: Record<ListingStatus, BadgeVariants['variant']> = {
  DRAFT: 'muted',
  LISTED: 'success',
  RESERVED: 'warn',
  SOLD: 'outline',
  ARCHIVED: 'outline',
};

const secondaryLine = computed(() => {
  if (props.listing.type === 'GECKO') return `${props.listing.gecko_count} gecko${props.listing.gecko_count === 1 ? '' : 's'}`;
  if (props.listing.type === 'PACKAGE') return `${props.listing.component_count} item${props.listing.component_count === 1 ? '' : 's'}`;
  return props.listing.sku || 'No SKU';
});
</script>

<template>
  <Card
    class="group overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 cursor-pointer transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
    @click="emit('edit', listing)"
  >
    <div class="relative h-40 bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center">
      <img
        v-if="listing.cover_photo_url"
        :src="listing.cover_photo_url"
        :alt="listing.title"
        class="w-full h-full object-cover"
      />
      <LowPolyGecko v-else :size="130" />
      <div class="absolute top-3 left-3">
        <Badge :variant="typeBadge[listing.type]" class="flex items-center gap-1">
          <component :is="typeIcon" class="size-3" />
          {{ LISTING_TYPE_LABEL[listing.type] }}
        </Badge>
      </div>
      <div class="absolute top-3 right-3">
        <Badge :variant="statusBadge[listing.status]">
          {{ LISTING_STATUS_LABEL[listing.status] }}
        </Badge>
      </div>
    </div>
    <div class="p-5 flex flex-col gap-2">
      <h3 class="font-serif text-xl text-brand-dark-950 leading-tight line-clamp-2">
        {{ listing.title }}
      </h3>
      <div class="text-xs text-brand-dark-600">{{ secondaryLine }}</div>
      <div class="flex items-baseline justify-between pt-2 border-t border-brand-cream-200">
        <div class="font-semibold text-brand-gold-700 text-lg">${{ listing.price_usd }}</div>
        <div v-if="listing.deposit_usd" class="text-[11px] text-brand-dark-500">
          deposit ${{ listing.deposit_usd }}
        </div>
      </div>
    </div>
  </Card>
</template>
