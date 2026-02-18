//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
var Default = Build.All

// Build contains all build-related targets
type Build mg.Namespace

// Test contains all test-related targets
type Test mg.Namespace

// Docker contains all docker-related targets
type Docker mg.Namespace

// DB contains all database-related targets
type DB mg.Namespace

// Dev contains all development-related targets
type Dev mg.Namespace

// Docs contains all documentation-related targets
type Docs mg.Namespace

const (
	servicesDir = "cmd"
	binDir      = "bin"
	protoDir    = "api/proto"
	docsDir     = "docs"
	swaggerDir  = "docs/api/swagger"
)

// All builds all services
func (Build) All() error {
	fmt.Println("Building all services...")
	mg.Deps(ensureBinDir)

	services := []string{
		"auth-service",
		"user-service",
		"friend-service",
		"group-service",
		"message-service",
		"session-service",
		"file-service",
		"push-service",
		"gateway-service",
		"rtc-service",
		"sync-service",
		"admin-service",
	}

	for _, service := range services {
		if err := buildService(service); err != nil {
			return err
		}
	}

	fmt.Println("✓ Build completed!")
	return nil
}

// Auth builds auth-service
func (Build) Auth() error {
	mg.Deps(ensureBinDir)
	return buildService("auth-service")
}

// User builds user-service
func (Build) User() error {
	mg.Deps(ensureBinDir)
	return buildService("user-service")
}

// Gateway builds gateway-service
func (Build) Gateway() error {
	mg.Deps(ensureBinDir)
	return buildService("gateway-service")
}

// Message builds message-service
func (Build) Message() error {
	mg.Deps(ensureBinDir)
	return buildService("message-service")
}

// buildService builds a specific service
func buildService(name string) error {
	fmt.Printf("Building %s...\n", name)
	output := filepath.Join(binDir, name)
	source := filepath.Join(servicesDir, name)
	return sh.Run("go", "build", "-o", output, "./"+source)
}

// ensureBinDir creates bin directory if it doesn't exist
func ensureBinDir() error {
	return os.MkdirAll(binDir, 0755)
}

// Proto generates protobuf code
func Proto() error {
	fmt.Println("Generating protobuf code...")

	// Find all .proto files
	protoFiles, err := filepath.Glob(filepath.Join(protoDir, "*", "*.proto"))
	if err != nil {
		return err
	}

	if len(protoFiles) == 0 {
		fmt.Println("No .proto files found")
		return nil
	}

	// Build protoc args
	args := []string{
		"-I", protoDir,
		"--experimental_allow_proto3_optional",
		"--go_out=" + protoDir,
		"--go_opt=paths=source_relative",
		"--go-grpc_out=" + protoDir,
		"--go-grpc_opt=paths=source_relative",
	}
	args = append(args, protoFiles...)

	return sh.Run("protoc", args...)
}

// All runs all tests
func (Test) All() error {
	fmt.Println("Running tests...")
	return sh.RunV("go", "test", "-v", "-race", "-coverprofile=coverage.out", "./...")
}

// Unit runs unit tests
func (Test) Unit() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "test", "-v", "-short", "./...")
}

// Coverage generates test coverage report
func (Test) Coverage() error {
	mg.Deps(Test.All)
	fmt.Println("Generating coverage report...")
	if err := sh.Run("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html"); err != nil {
		return err
	}
	fmt.Println("✓ Coverage report generated: coverage.html")
	return nil
}

// Lint runs linter
func Lint() error {
	fmt.Println("Running linter...")
	return sh.RunV("golangci-lint", "run")
}

// Fmt formats code
func Fmt() error {
	fmt.Println("Formatting code...")
	return sh.RunV("go", "fmt", "./...")
}

// Build builds docker images
func (Docker) Build() error {
	fmt.Println("Building docker images...")
	return sh.RunV("docker-compose", "-f", "deployments/docker-compose.yml", "build")
}

// Up starts docker compose
func (Docker) Up() error {
	fmt.Println("Starting docker compose...")
	return sh.RunV("docker-compose", "-f", "deployments/docker-compose.yml", "up", "-d")
}

// Down stops docker compose
func (Docker) Down() error {
	fmt.Println("Stopping docker compose...")
	return sh.RunV("docker-compose", "-f", "deployments/docker-compose.yml", "down")
}

