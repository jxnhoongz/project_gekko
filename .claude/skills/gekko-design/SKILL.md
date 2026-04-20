---
name: gekko-design
description: Use when building or modifying UI in any Zenetic Gekkos project (admin panel, storefront, or future). Defines the brand aesthetic, color system, typography, low-poly motif rules, component patterns, motion, spacing, and anti-patterns. Read before generating any new views, components, icons, illustrations, or visual elements.
---

# Gekko Design Manual

The visual language for **Zenetic Gekkos** — a gecko breeding business in Phnom Penh. Every UI surface (admin, storefront, marketing) must feel like it comes from the same brand.

## The one-liner

> **Warm, earthy, handcrafted low-poly — like a fancy nature magazine rendered in 3D.**

Think: natural history publication meets indie-game low-poly. Tactile, generous, confident. Not corporate, not childish, not glossy.

## Aesthetic pillars

1. **Low-poly is a flourish, not a texture.** True 3D (Three.js) lives on a few hero moments — login, landing heroes, major empty states. Everything else uses flat UI with SVG low-poly *accents* (faceted shapes, triangular highlights).
2. **Warm earth tones over cool tech tones.** Cream, tan, gold, dark brown. Never pure white, never pure black, never Silicon-Valley blue.
3. **Generous whitespace.** Magazine-like breathing room. Don't cram.
4. **Flat lighting, soft shadows.** Shadows are warm-tinted (`shadow-brand-dark-950/10`), not gray/black.
5. **One voice across screens.** A dashboard card and a public product card should feel related even if the layout differs.

## When to use Three.js vs SVG low-poly vs flat

| Surface | Treatment |
|---|---|
| Login page | Full Three.js animated backdrop (already built — `AnimatedBackground.vue`). Reference implementation. |
| Marketing hero sections (storefront home) | Three.js OR animated SVG low-poly — pick whichever loads faster on mobile. |
| Empty states ("No geckos yet") | SVG low-poly illustration. Never Three.js — it's overkill for an empty list. |
| Loading states | Simple spinner or skeleton. No 3D. |
| Forms, tables, lists, buttons | Flat UI only. No 3D. No faceting. Clean and functional. |
| Page headers / nav | Flat. Maybe a small SVG low-poly logo mark. |
| Dashboard stat cards | Flat with warm-tinted icons. No 3D. |
| Icons | Outline (heroicons-outline style). No filled/solid. No low-poly faceting. |

**Rule of thumb:** if the element must be interactive, fast, or dense — flat. If it's a decorative focal point with space to breathe — consider low-poly.

## Color system

Use the Tailwind brand tokens defined in `gekko-admin/tailwind.config.js`. **Never use raw hex or arbitrary Tailwind grays.**

### Palette roles

| Role | Tokens | Usage |
|---|---|---|
| Page background | `brand-cream-50` · `brand-cream-100` | Default canvas |
| Surface / card | `brand-cream-50` on `brand-cream-100` bg (subtle contrast) | Cards, panels |
| Surface border | `brand-cream-300` · `brand-cream-400` | Dividers, card borders |
| Primary action | `brand-gold-600` (bg) · `brand-gold-700` (hover) · `brand-gold-800` (active) | Primary buttons, active tabs, key CTAs |
| Primary text on gold | `white` | Button labels |
| Body text | `brand-dark-950` (headings) · `brand-dark-700` (body) · `brand-dark-600` (muted) | All text |
| Secondary action | `brand-dark-950` bg, white text | Destructive, logout, "Cancel" |
| Accent icon chip | `brand-gold-100` bg + `brand-gold-700` stroke | Stat icons, badge backgrounds |
| Focus ring | `brand-gold-500` | All focus states |
| Error | `red-600` text, `red-300` border, `red-500` focus | Validation, destructive confirms |
| Success | Use `brand-gold-600` — we don't add green unless absolutely necessary |

### Color rules

- **Never** use raw Tailwind grays (`gray-*`, `slate-*`, `zinc-*`). Use `brand-dark-*` or `brand-cream-*`.
- **Never** use `bg-white` on the main page canvas. Use `bg-brand-cream-50` or `bg-brand-cream-100`.
- Inputs may use `bg-white` for field contrast (see `BaseInput.vue`).
- Alert/status colors (red, amber, etc.) use Tailwind defaults, muted to match the palette temperature.

## Typography

### Fonts (proposed defaults — change in `index.html` + Tailwind config if different)

- **Headings**: `DM Serif Display` — warm, slightly magazine, free on Google Fonts
- **Body / UI**: `Inter` — neutral, readable, free on Google Fonts

