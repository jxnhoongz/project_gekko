<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import {
  Turtle,
  Heart,
  Egg,
  ClipboardList,
  Drumstick,
  ArrowRight,
  CircleAlert,
  Image as ImageIcon,
  Pause,
  AlertTriangle,
} from 'lucide-vue-next';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatCard from '@/components/layout/StatCard.vue';
import { useAuthStore } from '@/stores/auth';
import { useDashboard, type DashboardItem, type DashboardItemKind } from '@/composables/useDashboard';
import { timeAgo } from '@/lib/format';

const auth = useAuthStore();
const router = useRouter();
const { data, isLoading, isError, error, refetch } = useDashboard();

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

const needsAttention = computed<DashboardItem[]>(() => data.value?.needs_attention ?? []);
const recentActivity = computed<DashboardItem[]>(() => data.value?.recent_activity ?? []);
const stats = computed(() => data.value?.stats);

const attentionIcon: Record<DashboardItemKind, typeof Drumstick> = {
  waitlist_stale:   ClipboardList,
  hold_stale:       Pause,
  // unused in this panel but typed for completeness
  gecko_created:    Turtle,
  waitlist_created: ClipboardList,
  media_uploaded:   ImageIcon,
};

const activityIcon: Record<DashboardItemKind, typeof Drumstick> = {
  gecko_created:    Turtle,
  waitlist_created: ClipboardList,
  media_uploaded:   ImageIcon,
  // unused in this panel but typed for completeness
  waitlist_stale:   ClipboardList,
  hold_stale:       Pause,
};

function linkFor(item: DashboardItem) {
  if (item.ref_kind === 'gecko') {
    return { name: 'gecko-detail', params: { id: item.ref_id } };
  }
  return { name: 'waitlist' };
}
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

    <!-- Error -->
    <Card
      v-if="isError"
      class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3"
    >
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex-1 min-w-0">
        <div class="text-sm font-semibold text-red-900">Couldn't load the dashboard.</div>
        <div class="text-xs text-red-800 break-all">{{ (error as Error)?.message }}</div>
      </div>
      <Button variant="outline" size="sm" @click="refetch()">Retry</Button>
    </Card>

    <!-- Loading -->
    <section
      v-else-if="isLoading"
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5"
    >
      <Skeleton v-for="n in 4" :key="n" class="h-28 rounded-xl" />
    </section>

    <!-- Stat grid -->
    <section
      v-else-if="stats"
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5"
    >
      <StatCard
        label="Total geckos"
        :value="stats.total_geckos"
        :icon="Turtle"
        :delta="`${stats.breeding} breeding`"
        delta-tone="up"
        hint="Active in collection"
      />
      <StatCard
        label="Breeding"
        :value="stats.breeding"
        :icon="Heart"
        hint="Currently pairing or holdback"
      />
      <StatCard
        label="Available"
        :value="stats.available"
        :icon="Egg"
        hint="Listed for sale"
      />
      <StatCard
        label="Waitlist"
        :value="stats.waitlist"
        :icon="ClipboardList"
        hint="Interested buyers"
      />
    </section>

    <!-- Needs attention + Recent activity -->
    <section class="grid grid-cols-1 lg:grid-cols-5 gap-6" v-if="!isError">
      <!-- Needs attention -->
      <Card class="lg:col-span-3 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Needs attention</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">
              Stale waitlist signups and long-held geckos
            </p>
          </div>
          <Badge variant="soft">{{ needsAttention.length }}</Badge>
        </div>
        <Separator />

        <div v-if="isLoading" class="p-6 flex flex-col gap-3">
          <Skeleton v-for="n in 3" :key="n" class="h-14 w-full" />
        </div>

        <ul v-else-if="needsAttention.length" class="divide-y divide-brand-cream-200">
          <li
            v-for="item in needsAttention"
            :key="`${item.kind}-${item.ref_id}`"
            class="flex items-center gap-4 px-6 py-4 hover:bg-brand-cream-100/60 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-gold-500"
            tabindex="0"
            role="link"
            @click="router.push(linkFor(item))"
            @keydown.enter="router.push(linkFor(item))"
            @keydown.space.prevent="router.push(linkFor(item))"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800"
            >
              <component :is="attentionIcon[item.kind]" class="size-5" stroke-width="1.75" />
            </div>
            <div class="flex flex-col min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-brand-dark-950 text-sm">{{ item.title }}</span>
                <Badge variant="warn" class="gap-1">
                  <CircleAlert class="size-3" /> Stale
                </Badge>
              </div>
              <span class="text-xs text-brand-dark-600">{{ item.detail }}</span>
            </div>
            <span class="text-xs font-medium shrink-0 text-brand-dark-600">{{ timeAgo(item.at) }}</span>
          </li>
        </ul>

        <div v-else class="px-6 py-10 text-sm text-brand-dark-500 text-center">
          Nothing needs your attention right now.
        </div>
      </Card>

      <!-- Recent activity -->
      <Card class="lg:col-span-2 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Recent activity</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">Last few days across the colony</p>
          </div>
        </div>
        <Separator />

        <div v-if="isLoading" class="p-6 flex flex-col gap-3">
          <Skeleton v-for="n in 5" :key="n" class="h-8 w-full" />
        </div>

        <ol v-else-if="recentActivity.length" class="flex flex-col gap-5 p-6">
          <li
            v-for="a in recentActivity"
            :key="`${a.kind}-${a.ref_id}-${a.at}`"
            class="relative flex gap-3 pl-8 cursor-pointer hover:bg-brand-cream-100/40 -mx-2 px-2 rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-gold-500"
            tabindex="0"
            role="link"
            @click="router.push(linkFor(a))"
            @keydown.enter="router.push(linkFor(a))"
            @keydown.space.prevent="router.push(linkFor(a))"
          >
            <span
              class="absolute left-0 top-0.5 flex size-6 items-center justify-center rounded-full bg-brand-gold-100 text-brand-gold-800 border border-brand-gold-200"
            >
              <component :is="activityIcon[a.kind]" class="size-3.5" stroke-width="2" />
            </span>
            <div class="flex flex-col min-w-0 flex-1">
              <span class="text-sm font-medium text-brand-dark-950">{{ a.title }}</span>
              <span v-if="a.detail" class="text-xs text-brand-dark-600">{{ a.detail }}</span>
            </div>
            <span class="text-xs text-brand-dark-500 shrink-0">{{ timeAgo(a.at) }}</span>
          </li>
        </ol>

        <div v-else class="px-6 py-10 text-sm text-brand-dark-500 text-center">
          Nothing logged yet.
        </div>
      </Card>
    </section>
  </div>
</template>
