package platform

import "testing"

func TestExtractManagedBlock_Empty(t *testing.T) {
	if got := extractManagedBlock(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractManagedBlock_NoMarkers(t *testing.T) {
	content := "# some profile stuff\nSet-Alias ll Get-ChildItem\n"
	if got := extractManagedBlock(content); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractManagedBlock_OnlyStartMarker(t *testing.T) {
	content := "before\n" + ManagedBlockStart + "\nsome content\n"
	if got := extractManagedBlock(content); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractManagedBlock_SingleLine(t *testing.T) {
	content := ManagedBlockStart + "\nfnm env | Invoke-Expression\n" + ManagedBlockEnd + "\n"
	got := extractManagedBlock(content)
	want := "fnm env | Invoke-Expression"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractManagedBlock_MultiLine(t *testing.T) {
	content := "# user stuff\n" +
		ManagedBlockStart + "\n" +
		"line1\n" +
		"line2\n" +
		ManagedBlockEnd + "\n" +
		"# more user stuff\n"
	got := extractManagedBlock(content)
	want := "line1\nline2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractManagedBlock_EmptyBlock(t *testing.T) {
	content := ManagedBlockStart + "\n" + ManagedBlockEnd + "\n"
	if got := extractManagedBlock(content); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestReplaceManagedBlock_EmptyProfile(t *testing.T) {
	got := replaceManagedBlock("", "my content")
	want := ManagedBlockStart + "\nmy content\n" + ManagedBlockEnd + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceManagedBlock_AppendToExisting(t *testing.T) {
	profile := "# user config\nSet-Alias ll Get-ChildItem\n"
	got := replaceManagedBlock(profile, "fnm env | Invoke-Expression")
	want := "# user config\nSet-Alias ll Get-ChildItem\n" +
		ManagedBlockStart + "\nfnm env | Invoke-Expression\n" + ManagedBlockEnd + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceManagedBlock_AppendAddsNewline(t *testing.T) {
	profile := "no trailing newline"
	got := replaceManagedBlock(profile, "content")
	want := "no trailing newline\n" +
		ManagedBlockStart + "\ncontent\n" + ManagedBlockEnd + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceManagedBlock_ReplaceExisting(t *testing.T) {
	profile := "before\n" +
		ManagedBlockStart + "\nold content\n" + ManagedBlockEnd + "\n" +
		"after\n"
	got := replaceManagedBlock(profile, "new content")
	want := "before\n" +
		ManagedBlockStart + "\nnew content\n" + ManagedBlockEnd + "\n" +
		"after\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceManagedBlock_ReplaceMultiLine(t *testing.T) {
	profile := ManagedBlockStart + "\nline1\nline2\n" + ManagedBlockEnd + "\n"
	got := replaceManagedBlock(profile, "a\nb\nc")
	want := ManagedBlockStart + "\na\nb\nc\n" + ManagedBlockEnd + "\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRoundTrip_ExtractThenReplace(t *testing.T) {
	original := "# user config\n" +
		ManagedBlockStart + "\nfnm env | Invoke-Expression\n" + ManagedBlockEnd + "\n"

	block := extractManagedBlock(original)
	if block != "fnm env | Invoke-Expression" {
		t.Fatalf("extract: got %q", block)
	}

	updated := replaceManagedBlock(original, block+"\nextra line")
	got := extractManagedBlock(updated)
	want := "fnm env | Invoke-Expression\nextra line"
	if got != want {
		t.Errorf("round-trip: got %q, want %q", got, want)
	}
}
