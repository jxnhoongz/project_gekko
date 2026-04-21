import type { VariantProps } from 'class-variance-authority';
import { cva } from 'class-variance-authority';

export { default as Badge } from './Badge.vue';

export const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default:     'border-transparent bg-primary text-primary-foreground',
        secondary:   'border-transparent bg-secondary text-secondary-foreground',
        accent:      'border-transparent bg-accent text-accent-foreground',
        outline:     'border-border text-foreground',
        soft:        'border-transparent bg-brand-gold-100 text-brand-gold-800',
        success:     'border-transparent bg-brand-gold-200 text-brand-gold-900',
        warn:        'border-transparent bg-amber-100 text-amber-900',
        danger:      'border-transparent bg-red-100 text-red-800',
        muted:       'border-transparent bg-brand-cream-200 text-brand-dark-700',
      },
    },
    defaultVariants: { variant: 'default' },
  },
);

export type BadgeVariants = VariantProps<typeof badgeVariants>;
