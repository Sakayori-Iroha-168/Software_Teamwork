# Design

## Scope

Modify the frontend model profile page under `apps/web/src/pages/admin/model-profiles.tsx` and add focused page-level tests.

## UI Approach

- Follow the existing `parser-configs` management pattern.
- Keep the table `Status` cell as a read-only badge group:
  - `Enabled` / `Disabled`
  - optional `Default`
- Add checkboxes to the create and edit dialogs:
  - `Enabled`
  - `Set as default profile`

This resolves the screenshot confusion by keeping the table badge non-interactive while providing an obvious edit path via the existing edit action.

## Data Flow

- Extend the local model profile form shape with:
  - `enabled: boolean`
  - `isDefault: boolean`
- `EMPTY_FORM` starts with `enabled: true` and `isDefault: false`.
- `openEdit(profile)` copies both booleans from the selected `ModelProfile`.
- `formToCreateRequest(form)` sends both booleans.
- `formToUpdateRequest(form)` sends both booleans and continues to omit `apiKey` unless a new non-empty value was entered.
- Existing `useUpdateModelProfile` invalidates model profile list and detail caches, so no hook changes should be needed unless tests reveal a gap.

## Testing

Add `apps/web/src/pages/admin/model-profiles.test.tsx` using the existing `renderWithProviders` helper and mocked `fetch`.

Regression coverage:

- Load model profiles from the Gateway envelope.
- Open edit dialog for a profile.
- Verify current enabled/default checkbox states.
- Toggle both controls and save.
- Assert the captured PATCH request body includes `enabled` and `isDefault` and does not include `apiKey` when the key field is left blank.
- Include a create-dialog assertion if practical, or cover create payload in the same test file with a second focused case.

## Compatibility

- Reuses existing generated Gateway types via `@/lib/types`.
- Does not modify generated files.
- Does not change API endpoints or query keys.

## Risks

- The repository currently has some older pages with plain status pills but no edit form. The issue explicitly names `reports/templates` as a review boundary; this task should not expand into that page unless a direct contract/UI mismatch is found.
