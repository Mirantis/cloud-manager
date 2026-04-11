.PHONY: build build-frontend build-backend build-cli dev dev-stop dev-logs dev-compose dev-compose-stop dev-cli dev-frontend dev-backend test fmt lint install check install-linter tidy deps preview clean-build ci pre-commit build-prod build-release release help deploy-user remove-user create-role remove-role deploy-stackset update-stackset status-stackset remove-stackset delete-stackset cli-status check-aws-config unset-variables unset-variables-exec deploy install-k0s k0s-deploy install-ingress-nginx validate-prod-env podman-build podman-build-ghcr podman-build-multiarch podman-build-multiarch-push podman-push-ghcr lint-containerfile podman-run debug-probe-cloudmanager diagnose-k8s logs

# Remote kubectl for `make deploy` (non-interactive SSH often lacks /usr/local/bin and /snap/bin)
KUBECTL ?= kubectl
# PVC: default hostpath for single-node k0s / no StorageClass. Use K8S_STORAGE=dynamic on EKS/GKE (default StorageClass).
K8S_STORAGE ?= hostpath
# Env file → remote k8s/.env → Secret app-secrets (AWS + app creds). Path relative to this Makefile unless absolute.
MAKEFILE_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
ENV_FILE ?= .env.prod
DEPLOY_ENV_ABS := $(if $(filter /%,$(ENV_FILE)),$(ENV_FILE),$(MAKEFILE_DIR)/$(ENV_FILE))

# ingress-nginx bare-metal (NodePort). Override tag to pin a release.
INGRESS_NGINX_TAG ?= v1.11.2
INGRESS_BAREMETAL_URL = https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-$(INGRESS_NGINX_TAG)/deploy/static/provider/baremetal/deploy.yaml
# Set INGRESS_HOSTNETWORK=1 so the controller binds :80/:443 on the node (typical single-node k0s + public DNS to that IP).
INGRESS_HOSTNETWORK ?= 0

# Default image for Kubernetes / GHCR (override for forks, e.g. GHCR_IMAGE=ghcr.io/yourorg/cloud-manager:latest)
GHCR_IMAGE ?= ghcr.io/mirantis/cloud-manager:latest
# Initial lines shown before follow (make logs HOST=... TAIL=200)
TAIL ?= 100

# Default target
help:
	@echo "Cloud Manager - Unified Build System"
	@echo ""
	@echo "🏗️  Build Targets:"
	@echo "  build-frontend   - Build Vue.js frontend"
	@echo "  build-backend    - Build Go backend server"
	@echo "  build-cli        - Build Go CLI application"
	@echo "  build-go         - Build both backend and CLI"
	@echo "  build            - Build everything (Podman image)"
	@echo "  build-prod       - Production build with optimizations"
	@echo "  build-release    - Multi-platform release binaries"
	@echo ""
	@echo "🚀 Development:"
	@echo "  dev              - Build frontend + run Go server locally (no containers; needs .env.prod)"
	@echo "  dev-stop         - Remove legacy local k8s cloud-manager objects (optional cleanup)"
	@echo "  dev-logs         - Stream logs from legacy local k8s deployment (kubectl)"
	@echo "  dev-compose      - Run locally with podman compose"
	@echo "  dev-compose-stop - Stop podman compose"
	@echo "  dev-cli          - Run CLI in development mode"
	@echo "  dev-frontend     - Run frontend development server"
	@echo "  dev-backend      - Run backend in development mode"
	@echo ""
	@echo "🧪 Testing & Quality:"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  fmt              - Format all code"
	@echo "  lint             - Lint all code"
	@echo "  check            - Run all checks (fmt + lint + test)"
	@echo "  ci               - CI pipeline (install + check + build)"
	@echo ""
	@echo "🐳 Podman / container image:"
	@echo "  podman-build      - Build OCI image (localhost/cloud-manager:latest)"
	@echo "  podman-build-ghcr - Tag image as GHCR_IMAGE ($(GHCR_IMAGE))"
	@echo "  podman-build-multiarch - Multi-arch image (amd64, arm64)"
	@echo "  podman-build-multiarch-push - Multi-arch build and push to GHCR_IMAGE"
	@echo "  podman-push-ghcr  - Push GHCR_IMAGE (runs podman-build-ghcr first)"
	@echo "  podman-run        - Run container locally with .env.prod"
	@echo "  lint-containerfile - Lint Dockerfile with hadolint (podman if hadolint missing)"
	@echo ""
	@echo "☁️  AWS IAM Management:"
	@echo "  deploy-user      - Deploy IAM user and resources"
	@echo "  remove-user      - Remove IAM user and resources"
	@echo "  create-role      - Create IAM role for cross-account access"
	@echo "  remove-role      - Remove IAM role and resources"
	@echo "  deploy-stackset  - Deploy StackSet for organization setup"
	@echo "  update-stackset  - Update existing StackSet with new template"
	@echo "  status-stackset  - Show StackSet deployment status"
	@echo "  remove-stackset  - Remove StackSet and all instances"
	@echo "  delete-stackset  - Alias for remove-stackset"
	@echo "  cli-status       - Show current deployment status"
	@echo "  deploy HOST=... [DOMAIN=...] [USER=...] [K8S_STORAGE=dynamic] - Deploy to k8s (default hostpath PV; use dynamic on cloud with StorageClass)"
	@echo "                               Example: make deploy HOST=172.19.112.251 USER=ubuntu DOMAIN=iammanager.it.eu-cloud.mirantis.net"
	@echo "                               If remote says kubectl not found: KUBECTL=/snap/bin/kubectl (or install kubectl on the SSH host)"
	@echo "                               Requires .env.prod next to this Makefile (ENV_FILE=path overrides)"
	@echo "                               Ingress is HTTP-only (no cert-manager in deploy)"
	@echo "  install-k0s HOST=ssh-target [USER=ubuntu] [K0S_VERSION=v1.29.2+k0s.0] - Install k0s + kubectl on remote (sudo -n; alias: k0s-deploy)"
	@echo "  install-ingress-nginx HOST=ssh-target [USER=ubuntu] [INGRESS_HOSTNETWORK=1] - Install ingress-nginx (bare metal); use HOSTNETWORK=1 for :443 on node IP"
	@echo "  debug-probe-cloudmanager [PROBE_URL=...] - HTTPS probe; appends NDJSON to DEBUG_LOG_PATH (default nanoclaw .cursor/debug-3cd321.log)"
	@echo "  diagnose-k8s HOST=... [USER=] - SSH + kubectl: PVC, pods, endpoints, ingress, app logs"
	@echo "  logs HOST=... [USER=] [TAIL=100] - Stream app logs from remote kubectl (-f; same host as deploy)"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  clean            - Clean everything"
	@echo "  clean-build      - Clean build artifacts only"
	@echo ""
	@echo "🔧 Setup & Configuration:"
	@echo "  check-aws-config - Verify AWS credentials and configuration"
	@echo "  unset-variables  - Show command to unset AWS credential environment variables"
	@echo "  validate-prod-env - Validate ENV_FILE (.env.prod by default, same path rules as deploy)"
	@echo ""
	@echo "☁️  Azure AD (Optional):"
	@echo "  Azure support requires AZURE_TENANT_ID, AZURE_CLIENT_ID, and"
	@echo "  AZURE_CLIENT_SECRET in .env.prod. See README.md for setup instructions."
	@echo ""
	@echo "🚀 CI/CD & Quality:"
	@echo "  ci               - Run all CI checks locally (includes podman-build)"
	@echo "  test-coverage    - Generate test coverage report"
	@echo "  validate-workflows - Validate GitHub Actions workflows"
	@echo "  pre-commit       - Run all pre-commit checks"
	@echo "  release-build    - Create release build artifacts"
	@echo ""
	@echo "📖 CLI Usage Examples:"
	@echo "  cli-help         - Show CLI help"

