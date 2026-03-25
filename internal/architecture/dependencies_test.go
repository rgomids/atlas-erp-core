package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/rgomids/atlas-erp-core"

var boundedModules = map[string]struct{}{
	"billing":   {},
	"customers": {},
	"invoices":  {},
	"payments":  {},
}

func TestBoundedContextsImportOnlyPublicContractsFromOtherModules(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	internalRoot := filepath.Join(root, "internal")

	var violations []string

	if err := filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relativePath, err := filepath.Rel(internalRoot, path)
		if err != nil {
			return err
		}

		parts := strings.Split(filepath.ToSlash(relativePath), "/")
		if len(parts) < 2 {
			return nil
		}

		currentModule := parts[0]
		if _, ok := boundedModules[currentModule]; !ok {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			prefix := modulePath + "/internal/"
			if !strings.HasPrefix(importPath, prefix) {
				continue
			}

			remaining := strings.TrimPrefix(importPath, prefix)
			importParts := strings.Split(remaining, "/")
			if len(importParts) < 1 {
				continue
			}

			targetModule := importParts[0]
			if targetModule == currentModule {
				continue
			}
			if _, ok := boundedModules[targetModule]; !ok {
				continue
			}

			targetSuffix := strings.TrimPrefix(remaining, targetModule)
			if targetSuffix == "/public" || strings.HasPrefix(targetSuffix, "/public/") {
				continue
			}

			violations = append(violations, relativePath+" -> "+importPath)
		}

		return nil
	}); err != nil {
		t.Fatalf("walk internal tree: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("found forbidden cross-module imports:\n%s", strings.Join(violations, "\n"))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Dir(filepath.Dir(workingDirectory))
}
