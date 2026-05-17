package plan_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/wipnote/internal/launcher/mode"
	"github.com/shakestzd/wipnote/internal/launcher/plan"
)

// setupGitRepo creates a temp git repo with an initial commit on "main".
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	f, _ := os.Create(filepath.Join(dir, "README.md"))
	f.WriteString("# Test")
	f.Close()
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()
	return dir
}

func makeDirty(t *testing.T, dir string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, "dirty.txt"))
	if err != nil {
		t.Fatalf("makeDirty: %v", err)
	}
	f.WriteString("dirty")
	f.Close()
}

func TestLauncherPlan_DefaultsToWorktreeOnMain(t *testing.T) {
	dir := setupGitRepo(t)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "feat-abc12345",
		RuntimeMode: mode.RuntimeDevcontainer,
		InPlace:     false,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.IsolationMode != plan.IsolationManagedWorktree {
		t.Errorf("devcontainer+main+work-item: want IsolationManagedWorktree, got %v", p.IsolationMode)
	}
}

func TestLauncherPlan_InPlaceEscapeHatch(t *testing.T) {
	dir := setupGitRepo(t)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "feat-abc12345",
		RuntimeMode: mode.RuntimeDevcontainer,
		InPlace:     true,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.IsolationMode != plan.IsolationExplicitInPlace {
		t.Errorf("--in-place: want IsolationExplicitInPlace, got %v", p.IsolationMode)
	}
}

func TestHostProfile_StaysWarnOnly(t *testing.T) {
	dir := setupGitRepo(t)
	makeDirty(t, dir)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "feat-abc12345",
		RuntimeMode: mode.RuntimeHost,
		InPlace:     false,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.IsolationMode == plan.IsolationManagedWorktree {
		t.Error("host profile: must NOT default to managed-worktree (HIGH critique)")
	}
	if p.RefuseLaunch {
		t.Error("host profile: must be warn-only, not refuse (HIGH critique)")
	}
}

func TestLauncherPlan_DirtyMainWarns(t *testing.T) {
	dir := setupGitRepo(t)
	makeDirty(t, dir)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "feat-abc12345",
		RuntimeMode: mode.RuntimeHost,
		InPlace:     false,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.DirtyMainWarning == "" {
		t.Error("dirty main: expected non-empty DirtyMainWarning")
	}
}

func TestWorktreeLaunch_PreservesCanonicalRoot(t *testing.T) {
	dir := setupGitRepo(t)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "feat-abc12345",
		RuntimeMode: mode.RuntimeDevcontainer,
		InPlace:     false,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.IsolationMode != plan.IsolationManagedWorktree {
		t.Fatalf("expected managed worktree, got %v", p.IsolationMode)
	}
	rel, err := filepath.Rel(dir, p.PlannedWorktreePath)
	if err != nil || len(rel) == 0 || strings.HasPrefix(rel, "..") {
		t.Errorf("worktree path %q is not under repoRoot %q (rel=%q, err=%v)",
			p.PlannedWorktreePath, dir, rel, err)
	}
	if p.CanonicalRoot != dir {
		t.Errorf("CanonicalRoot: want %q, got %q", dir, p.CanonicalRoot)
	}
}

func TestLauncherPlan_NoWorkItemSkipsWorktree(t *testing.T) {
	dir := setupGitRepo(t)

	in := plan.Input{
		RepoRoot:    dir,
		WorkItemID:  "",
		RuntimeMode: mode.RuntimeDevcontainer,
		InPlace:     false,
	}
	p, err := plan.PlanLaunch(in)
	if err != nil {
		t.Fatalf("PlanLaunch: %v", err)
	}
	if p.IsolationMode == plan.IsolationManagedWorktree {
		t.Error("no work-item: must not select managed-worktree without an ID")
	}
}
