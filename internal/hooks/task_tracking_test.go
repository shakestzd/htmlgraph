package hooks

import "testing"

func TestBuildStepDesc_WithTeammate(t *testing.T) {
	// Simulate addTaskStep logic: subject + task tag + teammate prefix.
	subject := "Build widget"
	taskID := "task-001"
	teammateName := "reviewer"

	stepDesc := subject + " [task:" + taskID + "]"
	if teammateName != "" {
		stepDesc = "[" + teammateName + "] " + stepDesc
	}

	expected := "[reviewer] Build widget [task:task-001]"
	if stepDesc != expected {
		t.Errorf("stepDesc = %q, want %q", stepDesc, expected)
	}
}

func TestBuildStepDesc_NoTeammate(t *testing.T) {
	subject := "Build widget"
	taskID := "task-001"
	teammateName := ""

	stepDesc := subject + " [task:" + taskID + "]"
	if teammateName != "" {
		stepDesc = "[" + teammateName + "] " + stepDesc
	}

	expected := "Build widget [task:task-001]"
	if stepDesc != expected {
		t.Errorf("stepDesc = %q, want %q", stepDesc, expected)
	}
}

func TestBuildStepDesc_EmptySubjectWithTeammate(t *testing.T) {
	// When subject is empty, addTaskStep defaults to "Task <taskID>".
	taskID := "task-002"
	teammateName := "implementer"
	subject := "Task " + taskID // default

	stepDesc := subject + " [task:" + taskID + "]"
	if teammateName != "" {
		stepDesc = "[" + teammateName + "] " + stepDesc
	}

	expected := "[implementer] Task task-002 [task:task-002]"
	if stepDesc != expected {
		t.Errorf("stepDesc = %q, want %q", stepDesc, expected)
	}
}