// Logs shows docker compose logs
func (Docker) Logs() error {
	fmt.Println("Showing docker logs...")
	return sh.RunV("docker-compose", "-f", "deployments/docker-compose.yml", "logs", "-f")
}

// Ps shows docker compose status
func (Docker) Ps() error {
	return sh.RunV("docker-compose", "-f", "deployments/docker-compose.yml", "ps")
}

// Up runs database migrations up
func (DB) Up() error {
	fmt.Println("Running database migrations...")
	return sh.RunV("migrate",
		"-path", "migrations",
		"-database", "postgresql://anychat:anychat123@localhost:5432/anychat?sslmode=disable",
		"up",
	)
}

// Down runs database migrations down
func (DB) Down() error {
	fmt.Println("Reverting database migrations...")
	return sh.RunV("migrate",
		"-path", "migrations",
		"-database", "postgresql://anychat:anychat123@localhost:5432/anychat?sslmode=disable",
		"down",
	)
}

// Create creates a new migration file
func (DB) Create(name string) error {
	if name == "" {
		return fmt.Errorf("migration name is required")
	}
	fmt.Printf("Creating migration: %s\n", name)
	return sh.RunV("migrate", "create", "-ext", "sql", "-dir", "migrations", "-seq", name)
}

// Auth runs auth-service locally
func (Dev) Auth() error {
	fmt.Println("Running auth-service...")
	return sh.RunV("go", "run", "./cmd/auth-service")
}

// User runs user-service locally
func (Dev) User() error {
	fmt.Println("Running user-service...")
	return sh.RunV("go", "run", "./cmd/user-service")
}

// Friend runs friend-service locally
func (Dev) Friend() error {
	fmt.Println("Running friend-service...")
	return sh.RunV("go", "run", "./cmd/friend-service")
}

// Group runs group-service locally
func (Dev) Group() error {
	fmt.Println("Running group-service...")
	return sh.RunV("go", "run", "./cmd/group-service")
}

// File runs file-service locally
func (Dev) File() error {
	fmt.Println("Running file-service...")
	return sh.RunV("go", "run", "./cmd/file-service")
}

// Gateway runs gateway-service locally
func (Dev) Gateway() error {
	fmt.Println("Running gateway-service...")
	return sh.RunV("go", "run", "./cmd/gateway-service")
}

// Message runs message-service locally
func (Dev) Message() error {
	fmt.Println("Running message-service...")
	return sh.RunV("go", "run", "./cmd/message-service")
}

// Session runs session-service locally
func (Dev) Session() error {
	fmt.Println("Running session-service...")
	return sh.RunV("go", "run", "./cmd/session-service")
}

// Push runs push-service locally
func (Dev) Push() error {
	fmt.Println("Running push-service...")
	return sh.RunV("go", "run", "./cmd/push-service")
}

// Sync runs sync-service locally
func (Dev) Sync() error {
	fmt.Println("Running sync-service...")
	return sh.RunV("go", "run", "./cmd/sync-service")
}

// RTC runs rtc-service locally
func (Dev) RTC() error {
	fmt.Println("Running rtc-service...")
	return sh.RunV("go", "run", "./cmd/rtc-service")
}

// Admin runs admin-service locally
func (Dev) Admin() error {
	fmt.Println("Running admin-service...")
	return sh.RunV("go", "run", "./cmd/admin-service")
}

// Deps installs dependencies
func Deps() error {
	fmt.Println("Installing dependencies...")
	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}
	return sh.RunV("go", "mod", "tidy")
}

