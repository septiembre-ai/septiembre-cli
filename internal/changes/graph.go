package changes

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Edge is a source-file to imported-file relationship.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Status describes how a file changed relative to the base branch.
type Status string

const (
	StatusNew      Status = "new"      // added or untracked
	StatusModified Status = "modified" // edited in place
	StatusDeleted  Status = "deleted"  // removed
	StatusRenamed  Status = "renamed"  // moved to a new path
)

// Change is a file that differs from the base branch, tagged with its status.
type Change struct {
	Path   string
	Status Status
}

// Churn counts the lines added and deleted for a file relative to the base.
type Churn struct {
	Added   int `json:"added"`
	Deleted int `json:"deleted"`
}

// Graph contains changed files, internal Go import edges, and a per-file status
// map keyed by node path (values are Status strings). Import targets that were
// not themselves changed do not appear in Statuses; consumers render them as
// context nodes.
type Graph struct {
	Base     string            `json:"base"`
	Nodes    []string          `json:"nodes"`
	Edges    []Edge            `json:"edges"`
	Statuses map[string]string `json:"statuses,omitempty"`
	Diffs    map[string]string `json:"diffs,omitempty"`
	Churn    map[string]Churn  `json:"churn,omitempty"`
}

// Builder builds a changes graph for a repository.
type Builder struct {
	RepoRoot string
	Git      GitClient
}

// Build compares HEAD with base and builds a graph from changed files.
func (b Builder) Build(ctx context.Context, base string) (Graph, error) {
	if strings.TrimSpace(base) == "" {
		base = "main"
	}
	git := b.Git
	if git == nil {
		git = Git{RepoRoot: b.RepoRoot}
	}
	changed, err := git.ChangedFiles(ctx, base)
	if err != nil {
		return Graph{}, err
	}
	graph, err := BuildGraph(b.RepoRoot, base, changed)
	if err != nil {
		return Graph{}, err
	}
	diffs, err := git.Diffs(ctx, base)
	if err != nil {
		return Graph{}, err
	}
	graph.Diffs = diffs
	churn, err := git.Churn(ctx, base)
	if err != nil {
		return Graph{}, err
	}
	graph.Churn = churn
	return graph, nil
}

// BuildGraph builds a deterministic import graph for the supplied changes.
func BuildGraph(repoRoot, base string, changes []Change) (Graph, error) {
	modulePath, err := ReadModulePath(repoRoot)
	if err != nil {
		return Graph{}, err
	}
	changes = uniqueSortedChanges(changes)
	nodes := make([]string, 0, len(changes))
	statuses := make(map[string]string, len(changes))
	for _, change := range changes {
		nodes = append(nodes, change.Path)
		statuses[change.Path] = string(change.Status)
	}
	edges, err := BuildGoImportEdges(repoRoot, modulePath, changes)
	if err != nil {
		return Graph{}, err
	}
	return Graph{Base: base, Nodes: nodes, Edges: edges, Statuses: statuses}, nil
}

// ReadModulePath returns the module directive from go.mod.
func ReadModulePath(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod has no module directive")
}

// BuildGoImportEdges parses changed Go files and resolves module-internal imports to Go files.
// Deleted files are skipped: they no longer exist on disk and cannot be parsed.
func BuildGoImportEdges(repoRoot, modulePath string, changes []Change) ([]Edge, error) {
	seen := make(map[Edge]bool)
	var edges []Edge
	for _, change := range uniqueSortedChanges(changes) {
		if change.Status == StatusDeleted {
			continue
		}
		changed := change.Path
		if filepath.Ext(changed) != ".go" {
			continue
		}
		imports, err := parseFileImports(filepath.Join(repoRoot, filepath.FromSlash(changed)))
		if err != nil {
			return nil, fmt.Errorf("parse imports for %s: %w", changed, err)
		}
		for _, importPath := range imports {
			if !isInternalImport(modulePath, importPath) {
				continue
			}
			files, err := resolveImportFiles(repoRoot, modulePath, importPath)
			if err != nil {
				return nil, fmt.Errorf("resolve import %s from %s: %w", importPath, changed, err)
			}
			for _, target := range files {
				edge := Edge{From: changed, To: target}
				if seen[edge] {
					continue
				}
				seen[edge] = true
				edges = append(edges, edge)
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	return edges, nil
}

func parseFileImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		imports = append(imports, value)
	}
	sort.Strings(imports)
	return imports, nil
}

func isInternalImport(modulePath, importPath string) bool {
	prefix := modulePath + "/internal/"
	return strings.HasPrefix(importPath, prefix)
}

func resolveImportFiles(repoRoot, modulePath, importPath string) ([]string, error) {
	relDir := strings.TrimPrefix(importPath, modulePath+"/")
	absDir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.ToSlash(filepath.Join(relDir, name)))
	}
	sort.Strings(files)
	return files, nil
}

// statusRank orders statuses so that, when the same path shows up from several
// sources (committed diff, working tree, untracked), the most significant one
// wins: a deletion trumps an add, an add trumps a rename, a rename trumps a
// plain edit.
var statusRank = map[Status]int{
	StatusModified: 1,
	StatusRenamed:  2,
	StatusNew:      3,
	StatusDeleted:  4,
}

func uniqueSortedChanges(changes []Change) []Change {
	byPath := make(map[string]Status)
	for _, change := range changes {
		path := filepath.ToSlash(strings.TrimSpace(change.Path))
		if path == "" {
			continue
		}
		status := change.Status
		if status == "" {
			status = StatusModified
		}
		if current, ok := byPath[path]; !ok || statusRank[status] > statusRank[current] {
			byPath[path] = status
		}
	}
	out := make([]Change, 0, len(byPath))
	for path, status := range byPath {
		out = append(out, Change{Path: path, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