# ============================================================================
# BUILD TARGETS
# ============================================================================

# Frontend build
build-frontend:
	@echo "📦 Building frontend..."
	cd frontend && npm install && npm run build

# Backend build
build-backend:
	@echo "🔧 Building backend server..."
	mkdir -p bin
	go build -o bin/cloud-manager ./cmd/server

# CLI build
build-cli:
	@echo "⚙️  Building CLI..."
	@if command -v go >/dev/null 2>&1; then \
		mkdir -p bin; \
		go mod tidy; \
		go mod download; \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/iam-manager ./cmd/iam-manager; \
	else \
		echo "❌ Error: Go is not installed. Please install Go 1.21+"; \
		exit 1; \
	fi

# Build Go projects (backend + CLI)
build-go: build-backend build-cli

# Build everything (OCI image via Podman)
build: build-frontend podman-build

# Production build with optimizations
build-prod:
	@echo "🚀 Building for production..."
	$(MAKE) build-frontend
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/cloud-manager-prod ./cmd/server
	$(MAKE) build-cli

# Multi-platform release binaries
build-release:
	@echo "📦 Building release binaries for multiple platforms..."
	@mkdir -p bin/release
	$(MAKE) build-frontend
	# Backend server
	GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/cloud-manager-server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o bin/release/cloud-manager-server-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/cloud-manager-server-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o bin/release/cloud-manager-server-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/cloud-manager-server-windows-amd64.exe ./cmd/server
	# CLI binary
	GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/iam-manager-linux-amd64 ./cmd/iam-manager
	GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o bin/release/iam-manager-linux-arm64 ./cmd/iam-manager
	GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/iam-manager-darwin-amd64 ./cmd/iam-manager
	GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o bin/release/iam-manager-darwin-arm64 ./cmd/iam-manager
	GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/release/iam-manager-windows-amd64.exe ./cmd/iam-manager
	@echo "✅ Release binaries built in bin/release/ directory"

# ============================================================================
# DEVELOPMENT TARGETS
# ============================================================================

# Local dev: production frontend build + Go server (cwd = repo root; serves ./frontend/dist). No Kubernetes or Podman.
dev:
	@if [ ! -f "$(DEPLOY_ENV_ABS)" ]; then \
		echo "❌ Missing $(ENV_FILE) at $(DEPLOY_ENV_ABS). Copy from .env.example and edit."; \
		exit 1; \
	fi
	@echo "🚀 Local development (no containers)..."
	@echo "🧹 Unsetting AWS env vars so values from $(ENV_FILE) apply (godotenv does not override the shell)."
	@$(MAKE) build-frontend
	@echo "🔐 Ensuring admin password in $(ENV_FILE)..."
	@cd "$(MAKEFILE_DIR)" && ENVF="$(DEPLOY_ENV_ABS)" && \
		if ! grep -q "^ADMIN_PASSWORD=" "$$ENVF" 2>/dev/null || [ -z "$$(grep '^ADMIN_PASSWORD=' "$$ENVF" | cut -d'=' -f2)" ]; then \
			ADMIN_PASSWORD=$$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16); \
			echo "✅ Generated random admin password (stored in $$ENVF)"; \
			if grep -q "^ADMIN_PASSWORD=" "$$ENVF" 2>/dev/null; then \
				sed -i.bak "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$$ADMIN_PASSWORD|" "$$ENVF"; \
			else \
				echo "ADMIN_PASSWORD=$$ADMIN_PASSWORD" >> "$$ENVF"; \
			fi; \
			if ! grep -q "^ADMIN_USERNAME=" "$$ENVF" 2>/dev/null; then \
				echo "ADMIN_USERNAME=admin" >> "$$ENVF"; \
			fi; \
		else \
			echo "✅ Using existing admin password from $(ENV_FILE)"; \
		fi
	@ADMIN_USERNAME=$$(grep '^ADMIN_USERNAME=' "$(DEPLOY_ENV_ABS)" 2>/dev/null | cut -d'=' -f2 || echo "admin"); \
		echo "🔐 Admin username: $$ADMIN_USERNAME (password in $(ENV_FILE))"; \
		echo "💡 Open http://localhost:$${PORT:-8080}  (Ctrl+C to stop)"; \
		echo ""
	@cd "$(MAKEFILE_DIR)" && \
		unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_REGION AWS_PROFILE AWS_DEFAULT_REGION AWS_SSO_REGION && \
		set -a && . "$(DEPLOY_ENV_ABS)" && set +a && \
		(command -v air >/dev/null 2>&1 && exec air || exec go run ./cmd/server)

