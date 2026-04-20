<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import {
  Turtle,
  Heart,
  Egg,
  ClipboardList,
  Drumstick,
  Scale,
  Sparkles,
  HeartPulse,
  ArrowRight,
  CircleAlert,
} from 'lucide-vue-next';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatCard from '@/components/layout/StatCard.vue';
import { useAuthStore } from '@/stores/auth';
import { geckos, upcoming, activity } from '@/mock';
import type { ActivityKind, UpcomingKind } from '@/types';
import { timeAgo } from '@/lib/format';

const auth = useAuthStore();
const router = useRouter();

const stats = computed(() => ({
  total: geckos.length,
  breeding: geckos.filter((g) => g.status === 'Breeding').length,
  available: geckos.filter((g) => g.status === 'Available').length,
}));

const greetingName = computed(
  () => auth.admin?.name?.split(' ')[0] || auth.admin?.email?.split('@')[0] || 'there',
);

const greeting = computed(() => {
  const h = new Date().getHours();
  if (h < 5)  return 'Up late';
  if (h < 12) return 'Good morning';
  if (h < 17) return 'Good afternoon';
  if (h < 22) return 'Good evening';
  return 'Up late';
});

const upcomingSorted = computed(() =>
  [...upcoming].sort((a, b) => new Date(a.dueAt).getTime() - new Date(b.dueAt).getTime()),
);

const recentActivity = computed(() =>
  [...activity].sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime()).slice(0, 8),
);

const activityIcon: Record<ActivityKind, typeof Drumstick> = {
  feeding: Drumstick,
  weight: Scale,
  shed: Sparkles,
  waitlist: ClipboardList,
  sale: Heart,
  health: HeartPulse,
};

const upcomingIcon: Record<UpcomingKind, typeof Drumstick> = {
  feeding: Drumstick,
  weigh: Scale,
  'shed-check': Sparkles,
  pairing: Heart,
};
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      :eyebrow="greeting"
      :title="`Hello, ${greetingName}.`"
      subtitle="Everything you need to keep the colony moving today."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="router.push({ name: 'geckos' })">
          View geckos
          <ArrowRight class="size-4" />
        </Button>
      </template>
    </PageHeader>

    <section class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5">
      <StatCard
        label="Total geckos"
        :value="stats.total"
        :icon="Turtle"
        :delta="`${stats.breeding} breeding`"
        delta-tone="up"
        hint="Active in collection"
      />
      <StatCard
        label="Active pairings"
        value="1"
        :icon="Heart"
        delta="Apsara × Rithy"
        hint="Cycling since Apr 12"
      />
      <StatCard label="Eggs incubating" value="0" :icon="Egg" hint="Pre-breeding phase" />
      <StatCard
        label="Waitlist"
        value="5"
        :icon="ClipboardList"
        delta="+2 this week"
        delta-tone="up"
        hint="Active interest"
      />
    </section>

    <section class="grid grid-cols-1 lg:grid-cols-5 gap-6">
      <Card class="lg:col-span-3 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Upcoming today</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">
              Feedings, weigh-ins and shed checks due within 48 hours
            </p>
          </div>
          <Badge variant="soft">{{ upcomingSorted.length }}</Badge>
        </div>
        <Separator />
        <ul class="divide-y divide-brand-cream-200">
          <li
            v-for="item in upcomingSorted"
            :key="item.id"
            class="flex items-center gap-4 px-6 py-4 hover:bg-brand-cream-100/60 transition-colors"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-lg"
              :class="
                item.overdue
                  ? 'bg-red-100 text-red-700'
                  : 'bg-brand-gold-100 text-brand-gold-800'
              "
            >
              <component :is="upcomingIcon[item.kind]" class="size-5" stroke-width="1.75" />
            </div>
            <div class="flex flex-col min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-brand-dark-950 text-sm">{{ item.title }}</span>
                <Badge v-if="item.overdue" variant="danger" class="gap-1">
                  <CircleAlert class="size-3" /> Overdue
                </Badge>
              </div>
              <span class="text-xs text-brand-dark-600">{{ item.detail }}</span>
            </div>
            <span
              class="text-xs font-medium shrink-0"
              :class="item.overdue ? 'text-red-700' : 'text-brand-dark-600'"
              >{{ timeAgo(item.dueAt) }}</span
            >
          </li>
        </ul>
      </Card>

      <Card class="lg:col-span-2 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Recent activity</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">Last few days across the colony</p>
          </div>
        </div>
        <Separator />
        <ol class="flex flex-col gap-5 p-6">
          <li v-for="a in recentActivity" :key="a.id" class="relative flex gap-3 pl-8">
            <span
              class="absolute left-0 top-0.5 flex size-6 items-center justify-center rounded-full bg-brand-gold-100 text-brand-gold-800 border border-brand-gold-200"
            >
              <component :is="activityIcon[a.kind]" class="size-3.5" stroke-width="2" />
            </span>
            <div class="flex flex-col min-w-0 flex-1">
              <span class="text-sm font-medium text-brand-dark-950">{{ a.title }}</span>
              <span class="text-xs text-brand-dark-600">{{ a.detail }}</span>
            </div>
            <span class="text-xs text-brand-dark-500 shrink-0">{{ timeAgo(a.at) }}</span>
          </li>
        </ol>
      </Card>
    </section>
  </div>
</template>
