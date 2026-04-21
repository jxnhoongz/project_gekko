<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ArrowLeft, ArrowRight, Mars, Venus, HelpCircle } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import { usePublicGecko } from '@/composables/usePublicGeckos';
import { ageFromBirth, formatDate } from '@/lib/format';

const props = defineProps<{ code: string }>();
const router = useRouter();
const { data: gecko, isLoading, isError } = usePublicGecko(computed(() => props.code));

const selectedIdx = ref(0);
const mainPhoto = computed(() => {
  if (!gecko.value || !gecko.value.photos.length) return null;
  return gecko.value.photos[selectedIdx.value]?.url ?? gecko.value.photos[0].url;
});

const sexIcon = computed(() => {
  if (!gecko.value) return HelpCircle;
  const m = { M: Mars, F: Venus, U: HelpCircle } as const;
  return m[gecko.value.sex];
});

const sexLabel = computed(() => {
  if (!gecko.value) return '';
  const m = { M: 'Male', F: 'Female', U: 'Unsexed' } as const;
  return m[gecko.value.sex];
});

function onJoinWaitlist() {
  router.push({ name: 'waitlist', query: { interested_in: props.code } });
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-6xl w-full px-4 sm:px-6 py-10 flex-1">
      <!-- Loading -->
      <div v-if="isLoading" class="flex flex-col gap-4">
        <Skeleton class="h-6 w-28" />
        <Skeleton class="h-80 w-full rounded-xl" />
        <Skeleton class="h-10 w-1/2" />
      </div>

      <!-- 404 / error -->
      <div
        v-else-if="isError"
        class="rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <LowPolyGecko :size="160" class="animate-float opacity-70" />
        <h1 class="font-serif text-3xl">Not available</h1>
        <p class="text-brand-dark-700">
          This gecko isn't in the current collection. It may have been reserved or sold.
        </p>
        <Button @click="router.push({ name: 'geckos' })">
          <ArrowLeft class="size-4" /> Back to available
        </Button>
      </div>

      <!-- Success -->
      <div v-else-if="gecko" class="flex flex-col gap-10">
        <!-- Breadcrumb -->
        <nav class="text-sm text-brand-dark-600 flex items-center gap-2">
          <RouterLink :to="{ name: 'geckos' }" class="hover:text-brand-gold-700">Geckos</RouterLink>
          <span>/</span>
          <span class="text-brand-dark-800">{{ gecko.code }}</span>
        </nav>

        <!-- Hero -->
        <section class="grid grid-cols-1 md:grid-cols-[1fr_360px] gap-8">
          <div class="flex flex-col gap-3">
            <div class="relative aspect-[4/3] rounded-xl overflow-hidden border border-brand-cream-300 bg-brand-cream-100">
              <img
                v-if="mainPhoto"
                :src="mainPhoto"
                :alt="gecko.name"
                class="w-full h-full object-cover"
              />
              <div v-else class="w-full h-full flex items-center justify-center">
                <LowPolyGecko :size="260" />
              </div>
            </div>
            <div v-if="gecko.photos.length > 1" class="flex gap-2 overflow-x-auto">
              <button
                v-for="(p, i) in gecko.photos"
                :key="i"
                type="button"
                class="size-20 rounded-md overflow-hidden border-2 transition-colors shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-gold-500"
                :class="selectedIdx === i ? 'border-brand-gold-600' : 'border-brand-cream-300 hover:border-brand-cream-400'"
                @click="selectedIdx = i"
              >
                <img :src="p.url" :alt="p.caption" class="w-full h-full object-cover" />
              </button>
            </div>
          </div>

          <!-- Facts -->
          <aside class="flex flex-col gap-4">
            <div>
              <span class="text-xs uppercase tracking-wider text-brand-dark-600 font-semibold">{{ gecko.code }}</span>
              <h1 class="font-serif text-4xl mt-1">{{ gecko.name || 'Unnamed' }}</h1>
              <p class="text-brand-dark-700 mt-2">
                {{ gecko.species_name }} · <span class="text-brand-dark-950 font-medium">{{ gecko.morph }}</span>
              </p>
            </div>

            <dl class="grid grid-cols-2 gap-3 py-4 border-y border-brand-cream-200">
              <div>
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Sex</dt>
                <dd class="font-serif text-lg flex items-center gap-1.5">
                  <component :is="sexIcon" class="size-4" stroke-width="2" />
                  {{ sexLabel }}
                </dd>
              </div>
              <div v-if="gecko.hatch_date">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Age</dt>
                <dd class="font-serif text-lg">{{ ageFromBirth(gecko.hatch_date) }}</dd>
              </div>
              <div v-if="gecko.hatch_date">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Hatched</dt>
                <dd class="font-serif text-lg">{{ formatDate(gecko.hatch_date) }}</dd>
              </div>
              <div v-if="gecko.list_price_usd">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Price</dt>
                <dd class="font-serif text-2xl text-brand-gold-700">${{ gecko.list_price_usd }}</dd>
              </div>
            </dl>

            <Button size="lg" class="w-full" @click="onJoinWaitlist">
              Interested? Join the waitlist
              <ArrowRight class="size-4" />
            </Button>
          </aside>
        </section>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>
