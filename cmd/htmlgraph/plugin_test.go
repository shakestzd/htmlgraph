package main

import (
	"testing"
)

func TestPluginCmdStructure(t *testing.T) {
	cmd := pluginCmd()
	if cmd.Use != "plugin" {
		t.Errorf("pluginCmd Use = %q, want %q", cmd.Use, "plugin")
	}
}

func TestPluginCmdHasInstallSubcommand(t *testing.T) {
	cmd := pluginCmd()
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "install" {
			found = true
			break
		}
	}
	if !found {
		t.Error("pluginCmd should have an 'install' subcommand")
	}
}

func TestPluginInstallCmdStructure(t *testing.T) {
	cmd := pluginInstallCmd()
	if cmd.Use != "install" {
		t.Errorf("pluginInstallCmd Use = %q, want %q", cmd.Use, "install")
	}
	if cmd.Short == "" {
		t.Error("pluginInstallCmd Short description should not be empty")
	}
	if cmd.Long == "" {
		t.Error("pluginInstallCmd Long description should not be empty")
	}
	if cmd.RunE == nil {
		t.Error("pluginInstallCmd RunE should not be nil")
	}
}
