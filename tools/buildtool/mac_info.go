package main

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type macInfoPlistData struct {
	Version string
}

func RenderMacInfoPlist(src, dst, version string) error {
	tmpl, err := template.ParseFiles(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, macInfoPlistData{Version: macPlistVersion(version)})
}

func macPlistVersion(version string) string {
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(version), "v"), "V")
}