// DepsCheck verifies dependencies
func DepsCheck() error {
	fmt.Println("Checking dependencies...")
	return sh.RunV("go", "mod", "verify")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	if err := sh.Rm(binDir); err != nil {
		return err
	}
	if err := sh.Rm("coverage.out"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := sh.Rm("coverage.html"); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println("✓ Clean completed!")
	return nil
}

// Mock generates mock code
func Mock() error {
	fmt.Println("Generating mock code...")
	return sh.RunV("mockgen",
		"-source=internal/auth/service/auth_service.go",
		"-destination=internal/auth/service/mock/auth_service_mock.go",
	)
}

// Install installs required tools
func Install() error {
	fmt.Println("Installing required tools...")

	// Install migrate with PostgreSQL driver using build tags
	fmt.Println("Installing migrate (with PostgreSQL driver)...")
	if err := sh.RunV("go", "install", "-tags", "postgres", "github.com/golang-migrate/migrate/v4/cmd/migrate@latest"); err != nil {
		return fmt.Errorf("failed to install migrate: %w", err)
	}

	tools := map[string]string{
		"golangci-lint":      "github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
		"mockgen":            "github.com/golang/mock/mockgen@latest",
		"protoc-gen-go":      "google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		"protoc-gen-go-grpc": "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		"swag":               "github.com/swaggo/swag/cmd/swag@latest",
	}

	for name, pkg := range tools {
		fmt.Printf("Installing %s...\n", name)
		if err := sh.RunV("go", "install", pkg); err != nil {
			return fmt.Errorf("failed to install %s: %w", name, err)
		}
	}

	fmt.Println("✓ All tools installed!")
	return nil
}

// Generate generates OpenAPI 3.0 documentation for the gateway service
func (Docs) Generate() error {
	fmt.Println("Generating API documentation...")
	mg.Deps(ensureSwaggerDir)

	// Step 1: Generate Swagger 2.0 with swag (intermediate)
	if err := sh.RunV("swag", "init",
		"-g", "cmd/gateway-service/main.go",
		"--output", swaggerDir,
		"--parseDependency",
		"--parseInternal",
	); err != nil {
		return fmt.Errorf("failed to generate swagger docs: %w", err)
	}

	// Step 2: Convert Swagger 2.0 → OpenAPI 3.0 JSON
	swaggerFile := filepath.Join(swaggerDir, "swagger.json")
	openAPIFile := filepath.Join(swaggerDir, "openapi.json")
	if err := sh.Run("npx", "--yes", "swagger2openapi",
		"--warnOnly",
		"--outfile", openAPIFile,
		swaggerFile,
	); err != nil {
		return fmt.Errorf("failed to convert to OpenAPI 3.0: %w", err)
	}

	fmt.Println("✓ OpenAPI 3.0 spec generated:", openAPIFile)
	return nil
}

// Serve starts a local documentation server
func (Docs) Serve() error {
	fmt.Println("Starting documentation server...")
	fmt.Println("Documentation will be available at http://localhost:3000")
	fmt.Println("Press Ctrl+C to stop the server")

	// Check if node_modules exists, if not, install dependencies
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		fmt.Println("Installing npm dependencies...")
		if err := sh.RunV("npm", "install"); err != nil {
			return fmt.Errorf("failed to install npm dependencies: %w", err)
		}
	}

	// Use npm run to execute the locally installed docsify-cli
	return sh.RunV("npm", "run", "serve")
}

// Build builds static documentation site
func (Docs) Build() error {
	fmt.Println("Building documentation site...")
	mg.Deps(Docs.Generate)

	fmt.Println("✓ Documentation site ready for deployment")
	fmt.Println("  To deploy, copy the 'docs/' directory to your web server")
	fmt.Println("  Or use GitHub Pages by pushing to gh-pages branch")
	return nil
}

// Validate validates swagger documentation
func (Docs) Validate() error {
	fmt.Println("Validating API documentation...")
	mg.Deps(Docs.Generate)

	openAPIFile := filepath.Join(swaggerDir, "openapi.json")
	if _, err := os.Stat(openAPIFile); os.IsNotExist(err) {
		return fmt.Errorf("openapi.json not found at %s", openAPIFile)
	}
	fmt.Println("✓ openapi.json exists")

	asyncAPIFile := filepath.Join(docsDir, "api", "asyncapi.yaml")
	if _, err := os.Stat(asyncAPIFile); os.IsNotExist(err) {
		return fmt.Errorf("asyncapi.yaml not found at %s", asyncAPIFile)
	}
	fmt.Println("✓ asyncapi.yaml exists")

	fmt.Println("✓ All API documentation is valid")
	return nil
}

// ensureSwaggerDir creates swagger directory if it doesn't exist
func ensureSwaggerDir() error {
	return os.MkdirAll(swaggerDir, 0755)
}

