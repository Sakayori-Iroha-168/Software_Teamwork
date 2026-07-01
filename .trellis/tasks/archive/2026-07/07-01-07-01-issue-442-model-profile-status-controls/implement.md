# Implementation Plan

1. Add a failing regression test for model profile edit controls:
   - render profiles
   - open edit dialog
   - toggle `enabled` and `isDefault`
   - save
   - assert PATCH body includes correct booleans and omits blank `apiKey`
2. Run the focused test and confirm it fails because controls/payload are missing.
3. Update `model-profiles.tsx`:
   - extend `FormData`
   - update defaults
   - include booleans in create/update request mapping
   - populate booleans in `openEdit`
   - add create/edit checkboxes
   - show optional default badge in table status cell
4. Add or adjust tests for create payload if needed.
5. Run focused test until green.
6. Run self-checks:
   - `bun run --cwd apps/web test:unit -- src/pages/admin/model-profiles.test.tsx`
   - `bun run --cwd apps/web check`
   - `bun run --cwd apps/web build`
   - `git diff --check`
7. Inspect `admin/parser-configs` and `reports/templates`:
   - `parser-configs` already exposes enabled/default controls.
   - `reports/templates` currently displays status and only supports structure/delete in this page, so full enable/disable management is outside this issue unless the existing active contract has a page-level edit form in scope.