Load both in `<head>` of `index.html`:

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=DM+Serif+Display&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet" />
```

Add to `tailwind.config.js`:

```js
fontFamily: {
  display: ['"DM Serif Display"', 'serif'],
  sans: ['Inter', 'system-ui', 'sans-serif'],
},
```

### Type scale

| Use | Class | Notes |
|---|---|---|
| Page title (h1/h2) | `text-3xl font-display font-normal` | DM Serif Display, NOT bold — the face is already dramatic |
| Section heading | `text-xl font-semibold` | Inter |
| Card title | `text-lg font-semibold` | Inter |
| Body | `text-base font-normal` | Inter |
| Small / meta | `text-sm` | Inter |
| Caption | `text-xs text-brand-dark-600` | Inter |

Use `font-display` **sparingly** — only hero titles, major page headers, maybe gecko names on the storefront. Overuse = loses impact.

## Component system: shadcn-vue (themed)

We use **[shadcn-vue](https://www.shadcn-vue.com/)** as the component baseline. It's built on **[Reka UI](https://reka-ui.com/)** (a Vue port of Radix UI primitives) and styled with **[Tailwind CSS](https://tailwindcss.com/)** + **class-variance-authority**. Components are copied (not imported) into `src/components/ui/`, so we own them and restyle freely with our brand tokens. Never use shadcn's default neutral palette.

**Install target** (on the Linux setup):
```bash
bun add -D shadcn-vue
bunx shadcn-vue@latest init        # pick "new-york" style, configure paths
bunx shadcn-vue@latest add button card input label dialog table sonner dropdown-menu tabs
```

### Theming: override CSS variables with brand palette

shadcn components read `--background`, `--primary`, etc. as CSS variables. In `src/style.css` (or wherever the root stylesheet lives), **override the defaults with HSL triplets of our brand tokens**:

```css
@layer base {
  :root {
    /* Surface */
    --background:          39 40% 98%;   /* brand-cream-50  */
    --foreground:          215 28% 17%;  /* brand-dark-950  */
    --card:                36 38% 96%;   /* brand-cream-100 */
    --card-foreground:     215 28% 17%;
    --popover:             39 40% 98%;
    --popover-foreground:  215 28% 17%;

    /* Actions */
    --primary:             29 54% 56%;   /* brand-gold-600  */
    --primary-foreground:  0 0% 100%;
    --secondary:           215 28% 17%;  /* brand-dark-950  */
    --secondary-foreground: 0 0% 100%;
    --muted:               34 32% 90%;   /* brand-cream-300 */
    --muted-foreground:    0 0% 40%;     /* brand-dark-700  */
    --accent:              33 58% 82%;   /* brand-gold-200  */
    --accent-foreground:   27 46% 48%;   /* brand-gold-700  */
    --destructive:         0 72% 51%;
    --destructive-foreground: 0 0% 100%;

    /* Borders / inputs */
    --border:              32 35% 84%;   /* brand-cream-400 */
    --input:               32 35% 84%;
    --ring:                29 55% 64%;   /* brand-gold-500  */

    --radius: 0.75rem;                    /* 12px — warmer than default 0.5rem */
  }
}
```

**Rule**: never add a component with raw `bg-neutral-*` or `bg-slate-*` from shadcn defaults. If a component ships with those, fix it to use `bg-background`, `bg-primary`, `bg-muted` etc., which flow from our CSS variables.

### Component usage rules

These rules apply on top of shadcn primitives:

#### Button
- Variants: shadcn's `default` = our gold primary. `secondary` = dark. `outline` = gold-bordered. `ghost` = transparent gold. `destructive` = red (rare). `link` = text-only, gold.
- **One `variant="default"` primary button per screen.** Multiple = confused hierarchy.
- Use `size="lg"` for hero CTAs, `default` for most, `sm` for table row actions.

#### Card
- Default shadcn Card gets rounded-xl via our `--radius`. Keep.
- For login and hero surfaces, apply a glass variant manually: `class="bg-card/80 backdrop-blur-lg border-border/50"` on top of Three.js background.
- Hoverable cards: add `transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg`.

#### Input / Label
- Use shadcn `<Input>` and `<Label>`. Our theme will give it cream-400 border + gold-500 focus ring automatically.
- Required marker: append `<span class="text-destructive ml-0.5">*</span>` to Label manually.

#### Dialog / Sheet / DropdownMenu / Table / Toast (sonner)
- Use as-is. Our theme tokens flow through.
- For Table, wrap in `<Card>` for the magazine-card feel, or use a full-bleed table with cream-200 dividers.

#### Forms
- Pair shadcn `<Form>` primitives with **vee-validate** (form state) + **zod** (schema validation). Install: `bun add vee-validate @vee-validate/zod zod`.
- Layout: labels above fields, `space-y-4` between fields, submit button bottom-right with `variant="default"`, cancel as ghost to its left.

### Patterns: dashboards, lists, forms

**Stat card**:
```
[icon chip]  [Label]
             [Big number]
