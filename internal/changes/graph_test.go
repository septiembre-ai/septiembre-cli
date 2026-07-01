package changes

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestBuildGraph(t *testing.T) {
	repo := newGraphFixture(t)

	tests := []struct {
		name    string
		changed []Change
		want    Graph
	}{
		{
			name:    "changed go file resolves internal imports to repo files",
			changed: []Change{{Path: "internal/cli/changes.go", Status: StatusModified}},
			want: Graph{
				Base:  "main",
				Nodes: []string{"internal/cli/changes.go"},
				Edges: []Edge{
					{From: "internal/cli/changes.go", To: "internal/changes/git.go"},
					{From: "internal/cli/changes.go", To: "internal/changes/graph.go"},
				},
				Statuses: map[string]string{"internal/cli/changes.go": "modified"},
			},
		},
		{
			name:    "new go file is tagged and resolves imports",
			changed: []Change{{Path: "internal/cli/changes.go", Status: StatusNew}},
			want: Graph{
				Base:  "main",
				Nodes: []string{"internal/cli/changes.go"},
				Edges: []Edge{
					{From: "internal/cli/changes.go", To: "internal/changes/git.go"},
					{From: "internal/cli/changes.go", To: "internal/changes/graph.go"},
				},
				Statuses: map[string]string{"internal/cli/changes.go": "new"},
			},
		},
		{
			name:    "deleted go file is a node but is not parsed for imports",
			changed: []Change{{Path: "internal/cli/changes.go", Status: StatusDeleted}},
			want: Graph{
				Base:     "main",
				Nodes:    []string{"internal/cli/changes.go"},
				Edges:    nil,
				Statuses: map[string]string{"internal/cli/changes.go": "deleted"},
			},
		},
		{
			name:    "changed non go file remains a node without edges",
			changed: []Change{{Path: "README.md", Status: StatusModified}},
			want: Graph{
				Base:     "main",
				Nodes:    []string{"README.md"},
				Edges:    nil,
				Statuses: map[string]string{"README.md": "modified"},
			},
		},
		{
			name:    "external imports are ignored and nodes are sorted",
			changed: []Change{{Path: "README.md", Status: StatusModified}, {Path: "internal/cli/external.go", Status: StatusNew}},
			want: Graph{
				Base:     "main",
				Nodes:    []string{"README.md", "internal/cli/external.go"},
				Edges:    nil,
				Statuses: map[string]string{"README.md": "modified", "internal/cli/external.go": "new"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildGraph(repo, "main", tt.changed)
			if err != nil {
				t.Fatalf("BuildGraph: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("BuildGraph() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBuildUsesGitChangedFiles(t *testing.T) {
	repo := newGraphFixture(t)
	git := &fakeGit{
		files: []Change{{Path: "internal/cli/changes.go", Status: StatusModified}},
		diffs: map[string]string{"internal/cli/changes.go": "diff --git a/x b/x\n+added"},
		churn: map[string]Churn{"internal/cli/changes.go": {Added: 5, Deleted: 2}},
	}

	graph, err := (Builder{RepoRoot: repo, Git: git}).Build(context.Background(), "develop")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if git.base != "develop" {
		t.Fatalf("git base = %q, want develop", git.base)
	}
	if graph.Base != "develop" {
		t.Fatalf("graph base = %q, want develop", graph.Base)
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("edges len = %d, want 2", len(graph.Edges))
	}
	if graph.Diffs["internal/cli/changes.go"] != "diff --git a/x b/x\n+added" {
		t.Fatalf("graph diffs = %#v, want attached diff", graph.Diffs)
	}
	if got := graph.Churn["internal/cli/changes.go"]; got != (Churn{Added: 5, Deleted: 2}) {
		t.Fatalf("graph churn = %#v, want {5 2}", got)
	}
}

func TestParseNumstat(t *testing.T) {
	out := []byte(strings.Join([]string{
		"5\t2\tinternal/a.go",
		"-\t-\tassets/logo.png",
		"3\t0\tinternal/old.go => internal/new.go",
	}, "\n"))

	churn := parseNumstat(out)
	if got := churn["internal/a.go"]; got != (Churn{Added: 5, Deleted: 2}) {
		t.Fatalf("a.go churn = %#v, want {5 2}", got)
	}
	if got := churn["assets/logo.png"]; got != (Churn{}) {
		t.Fatalf("binary churn = %#v, want zero", got)
	}
	if got := churn["internal/new.go"]; got != (Churn{Added: 3, Deleted: 0}) {
		t.Fatalf("rename churn keyed by new path = %#v, want {3 0}", got)
	}
}

func TestParseUnifiedDiff(t *testing.T) {
	out := []byte(strings.Join([]string{
		"diff --git a/internal/a.go b/internal/a.go",
		"index 111..222 100644",
		"--- a/internal/a.go",
		"+++ b/internal/a.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"diff --git a/README.md b/README.md",
		"index 333..444 100644",
		"--- a/README.md",
		"+++ b/README.md",
		"@@ -0,0 +1 @@",
		"+docs",
	}, "\n"))

	diffs := parseUnifiedDiff(out)
	if len(diffs) != 2 {
		t.Fatalf("parsed %d files, want 2: %#v", len(diffs), diffs)
	}
	if !strings.Contains(diffs["internal/a.go"], "+new") || !strings.Contains(diffs["internal/a.go"], "-old") {
		t.Fatalf("internal/a.go diff missing hunk lines: %q", diffs["internal/a.go"])
	}
	if !strings.HasPrefix(diffs["README.md"], "diff --git a/README.md") {
		t.Fatalf("README.md diff should start at its own header: %q", diffs["README.md"])
	}
	if strings.Contains(diffs["README.md"], "internal/a.go") {
		t.Fatalf("README.md diff leaked another file's section: %q", diffs["README.md"])
	}
}

func TestGitChangedFiles(t *testing.T) {
	tests := []struct {
		name    string
		runner  *fakeRunner
		want    []Change
		wantErr bool
	}{
		{
			name: "unions committed, working tree and untracked with status precedence",
			runner: &fakeRunner{outputs: [][]byte{
				[]byte("abc123\n"),
				[]byte("M\tinternal/b.go\nA\tinternal/a.go\nD\tinternal/e.go\n"),
				[]byte("M\tinternal/c.go\nM\tinternal/a.go\nR100\tinternal/old.go\tinternal/f.go\n"),
				[]byte("internal/d.go\n"),
			}},
			want: []Change{
				{Path: "internal/a.go", Status: StatusNew},
				{Path: "internal/b.go", Status: StatusModified},
				{Path: "internal/c.go", Status: StatusModified},
				{Path: "internal/d.go", Status: StatusNew},
				{Path: "internal/e.go", Status: StatusDeleted},
				{Path: "internal/f.go", Status: StatusRenamed},
			},
		},
		{
			name:    "merge base failure returns error",
			runner:  &fakeRunner{errAt: 1, outputs: [][]byte{[]byte("fatal: no merge base")}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (Git{RepoRoot: "/repo", Runner: tt.runner}).ChangedFiles(context.Background(), "main")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ChangedFiles: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ChangedFiles() = %#v, want %#v", got, tt.want)
			}
			wantCommands := [][]string{
				{"git", "merge-base", "main", "HEAD"},
				{"git", "diff", "--name-status", "--diff-filter=ACMRD", "abc123", "HEAD"},
				{"git", "diff", "--name-status", "--diff-filter=ACMRD", "HEAD"},
				{"git", "ls-files", "--others", "--exclude-standard"},
			}
			if !reflect.DeepEqual(tt.runner.commands, wantCommands) {
				t.Fatalf("commands = %#v, want %#v", tt.runner.commands, wantCommands)
			}
		})
	}
}

func TestDiscoverRepoRoot(t *testing.T) {
	runner := &fakeRunner{outputs: [][]byte{[]byte("/repo\n")}}

	got, err := DiscoverRepoRoot(context.Background(), "/repo/internal/cli", runner)
	if err != nil {
		t.Fatalf("DiscoverRepoRoot: %v", err)
	}
	if got != "/repo" {
		t.Fatalf("root = %q, want /repo", got)
	}
	wantCommands := [][]string{{"git", "rev-parse", "--show-toplevel"}}
	if !reflect.DeepEqual(runner.commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, wantCommands)
	}
	if !reflect.DeepEqual(runner.dirs, []string{"/repo/internal/cli"}) {
		t.Fatalf("dirs = %#v, want start dir", runner.dirs)
	}
}

func TestDiscoverRepoRootFailure(t *testing.T) {
	runner := &fakeRunner{errAt: 1, outputs: [][]byte{[]byte("fatal: not a git repository")}}

	_, err := DiscoverRepoRoot(context.Background(), "/tmp", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteHTMLGraphIncludesSafeDeterministicNodesAndEdges(t *testing.T) {
	graph := Graph{
		Base:  "main",
		Nodes: []string{"internal/cli/<changes>.go", "README.md"},
		Edges: []Edge{{From: "internal/cli/<changes>.go", To: "internal/changes/graph.go"}},
	}
	var first bytes.Buffer
	if err := WriteHTMLGraph(&first, graph); err != nil {
		t.Fatalf("WriteHTMLGraph: %v", err)
	}
	var second bytes.Buffer
	if err := WriteHTMLGraph(&second, graph); err != nil {
		t.Fatalf("WriteHTMLGraph second: %v", err)
	}
	if first.String() != second.String() {
		t.Fatal("WriteHTMLGraph output is not deterministic")
	}
	html := first.String()
	if strings.Contains(html, "internal/cli/<changes>.go") {
		t.Fatalf("HTML contains unescaped raw node label: %s", html)
	}
	if !strings.Contains(html, "application/octet-stream") {
		t.Fatalf("HTML missing embedded graph data script: %s", html)
	}
	decoded := decodeEmbeddedGraphData(t, html)
	for _, want := range []string{
		`"base":"main"`,
		`"id":"README.md","status":"modified"`,
		`"id":"internal/changes/graph.go","status":"context"`,
		`"from":"internal/cli/\u003cchanges\u003e.go","to":"internal/changes/graph.go"`,
	} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("decoded graph data missing %q in %s", want, decoded)
		}
	}
}

func newGraphFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, repo, "go.mod", "module example.com/project\n\ngo 1.25\n")
	writeFile(t, repo, "README.md", "docs\n")
	writeFile(t, repo, "internal/cli/changes.go", `package cli

import (
	"fmt"
	"example.com/project/internal/changes"
	"github.com/spf13/cobra"
)

var _, _ = fmt.Println, changes.Graph{}
var _ = cobra.Command{}
`)
	writeFile(t, repo, "internal/cli/external.go", `package cli

import "github.com/spf13/cobra"

var _ = cobra.Command{}
`)
	writeFile(t, repo, "internal/changes/git.go", "package changes\n")
	writeFile(t, repo, "internal/changes/graph.go", "package changes\n")
	writeFile(t, repo, "internal/changes/graph_test.go", "package changes\n")
	return repo
}

func decodeEmbeddedGraphData(t *testing.T, html string) string {
	t.Helper()
	match := regexp.MustCompile(`<script id="graph-data" type="application/octet-stream">([^<]+)</script>`).FindStringSubmatch(html)
	if len(match) != 2 {
		t.Fatalf("graph data script not found in %s", html)
	}
	data, err := base64.URLEncoding.DecodeString(match[1])
	if err != nil {
		t.Fatalf("decode graph data: %v", err)
	}
	return string(data)
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fakeGit struct {
	base       string
	files      []Change
	diffs      map[string]string
	churn      map[string]Churn
	commits    []Commit
	prevTag    string
	releaseTag string
	rangeFrom  string
	rangeTo    string
	err        error
}

func (f *fakeGit) ChangedFiles(_ context.Context, base string) ([]Change, error) {
	f.base = base
	return f.files, f.err
}

func (f *fakeGit) Diffs(_ context.Context, _ string) (map[string]string, error) {
	return f.diffs, f.err
}

func (f *fakeGit) Churn(_ context.Context, _ string) (map[string]Churn, error) {
	return f.churn, f.err
}

func (f *fakeGit) PreviousTag(_ context.Context, tag string) (string, error) {
	f.releaseTag = tag
	return f.prevTag, f.err
}

func (f *fakeGit) ChangedFilesRange(_ context.Context, from, to string) ([]Change, error) {
	f.rangeFrom, f.rangeTo = from, to
	return f.files, f.err
}

func (f *fakeGit) DiffsRange(_ context.Context, _, _ string) (map[string]string, error) {
	return f.diffs, f.err
}

func (f *fakeGit) ChurnRange(_ context.Context, _, _ string) (map[string]Churn, error) {
	return f.churn, f.err
}

func (f *fakeGit) Log(_ context.Context, _, _ string) ([]Commit, error) {
	return f.commits, f.err
}

type fakeRunner struct {
	outputs  [][]byte
	errAt    int
	commands [][]string
	dirs     []string
}

func (f *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) ([]byte, error) {
	f.dirs = append(f.dirs, dir)
	command := append([]string{name}, args...)
	f.commands = append(f.commands, command)
	call := len(f.commands)
	if f.errAt == call {
		out := []byte(nil)
		if call <= len(f.outputs) {
			out = f.outputs[call-1]
		}
		return out, errors.New("command failed")
	}
	if call <= len(f.outputs) {
		return f.outputs[call-1], nil
	}
	return nil, nil
}
