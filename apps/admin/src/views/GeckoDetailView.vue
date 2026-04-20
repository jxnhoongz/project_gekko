<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { Card } from '@/components/ui/card';
import { Badge, type BadgeVariants } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import {
  ArrowLeft,
  Mars,
  Venus,
  HelpCircle,
  Drumstick,
  Scale,
  Sparkles,
  HeartPulse,
  Image as ImageIcon,
  Plus,
} from 'lucide-vue-next';
import EmptyState from '@/components/layout/EmptyState.vue';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import WeightSparkline from '@/components/WeightSparkline.vue';
import { findGecko, feedings, weights, sheds, healthLogs } from '@/mock';
import type { GeckoStatus, Sex } from '@/types';
import { formatDate, ageFromBirth, timeAgo } from '@/lib/format';

const props = defineProps<{ id: string }>();
const router = useRouter();

const gecko = computed(() => findGecko(props.id));

const geckoFeedings = computed(() =>
  gecko.value
    ? feedings
        .filter((f) => f.geckoId === gecko.value!.id)
        .sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
    : [],
);

const geckoWeights = computed(() =>
  gecko.value
    ? weights
        .filter((w) => w.geckoId === gecko.value!.id)
        .sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())
    : [],
);

const geckoSheds = computed(() =>
  gecko.value
    ? sheds
        .filter((s) => s.geckoId === gecko.value!.id)
        .sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
    : [],
);

const geckoHealth = computed(() =>
  gecko.value
    ? healthLogs
        .filter((h) => h.geckoId === gecko.value!.id)
        .sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
    : [],
);

const weightDelta = computed(() => {
  const ws = geckoWeights.value;
  if (ws.length < 2) return null;
  return ws[ws.length - 1].grams - ws[0].grams;
});

const statusVariant: Record<GeckoStatus, BadgeVariants['variant']> = {
  Breeding:  'soft',
  Available: 'success',
  Hold:      'warn',
  Personal:  'muted',
  Sold:      'outline',
};

const sexIcon: Record<Sex, typeof Mars> = { Male: Mars, Female: Venus, Unsexed: HelpCircle };

const severityVariant: Record<string, BadgeVariants['variant']> = {
  Note:  'muted',
  Watch: 'warn',
  Alert: 'danger',
};

function back() {
  router.push({ name: 'geckos' });
}
</script>

