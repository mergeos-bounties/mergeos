# Bounty #12: Project Category Selection Step

## Summary
Added a project category selection step (step 0) to the create-project wizard that clearly separates "New project" from "Fix bug in existing project" BEFORE the existing step 1.

## Changes Made

### `/frontend/src/App.vue`
1. **Added step 0 UI** — Two selectable cards: "New project" (Sparkles icon) and "Fix bug in existing project" (Bug icon), with a subtitle "What kind of project do you want to create?"
2. **Added `projectCategory` field** to `projectSetupForm` reactive object, initialized to `''`
3. **Updated `normalizeProjectWizardStep`** — now allows step 0 as valid (minimum step is 0 instead of 1)
4. **Added step 0 route path** in `projectWizardStepPaths` constant
5. **Updated `openProjectWizard`** — starts at step 0 instead of 1
6. **Updated `closeProjectWizard`** — resets to step 0
7. **"New project" card** clears `projectType` if it was previously set to "Repo Issue Fix"
8. **"Fix bug" card** auto-sets `projectType` to "Repo Issue Fix" and advances to step 1
9. **"New project" card** clears projectType if it was 'Repo Issue Fix' and advances to step 1

### `/frontend/src/styles.css`
1. Added `.project-category-step` styles
2. Added `.project-category-cards` flex layout
3. Added `.project-category-card` styles with hover/selected states
4. Added `.project-category-card svg` color rules

## Test Status
- 5/5 unit tests passing (server.test.js failure is pre-existing empty test file)
- Build succeeds
