<template>
  <footer class="mt-16 border-t border-brand-cream-300 bg-brand-cream-50">
    <div class="mx-auto max-w-6xl px-4 sm:px-6 py-8 text-sm text-brand-dark-600 flex flex-col sm:flex-row gap-4 sm:items-center sm:justify-between">
      <div>© {{ new Date().getFullYear() }} Zenetic Gekkos · Phnom Penh</div>
      <div class="flex items-center gap-4">
        <RouterLink :to="{ name: 'geckos' }" class="hover:text-brand-dark-950">Geckos</RouterLink>
        <RouterLink :to="{ name: 'waitlist' }" class="hover:text-brand-dark-950">Waitlist</RouterLink>
      </div>
    </div>
  </footer>
</template>