# Run locally with podman compose (uses .env.prod)
dev-compose:
	@if [ ! -f .env.prod ]; then \
		echo "❌ Error: .env.prod file not found. Create it with your environment variables."; \
		exit 1; \
	fi
	@echo "📦 Starting with podman compose..."
	podman compose up --build

# Stop podman compose
dev-compose-stop:
	@echo "🛑 Stopping podman compose..."
	podman compose down

# Optional: tear down legacy in-cluster dev from older Makefile (not used by `make dev` today)
dev-stop:
	@echo "🛑 Removing legacy cloud-manager objects from local Kubernetes (if any)..."
	@kubectl delete deployment cloud-manager -n cloud-manager --ignore-not-found
	@kubectl delete secret app-secrets -n cloud-manager --ignore-not-found
	@echo "✅ Done (ignored if cluster/namespace absent)"

# Legacy: follow logs from in-cluster deployment (use terminal output for `make dev`)
dev-logs:
	@kubectl logs -f -n cloud-manager -l app.kubernetes.io/name=cloud-manager

# Frontend development server (standalone)
dev-frontend:
	@echo "🎨 Starting frontend development server..."
	@if [ -f .env.prod ]; then \
		set -a && . ./.env.prod && set +a && cd frontend && npm run dev; \
	else \
		cd frontend && npm run dev; \
	fi

# Backend development server (standalone)
dev-backend:
	@echo "🔧 Starting backend development server..."
	@if [ -f .env.prod ]; then \
		set -a && . ./.env.prod && set +a && (which air > /dev/null && air || go run ./cmd/server); \
	else \
		which air > /dev/null && air || go run ./cmd/server; \
	fi

# CLI development
dev-cli:
	@if command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager; \
	else \
		echo "❌ Error: Go is not installed. Cannot run dev-cli target."; \
		exit 1; \
	fi

# Preview production frontend build
preview:
	@echo "👀 Previewing frontend production build..."
	cd frontend && npm run preview

# ============================================================================
# TESTING & QUALITY TARGETS
# ============================================================================

# Run all tests
test:
	@echo "🧪 Running tests..."
	go test $(shell go list ./... | grep -v cmd/iam-manager)
	cd frontend && npm test

# Run tests with coverage
test-coverage-basic:
	@echo "🧪 Running tests with basic coverage..."
	go test -cover $(shell go list ./... | grep -v cmd/iam-manager)
	cd frontend && npm run test -- --coverage

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running verbose tests..."
	go test -v $(shell go list ./... | grep -v cmd/iam-manager)

# Format all code
fmt:
	@echo "✨ Formatting code..."
	@if command -v go >/dev/null 2>&1; then \
		echo "  📝 Formatting Go code..."; \
		go fmt ./...; \
	else \
		echo "⚠️  Go not installed, skipping Go formatting"; \
	fi
	@echo "  📝 Formatting frontend code..."
	@cd frontend && (which prettier > /dev/null && npm run format || echo "    Prettier not configured")

# Lint all code
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "  🔍 Running golangci-lint on Go code..."; \
		golangci-lint run ./cmd/server/... ./internal/...; \
	else \
		echo "  ⚠️  golangci-lint not found, using basic go vet"; \
		go vet ./cmd/server/... ./internal/...; \
	fi
	@echo "  🔍 Linting frontend code..."
	@cd frontend && (which eslint > /dev/null && npm run lint || echo "    ESLint not configured")

# Run all checks (pre-commit)
check: fmt lint test
	@echo "✅ All checks passed!"

# CI pipeline
ci-basic: tidy deps check build
	@echo "✅ Basic CI pipeline completed successfully!"

# Pre-commit checks (lighter than CI)
pre-commit: fmt lint test

# ============================================================================
# DEPENDENCY MANAGEMENT
# ============================================================================

# Tidy Go dependencies
tidy:
	@echo "🧹 Tidying Go dependencies..."
	go mod tidy

# Download Go dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	cd frontend && npm install

# Install frontend dependencies (production)
install-frontend:
	@echo "📦 Installing frontend dependencies..."
	cd frontend && npm ci

# ============================================================================
# DOCKER OPERATIONS
# ============================================================================

# ============================================================================
# CLI USAGE EXAMPLES
# ============================================================================

cli-help: build-cli
	@echo "📖 Showing CLI help..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager --help; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager --help; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# ============================================================================
# SETUP & CONFIGURATION TARGETS
# ============================================================================

