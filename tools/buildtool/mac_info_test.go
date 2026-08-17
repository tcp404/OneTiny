package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMacInfoPlistWritesVersionWithoutTagPrefix(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Info.plist.tmpl")
	dst := filepath.Join(dir, "out", "Info.plist")
	template := `<plist version="1.0">
<dict>
	<key>CFBundleShortVersionString</key>
	<string>{{ .Version }}</string>
	<key>CFBundleVersion</key>
	<string>{{ .Version }}</string>
</dict>
</plist>
`
	if err := os.WriteFile(src, []byte(template), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	if err := RenderMacInfoPlist(src, dst, "v0.11.0"); err != nil {
		t.Fatalf("RenderMacInfoPlist returned error: %v", err)
	}

	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read rendered plist: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "{{") {
		t.Fatalf("rendered plist still contains template marker: %q", text)
	}
	if got := strings.Count(text, "<string>0.11.0</string>"); got != 2 {
		t.Fatalf("rendered plist version count = %d, want 2 in %q", got, text)
	}
}
