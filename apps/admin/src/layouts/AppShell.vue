<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import {
  LayoutDashboard,
  Turtle,
  ClipboardList,
  DollarSign,
  Image,
  Database,
  Settings,
  Menu,
  LogOut,
  Bell,
} from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Avatar } from '@/components/ui/avatar';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import BrandLogo from '@/components/BrandLogo.vue';
import CommandPalette from '@/components/CommandPalette.vue';
import { Search } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const mobileOpen = ref(false);
const palette = ref<InstanceType<typeof CommandPalette> | null>(null);

interface NavItem {
  name: string;
  label: string;
  icon: typeof LayoutDashboard;
}
const nav: NavItem[] = [
  { name: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'geckos',    label: 'Geckos',    icon: Turtle },
  { name: 'waitlist',  label: 'Waitlist',  icon: ClipboardList },
  { name: 'sales',     label: 'Sales',     icon: DollarSign },
  { name: 'photos',    label: 'Photos',    icon: Image },
  { name: 'schema',    label: 'Schema',    icon: Database },
  { name: 'settings',  label: 'Settings',  icon: Settings },
];

const initials = computed(() => {
  const name = auth.admin?.name || auth.admin?.email || '';
  if (!name) return 'ZG';
  return name
    .split(/[\s@.]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join('');
});

function go(name: string) {
  mobileOpen.value = false;
  router.push({ name });
}

function onLogout() {
  auth.logout();
  router.push('/login');
}

const pageTitle = computed(() => {
  const m = nav.find((n) => n.name === route.name);
  return m?.label ?? '';
});
</script>

<template>
  <div class="min-h-screen flex bg-brand-cream-100 text-brand-dark-950">
    <!-- Desktop sidebar -->
    <aside
      class="hidden lg:flex lg:flex-col w-60 shrink-0 bg-brand-cream-50 border-r border-brand-cream-300"
    >
      <div class="flex items-center px-5 h-16 border-b border-brand-cream-300">
        <BrandLogo :size="36" />
      </div>
      <nav class="flex flex-col gap-1 p-3 flex-1 overflow-y-auto">
        <RouterLink
          v-for="n in nav"
          :key="n.name"
          :to="{ name: n.name }"
          class="group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-brand-dark-700 transition-colors hover:bg-brand-cream-100 hover:text-brand-dark-950"
          active-class="!bg-brand-gold-100 !text-brand-gold-900 shadow-sm"
        >
          <component :is="n.icon" class="size-5 shrink-0" stroke-width="1.75" />
          <span>{{ n.label }}</span>
        </RouterLink>
      </nav>
      <div class="border-t border-brand-cream-300 p-3">
        <div class="flex items-center gap-3 rounded-lg px-2 py-2">
          <Avatar>{{ initials }}</Avatar>
          <div class="flex flex-col min-w-0 flex-1">
            <span class="text-sm font-semibold text-brand-dark-950 truncate">
              {{ auth.admin?.name || 'Admin' }}
            </span>
            <span class="text-xs text-brand-dark-600 truncate">
              {{ auth.admin?.email }}
            </span>
          </div>
          <Button variant="ghost" size="icon-sm" aria-label="Log out" @click="onLogout">
            <LogOut class="size-4" />
          </Button>
        </div>
      </div>
    </aside>

    <!-- Main column -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Topbar -->
      <header
        class="h-16 bg-brand-cream-50 border-b border-brand-cream-300 px-4 sm:px-6 flex items-center justify-between gap-4"
      >
        <div class="flex items-center gap-3 min-w-0">
          <Button
            variant="ghost"
            size="icon"
            class="lg:hidden"
            aria-label="Open menu"
            @click="mobileOpen = true"
          >
            <Menu class="size-5" />
          </Button>
          <div class="lg:hidden">
            <BrandLogo :size="30" text-class="font-serif text-lg leading-none" />
          </div>
          <span
            class="hidden lg:block text-sm text-brand-dark-600 font-medium tracking-wide uppercase"
          >
            {{ pageTitle }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="hidden sm:flex items-center gap-2 h-9 px-3 rounded-md border border-brand-cream-300 bg-white text-brand-dark-600 hover:bg-brand-cream-100 text-sm transition-colors"
            aria-label="Open command palette"
            @click="palette?.open()"
          >
            <Search class="size-4" />
            <span>Quick jump</span>
            <span class="inline-flex items-center gap-0.5 ml-2 text-[10px] text-brand-dark-500">
              <kbd class="rounded border border-brand-cream-300 bg-brand-cream-50 px-1">⌘</kbd>
              <kbd class="rounded border border-brand-cream-300 bg-brand-cream-50 px-1">K</kbd>
            </span>
          </button>
          <Button
            variant="ghost"
            size="icon"
            class="sm:hidden"
            aria-label="Command palette"
            @click="palette?.open()"
          >
            <Search class="size-5" />
          </Button>
          <Button variant="ghost" size="icon" aria-label="Notifications">
            <Bell class="size-5" />
          </Button>
          <Avatar class="size-8 text-xs">{{ initials }}</Avatar>
        </div>
      </header>

      <!-- Page -->
      <main
        class="flex-1 overflow-y-auto bg-brand-cream-50 px-4 sm:px-6 lg:px-8 py-6 lg:py-10"
      >
        <div class="mx-auto w-full max-w-7xl">
          <RouterView v-slot="{ Component, route: r }">
            <transition name="route" mode="out-in">
              <component :is="Component" :key="r.path" />
            </transition>
          </RouterView>
        </div>
      </main>
    </div>

    <!-- Global command palette -->
    <CommandPalette ref="palette" />

    <!-- Mobile drawer -->
    <Sheet v-model:open="mobileOpen">
      <SheetContent side="left" class="flex flex-col gap-0 p-0">
        <div class="flex items-center px-5 h-16 border-b border-brand-cream-300">
          <BrandLogo :size="36" />
        </div>
        <nav class="flex flex-col gap-1 p-3 flex-1 overflow-y-auto">
          <button
            v-for="n in nav"
            :key="n.name"
            class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-brand-dark-700 transition-colors hover:bg-brand-cream-100 hover:text-brand-dark-950 text-left"
            :class="{
              '!bg-brand-gold-100 !text-brand-gold-900 shadow-sm': route.name === n.name,
            }"
            @click="go(n.name)"
          >
            <component :is="n.icon" class="size-5 shrink-0" stroke-width="1.75" />
            <span>{{ n.label }}</span>
          </button>
        </nav>
        <div class="border-t border-brand-cream-300 p-3">
          <div class="flex items-center gap-3 px-2 py-2">
            <Avatar>{{ initials }}</Avatar>
            <div class="flex flex-col min-w-0 flex-1">
              <span class="text-sm font-semibold text-brand-dark-950 truncate">{{
                auth.admin?.name || 'Admin'
              }}</span>
              <span class="text-xs text-brand-dark-600 truncate">{{ auth.admin?.email }}</span>
            </div>
            <Button variant="ghost" size="icon-sm" aria-label="Log out" @click="onLogout">
              <LogOut class="size-4" />
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  </div>
</template>