# Check AWS credentials and configuration
check-aws-config:
	@echo "🔍 Checking AWS configuration..."
	@echo ""
	@echo "📋 AWS CLI Configuration:"
	@aws configure list || echo "❌ AWS CLI not configured"
	@echo ""
	@echo "🌍 Environment Variables:"
	@echo "  AWS_REGION: $${AWS_REGION:-<not set>}"
	@echo "  AWS_ACCESS_KEY_ID: $${AWS_ACCESS_KEY_ID:-<not set>}"
	@echo "  AWS_SECRET_ACCESS_KEY: $${AWS_SECRET_ACCESS_KEY:+<set>}$${AWS_SECRET_ACCESS_KEY:-<not set>}"
	@echo ""
	@echo "💡 Quick Setup Guide:"
	@echo "   1. Configure AWS CLI: aws configure"
	@echo "   2. Or set environment variables:"
	@echo "      export AWS_ACCESS_KEY_ID=your_key"
	@echo "      export AWS_SECRET_ACCESS_KEY=your_secret" 
	@echo "      export AWS_REGION=us-east-1"
	@echo "   3. Or copy .env.example to .env and set your credentials"
	@echo ""
	@echo "🧪 Testing AWS connectivity..."
	@if command -v aws >/dev/null 2>&1; then \
		aws sts get-caller-identity 2>/dev/null || echo "❌ AWS connectivity test failed - credentials may be invalid"; \
	else \
		echo "⚠️  AWS CLI not installed - install it for easier credential management"; \
	fi

# Unset AWS credential environment variables
unset-variables:
	@echo "🧹 Unsetting AWS credential environment variables..."
	@echo ""
	@echo "Run this command in your shell to unset AWS credentials:"
	@echo ""
	@echo "  unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN"
	@echo ""
	@echo "Or source this command:"
	@echo "  eval \"\$$(make unset-variables-exec)\""
	@echo ""
	@echo "To verify credentials are unset, run:"
	@echo "  env | grep AWS_"

# Internal target that outputs unset commands (for use with eval)
unset-variables-exec:
	@echo "unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN; echo '✅ AWS credentials unset'"

# ============================================================================
# AWS IAM MANAGEMENT TARGETS
# ============================================================================

# Deploy IAM user and resources
deploy-user: build-cli
	@echo "🚀 Deploying IAM user and resources..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager deploy; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager deploy; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi
	@if [ -f /tmp/iam-manager.env ]; then \
		cp /tmp/iam-manager.env .env.prod; \
		chmod 600 .env.prod; \
		echo "✅ Credentials saved to .env.prod"; \
	else \
		echo "⚠️  Warning: /tmp/iam-manager.env not found, .env.prod not created"; \
	fi
	@echo "🔐 Attaching SSO policy to IAM user..."
	@USER_NAME=$${IAM_USER_NAME:-iam-manager}; \
	POLICY_FILE="cloudformation/iam-manager-user-sso-policy.json"; \
	if [ -f "$$POLICY_FILE" ]; then \
		if aws iam put-user-policy \
			--user-name "$$USER_NAME" \
			--policy-name SSOIdentityCenterManagement \
			--policy-document "file://$$POLICY_FILE" 2>/dev/null; then \
			echo "✅ SSO policy attached successfully"; \
		else \
			echo "⚠️  Warning: Failed to attach SSO policy. You may need to attach it manually:"; \
			echo "   aws iam put-user-policy --user-name $$USER_NAME --policy-name SSOIdentityCenterManagement --policy-document file://$$POLICY_FILE"; \
		fi \
	else \
		echo "⚠️  Warning: SSO policy file not found at $$POLICY_FILE"; \
	fi

# Remove IAM user and resources
remove-user: build-cli
	@echo "🗑️  Removing IAM user and resources..."
	@USER_NAME=$${IAM_USER_NAME:-iam-manager}; \
	echo "🗑️  Removing SSO policy from IAM user..."; \
	if aws iam delete-user-policy \
		--user-name "$$USER_NAME" \
		--policy-name SSOIdentityCenterManagement 2>/dev/null; then \
		echo "✅ SSO policy removed"; \
	else \
		echo "ℹ️  SSO policy not found or already removed"; \
	fi
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager remove; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager remove; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi
	@if [ -f .env.prod ]; then \
		rm .env.prod; \
		echo "✅ Removed .env.prod"; \
	fi

# Create IAM role for cross-account access
create-role: build-cli
	@echo "🔐 Creating IAM role for cross-account access..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager create-role; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager create-role; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Remove IAM role and resources
remove-role: build-cli
	@echo "🗑️  Removing IAM role and resources..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager remove-role; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager remove-role; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Deploy StackSet for organization setup
deploy-stackset: build-cli
	@echo "📦 Deploying StackSet for organization setup..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager stackset-deploy; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager stackset-deploy; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Update existing StackSet with new template
