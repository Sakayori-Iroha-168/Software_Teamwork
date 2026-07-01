# F-021 Model Profile Status Controls

## Goal

Resolve issue #442 by making `enabled` and `isDefault` reachable in the admin model profile UI while keeping the status table badge visually read-only and preserving API key safety.

## Background

- Issue: <https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/442>
- User screenshot circles the `Status` column on the model profile table. The green `Enabled` badge looks like it might be clickable, but it is display-only.
- Gateway public OpenAPI already exposes active `PATCH /api/v1/admin/model-profiles/{profileId}` with `UpdateModelProfileRequest.enabled?: boolean` and `isDefault?: boolean`.
- `apps/web/src/api/admin.ts` and `apps/web/src/features/admin-config/hooks/use-model-profiles.ts` already wrap the update mutation and invalidate model profile list/detail caches.
- `apps/web/src/pages/admin/parser-configs.tsx` is the nearest working comparison: status badges remain display-only and create/edit forms expose `enabled` and `isDefault` checkboxes.

## Requirements

- Add explicit create/edit form controls for model profile `enabled`.
- Add explicit create/edit form controls for model profile `isDefault`.
- Initialize edit form controls from the selected profile's current `enabled` and `isDefault` values.
- Send `enabled` and `isDefault` in create and update requests.
- Keep API key edit semantics unchanged: do not echo the original key; an empty API key field means keep the existing key.
- Keep the table status badge read-only, and show default status there as an additional display badge when applicable.
- Preserve existing mutation error feedback and TanStack Query cache invalidation behavior.
- Check `admin/parser-configs` and `reports/templates` for similar status-display gaps and report the boundary in the final summary.

## Acceptance Criteria

- [x] An admin can edit an enabled model profile and uncheck `enabled`; the PATCH body contains `enabled: false`.
- [x] An admin can edit a disabled model profile and check `enabled`; the PATCH body contains `enabled: true`.
- [x] An admin can set or unset a profile's default flag; the PATCH body contains the selected `isDefault` value.
- [x] New model profile creation can select `enabled` and `isDefault`; the POST body reflects the selected values.
- [x] The model profile table status column remains clearly display-only and includes a `Default` badge when `isDefault` is true.
- [x] Failed update requests display an error notification instead of failing silently.
- [x] API key edit field remains blank for existing profiles, and blank submit does not send `apiKey`.
- [x] A frontend regression test covers model profile status/default controls and request payloads.
- [x] `bun run --cwd apps/web check`, `bun run --cwd apps/web build`, and `git diff --check` are run or any inability to run them is documented.

## Verification Notes

- `bun run --cwd apps/web test:unit -- src/pages/admin/model-profiles.test.tsx`: passed, 2 tests.
- `bun run --cwd apps/web test:unit`: passed, 20 files / 68 tests.
- `bun run --cwd apps/web build`: passed with the existing Vite chunk-size warning.
- `bun run --cwd apps/web prettier src/pages/admin/model-profiles.tsx src/pages/admin/model-profiles.test.tsx --check`: passed.
- `git diff --check`: passed.
- `bun run --cwd apps/web check`: typecheck, test typecheck, and lint passed; `format:check` still fails on 41 pre-existing files outside this task's touched set.

## Out Of Scope

- No Gateway or AI Gateway OpenAPI changes.
- No direct browser use of internal AI Gateway OpenAPI.
- No redesign of report template/material management.
- No inline status toggle in the table unless the form-based path cannot satisfy the issue.
