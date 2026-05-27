// Package index parses a Go source tree into a symbol and reference graph.
package index

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kind labels the sort of definition a symbol is.
const (
	KindFunc   = "func"
	KindMethod = "method"
	KindType   = "type"
	KindVar    = "var"
	KindConst  = "const"
)

// Position is a 1-based line and column in a file.
type Position struct {
	Line int
	Col  int
}

// Def is a parsed definition.
type Def struct {
	Name     string
	Kind     string
	Path     string
	Package  string
	Pos      Position
	Receiver string
}

// Reference is a parsed use site keyed by the resolved definition name.
type Reference struct {
	Name string
	Path string
	Pos  Position
}

// CallEdge connects a caller name to a callee name.
type CallEdge struct {
	CallerName string
	CalleeName string
}

// Graph is the parsed result before it is persisted.
type Graph struct {
	Files []FileInfo
	Defs  []Def
	Refs  []Reference
	Calls []CallEdge
}

// FileInfo describes a parsed file.
type FileInfo struct {
	Path    string
	Package string
	Lines   int
}

// Parse walks root, parsing every .go file (test files excluded) and building
// a graph. Reference resolution is intentionally name-based: this tool indexes
// definitions and use sites by name, which is sufficient for jump-to-definition
// and find-references within a single-package corpus.
func Parse(root string) (*Graph, error) {
	fset := token.NewFileSet()
	g := &Graph{}
	defNames := map[string]bool{}

	var goFiles []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			goFiles = append(goFiles, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(goFiles)

	type fileAST struct {
		path string
		rel  string
		f    *ast.File
		lines int
	}
	var parsed []fileAST

	// Pass 1: parse and collect definitions.
	for _, p := range goFiles {
		src, rerr := os.ReadFile(p) //nolint:gosec // corpus paths are walked, not user input
		if rerr != nil {
			return nil, rerr
		}
		f, perr := parser.ParseFile(fset, p, src, 0)
		if perr != nil {
			return nil, perr
		}
		rel, _ := filepath.Rel(root, p)
		lines := lineCount(src)
		parsed = append(parsed, fileAST{path: p, rel: rel, f: f, lines: lines})
		g.Files = append(g.Files, FileInfo{Path: rel, Package: f.Name.Name, Lines: lines})

		for _, decl := range f.Decls {
			switch dd := decl.(type) {
			case *ast.FuncDecl:
				kind := KindFunc
				recv := ""
				if dd.Recv != nil && len(dd.Recv.List) > 0 {
					kind = KindMethod
					recv = receiverName(dd.Recv.List[0].Type)
				}
				pos := fset.Position(dd.Name.Pos())
				g.Defs = append(g.Defs, Def{
					Name: dd.Name.Name, Kind: kind, Path: rel, Package: f.Name.Name,
					Pos: Position{Line: pos.Line, Col: pos.Column}, Receiver: recv,
				})
				defNames[dd.Name.Name] = true
			case *ast.GenDecl:
				collectGenDecl(fset, dd, rel, f.Name.Name, g, defNames)
			}
		}
	}

	// Pass 2: collect references and call edges against known definition names.
	for _, fa := range parsed {
		enclosing := ""
		ast.Inspect(fa.f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				enclosing = node.Name.Name
			case *ast.Ident:
				if node.Obj != nil && node.Obj.Decl == node {
					return true // skip the definition ident itself
				}
				if defNames[node.Name] {
					pos := fset.Position(node.Pos())
					g.Refs = append(g.Refs, Reference{
						Name: node.Name, Path: fa.rel,
						Pos: Position{Line: pos.Line, Col: pos.Column},
					})
				}
			case *ast.CallExpr:
				if name := calleeName(node.Fun); name != "" && defNames[name] && enclosing != "" {
					g.Calls = append(g.Calls, CallEdge{CallerName: enclosing, CalleeName: name})
				}
			}
			return true
		})
	}
	return g, nil
}

func collectGenDecl(fset *token.FileSet, dd *ast.GenDecl, rel, pkg string, g *Graph, defNames map[string]bool) {
	for _, spec := range dd.Specs {
		switch sp := spec.(type) {
		case *ast.TypeSpec:
			pos := fset.Position(sp.Name.Pos())
			g.Defs = append(g.Defs, Def{
				Name: sp.Name.Name, Kind: KindType, Path: rel, Package: pkg,
				Pos: Position{Line: pos.Line, Col: pos.Column},
			})
			defNames[sp.Name.Name] = true
		case *ast.ValueSpec:
			kind := KindVar
			if dd.Tok == token.CONST {
				kind = KindConst
			}
			for _, nm := range sp.Names {
				if nm.Name == "_" {
					continue
				}
				pos := fset.Position(nm.Pos())
				g.Defs = append(g.Defs, Def{
					Name: nm.Name, Kind: kind, Path: rel, Package: pkg,
					Pos: Position{Line: pos.Line, Col: pos.Column},
				})
				defNames[nm.Name] = true
			}
		}
	}
}

func calleeName(fun ast.Expr) string {
	switch e := fun.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

func receiverName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return receiverName(e.X)
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return receiverName(e.X)
	}
	return ""
}

func lineCount(src []byte) int {
	if len(src) == 0 {
		return 0
	}
	n := 1
	for _, b := range src {
		if b == '\n' {
			n++
		}
	}
	return n
}