update-stackset:
	@echo "🔄 Updating StackSet with new template..."
	@if [ ! -f cloudformation/iam-manager-role.yaml ]; then \
		echo "❌ Error: cloudformation/iam-manager-role.yaml not found"; \
		exit 1; \
	fi
	@echo "📋 Template: cloudformation/iam-manager-role.yaml"
	@echo "🔍 Checking StackSet exists..."
	@aws cloudformation describe-stack-set --stack-set-name IAMManagerRoleStackSet >/dev/null 2>&1 || \
		(echo "❌ Error: StackSet 'IAMManagerRoleStackSet' not found. Run 'make deploy-stackset' first."; exit 1)
	@echo "🔍 Getting current account ID..."
	@MASTER_ACCOUNT_ID=$$(aws sts get-caller-identity --query Account --output text 2>/dev/null); \
	if [ -z "$$MASTER_ACCOUNT_ID" ]; then \
		echo "❌ Error: Failed to get current account ID. Check AWS credentials."; \
		exit 1; \
	fi; \
	MASTER_USER_NAME=$${IAM_USER_NAME:-iam-manager}; \
	ROLE_NAME=$${IAM_ORG_ROLE_NAME:-IAMManagerCrossAccountRole}; \
	echo "📊 StackSet Parameters:"; \
	echo "  Master Account ID: $$MASTER_ACCOUNT_ID"; \
	echo "  Master User Name: $$MASTER_USER_NAME"; \
	echo "  Role Name: $$ROLE_NAME"; \
	echo ""; \
	echo "📤 Updating StackSet in parallel across all accounts..."; \
	aws cloudformation update-stack-set \
		--stack-set-name IAMManagerRoleStackSet \
		--template-body file://cloudformation/iam-manager-role.yaml \
		--capabilities CAPABILITY_NAMED_IAM \
		--parameters \
			ParameterKey=MasterAccountId,ParameterValue=$$MASTER_ACCOUNT_ID \
			ParameterKey=RoleName,ParameterValue=$$ROLE_NAME \
			ParameterKey=MasterUserName,ParameterValue=$$MASTER_USER_NAME \
		--operation-preferences FailureToleranceCount=0,MaxConcurrentPercentage=100,RegionConcurrencyType=PARALLEL \
		--output text > /tmp/stackset-operation-id.txt || \
		(echo "⚠️  No updates to perform or update failed"; exit 0)
	@if [ -s /tmp/stackset-operation-id.txt ]; then \
		OPERATION_ID=$$(cat /tmp/stackset-operation-id.txt); \
		echo "✅ StackSet update initiated"; \
		echo "📊 Operation ID: $$OPERATION_ID"; \
		echo ""; \
		echo "🚀 Updating all accounts in parallel (MaxConcurrentPercentage=100, RegionConcurrencyType=PARALLEL)"; \
		echo "⏳ Waiting for update to complete across all accounts..."; \
		echo "💡 Accounts are being updated concurrently for faster completion"; \
		echo ""; \
		while true; do \
			STATUS=$$(aws cloudformation describe-stack-set-operation \
				--stack-set-name IAMManagerRoleStackSet \
				--operation-id $$OPERATION_ID \
				--query 'StackSetOperation.Status' \
				--output text 2>/dev/null || echo "UNKNOWN"); \
			if [ "$$STATUS" = "SUCCEEDED" ]; then \
				echo "✅ StackSet update completed successfully!"; \
				break; \
			elif [ "$$STATUS" = "FAILED" ] || [ "$$STATUS" = "STOPPED" ]; then \
				echo "❌ StackSet update failed with status: $$STATUS"; \
				echo "🔍 Check details with: make status-stackset"; \
				exit 1; \
			elif [ "$$STATUS" = "RUNNING" ]; then \
				echo "⏳ Update in progress... (Status: $$STATUS)"; \
				sleep 10; \
			else \
				echo "⚠️  Unknown status: $$STATUS"; \
				sleep 10; \
			fi; \
		done; \
		echo ""; \
		echo "🎉 All account stacks updated with new IAM permissions"; \
		echo "📊 Check status with: make status-stackset"; \
	fi
	@rm -f /tmp/stackset-operation-id.txt

# Show StackSet deployment status
status-stackset: build-cli
	@echo "📊 Checking StackSet deployment status..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager stackset-status; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager stackset-status; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Remove StackSet and all instances
remove-stackset: build-cli
	@echo "🗑️  Removing StackSet and all instances..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager stackset-delete; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager stackset-delete; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Delete StackSet (alias for remove-stackset)
delete-stackset: remove-stackset

# Show current deployment status
cli-status: build-cli
	@echo "📋 Showing current deployment status..."
	@if [ -f bin/iam-manager ]; then \
		./bin/iam-manager status; \
	elif command -v go >/dev/null 2>&1; then \
		go run ./cmd/iam-manager status; \
	else \
		echo "❌ Error: Neither binary nor Go found. Run 'make build-cli' first."; \
		exit 1; \
	fi

# Deploy application to specified host using Kubernetes
# Optional: DEBUG_LOG_PATH, PROBE_URL, DEBUG_SESSION_ID, DEBUG_RUN_ID
debug-probe-cloudmanager:
	@python3 "$(CURDIR)/scripts/k8s_debug_probe.py"

# One-shot remote diagnostics for 503 / scheduling (needs same HOST/USER as deploy)
diagnose-k8s:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Usage: make diagnose-k8s HOST=ssh-target [USER=ubuntu]  (same as make deploy)"; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	@echo "🔍 Diagnostics on $(TARGET_HOST) (namespace cloud-manager)..."
	@ssh "$(TARGET_HOST)" 'export PATH="$$PATH:/usr/local/bin:/snap/bin"; \
		command -v $(KUBECTL) >/dev/null 2>&1 || { echo "kubectl not found"; exit 127; }; \
		echo "=== PVC / PV (if Pending: delete pvc + deployment, redeploy; default make deploy uses hostpath PV) ===" && \
		$(KUBECTL) get pvc -n cloud-manager -o wide 2>/dev/null; \
		$(KUBECTL) get pv 2>/dev/null | grep -E "NAME|cloud-manager" || true; \
		$(KUBECTL) describe pvc cloud-manager-data -n cloud-manager 2>/dev/null | tail -25; \
		echo "" && echo "=== pods / svc / endpoints ===" && \
		$(KUBECTL) get pods,svc,endpoints -n cloud-manager -o wide 2>/dev/null && \
		echo "" && echo "=== ingress / app secret ===" && \
		$(KUBECTL) get ingress -n cloud-manager 2>/dev/null; \
		$(KUBECTL) get secret app-secrets -n cloud-manager 2>/dev/null; \
		echo "" && echo "=== app pod describe (first app pod) ===" && \
		POD=$$($(KUBECTL) get pods -n cloud-manager -l app.kubernetes.io/component=app -o jsonpath="{.items[0].metadata.name}" 2>/dev/null); \
		if [ -n "$$POD" ]; then $(KUBECTL) describe pod -n cloud-manager "$$POD" | tail -40; else echo "(no app pod)"; fi; \
		echo "" && echo "=== app logs (last 60 lines) ===" && \
		$(KUBECTL) logs -n cloud-manager -l app.kubernetes.io/component=app --tail=60 2>/dev/null || echo "(no logs)"'

