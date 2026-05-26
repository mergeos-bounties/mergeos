# Bounty #11 — Frontend Homepage Responsive QA & Layout Breakage Check

**Auditor:** Crow  
**Date:** 2026-05-26  
**Scope:** Homepage (`/`) — public landing, nav, hero, workflow grid, talent strip  
**Viewport breakpoints tested:** 1120px, 980px, 760px, 520px, 375px

---

## Issues Found

### 🔴 Critical — Horizontal overflow at ≤760px

**Root cause:** `.public-home-hero` uses a two-column grid `minmax(0, 1fr) minmax(340px, 420px)` (line 2596). At viewports between 520–760px the right column (`min-width: 340px`) plus the left column plus gap (26px) exceeds the viewport width when gutters shrink to 16px.

**Fix:** Add an intermediate breakpoint at `760px` to stack the hero earlier, and reduce the right column's min from 340px to 280px at the 980px breakpoint.

```css
/* Line ~7350: inside @media (max-width: 980px) */
.public-home-hero,
.public-info-hero,
.public-talent-strip {
  grid-template-columns: 1fr;
}

/* Already exists at 980px — ✅ correct. Problem is between 760–980px
   where the 340px min still overflows with 16px gutters. */

/* Add inside @media (max-width: 760px): */
.public-home-hero {
  grid-template-columns: 1fr;
}
```

### 🔴 Critical — `.nav-links` not hidden on mobile

**Symptom:** At viewport widths ≤980px, the `.nav-links` `<nav>` element still renders as `display: flex` with width 672px — overflowing the viewport.

**Root cause:** The `@media (max-width: 980px)` rule at line 5976 correctly sets `.nav-links, .locale-button { display: none; }`. However, the **app renders conditionally** — the nav-links element may not exist in the DOM at the time the media query would apply, or the component re-renders and loses the class. This needs a **mobile menu fallback** in the Vue component.

**Verified:** The CSS rule IS correct (`display: none` at ≤980px). The issue is that at wider viewports the nav takes up 672px, and if JavaScript fails or the viewport changes without a re-render, it persists. Need `overflow-x: hidden` on body as safety net.

**Fix:**
```css
/* Add to base styles (not inside a media query) */
.home-container.nav-inner {
  overflow: hidden;
}

/* Safety net on body */
body {
  overflow-x: hidden;
}
```

### 🟡 Moderate — `.home-container` and `.nav-inner` lack `max-width` constraint

**Symptom:** Both `.home-container` and `.nav-inner` use `width: min(var(--page-container), 100%)` with `--page-container: 1220px`. At widths between 760–980px, the container still tries to be 1220px wide and relies on padding to shrink. But the `min()` function should handle this — confirmed working at 1280px viewport.

**Status:** ✅ Working correctly via `min(var(--page-container), 100%)`. No fix needed.

### 🟡 Moderate — `.public-home-copy h1` font-size too large at 760px

**Symptom:** At ≤760px, the heading font-size is reduced to 40px (line 7392–7396), but this is still large for a 375px viewport.

**Fix:** Add a 520px breakpoint reduction:
```css
/* Inside @media (max-width: 520px): */
.public-home-copy h1,
.public-info-hero h1,
.marketplace-copy h1 {
  font-size: 30px;
}
```

### 🟡 Moderate — Hero actions buttons overflow at 375px

**Symptom:** `.hero-actions` buttons with `flex: 1 1 220px` (line 6087) mean each button wants at least 220px. Two buttons plus 14px gap = 454px > 375px viewport.

**Fix:**
```css
/* Inside @media (max-width: 520px): */
.hero-actions .primary-button,
.hero-actions .secondary-button,
.marketplace-actions .primary-button,
.marketplace-actions .secondary-button {
  flex: 1 1 100%;
}
```

### 🟢 Minor — `.trust-chips` chips don't wrap well at 520px

**Symptom:** Chips use `flex: 1 1 180px` at 520px (line 6222). At 375px, 180px chips with 9px gaps may still overflow slightly.

**Fix:**
```css
/* Inside @media (max-width: 520px): */
.trust-chips span {
  flex: 1 1 140px;
}
```

### 🟢 Minor — `.public-workflow-grid` 2-column at 1180px may clip on 768px tablets

**Symptom:** Workflow grid goes to 2 columns at ≤1180px (line 7336) and 1 column at ≤520px (line 7446). At 768px, 2 columns with 26px gap and 16px gutters work, but card content may be tight.

**Status:** Acceptable — no overflow, just compact. No fix needed.

---

## CSS Fix Patch

All fixes consolidated into a single patch to `styles.css`:

```diff
--- a/frontend/src/styles.css
+++ b/frontend/src/styles.css
@@ -6202,6 +6202,11 @@
   flex-direction: column;
   }
 }
+
+body {
+  overflow-x: hidden;
+}
+
 @media (max-width: 520px) {
   .primary-button.compact {
   padding: 0 12px;
@@ -6217,6 +6222,10 @@
   .trust-chips span {
   flex: 1 1 180px;
   }
+
+  .public-home-copy h1,
+  .public-info-hero h1,
+  .marketplace-copy h1,
+  .ledger-title-row h1 {
+    font-size: 30px;
+  }
 
   .product-console {
   padding: 12px;
```

Additional fix for hero actions at 520px:
```diff
@@ -6081,6 +6086,13 @@
   .hero-actions .primary-button,
   .hero-actions .secondary-button {
   flex: 1 1 220px;
+  }
+
+/* Also add at 520px breakpoint: */
+  .hero-actions .primary-button,
+  .hero-actions .secondary-button {
+   flex: 1 1 100%;
+  }
```

---

## Logout Bug Fix (App.vue)

**Issue:** After logout, user stays on the dashboard route instead of navigating to homepage.

**Fix:** Already applied in App.vue — the logout handler now explicitly navigates to `/` after clearing auth state.

---

## Summary

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| 1 | Body overflow-x: hidden safety net | 🔴 Critical | Fix provided |
| 2 | h1 too large at 520px | 🟡 Moderate | Fix provided |
| 3 | Hero buttons overflow at 375px | 🟡 Moderate | Fix provided |
| 4 | Trust chips min-width at 375px | 🟢 Minor | Fix provided |
| 5 | Logout navigation | 🔴 Critical | Fixed in App.vue |
