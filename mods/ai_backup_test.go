package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempAIBackupPaths(t *testing.T) string {
	t.Helper()
	dir := withTempOpenAIPaths(t)
	oldDir, oldCleared, oldDone := aiBackupDir, aiDataClearedFlag, aiRestoreDoneFlag
	aiBackupDir = filepath.Join(dir, "persist", "ai-backup")
	aiDataClearedFlag = filepath.Join(dir, "tmp", "data_cleared")
	aiRestoreDoneFlag = filepath.Join(dir, "run", "ai-restored")
	_ = os.MkdirAll(filepath.Dir(aiDataClearedFlag), 0755)
	t.Cleanup(func() {
		aiBackupDir, aiDataClearedFlag, aiRestoreDoneFlag = oldDir, oldCleared, oldDone
	})
	return dir
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAIBackupMirrorsValidConfigOnly(t *testing.T) {
	withTempAIBackupPaths(t)
	writeFile(t, xiaozhiConfigPath, `{"backend":"gemini"}`)
	writeFile(t, openAIConfigPath, `{"api_key":"sk-trunc`)

	backupAIConfig()

	if got := readFile(t, filepath.Join(aiBackupDir, "xiaozhi.json")); got != `{"backend":"gemini"}` {
		t.Fatalf("xiaozhi backup = %q", got)
	}
	if _, err := os.Stat(filepath.Join(aiBackupDir, "openai.json")); !os.IsNotExist(err) {
		t.Fatal("truncated openai.json must not replace the backup")
	}
	if fi, _ := os.Stat(filepath.Join(aiBackupDir, "xiaozhi.json")); fi.Mode().Perm() != 0600 {
		t.Fatalf("backup mode %v, want 0600", fi.Mode().Perm())
	}
}

func TestAIRestoreAfterClearOverwritesDefaults(t *testing.T) {
	withTempAIBackupPaths(t)
	writeFile(t, filepath.Join(aiBackupDir, "xiaozhi.json"), `{"backend":"gemini"}`)
	writeFile(t, filepath.Join(aiBackupDir, "openai.json"), `{"gemini_api_key":"k"}`)
	// vic-cloud already wrote defaults before wired came up.
	writeFile(t, xiaozhiConfigPath, `{"backend":"xiaozhi"}`)
	writeFile(t, aiDataClearedFlag, "1")

	if !restoreAIConfig() {
		t.Fatal("expected a restore after Clear User Data")
	}
	if got := readFile(t, xiaozhiConfigPath); got != `{"backend":"gemini"}` {
		t.Fatalf("xiaozhi.json = %q", got)
	}
	if got := readFile(t, openAIConfigPath); got != `{"gemini_api_key":"k"}` {
		t.Fatalf("openai.json = %q", got)
	}

	// Later changes in the same boot are never rolled back.
	writeFile(t, xiaozhiConfigPath, `{"backend":"grok"}`)
	if restoreAIConfig() {
		t.Fatal("restore must run once per boot")
	}
	if got := readFile(t, xiaozhiConfigPath); got != `{"backend":"grok"}` {
		t.Fatalf("xiaozhi.json rolled back to %q", got)
	}
}

func TestAIRestoreWithoutClearOnlyFillsMissing(t *testing.T) {
	withTempAIBackupPaths(t)
	writeFile(t, filepath.Join(aiBackupDir, "xiaozhi.json"), `{"backend":"gemini"}`)
	writeFile(t, filepath.Join(aiBackupDir, "openai.json"), `{"gemini_api_key":"k"}`)
	writeFile(t, xiaozhiConfigPath, `{"backend":"grok"}`)

	if !restoreAIConfig() {
		t.Fatal("missing openai.json should be restored")
	}
	if got := readFile(t, xiaozhiConfigPath); got != `{"backend":"grok"}` {
		t.Fatalf("existing xiaozhi.json overwritten: %q", got)
	}
	if got := readFile(t, openAIConfigPath); got != `{"gemini_api_key":"k"}` {
		t.Fatalf("openai.json = %q", got)
	}
}