```
- Wrap in shadcn `<Card>` with `class="p-6 hover:-translate-y-0.5 hover:shadow-lg transition-all"`.
- Icon chip = 48px rounded square, `bg-accent`, icon in `text-accent-foreground`. Decorate with a small faceted SVG triangle in the top-right corner of the chip for the low-poly flourish.
- Value = `text-2xl font-bold text-foreground`.
- Label = `text-sm text-muted-foreground`.

**List row** (future — geckos table, waitlist table):
- Use shadcn `<Table>`. Theme gives `bg-background` rows with `border-border` dividers.
- Row hover = `hover:bg-muted/50`.
- Actions column on the right, `<Button variant="ghost" size="sm">` for each action.

**Form section**:
- Wrap in `<Card class="p-8">`.
- Group related fields with `<div class="space-y-4">`.
- Submit `<Button variant="default">` bottom-right, `<Button variant="ghost">` Cancel to its left.

## Motion

Existing animations in `tailwind.config.js`:

- `animate-pulse-slow` — 4s pulse for loading/attention (use on waitlist CTA, pending states)
- `animate-float` — 6s gentle vertical bob (use on hero illustrations, mascots)

### Motion rules

- **Durations**: 200ms for UI feedback (hover, focus). 400–600ms for page transitions. Slow, organic.
- **Easing**: `cubic-bezier(0.4, 0, 0.6, 1)` (Tailwind's default ease-in-out) or `ease-out`. Never snappy / bouncy Material spring.
- **No scale pops > 1.02.** Card hover is `translate-y -0.5` (2px lift), not a scale transform.
- **Respect `prefers-reduced-motion`.** Three.js components should pause or swap to static image. Keyframe animations in Tailwind honor this by default.

## Spacing & layout

- Max page width: `max-w-7xl` (1280px). Generous but not endless.
- Page padding: `px-4 sm:px-6 lg:px-8 py-8`.
- Card gap in grids: `gap-6`.
- Section gap (vertical between major regions): `mb-8` or `space-y-8`.
- Inside a card: `padding="md"` (24px) default, `padding="lg"` (32px) for forms/hero content.

## Iconography

- **Library**: `lucide-vue-next` — the default icon set recommended by shadcn-vue. Install: `bun add lucide-vue-next`. Line style, consistent stroke.
- **Stroke**: `stroke-width="2"` (lucide default). Only adjust for tiny sizes (`stroke-width="1.75"` at `w-4`).
- **Size**: `w-6 h-6` for stat icons, `w-5 h-5` for buttons/inline, `w-4 h-4` for tiny meta.
- **Color**: `text-accent-foreground` (brand-gold-700) or `text-muted-foreground`. Never pure black/white.
- **Never** mix line + solid icon styles in the same view.
- **Low-poly flourish idea**: for hero/empty-state icons, wrap the lucide icon in a small faceted SVG background (triangular gradient splat behind) to nod at the aesthetic.

## SVG low-poly illustration rules

When creating faceted/low-poly SVG illustrations (empty states, section decorations, gecko silhouettes):

- **Geometry**: 8–40 triangles/polygons per illustration. Less is more.
- **Palette**: 3–5 shades from `brand-cream` or `brand-gold`, plus one `brand-dark` for shadow facets.
- **Lighting**: Single light source from top-right. Lighter shades on facets facing up-right, darker facets on bottom-left.
- **Outline**: Subtle `stroke-brand-cream-400` at 0.5px, OR no outline. Never heavy black outlines.
- **Subject ideas**: gecko silhouette, egg, leaf, branch, landscape, moon.

## Anti-patterns (don't do)

- ❌ Pure-black text (`text-black` or `#000`) — use `brand-dark-950`
- ❌ Pure-white page backgrounds — use `brand-cream-50`
- ❌ Material Design spring bounces
- ❌ Gradient backgrounds on buttons (flat gold only)
- ❌ Filled/solid icons
- ❌ Generic stock illustrations (Undraw, Storyset) — commission or generate custom low-poly SVG
- ❌ Multiple primary CTAs on one screen
- ❌ Three.js on pages that need to be fast/indexed/scrollable
- ❌ Emoji used as icons (use lucide)
- ❌ Default Tailwind grays
- ❌ Using shadcn components with their **default neutral palette** — always theme via CSS variables with our brand tokens
- ❌ Mixing shadcn components with a second UI library (MUI, Vuetify, Naive, etc.) — shadcn only

## Canonical reference screens

When in doubt, mimic these:

- **`apps/admin/src/views/LoginView.vue`** — The gold standard for hero aesthetic (Three.js + glass card). Re-implemented with shadcn-vue `<Card>` primitives but the *look* stays identical.
- **`apps/admin/src/components/AnimatedBackground.vue`** — Reference Three.js low-poly scene (animated wave). Reuse for other hero/landing surfaces.
- **[shadcn-vue docs](https://www.shadcn-vue.com/docs)** and **[reka-ui docs](https://reka-ui.com/)** — for component installation steps and primitive APIs. Copy the patterns, **not** the color palette.

## When this skill applies

Read this skill before:

- Creating a new Vue view or storefront page in any gekko repo
- Installing a new shadcn-vue component (run `bunx shadcn-vue@latest add <component>`, then theme-check)
- Editing CSS variables, Tailwind config, or `components.json`
- Designing an empty state, loading state, or error state
- Choosing colors, spacing, typography for any UI
- Picking or creating an illustration / icon set
- Deciding whether to use 3D / Three.js for a surface

## Change control

This manual is the single source of truth for gekko brand UI. To change a rule:

1. Open a discussion with the user — explain the tradeoff
2. Get explicit agreement
3. Update this file in the same commit as the code that uses the new rule

Do not silently drift from these rules.