# Stream cloud-manager app logs (SSH host must have kubeconfig for the cluster; same HOST/USER as deploy)
logs:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Usage: make logs HOST=ssh-target [USER=ubuntu] [TAIL=100] [KUBECTL=kubectl]"; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	@echo "📋 Following cloud-manager logs on $(TARGET_HOST) (namespace cloud-manager, Ctrl+C to stop)..."
	@ssh -t "$(TARGET_HOST)" 'export PATH="$$PATH:/usr/local/bin:/snap/bin"; \
		command -v $(KUBECTL) >/dev/null 2>&1 || { echo "kubectl not found"; exit 127; }; \
		exec $(KUBECTL) logs -n cloud-manager -l app.kubernetes.io/component=app -f --tail=$(TAIL)'

deploy:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Error: HOST is required (SSH target). Usage: make deploy HOST=bastion.example.com DOMAIN=app.example.com [USER=ubuntu]"; \
		exit 1; \
	fi
	@if [ ! -f "$(DEPLOY_ENV_ABS)" ]; then \
		echo "❌ Deploy env file missing: $(DEPLOY_ENV_ABS)"; \
		echo "   Create $(ENV_FILE) with AWS and app credentials (see .env.example), then re-run deploy."; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	$(eval APP_DOMAIN := $(if $(DOMAIN),$(DOMAIN),$(HOST)))
	@echo "☸️  Deploying application to $(TARGET_HOST) using Kubernetes..."
	@echo "🔐 Using env file: $(DEPLOY_ENV_ABS)"
	@echo "🌐 Public hostname (Ingress HTTP): $(APP_DOMAIN)"
	@echo "💾 PVC mode: $(K8S_STORAGE) (hostpath = static PV on node /var/lib/cloud-manager-data)"
	@echo "📤 Copying Kubernetes manifests to $(TARGET_HOST)..."
	@ssh "$(TARGET_HOST)" 'mkdir -p "$$HOME/cloud-manager"'
	@scp -r k8s/ "$(TARGET_HOST)":~/cloud-manager/
	@echo "📤 Copying $(ENV_FILE) → remote k8s/.env (Kubernetes secret app-secrets)..."
	@scp "$(DEPLOY_ENV_ABS)" "$(TARGET_HOST)":~/cloud-manager/k8s/.env
	@echo "☸️  Configuring secrets and deploying to Kubernetes..."
	@ssh "$(TARGET_HOST)" 'export PATH="$$PATH:/usr/local/bin:/snap/bin"; \
		command -v $(KUBECTL) >/dev/null 2>&1 || { echo "kubectl not found on remote host (non-interactive PATH). Install kubectl or retry with e.g. KUBECTL=/snap/bin/kubectl"; exit 127; }; \
		cd ~/cloud-manager && \
		echo "🔐 Generating admin password..." && \
		if ! grep -q "^ADMIN_PASSWORD=" k8s/.env 2>/dev/null || [ -z "$$(grep '\''^ADMIN_PASSWORD='\'' k8s/.env | cut -d'\''='\'' -f2)" ]; then \
			ADMIN_PASSWORD=$$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16); \
			echo "✅ Generated random admin password (stored in k8s/.env)"; \
			if grep -q "^ADMIN_PASSWORD=" k8s/.env 2>/dev/null; then \
				sed -i.bak "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$$ADMIN_PASSWORD|" k8s/.env; \
			else \
				echo "ADMIN_PASSWORD=$$ADMIN_PASSWORD" >> k8s/.env; \
			fi; \
			if ! grep -q "^ADMIN_USERNAME=" k8s/.env 2>/dev/null; then \
				echo "ADMIN_USERNAME=admin" >> k8s/.env; \
			fi; \
		else \
			echo "✅ Using existing admin password from k8s/.env"; \
		fi; \
		ADMIN_USERNAME=$$(grep '\''^ADMIN_USERNAME='\'' k8s/.env 2>/dev/null | cut -d'\''='\'' -f2 || echo "admin"); \
		echo "☸️  Creating namespace first..." && \
		$(KUBECTL) apply -f k8s/namespace.yaml && \
		echo "🔐 Creating Kubernetes secrets from environment file..." && \
		$(KUBECTL) create secret generic app-secrets --namespace=cloud-manager \
			--from-env-file=k8s/.env \
			--dry-run=client -o yaml | $(KUBECTL) apply -f - && \
		echo "🔧 Creating ingress (HTTP only, no TLS)..." && \
		sed "s|DOMAIN_PLACEHOLDER|$(APP_DOMAIN)|g" k8s/ingress.yaml | $(KUBECTL) apply -f - && \
		echo "☸️  Applying remaining Kubernetes manifests..." && \
		$(KUBECTL) apply -f k8s/configmap.yaml && \
		if [ "$(K8S_STORAGE)" = "hostpath" ]; then \
			$(KUBECTL) apply -f k8s/pv-hostpath.yaml && \
			$(KUBECTL) apply -f k8s/pvc-hostpath.yaml; \
		else \
			$(KUBECTL) apply -f k8s/pvc.yaml; \
		fi && \
		$(KUBECTL) apply -f k8s/app-deployment.yaml && \
		echo "🔄 Restarting deployment to pull and run the latest image (same tag may have a new digest)..." && \
		$(KUBECTL) rollout restart deployment/cloud-manager -n cloud-manager && \
		$(KUBECTL) rollout status deployment/cloud-manager -n cloud-manager --timeout=180s && \
		$(KUBECTL) apply -f k8s/service.yaml'
	@echo "✅ Application deployed successfully to Kubernetes cluster on $(TARGET_HOST)"
	@echo ""
	@echo "🔐 Admin credentials configured"
	@ssh "$(TARGET_HOST)" 'cd ~/cloud-manager && \
		ADMIN_USERNAME=$$(grep '\''^ADMIN_USERNAME='\'' k8s/.env 2>/dev/null | cut -d'\''='\'' -f2 || echo "admin"); \
		echo "  Username: $$ADMIN_USERNAME"; \
		echo "  Password stored in k8s/.env file (not displayed for security)"'
	@echo ""
	@echo "🌐 External Access Information:"
	@echo "  📍 HTTP (port 80): http://$(APP_DOMAIN)"
	@echo "  📍 Nginx Ingress → cloud-manager service → backend (TLS not configured by deploy)"
	@echo ""
	@echo "🔍 Checking deployment status..."
	@ssh "$(TARGET_HOST)" 'export PATH="$$PATH:/usr/local/bin:/snap/bin"; command -v $(KUBECTL) >/dev/null 2>&1 && $(KUBECTL) get pods -n cloud-manager && $(KUBECTL) get services -n cloud-manager && $(KUBECTL) get ingress -n cloud-manager'
	@echo ""
	@echo "💡 External Access:"
	@echo "   • Use: http://$(APP_DOMAIN)"
	@echo "   • Requires: Nginx Ingress Controller (TLS optional; k8s/cert-manager.yaml not applied by deploy)"
	@echo "   • Authentication is handled by the application (admin username/password)"
	@echo "   • Open port 80 to the ingress (443 only if you add TLS later)"

