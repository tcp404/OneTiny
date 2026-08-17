set dotenv-load := true
set shell := ["bash", "-euo", "pipefail", "-c"]

default_version := `go run ./tools/buildtool current-version`
version := env("VERSION", default_version)
target := env("TARGET", "")

alias g := build-gui
alias c := dev-clean
alias gui := build-gui
alias cli := build-cli

[default]
[doc("List available recipes")]
[group("Main")]
help:
    @just --list --no-aliases



[doc("Build GUI binary and refresh build/bin")]
[group("Build")]
build-gui target=target:
    task build:gui TARGET="{{ target }}" VERSION="{{ version }}"
    go run ./tools/buildtool copy-file --src "$(go run ./tools/buildtool metadata --target "{{ target }}" --kind gui --field binary-path)" --dst "build/bin/$(go run ./tools/buildtool metadata --target "{{ target }}" --kind gui --field binary-name)"

[doc("Build CLI binary and refresh build/bin")]
[group("Build")]
build-cli target=target:
    task build:cli TARGET="{{ target }}" VERSION="{{ version }}"
    go run ./tools/buildtool copy-file --src "$(go run ./tools/buildtool metadata --target "{{ target }}" --kind cli --field binary-path)" --dst "build/bin/$(go run ./tools/buildtool metadata --target "{{ target }}" --kind cli --field binary-name)"



[doc("Produce the installable CLI artifact for TARGET")]
[group("Build")]
package-cli target=target:
    task package:cli TARGET="{{ target }}" VERSION="{{ version }}"

[doc("Produce the installable GUI artifact for TARGET")]
[group("Build")]
package-gui target=target:
    task package:gui TARGET="{{ target }}" VERSION="{{ version }}"



[doc("Build and archive CLI release artifact")]
[group("Build")]
dist-cli target=target:
    task dist:cli TARGET="{{ target }}" VERSION="{{ version }}"

[doc("Build and archive GUI release artifact")]
[group("Build")]
dist-gui target=target:
    task dist:gui TARGET="{{ target }}" VERSION="{{ version }}"

[doc("Generate release checksums")]
[group("Build")]
dist-checksums:
    task dist:checksums



[doc("Build release GUI artifact for TARGET and update checksums")]
[group("Build")]
release target=target:
    task release TARGET="{{ target }}" VERSION="{{ version }}"



[doc("Remove build and dist artifacts")]
[group("Dev")]
dev-clean:
    task dev:clean

[doc("Show computed build settings")]
[group("Dev")]
dev-info target=target:
    task dev:info TARGET="{{ target }}" VERSION="{{ version }}"



[doc("Format Go source files")]
[group("Quality")]
go-format:
    task go:format

[doc("Apply Go source modernization fixes")]
[group("Quality")]
go-fix:
    task go:fix

[doc("Run Go lint baseline")]
[group("Quality")]
go-lint:
    task go:lint

[doc("Run Go tests")]
[group("Quality")]
go-test:
    task go:test



[doc("Run baseline checks")]
[group("Quality")]
check:
    task check

[doc("Run required checks before commit")]
[group("Quality")]
check-precommit:
    task check:precommit

[doc("Run required checks before push")]
[group("Quality")]
check-prepush:
    task check:prepush



[doc("Install repository Git hooks")]
[group("Quality")]
hooks-install:
    task hooks:install
