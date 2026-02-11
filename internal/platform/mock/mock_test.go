package mock

import (
	"testing"

	"github.com/druarnfield/shhh/internal/platform"
)

func TestUserEnv_SetAndGet(t *testing.T) {
	env := NewUserEnv()

	if err := env.Set("HTTP_PROXY", "http://proxy:8080"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, source, err := env.Get("HTTP_PROXY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "http://proxy:8080" {
		t.Errorf("value = %q, want %q", val, "http://proxy:8080")
	}
	if source != platform.SourceUser {
		t.Errorf("source = %v, want SourceUser", source)
	}
}

func TestUserEnv_GetMissing(t *testing.T) {
	env := NewUserEnv()
	_, _, err := env.Get("NONEXISTENT")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestUserEnv_Delete(t *testing.T) {
	env := NewUserEnv()
	env.Set("FOO", "bar")
	if err := env.Delete("FOO"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, _, err := env.Get("FOO")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestUserEnv_AppendPath(t *testing.T) {
	env := NewUserEnv()

	if err := env.AppendPath("/usr/local/bin"); err != nil {
		t.Fatalf("AppendPath: %v", err)
	}
	if err := env.AppendPath("/usr/bin"); err != nil {
		t.Fatalf("AppendPath: %v", err)
	}

	entries, err := env.ListPath()
	if err != nil {
		t.Fatalf("ListPath: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Dir != "/usr/local/bin" {
		t.Errorf("entries[0] = %q, want %q", entries[0].Dir, "/usr/local/bin")
	}
}

func TestUserEnv_AppendPath_Dedup(t *testing.T) {
	env := NewUserEnv()
	env.AppendPath("/usr/bin")
	env.AppendPath("/usr/bin")

	entries, _ := env.ListPath()
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1 (should dedup)", len(entries))
	}
}

func TestUserEnv_RemovePath(t *testing.T) {
	env := NewUserEnv()
	env.AppendPath("/usr/bin")
	env.AppendPath("/usr/local/bin")

	if err := env.RemovePath("/usr/bin"); err != nil {
		t.Fatalf("RemovePath: %v", err)
	}

	entries, _ := env.ListPath()
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
}

func TestProfileManager_ManagedBlock(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1")

	if err := pm.EnsureExists(); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}

	block := "$env:HTTP_PROXY = \"http://proxy:8080\"\n$env:HTTPS_PROXY = \"http://proxy:8080\""

	if err := pm.SetManagedBlock(block); err != nil {
		t.Fatalf("SetManagedBlock: %v", err)
	}

	got, err := pm.ManagedBlock()
	if err != nil {
		t.Fatalf("ManagedBlock: %v", err)
	}
	if got != block {
		t.Errorf("ManagedBlock = %q, want %q", got, block)
	}

	full, _ := pm.Read()
	if full == "" {
		t.Error("Read() returned empty")
	}
}

func TestProfileManager_AppendToManagedBlock(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1")
	pm.EnsureExists()

	pm.SetManagedBlock("$env:FOO = \"bar\"")
	pm.AppendToManagedBlock("$env:BAZ = \"qux\"")

	got, _ := pm.ManagedBlock()
	want := "$env:FOO = \"bar\"\n$env:BAZ = \"qux\""
	if got != want {
		t.Errorf("ManagedBlock = %q, want %q", got, want)
	}
}

func TestProfileManager_PreservesUserContent(t *testing.T) {
	pm := NewProfileManager("/tmp/test_profile.ps1").(*ProfileManager)
	pm.exists = true
	pm.content = "# my custom stuff\nSet-Alias ll ls\n"

	pm.SetManagedBlock("$env:FOO = \"bar\"")

	full, _ := pm.Read()
	if full == "" {
		t.Error("Read() returned empty")
	}

	pm.SetManagedBlock("$env:FOO = \"updated\"")
	got, _ := pm.ManagedBlock()
	if got != "$env:FOO = \"updated\"" {
		t.Errorf("ManagedBlock = %q", got)
	}
}
