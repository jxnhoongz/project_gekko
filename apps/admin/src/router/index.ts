import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '',              name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
      { path: 'geckos',        name: 'geckos',    component: () => import('@/views/GeckosView.vue') },
      { path: 'geckos/:id',    name: 'gecko-detail', component: () => import('@/views/GeckoDetailView.vue'), props: true },
      { path: 'waitlist',      name: 'waitlist',  component: () => import('@/views/WaitlistView.vue') },
      { path: 'sales',         name: 'sales',     component: () => import('@/views/SalesView.vue') },
      { path: 'photos',        name: 'photos',    component: () => import('@/views/PhotosView.vue') },
      { path: 'schema',        name: 'schema',    component: () => import('@/views/SchemaView.vue') },
      { path: 'settings',      name: 'settings',  component: () => import('@/views/SettingsView.vue') },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
  },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  const requiresAuth = to.matched.some((r) => r.meta.requiresAuth);

  if (requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' };
  }
  return true;
});
