# Add Desktop Document to Repository

## Goal

Add the desktop PDF document to this repository under a directory name that describes the document content better than a generic `docs` folder, so the team can review it through the normal branch and PR workflow.

## What I Already Know

* The user wants to add a document from the Desktop into the repository.
* The Desktop contains `C:/Users/liu/Desktop/报告生成需求说明书.pdf`.
* The repository currently has no existing business documentation directory.
* The repository root currently contains `AGENTS.md`, `.gitignore`, and Trellis/Codex support directories.
* The user wants a directory name more specific than `docs`.

## Assumptions (Temporary)

* The PDF is a requirements/specification document for report generation.
* The file should be copied into the repository, not moved away from the Desktop.
* A focused directory such as `requirements/` or `specs/` is preferable to a broad `docs/`.

## Open Questions

* None.

## Requirements (Evolving)

* Add the desktop PDF to a new repository directory.
* Use `report-generation/` as the target directory name.
* Keep the change suitable for a small PR.

## Acceptance Criteria (Evolving)

* [ ] The PDF exists in the repository under the chosen directory.
* [ ] The original Desktop file remains untouched.
* [ ] The repository has a clean git status aside from the intended task changes.
* [ ] The work is committed on a feature/documentation branch and ready for PR.

## Definition of Done (Team Quality Bar)

* Verify the file exists at the expected path.
* Verify git status and diff show only intended changes.
* Push the feature branch to `origin`.
* Provide the PR direction from the fork branch to upstream `main`.

## Out of Scope

* Editing the PDF contents.
* Converting the PDF into Markdown.
* Adding a full documentation site.

## Technical Notes

* Desktop inspection found `报告生成需求说明书.pdf` with size 2,833,723 bytes and last modified time `2026/6/27 13:49:19`.
* Existing repository Markdown files: `AGENTS.md` only.
* Candidate directory names:
  * `requirements/` - clear, common, and matches "需求说明书".
  * `specs/` - concise and technical, but slightly less obvious in Chinese-team context.
  * `report-generation/` - selected by the user; very specific and useful if this folder may hold all report-generation materials.

## Decision (ADR-lite)

**Context**: The user wants a directory name more specific than `docs/` for a PDF named `报告生成需求说明书.pdf`.

**Decision**: Use `report-generation/`.

**Consequences**: The directory name is scoped to the report-generation feature area. If future repository documentation grows, broader folders such as `requirements/` or `docs/` can still be added later without renaming this specific feature folder.
