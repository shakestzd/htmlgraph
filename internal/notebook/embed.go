package notebook

import (
	"embed"
	"os"
	"path/filepath"
)

// Files contains the plan review notebook and its dependencies.
// These files are copied from prototypes/ at build time by plugin/build.sh.
//
//go:embed files/plan_notebook.py files/plan_ui.py files/plan_persistence.py files/critique_renderer.py files/dagre_widget.py files/chat_widget.py files/claude_chat.py files/amendment_parser.py
var Files embed.FS

// WriteToDir extracts all embedded notebook files to the given directory.
func WriteToDir(dir string) error {
	entries, err := Files.ReadDir("files")
	if err != nil {
		return err
	}
	for _, e := range entries {
		data, err := Files.ReadFile("files/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0644); err != nil {
			return err
		}
	}
	return nil
}
