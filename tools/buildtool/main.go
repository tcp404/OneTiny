package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: buildtool <command> [options]")
	}

	switch args[0] {
	case "metadata":
		return runMetadata(args[1:])
	case "current-version":
		return runCurrentVersion(args[1:])
	case "validate-version":
		return runValidateVersion(args[1:])
	case "ensure-dir":
		return runEnsureDir(args[1:])
	case "copy-file":
		return runCopyFile(args[1:])
	case "render-mac-info":
		return runRenderMacInfo(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "require-target-os":
		return runRequireTargetOS(args[1:])
	case "go-build":
		return runGoBuild(args[1:])
	case "archive":
		return runArchive(args[1:])
	case "checksums":
		return runChecksums(args[1:])
	case "remove":
		return runRemove(args[1:])
	default:
		return errors.Errorf("unknown command %q", args[0])
	}
}

func runCurrentVersion(args []string) error {
	fs := flag.NewFlagSet("current-version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	version, err := DefaultVersion()
	if err != nil {
		return err
	}
	fmt.Println(version)
	return nil
}

func runMetadata(args []string) error {
	fs := flag.NewFlagSet("metadata", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	targetValue := fs.String("target", "", "target in GOOS-GOARCH format")
	kindValue := fs.String("kind", "cli", "artifact kind: cli or gui")
	version := fs.String("version", "", "version value for ldflags metadata")
	field := fs.String("field", "", "single field to print")
	format := fs.String("format", "env", "output format: env or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	target, err := parseTargetOrCurrent(*targetValue)
	if err != nil {
		return err
	}
	kind, err := ParseKind(*kindValue)
	if err != nil {
		return err
	}
	spec, err := NewArtifactSpec(kind, target, DefaultConfig())
	if err != nil {
		return err
	}
	resolvedVersion, err := resolveVersion(*version)
	if err != nil {
		return err
	}

	values := map[string]string{
		"target":       target.Name(),
		"goos":         target.OS,
		"goarch":       target.Arch,
		"label":        target.Label,
		"kind":         string(kind),
		"base-name":    spec.BaseName,
		"staging-dir":  spec.StagingDir,
		"zip-path":     spec.ZipPath,
		"binary-name":  spec.BinaryName,
		"binary-path":  spec.BinaryPath,
		"mac-app-path": spec.MacAppPath,
		"ldflags":      BuildLdflags(kind, target, resolvedVersion),
	}

	if *field != "" {
		value, ok := values[*field]
		if !ok {
			return errors.Errorf("unknown metadata field %q", *field)
		}
		fmt.Println(value)
		return nil
	}

	switch *format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(values)
	case "env":
		for _, key := range []string{"target", "goos", "goarch", "label", "kind", "base-name", "staging-dir", "zip-path", "binary-name", "binary-path", "mac-app-path", "ldflags"} {
			fmt.Printf("%s=%s\n", key, values[key])
		}
		return nil
	default:
		return errors.Errorf("unsupported metadata format %q", *format)
	}
}

func parseTargetOrCurrent(value string) (Target, error) {
	if value == "" {
		return CurrentTarget()
	}
	return ParseTarget(value)
}

func runValidateVersion(args []string) error {
	fs := flag.NewFlagSet("validate-version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	version := fs.String("version", "", "release version tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return ValidateVersion(*version)
}

func resolveVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultVersion()
	}
	return version, nil
}

func runEnsureDir(args []string) error {
	fs := flag.NewFlagSet("ensure-dir", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var paths stringList
	fs.Var(&paths, "path", "directory path to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return EnsureDirs(paths)
}

func runCopyFile(args []string) error {
	fs := flag.NewFlagSet("copy-file", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	src := fs.String("src", "", "source file")
	dst := fs.String("dst", "", "destination file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *dst == "" {
		return errors.New("copy-file requires --src and --dst")
	}
	return CopyFile(*src, *dst)
}

func runRenderMacInfo(args []string) error {
	fs := flag.NewFlagSet("render-mac-info", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	src := fs.String("src", "", "source Info.plist template")
	dst := fs.String("dst", "", "destination Info.plist")
	version := fs.String("version", "", "release version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *dst == "" || *version == "" {
		return errors.New("render-mac-info requires --src, --dst, and --version")
	}
	return RenderMacInfoPlist(*src, *dst, *version)
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "path to verify")
	kind := fs.String("kind", "any", "expected kind: any, file, or dir")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("verify requires --path")
	}
	return VerifyPath(*path, *kind)
}

func runRequireTargetOS(args []string) error {
	fs := flag.NewFlagSet("require-target-os", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	targetValue := fs.String("target", "", "target in GOOS-GOARCH format")
	goos := fs.String("os", "", "required GOOS")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *goos == "" {
		return errors.New("require-target-os requires --os")
	}
	target, err := parseTargetOrCurrent(*targetValue)
	if err != nil {
		return err
	}
	return RequireTargetOS(target, *goos)
}

func runGoBuild(args []string) error {
	fs := flag.NewFlagSet("go-build", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	targetValue := fs.String("target", "", "target in GOOS-GOARCH format")
	kindValue := fs.String("kind", "", "artifact kind: cli or gui")
	version := fs.String("version", "", "version value for ldflags")
	outputPath := fs.String("output", "", "output binary path")
	packagePath := fs.String("package", "", "Go package path")
	cgoEnabled := fs.String("cgo-enabled", "", "optional CGO_ENABLED value")
	if err := fs.Parse(args); err != nil {
		return err
	}

	target, err := parseTargetOrCurrent(*targetValue)
	if err != nil {
		return err
	}
	kind, err := ParseKind(*kindValue)
	if err != nil {
		return err
	}
	resolvedVersion, err := resolveVersion(*version)
	if err != nil {
		return err
	}

	return RunGoBuild(GoBuildOptions{
		Kind:       kind,
		Target:     target,
		Version:    resolvedVersion,
		OutputPath: *outputPath,
		Package:    *packagePath,
		CGOEnabled: *cgoEnabled,
	})
}

func runArchive(args []string) error {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "", "input file or directory")
	output := fs.String("output", "", "output zip path")
	entryName := fs.String("entry-name", "", "zip entry name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" || *output == "" {
		return errors.New("archive requires --input and --output")
	}
	return ArchiveZip(*input, *output, *entryName)
}

func runChecksums(args []string) error {
	fs := flag.NewFlagSet("checksums", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", "dist", "directory containing zip files")
	output := fs.String("output", "dist/onetiny-checksums.txt", "checksum file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return WriteChecksums(*dir, *output)
}

func runRemove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var paths stringList
	fs.Var(&paths, "path", "path to remove")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return RemovePaths(paths)
}

type stringList []string

func (s *stringList) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}
