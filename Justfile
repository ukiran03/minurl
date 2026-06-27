# List available recipes
default:
    @just --list

set dotenv-load := true
set dotenv-filename := ".envrc"

# Tidy module dependencies
tidy:
    @echo '> Tidying module dependencies...'
    go mod tidy
    @echo '> Verifying module dependencies...'
    go mod verify
    @echo '> Formatting .go files...'
    go fmt ./...

# Run quality control checks
audit:
    @echo '> Checking module dependencies...'
    go mod tidy -diff
    go mod verify
    @echo '> Vetting code...'
    go vet ./...
    staticcheck ./...
    # @echo '> Running tests...'
    # go test -race -vet=off ./...

# Comprehensive golangci-lint
ci-lint:
    golangci-lint run ./...

# Automatic refactors and formatting
ci-fix:
    golangci-lint run --fix ./...
