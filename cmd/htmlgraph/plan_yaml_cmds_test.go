package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo creates a git repo in dir and configures a test user identity.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@example.com"},
		{"-C", dir, "config", "user.name", "Test User"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// writePlanFiles writes minimal YAML and HTML plan files under dir/plans/.
func writePlanFiles(t *testing.T, dir, planID string) (yamlPath, htmlPath string) {
	t.Helper()
	plansDir := filepath.Join(dir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	yamlPath = filepath.Join(plansDir, planID+".yaml")
	htmlPath = filepath.Join(plansDir, planID+".html")
	if err := os.WriteFile(yamlPath, []byte("meta:\n  id: "+planID+"\n"), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	if err := os.WriteFile(htmlPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}
	return yamlPath, htmlPath
}

// gitLog runs git log --oneline and returns the output.
func gitLog(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	return string(out)
}

func TestAutocommitPlan_CreatesCommit(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	yamlPath, _ := writePlanFiles(t, dir, "plan-test1234")

	if err := commitPlanChange(yamlPath, "plan(plan-test1234): test commit"); err != nil {
		t.Fatalf("commitPlanChange: %v", err)
	}

	log := gitLog(t, dir)
	if !strings.Contains(log, "plan(plan-test1234): test commit") {
		t.Errorf("expected commit subject in log, got:\n%s", log)
	}

	// Verify the commit includes both plan files.
	showOut, err := exec.Command("git", "-C", dir, "show", "--stat", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	showStr := string(showOut)
	if !strings.Contains(showStr, "plan-test1234.yaml") {
		t.Errorf("expected yaml in commit stat, got:\n%s", showStr)
	}
	if !strings.Contains(showStr, "plan-test1234.html") {
		t.Errorf("expected html in commit stat, got:\n%s", showStr)
	}
}

func TestAutocommitPlan_SkipsWhenNoGitRepo(t *testing.T) {
	dir := t.TempDir()
	// Do NOT init git.

	yamlPath, _ := writePlanFiles(t, dir, "plan-nogit12")

	if err := commitPlanChange(yamlPath, "should be skipped"); err != nil {
		t.Fatalf("expected nil error in non-git dir, got: %v", err)
	}
	// No assertions on git state — there is no repo. The function returning nil is the spec.
}

func TestAutocommitPlan_PreservesUnrelatedStagedChanges(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	yamlPath, _ := writePlanFiles(t, dir, "plan-isol5678")

	// Stage an unrelated file BEFORE calling commitPlanChange.
	unrelated := filepath.Join(dir, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write unrelated: %v", err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", "unrelated.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add unrelated: %v\n%s", err, out)
	}

	if err := commitPlanChange(yamlPath, "plan(plan-isol5678): isolation test"); err != nil {
		t.Fatalf("commitPlanChange: %v", err)
	}

	// The commit should NOT contain unrelated.txt.
	showOut, err := exec.Command("git", "-C", dir, "show", "--stat", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if strings.Contains(string(showOut), "unrelated.txt") {
		t.Errorf("unrelated.txt was included in the plan commit:\n%s", showOut)
	}

	// unrelated.txt should still be staged (index A).
	statusOut, err := exec.Command("git", "-C", dir, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if !strings.Contains(string(statusOut), "A  unrelated.txt") {
		t.Errorf("expected unrelated.txt to remain staged, got:\n%s", statusOut)
	}
}

func TestAutocommitPlan_NoOpCommit(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	yamlPath, _ := writePlanFiles(t, dir, "plan-noop9876")

	// Commit both files manually so the tree is clean.
	if out, err := exec.Command("git", "-C", dir, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", dir, "commit", "-m", "initial").CombinedOutput(); err != nil {
		t.Fatalf("git commit initial: %v\n%s", err, out)
	}

	// Count commits before.
	beforeOut, _ := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD").CombinedOutput()
	before := strings.TrimSpace(string(beforeOut))

	if err := commitPlanChange(yamlPath, "should be no-op"); err != nil {
		t.Fatalf("commitPlanChange: %v", err)
	}

	// Count commits after — should be unchanged.
	afterOut, _ := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD").CombinedOutput()
	after := strings.TrimSpace(string(afterOut))

	if before != after {
		t.Errorf("expected no new commit (count %s → %s)", before, after)
	}
}