# Install k0s (single-controller + embedded etcd) and upstream kubectl on a remote host. Requires passwordless sudo (sudo -n).
# Optional K0S_VERSION pins get.k0s.sh (e.g. v1.29.2+k0s.0); kubectl version is derived (same minor) or stable.
install-k0s:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Usage: make install-k0s HOST=host [USER=ubuntu] [K0S_VERSION=v1.29.2+k0s.0]"; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	@echo "📦 Installing k0s + kubectl on $(TARGET_HOST) (single-node controller)..."
	@ssh -o BatchMode=yes "$(TARGET_HOST)" 'set -e; \
		export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"; \
		$(if $(K0S_VERSION),export K0S_VERSION="$(K0S_VERSION)";,) \
		if ! sudo -n true 2>/dev/null; then \
			echo "❌ This host needs non-interactive sudo (configure NOPASSWD for the SSH user, or run install manually)."; \
			exit 1; \
		fi; \
		if [ -f /etc/systemd/system/k0scontroller.service ] || [ -f /lib/systemd/system/k0scontroller.service ] || [ -f /usr/lib/systemd/system/k0scontroller.service ]; then \
			echo "✅ k0s controller unit already present; ensuring service is started..."; \
			sudo k0s start || sudo systemctl start k0scontroller.service || true; \
		else \
			curl -sSLf https://get.k0s.sh | sudo sh; \
			sudo k0s install controller --single; \
			sudo k0s start; \
		fi; \
		echo "⏳ Waiting for node Ready..."; \
		n=0; \
		until sudo k0s kubectl get nodes 2>/dev/null | grep -qE "\sReady\s"; do \
			n=$$((n+1)); \
			if [ "$$n" -gt 90 ]; then echo "❌ Timeout waiting for k0s node"; exit 1; fi; \
			sleep 2; \
		done; \
		sudo k0s kubectl get nodes; \
		mkdir -p "$$HOME/.kube"; \
		sudo k0s kubeconfig admin > "$$HOME/.kube/config"; \
		chmod 600 "$$HOME/.kube/config"; \
		echo "📥 Installing kubectl (kubernetes.io release)..."; \
		ARCH=$$(uname -m); \
		case $$ARCH in x86_64) K_ARCH=amd64;; aarch64|arm64) K_ARCH=arm64;; *) echo "❌ unsupported arch: $$ARCH"; exit 1;; esac; \
		KUBE_VER=$$(echo "$${K0S_VERSION:-}" | cut -d+ -f1); \
		if [ -z "$$KUBE_VER" ]; then \
			KUBE_VER=$$(sudo k0s version 2>/dev/null | tr -d '\r' | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
		fi; \
		if [ -z "$$KUBE_VER" ]; then \
			KUBE_VER=$$(curl -Ls https://dl.k8s.io/release/stable.txt); \
		fi; \
		curl -sSLf -o /tmp/kubectl.bin "https://dl.k8s.io/release/$${KUBE_VER}/bin/linux/$${K_ARCH}/kubectl"; \
		sudo install -m 0755 /tmp/kubectl.bin /usr/local/bin/kubectl; \
		rm -f /tmp/kubectl.bin; \
		kubectl version --client; \
		echo "✅ kubeconfig: ~/.kube/config — run: kubectl get nodes"'

k0s-deploy: install-k0s

# Install ingress-nginx controller (bare-metal / NodePort manifest). Run on the SSH host that has kubectl.
# INGRESS_HOSTNETWORK=1 patches the controller to listen on the node’s real :80/:443 (needed if DNS points to the node and you see “connection refused”).
install-ingress-nginx:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Usage: make install-ingress-nginx HOST=host [USER=ubuntu] [INGRESS_HOSTNETWORK=1] [INGRESS_NGINX_TAG=v1.11.2]"; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	@echo "📥 Installing ingress-nginx ($(INGRESS_NGINX_TAG)) on $(TARGET_HOST)..."
	@if [ "$(INGRESS_HOSTNETWORK)" = "1" ]; then \
		scp -q k8s/ingress-nginx-hostnetwork-patch.json "$(TARGET_HOST)":/tmp/ing-hostnet-patch.json; \
	fi
	@ssh -o BatchMode=yes "$(TARGET_HOST)" 'set -e; \
		export PATH="$$PATH:/usr/local/bin:/snap/bin"; \
		command -v $(KUBECTL) >/dev/null 2>&1 || { echo "❌ kubectl not found on remote"; exit 127; }; \
		$(KUBECTL) apply -f "$(INGRESS_BAREMETAL_URL)" && \
		$(KUBECTL) wait --namespace ingress-nginx --for=condition=available deployment/ingress-nginx-controller --timeout=300s && \
		if [ "$(INGRESS_HOSTNETWORK)" = "1" ]; then \
			echo "🔧 Patching ingress-nginx for hostNetwork (bind :80/:443 on this node)..." && \
			$(KUBECTL) patch deployment ingress-nginx-controller -n ingress-nginx --type=strategic --patch-file /tmp/ing-hostnet-patch.json && \
			$(KUBECTL) rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=300s; \
		fi && \
		echo "✅ ingress-nginx ready. Check: $(KUBECTL) get svc,pods -n ingress-nginx"'

# Validate production environment file (same defaults as deploy: ENV_FILE, path vs Makefile dir)
validate-prod-env:
	@echo "🔍 Validating production environment configuration..."
	@if [ ! -f "$(DEPLOY_ENV_ABS)" ]; then \
		echo "❌ $(ENV_FILE) not found at $(DEPLOY_ENV_ABS). Create it from .env.example:"; \
		echo "   cp .env.example $(ENV_FILE)"; \
		echo "   # Edit with production values"; \
		exit 1; \
	fi
	@echo "✅ $(DEPLOY_ENV_ABS) exists"
	@echo "🔍 Checking required variables..."
	@if ! grep -q "^ADMIN_PASSWORD=" "$(DEPLOY_ENV_ABS)" || grep -q "^ADMIN_PASSWORD=$$" "$(DEPLOY_ENV_ABS)"; then \
		echo "⚠️  ADMIN_PASSWORD not set in $(ENV_FILE) (deploy will auto-generate on remote if empty)"; \
	fi
	@echo "✅ Production environment validation passed"

# ============================================================================
# CLEANUP TARGETS
# ============================================================================

# Clean everything (build artifacts)
clean:
	@echo "🧹 Cleaning everything..."
	rm -rf bin/
	rm -rf frontend/dist/
	rm -rf frontend/node_modules/

# Clean only build artifacts
clean-build:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf frontend/dist/

# ============================================================================
# INSTALLATION TARGETS
# ============================================================================

# Install CLI globally
install:
	@echo "📦 Installing CLI globally..."
	go install ./cmd/iam-manager

# Install golangci-lint
install-linter:
	@echo "📦 Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2
	@echo "✅ golangci-lint installed to $$(go env GOPATH)/bin"


# ============================================================================
# CI/CD TARGETS
# ============================================================================

# Run all CI checks locally
ci: check test-coverage podman-build
	@echo "✅ All CI checks completed successfully"

# Generate test coverage report
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v cmd/iam-manager)
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Lint Dockerfile with hadolint
lint-containerfile:
	@echo "🐳 Linting Dockerfile..."
	@if command -v hadolint >/dev/null 2>&1; then \
		hadolint Dockerfile; \
	else \
		podman run --rm -i docker.io/hadolint/hadolint:latest < Dockerfile; \
	fi