<template>
  <div v-if="!gecko" class="flex flex-col gap-6">
    <EmptyState title="Gecko not found" description="That record doesn't exist in your collection.">
      <template #actions>
        <Button variant="outline" size="sm" @click="back">
          <ArrowLeft class="size-4" />
          Back to geckos
        </Button>
      </template>
    </EmptyState>
  </div>

  <div v-else class="flex flex-col gap-8">
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" @click="back">
        <ArrowLeft class="size-4" /> Geckos
      </Button>
    </div>

    <!-- Hero -->
    <Card
      class="relative overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0"
    >
      <div
        class="grid grid-cols-1 md:grid-cols-[280px_1fr] gap-0"
      >
        <div
          class="relative bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center aspect-square md:aspect-auto md:h-full p-8"
        >
          <LowPolyGecko :size="240" class="animate-float" />
        </div>
        <div class="p-6 md:p-8 flex flex-col gap-4">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="statusVariant[gecko.status]">{{ gecko.status }}</Badge>
            <Badge variant="outline">{{ gecko.code }}</Badge>
            <span
              class="inline-flex items-center gap-1 text-xs text-brand-dark-600"
            >
              <component :is="sexIcon[gecko.sex]" class="size-3.5" stroke-width="2" />
              {{ gecko.sex }}
            </span>
          </div>
          <div class="flex flex-col gap-1">
            <h1 class="font-serif text-4xl md:text-5xl leading-none">{{ gecko.name }}</h1>
            <p class="text-brand-dark-600">
              {{ gecko.species }} · <span class="text-brand-dark-700 font-medium">{{ gecko.morph }}</span>
            </p>
          </div>
          <p v-if="gecko.notes" class="text-sm text-brand-dark-700 max-w-xl">{{ gecko.notes }}</p>

          <dl
            class="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-4 border-t border-brand-cream-200 mt-auto"
          >
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Age</dt>
              <dd class="font-serif text-xl text-brand-dark-950">{{ ageFromBirth(gecko.bornAt) }}</dd>
            </div>
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Weight</dt>
              <dd class="font-serif text-xl text-brand-dark-950">{{ gecko.weightG }} g</dd>
            </div>
            <div class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Born</dt>
              <dd class="font-serif text-xl text-brand-dark-950">{{ formatDate(gecko.bornAt) }}</dd>
            </div>
            <div v-if="gecko.priceUsd" class="flex flex-col">
              <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Price</dt>
              <dd class="font-serif text-xl text-brand-gold-700">${{ gecko.priceUsd }}</dd>
            </div>
          </dl>
        </div>
      </div>
    </Card>

    <!-- Tabs -->
    <Tabs default-value="overview">
      <TabsList class="w-full sm:w-auto">
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="weights">Weights</TabsTrigger>
        <TabsTrigger value="feedings">Feedings</TabsTrigger>
        <TabsTrigger value="sheds">Sheds</TabsTrigger>
        <TabsTrigger value="health">Health</TabsTrigger>
        <TabsTrigger value="photos">Photos</TabsTrigger>
      </TabsList>

      <!-- Overview -->
      <TabsContent value="overview">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
            <div class="px-6 py-5 flex items-center gap-2">
              <Scale class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">Weight trend</h3>
              <span
                v-if="weightDelta !== null"
                class="ml-auto text-xs px-2 py-0.5 rounded-full"
                :class="weightDelta >= 0 ? 'bg-brand-gold-100 text-brand-gold-800' : 'bg-red-100 text-red-700'"
                >{{ weightDelta >= 0 ? '+' : '' }}{{ weightDelta }} g</span
              >
            </div>
            <Separator />
            <div class="p-6">
              <WeightSparkline v-if="geckoWeights.length >= 2" :points="geckoWeights" />
              <p v-else class="text-sm text-brand-dark-600">Not enough data yet.</p>
            </div>
          </Card>

          <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
            <div class="px-6 py-5 flex items-center gap-2">
              <Drumstick class="size-4 text-brand-gold-700" />
              <h3 class="font-serif text-lg">Recent feedings</h3>
            </div>
            <Separator />
            <ul class="divide-y divide-brand-cream-200">
              <li
                v-for="f in geckoFeedings.slice(0, 5)"
                :key="f.id"
                class="flex items-center justify-between px-6 py-3 text-sm"
              >
                <div class="flex flex-col">
                  <span class="font-medium text-brand-dark-950">{{ f.prey }}</span>
                  <span class="text-xs text-brand-dark-600">×{{ f.quantity }}{{ f.note ? ` — ${f.note}` : '' }}</span>
                </div>
                <span class="text-xs text-brand-dark-600">{{ timeAgo(f.at) }}</span>
              </li>
              <li v-if="!geckoFeedings.length" class="px-6 py-6 text-sm text-brand-dark-600">
                No feedings logged yet.
              </li>
            </ul>
          </Card>
        </div>
      </TabsContent>

      <!-- Weights -->
      <TabsContent value="weights">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
          <div class="px-6 py-5 flex items-center justify-between">
            <h3 class="font-serif text-lg">Weight history</h3>
            <Button size="sm"><Plus class="size-4" /> Log weight</Button>
          </div>
          <Separator />
          <div class="p-6">
            <WeightSparkline v-if="geckoWeights.length >= 2" :points="geckoWeights" :height="180" />
          </div>
          <Separator />
          <ul class="divide-y divide-brand-cream-200">
            <li
              v-for="w in [...geckoWeights].reverse()"
              :key="w.id"
              class="flex items-center justify-between px-6 py-3 text-sm"
            >
              <span class="text-brand-dark-950 font-medium">{{ formatDate(w.at) }}</span>
              <span class="font-serif text-lg text-brand-dark-950">{{ w.grams }} g</span>
            </li>
          </ul>
        </Card>
      </TabsContent>

      <!-- Feedings -->
      <TabsContent value="feedings">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
          <div class="px-6 py-5 flex items-center justify-between">
            <h3 class="font-serif text-lg">Feeding log</h3>
            <Button size="sm"><Plus class="size-4" /> Log feeding</Button>
          </div>
          <Separator />
          <ul class="divide-y divide-brand-cream-200">
            <li
              v-for="f in geckoFeedings"
              :key="f.id"
              class="flex items-center justify-between px-6 py-4"
            >
              <div class="flex items-center gap-3">
                <span class="flex size-9 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
                  <Drumstick class="size-4" />
                </span>
                <div class="flex flex-col">
                  <span class="font-medium text-brand-dark-950">{{ f.prey }} ×{{ f.quantity }}</span>
                  <span v-if="f.note" class="text-xs text-brand-dark-600">{{ f.note }}</span>
                </div>
              </div>
              <div class="text-right">
                <div class="text-sm text-brand-dark-950">{{ formatDate(f.at) }}</div>
                <div class="text-xs text-brand-dark-600">{{ timeAgo(f.at) }}</div>
              </div>
            </li>
            <li v-if="!geckoFeedings.length" class="px-6 py-12 text-center text-sm text-brand-dark-600">
              No feedings yet.
            </li>
          </ul>
        </Card>
      </TabsContent>

      <!-- Sheds -->
      <TabsContent value="sheds">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
          <div class="px-6 py-5 flex items-center justify-between">
            <h3 class="font-serif text-lg">Shed log</h3>
            <Button size="sm"><Plus class="size-4" /> Record shed</Button>
          </div>
          <Separator />
          <ul class="divide-y divide-brand-cream-200">
            <li
              v-for="s in geckoSheds"
              :key="s.id"
              class="flex items-center justify-between px-6 py-4"
            >
              <div class="flex items-center gap-3">
                <span class="flex size-9 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
                  <Sparkles class="size-4" />
                </span>
                <div class="flex flex-col">
                  <Badge
                    :variant="s.completeness === 'Complete' ? 'success' : s.completeness === 'Partial' ? 'warn' : 'danger'"
                  >
                    {{ s.completeness }}
                  </Badge>
                  <span v-if="s.note" class="text-xs text-brand-dark-600 mt-1">{{ s.note }}</span>
                </div>
              </div>
              <div class="text-sm text-brand-dark-600">{{ formatDate(s.at) }}</div>
            </li>
            <li v-if="!geckoSheds.length" class="px-6 py-12 text-center text-sm text-brand-dark-600">
              No sheds recorded.
            </li>
          </ul>
        </Card>
      </TabsContent>

      <!-- Health -->
      <TabsContent value="health">
        <Card class="border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0">
          <div class="px-6 py-5 flex items-center justify-between">
            <h3 class="font-serif text-lg">Health log</h3>
            <Button size="sm"><Plus class="size-4" /> Add note</Button>
          </div>
          <Separator />
          <ul v-if="geckoHealth.length" class="divide-y divide-brand-cream-200">
            <li
              v-for="h in geckoHealth"
              :key="h.id"
              class="flex items-start gap-3 px-6 py-4"
            >
              <span class="mt-1 flex size-9 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800">
                <HeartPulse class="size-4" />
              </span>
              <div class="flex-1 flex flex-col gap-1 min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-medium text-brand-dark-950">{{ h.title }}</span>
                  <Badge :variant="severityVariant[h.severity]">{{ h.severity }}</Badge>
                </div>
                <p class="text-sm text-brand-dark-700">{{ h.detail }}</p>
                <span class="text-xs text-brand-dark-500">{{ formatDate(h.at) }} · {{ timeAgo(h.at) }}</span>
              </div>
            </li>
          </ul>
          <div v-else class="p-6">
            <EmptyState
              title="All clear."
              description="No health notes on file for this gecko."
            />
          </div>
        </Card>
      </TabsContent>

      <!-- Photos -->
      <TabsContent value="photos">
        <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
          <EmptyState
            title="No photos yet."
            description="Upload photos to build a growth gallery for this gecko."
          >
            <template #actions>
              <Button variant="default" size="sm"><ImageIcon class="size-4" /> Upload photos</Button>
            </template>
          </EmptyState>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>

<style scoped>
.animate-float {
  animation: gecko-float 6s ease-in-out infinite;
}
@keyframes gecko-float {
  0%, 100% { transform: translateY(0); }
  50%      { transform: translateY(-6px); }
}
</style>
