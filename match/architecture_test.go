package match

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEngineDoesNotImportApplication guards the library boundary: the
// engine must stay reusable and cannot depend on Ovahol's internals. The
// test walks the whole engine module (the parent of this package) and
// fails on any import of the application module.
func TestEngineDoesNotImportApplication(t *testing.T) {
	const appPrefix = "github.com/ovahol/ovahol"

	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			pkg := strings.Trim(imp.Path.Value, `"`)
			if pkg == appPrefix || strings.HasPrefix(pkg, appPrefix+"/") {
				t.Errorf("%s imports application package %q", path, pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk engine module: %v", err)
	}
}
