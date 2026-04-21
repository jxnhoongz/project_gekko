import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  { path: '/',              name: 'home',         component: () => import('@/views/HomeView.vue') },
  { path: '/geckos',        name: 'geckos',       component: () => import('@/views/GeckosView.vue') },
  { path: '/geckos/:code',  name: 'gecko-detail', component: () => import('@/views/GeckoDetailView.vue'), props: true },
  { path: '/waitlist',      name: 'waitlist',     component: () => import('@/views/WaitlistView.vue') },
  { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});
