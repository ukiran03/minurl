# List available commands
default:
    @just --list

#Variables
TEMPLUI_PATH := `FORMAT='{{.Dir}}'; go list -mod=mod -m -f "$FORMAT" github.com/templui/templui`
PNPM_TAILWIND := `echo $PNPM_HOME` + "/tailwindcss"

# Initializations and Checks
init: initialise checks

# Initialise few things
initialise:
    pnpm add tailwindcss

# Runs tool checks
checks: (_check-dep "wgo") (_check-dep "pnpm") (_check-dep "templ") (_check-dep "fd")
# The helper recipe
_check-dep tool:
    @command -v {{tool}} >/dev/null || (echo "Missing dependency: {{tool}}"; exit 1)

start +args="":
    wgo -file=.go -file=.templ -xfile=_templ.go \
    templ generate ./ui/... :: go run ./cmd/web {{args}}

# Watch Tailwind CSS changes
tailwind:
    #!/usr/bin/env bash
    printf '%s\n' \
      '@source "./**/*.templ";' \
      '@source "./**/*.js";' \
      "@source \"{{TEMPLUI_PATH}}/components/**/*.templ\";" \
      "@source \"{{TEMPLUI_PATH}}/components/**/*.js\";" \
      > ./ui/assets/css/sources.generated.css
     {{PNPM_TAILWIND}} -i ./ui/assets/css/input.css -o ./ui/assets/css/output.css --watch

# Clean up build artifacts
clean:
    rm -rf bin/
    fd -I "_templ.go$" -x rm

# Start development server with hot reload
[parallel]
dev: start tailwind