# Build OCI image (Podman)
podman-build: build-frontend
	@echo "🐳 Building container image with Podman..."
	podman build -t cloud-manager:latest .

# Build and tag for GHCR (see GHCR_IMAGE)
podman-build-ghcr: build-frontend
	@echo "🐳 Building container image for GHCR ($(GHCR_IMAGE))..."
	podman build -t $(GHCR_IMAGE) .

# Multi-architecture image (local manifest)
podman-build-multiarch: build-frontend
	@echo "🐳 Building multi-architecture image..."
	podman build --platform linux/amd64,linux/arm64 -t cloud-manager:latest .

# Multi-architecture build and push (override GHCR_IMAGE for tag)
podman-build-multiarch-push: build-frontend
	@echo "🐳 Building and pushing multi-architecture image to $(GHCR_IMAGE)..."
	podman build --platform linux/amd64,linux/arm64 -t $(GHCR_IMAGE) .
	podman push $(GHCR_IMAGE)

# Push to GHCR
podman-push-ghcr: podman-build-ghcr
	@echo "📤 Pushing $(GHCR_IMAGE)..."
	podman push $(GHCR_IMAGE)

# Run container locally
podman-run:
	@echo "🐳 Running container locally..."
	podman run -d -p 8080:8080 --name cloud-manager \
		--env-file .env.prod \
		cloud-manager:latest
	@echo "✅ Container started successfully"
	@echo "📍 Application available at: http://localhost:8080"
	@echo "💡 To stop: podman stop cloud-manager"
	@echo "💡 To remove: podman rm cloud-manager"

# Validate GitHub Actions workflows
validate-workflows:
	@echo "🔧 Validating GitHub Actions workflows..."
	@if command -v actionlint >/dev/null 2>&1; then \
		actionlint; \
	else \
		echo "Installing actionlint..."; \
		go install github.com/rhymond/actionlint/cmd/actionlint@latest; \
		actionlint; \
	fi

# Pre-commit checks (run before committing)
pre-commit: fmt lint test lint-containerfile validate-workflows
	@echo "✅ All pre-commit checks passed"

# Create release build
release-build: clean build-prod
	@echo "📦 Creating release artifacts..."
	@mkdir -p dist
	@cp cloud-manager dist/
	@cp -r frontend/dist dist/frontend
	@tar -czf dist/cloud-manager-release.tar.gz -C dist cloud-manager frontend
	@echo "✅ Release build created: dist/cloud-manager-release.tar.gz"