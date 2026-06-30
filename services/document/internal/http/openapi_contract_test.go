package httpapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDocumentOpenAPIReportBaseResourcesMatchImplementedEnvelope(t *testing.T) {
	spec := readDocumentDocsOpenAPI(t)

	for _, ref := range []string{
		"#/components/schemas/ReportTypeListResponse",
		"#/components/schemas/ReportTemplateResponse",
		"#/components/schemas/ReportTemplateListResponse",
		"#/components/schemas/ReportTemplateStructureResponse",
		"#/components/schemas/ReportMaterialResponse",
		"#/components/schemas/ReportMaterialListResponse",
		"#/components/schemas/ReportDailyStatisticsResponse",
		"#/components/schemas/ReportOperationLogListResponse",
	} {
		if !strings.Contains(spec, ref) {
			t.Fatalf("document OpenAPI missing implemented response schema ref %s", ref)
		}
	}

	assertSchemaHasFields(t, spec, "ReportTypeListResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateListResponse", "data:", "page:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateStructureResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportMaterialResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportMaterialListResponse", "data:", "page:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportDailyStatisticsResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportOperationLogListResponse", "data:", "page:", "requestId:")

	templateSchema := openAPISchemaBlock(t, spec, "ReportTemplate")
	for _, field := range []string{"templateName:", "version:", "enabled:"} {
		if !strings.Contains(templateSchema, field) {
			t.Fatalf("ReportTemplate schema missing %s in:\n%s", field, templateSchema)
		}
	}
	for _, staleField := range []string{"name:", "status:"} {
		if containsYAMLField(templateSchema, staleField) {
			t.Fatalf("ReportTemplate schema still contains stale field %s in:\n%s", staleField, templateSchema)
		}
	}

	materialSchema := openAPISchemaBlock(t, spec, "ReportMaterial")
	for _, field := range []string{"id:", "materialName:", "enabled:"} {
		if !strings.Contains(materialSchema, field) {
			t.Fatalf("ReportMaterial schema missing %s in:\n%s", field, materialSchema)
		}
	}
	for _, staleField := range []string{"materialId:", "name:"} {
		if containsYAMLField(materialSchema, staleField) {
			t.Fatalf("ReportMaterial schema still contains stale field %s in:\n%s", staleField, materialSchema)
		}
	}
}

func readDocumentDocsOpenAPI(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller failed; skipping OpenAPI contract test")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		candidate := filepath.Join(dir, "docs", "services", "document", "api", "openapi.yaml")
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("docs/services/document/api/openapi.yaml not found; skipping OpenAPI contract test")
	return ""
}

func assertSchemaHasFields(t *testing.T, spec string, schema string, fields ...string) {
	t.Helper()
	block := openAPISchemaBlock(t, spec, schema)
	for _, field := range fields {
		if !strings.Contains(block, field) {
			t.Fatalf("%s schema missing %s in:\n%s", schema, field, block)
		}
	}
}

func openAPISchemaBlock(t *testing.T, spec string, schema string) string {
	t.Helper()
	lines := strings.Split(spec, "\n")
	start := -1
	startIndent := 0
	for i, line := range lines {
		if strings.TrimSpace(line) != schema+":" {
			continue
		}
		start = i
		startIndent = leadingSpaces(line)
		break
	}
	if start == -1 {
		t.Fatalf("schema %s not found in document OpenAPI", schema)
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if leadingSpaces(lines[i]) <= startIndent {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func leadingSpaces(value string) int {
	return len(value) - len(strings.TrimLeft(value, " "))
}

func containsYAMLField(block string, field string) bool {
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == field {
			return true
		}
	}
	return false
}
