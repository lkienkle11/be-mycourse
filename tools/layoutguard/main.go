package main

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var allowedTopLevelDirs = map[string]bool{
	"cmd":        true,
	"config":     true,
	"internal":   true,
	"pkg":        true,
	"runtime":    true,
	"tools":      true,
	"migrations": true,
}

var analyzer = &analysis.Analyzer{
	Name: "layoutguard",
	Doc:  "enforces Go file placement: no random Go packages at the project root",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		checkFile(pass, file, filename)
	}

	return nil, nil
}

func checkFile(pass *analysis.Pass, file *ast.File, filename string) {
	clean := filepath.ToSlash(filepath.Clean(filename))

	if strings.HasSuffix(clean, "_test.go") {
		return
	}

	rel := toProjectRelative(clean)

	// Ignore files outside current project.
	if rel == "" || strings.HasPrefix(rel, "../") {
		return
	}

	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return
	}

	// Root-level Go file.
	if len(parts) == 1 {
		checkRootGoFile(pass, file, rel)
		return
	}

	top := parts[0]
	if !allowedTopLevelDirs[top] {
		pass.Reportf(
			file.Package,
			"go file is outside allowed project layout: %s; move it under cmd/, config/, internal/, pkg/, runtime/, tools/, or migrations/",
			rel,
		)
	}
}

func checkRootGoFile(pass *analysis.Pass, file *ast.File, rel string) {
	// Allow only root main.go with package main.
	if rel == "main.go" && file.Name.Name == "main" {
		return
	}

	pass.Reportf(
		file.Package,
		"go file must not be placed at project root: %s; move it under internal/<feature>/<layer>/",
		rel,
	)
}

func toProjectRelative(clean string) string {
	// Prefer working from the module root. singlechecker runs from the command cwd.
	// If the path already looks relative, keep it.
	if !filepath.IsAbs(clean) {
		return strings.TrimPrefix(clean, "./")
	}

	wd, err := filepath.Abs(".")
	if err != nil {
		return ""
	}

	wd = filepath.ToSlash(filepath.Clean(wd))

	if clean == wd {
		return ""
	}

	prefix := wd + "/"
	if strings.HasPrefix(clean, prefix) {
		return strings.TrimPrefix(clean, prefix)
	}

	return ""
}

func main() {
	// Keep token imported intentionally when extending diagnostics later.
	_ = token.NoPos

	singlechecker.Main(analyzer)
}
