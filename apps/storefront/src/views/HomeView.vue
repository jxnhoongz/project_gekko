<script setup lang="ts">
import { useRouter } from 'vue-router';
import { ArrowRight } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import PublicGeckoCard from '@/components/PublicGeckoCard.vue';
import { useAvailableTeaser } from '@/composables/usePublicGeckos';

const router = useRouter();
const { data: teaser, isLoading: teaserLoading } = useAvailableTeaser(3);
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <!-- Hero -->
    <section class="relative overflow-hidden">
      <div class="mx-auto max-w-6xl px-4 sm:px-6 py-16 sm:py-24 grid grid-cols-1 md:grid-cols-2 gap-10 items-center">
        <div class="flex flex-col gap-6">
          <span class="inline-flex items-center gap-2 text-xs font-semibold tracking-[0.16em] uppercase text-brand-gold-700">
            <LowPolyAccent :size="18" /> Zenetic Gekkos
          </span>
          <h1 class="font-serif text-5xl sm:text-6xl text-brand-dark-950 leading-tight">
            Small-batch gecko breedery in Phnom Penh.
          </h1>
          <p class="text-lg text-brand-dark-700 max-w-md">
            Hand-raised leopard, crested, and African fat-tail geckos — health-first, paired for pattern and temperament.
          </p>
          <div class="flex flex-col sm:flex-row gap-3">
            <Button size="lg" @click="router.push({ name: 'waitlist' })">
              Join the waitlist
              <ArrowRight class="size-4" />
            </Button>
            <Button variant="outline" size="lg" @click="router.push({ name: 'geckos' })">
              Browse available
            </Button>
          </div>
        </div>
        <div class="flex items-center justify-center">
          <LowPolyGecko :size="320" class="animate-float" />
        </div>
      </div>
    </section>

    <!-- Available teaser -->
    <section class="mx-auto max-w-6xl px-4 sm:px-6 py-12">
      <div class="flex items-end justify-between mb-6">
        <div>
          <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">Currently available</span>
          <h2 class="font-serif text-3xl mt-1">Ready for new homes</h2>
        </div>
        <RouterLink :to="{ name: 'geckos' }" class="text-sm text-brand-gold-700 hover:underline hidden sm:inline">
          See all →
        </RouterLink>
      </div>
      <div v-if="teaserLoading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <Skeleton v-for="n in 3" :key="n" class="h-80 rounded-xl" />
      </div>
      <div v-else-if="teaser && teaser.length" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <PublicGeckoCard v-for="g in teaser" :key="g.code" :gecko="g" />
      </div>
      <div
        v-else
        class="rounded-xl border border-dashed border-brand-cream-400 bg-brand-cream-50 p-10 text-center"
      >
        <p class="text-brand-dark-700">
          Nothing available right now — we'll announce new hatchlings to the waitlist first.
        </p>
        <Button variant="default" size="sm" class="mt-4" @click="router.push({ name: 'waitlist' })">
          Join the waitlist
        </Button>
      </div>
    </section>

    <!-- About -->
    <section class="mx-auto max-w-3xl px-4 sm:px-6 py-12 text-center">
      <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">About</span>
      <h2 class="font-serif text-3xl mt-1">Bred for health, priced for keepers.</h2>
      <p class="text-brand-dark-700 mt-4">
        Zenetic is a small, transparent operation. Every gecko we sell is a holdback we chose to raise ourselves — proven eaters, well-sheds, and genetics we're proud of. Ask us anything.
      </p>
    </section>

    <!-- CTA band -->
    <section class="bg-brand-gold-100 border-y border-brand-cream-300">
      <div class="mx-auto max-w-4xl px-4 sm:px-6 py-10 text-center flex flex-col items-center gap-4">
        <h2 class="font-serif text-2xl sm:text-3xl text-brand-dark-950">
          Want first dibs on our next clutch?
        </h2>
        <Button size="lg" @click="router.push({ name: 'waitlist' })">
          Join the waitlist
          <ArrowRight class="size-4" />
        </Button>
      </div>
    </section>

    <SiteFooter />
  </div>
</template>
