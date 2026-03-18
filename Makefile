.PHONY: build build-frontend build-backend build-cli dev dev-stop dev-logs dev-compose dev-compose-stop dev-cli dev-frontend dev-backend test fmt lint install check install-linter tidy deps preview clean-build ci pre-commit build-prod build-release release help deploy-user remove-user create-role remove-role deploy-stackset update-stackset status-stackset remove-stackset delete-stackset cli-status check-aws-config unset-variables unset-variables-exec deploy validate-prod-env docker-build docker-build-ghcr docker-build-multiarch docker-build-multiarch-push docker-push-ghcr lint-docker docker-run


# Default target
help:
	@echo "Cloud Manager - Unified Build System"
	@echo ""
	@echo "🏗️  Build Targets:"
	@echo "  build-frontend   - Build Vue.js frontend"
	@echo "  build-backend    - Build Go backend server"
	@echo "  build-cli        - Build Go CLI application"
	@echo "  build-go         - Build both backend and CLI"
	@echo "  build            - Build everything (Docker)"
	@echo "  build-prod       - Production build with optimizations"
	@echo "  build-release    - Multi-platform release binaries"
	@echo ""
	@echo "🚀 Development:"
	@echo "  dev              - Build and deploy to local k8s with .env.prod"
	@echo "  dev-stop         - Stop and remove local k8s deployment"
	@echo "  dev-logs         - Show logs from local k8s deployment"
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
	@echo "🐳 Docker Operations:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-build-ghcr - Build Docker image for GitHub Container Registry"
	@echo "  docker-build-multiarch - Build multi-architecture Docker image"
	@echo "  docker-push-ghcr - Push Docker image to GitHub Container Registry"
	@echo "  docker-run       - Run Docker container locally"
	@echo "  lint-docker      - Lint Dockerfile"
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
	@echo "  deploy HOST=host [USER=user] - Deploy application to specified host using Kubernetes"
	@echo "                               (automatically uses .env.prod if available)"
	@echo "                               Includes Let's Encrypt SSL certificate setup"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  clean            - Clean everything"
	@echo "  clean-build      - Clean build artifacts only"
	@echo ""
	@echo "🔧 Setup & Configuration:"
	@echo "  check-aws-config - Verify AWS credentials and configuration"
	@echo "  unset-variables  - Show command to unset AWS credential environment variables"
	@echo "  validate-prod-env - Validate production environment file (.env.prod)"
	@echo ""
	@echo "☁️  Azure AD (Optional):"
	@echo "  Azure support requires AZURE_TENANT_ID, AZURE_CLIENT_ID, and"
	@echo "  AZURE_CLIENT_SECRET in .env.prod. See README.md for setup instructions."
	@echo ""
	@echo "🚀 CI/CD & Quality:"
	@echo "  ci               - Run all CI checks locally"
	@echo "  test-coverage    - Generate test coverage report"
	@echo "  security-scan    - Run security analysis with gosec"
	@echo "  lint-docker      - Lint Dockerfile with hadolint"
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

# Build everything with Docker
build: build-frontend docker-build

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

# Development mode - builds and deploys to local k8s cluster with .env.prod env vars
# For Podman, minikube (with eval $(minikube podman-env)), or kind (with kind load)
dev:
	@echo "🚀 Starting development environment in Kubernetes..."
	@echo "🧹 Unsetting AWS environment variables..."
	@if [ ! -f .env.prod ]; then \
		echo "❌ Error: .env.prod file not found. Create it with your environment variables."; \
		exit 1; \
	fi
	@echo "📦 Building frontend..."
	@cd frontend && npm run build
	@echo "📦 Building Podman image locally (no cache)..."
	@(unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_REGION AWS_PROFILE AWS_DEFAULT_REGION AWS_SSO_REGION; \
	podman build \
		--no-cache \
		--network=host \
		-t cloud-manager:dev .) || \
		(echo "❌ Podman build failed. Trying without network isolation..." && \
		 unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_REGION AWS_PROFILE AWS_DEFAULT_REGION AWS_SSO_REGION && \
		 podman build --no-cache --network=host -t cloud-manager:dev .)
	@echo "☸️  Deploying to Kubernetes cluster..."
	@kubectl apply -f k8s/namespace.yaml
	@echo "🔐 Generating admin password..."
	@if ! grep -q "^ADMIN_PASSWORD=" .env.prod 2>/dev/null || [ -z "$$(grep '^ADMIN_PASSWORD=' .env.prod | cut -d'=' -f2)" ]; then \
		ADMIN_PASSWORD=$$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16); \
		echo "✅ Generated random admin password (stored in .env.prod)"; \
		if grep -q "^ADMIN_PASSWORD=" .env.prod 2>/dev/null; then \
			sed -i.bak "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$$ADMIN_PASSWORD|" .env.prod; \
		else \
			echo "ADMIN_PASSWORD=$$ADMIN_PASSWORD" >> .env.prod; \
		fi; \
		if ! grep -q "^ADMIN_USERNAME=" .env.prod 2>/dev/null; then \
			echo "ADMIN_USERNAME=admin" >> .env.prod; \
		fi; \
	else \
		echo "✅ Using existing admin password from .env.prod"; \
	fi; \
	ADMIN_USERNAME=$$(grep '^ADMIN_USERNAME=' .env.prod 2>/dev/null | cut -d'=' -f2 || echo "admin"); \
	kubectl create secret generic app-secrets --namespace=cloud-manager \
		--from-env-file=.env.prod \
		--dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/configmap.yaml
	@sed 's|image: ghcr.io/rusik69/cloud-manager:latest|image: cloud-manager:dev\n        imagePullPolicy: Never|' \
		k8s/app-deployment.yaml | kubectl apply -f -
	@kubectl apply -f k8s/service.yaml
	@echo "🔄 Forcing pod restart to pick up new image..."
	@kubectl rollout restart deployment/cloud-manager -n cloud-manager
	@echo "⏳ Waiting for deployment to be ready..."
	@kubectl rollout status deployment/cloud-manager -n cloud-manager --timeout=120s
	@echo "✅ Deployment ready!"
	@echo ""
	@ADMIN_USERNAME=$$(grep '^ADMIN_USERNAME=' .env.prod 2>/dev/null | cut -d'=' -f2 || echo "admin"); \
	echo "🔐 Admin credentials configured (username: $$ADMIN_USERNAME)"
	@echo "   Password stored in .env.prod file (not displayed for security)"
	@echo ""
	@echo "💡 Access the app at http://localhost:8080"
	@echo "🔌 Starting port-forward and showing logs (Ctrl+C to stop)..."
	@trap 'kill 0' INT TERM; \
	kubectl port-forward -n cloud-manager svc/cloud-manager 8080:8080 & \
	sleep 2 && kubectl logs -f -n cloud-manager -l app.kubernetes.io/name=cloud-manager & \
	wait

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

# Stop local k8s development deployment
dev-stop:
	@echo "🛑 Stopping local Kubernetes deployment..."
	@kubectl delete deployment cloud-manager -n cloud-manager --ignore-not-found
	@kubectl delete secret app-secrets -n cloud-manager --ignore-not-found
	@echo "✅ Local deployment stopped"

# Show logs from local k8s deployment
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
	go test ./...
	cd frontend && npm test

# Run tests with coverage
test-coverage-basic:
	@echo "🧪 Running tests with basic coverage..."
	go test -cover ./...
	cd frontend && npm run test -- --coverage

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running verbose tests..."
	go test -v ./...

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
		golangci-lint run ./...; \
	else \
		echo "  ⚠️  golangci-lint not found, using basic go vet"; \
		go vet ./...; \
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
deploy:
	@if [ -z "$(HOST)" ]; then \
		echo "❌ Error: HOST parameter is required. Usage: make deploy HOST=your-host [USER=username]"; \
		exit 1; \
	fi
	$(eval TARGET_HOST := $(if $(USER),$(USER)@$(HOST),$(HOST)))
	@echo "☸️  Deploying application to $(TARGET_HOST) using Kubernetes..."
	@echo "📤 Copying Kubernetes manifests to $(TARGET_HOST)..."
	@scp -r k8s/ $(TARGET_HOST):~/cloud-manager/
	@if [ -f .env.prod ]; then \
		echo "📤 Copying production environment file (.env.prod)..."; \
		scp .env.prod $(TARGET_HOST):~/cloud-manager/k8s/.env; \
	else \
		echo "⚠️  No .env.prod found, creating from .env.example"; \
		scp .env.example $(TARGET_HOST):~/cloud-manager/k8s/.env; \
	fi
	@echo "☸️  Configuring secrets and deploying to Kubernetes..."
	@ssh $(TARGET_HOST) 'cd ~/cloud-manager && \
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
		kubectl apply -f k8s/namespace.yaml && \
		echo "🔐 Creating Kubernetes secrets from environment file..." && \
		kubectl create secret generic app-secrets --namespace=cloud-manager \
			--from-env-file=k8s/.env \
			--dry-run=client -o yaml | kubectl apply -f - && \
		echo "☸️  Applying cert-manager configuration..." && \
		kubectl apply -f k8s/cert-manager.yaml && \
		echo "🔧 Creating certificate for domain $(HOST)..." && \
		sed "s/DOMAIN_PLACEHOLDER/$(HOST)/g" k8s/certificate.yaml | kubectl apply -f - && \
		echo "🔧 Creating ingress for domain $(HOST)..." && \
		sed "s/DOMAIN_PLACEHOLDER/$(HOST)/g" k8s/ingress.yaml | kubectl apply -f - && \
		echo "☸️  Applying remaining Kubernetes manifests..." && \
		kubectl apply -f k8s/configmap.yaml && \
		kubectl apply -f k8s/app-deployment.yaml && \
		kubectl apply -f k8s/service.yaml'
	@echo "✅ Application deployed successfully to Kubernetes cluster on $(TARGET_HOST)"
	@echo ""
	@echo "🔐 Admin credentials configured"
	@ssh $(TARGET_HOST) 'cd ~/cloud-manager && \
		ADMIN_USERNAME=$$(grep '\''^ADMIN_USERNAME='\'' k8s/.env 2>/dev/null | cut -d'\''='\'' -f2 || echo "admin"); \
		echo "  Username: $$ADMIN_USERNAME"; \
		echo "  Password stored in k8s/.env file (not displayed for security)"'
	@echo ""
	@echo "🌐 External Access Information:"
	@echo "  📍 HTTPS Access (Port 443): https://$(HOST)"
	@echo "  📍 HTTP Access (Port 80): http://$(HOST) (redirects to HTTPS)"
	@echo "  📍 Via Nginx Ingress Controller with Let's Encrypt SSL"
	@echo "  📍 Traffic flows: Internet → Ingress (SSL termination) → cloud-manager service → backend"
	@echo ""
	@echo "🔍 Checking deployment status..."
	@ssh $(TARGET_HOST) 'kubectl get pods -n cloud-manager && kubectl get services -n cloud-manager && kubectl get ingress -n cloud-manager && kubectl get certificates -n cloud-manager'
	@echo ""
	@echo "🔒 SSL Certificate Information:"
	@echo "   • Let's Encrypt certificate will be automatically provisioned"
	@echo "   • Certificate status: kubectl get certificates -n cloud-manager"
	@echo "   • Certificate issuer: letsencrypt-prod"
	@echo ""
	@echo "💡 External Access:"
	@echo "   • Primary: https://$(HOST) (HTTPS with Let's Encrypt SSL)"
	@echo "   • Fallback: http://$(HOST) (redirects to HTTPS)"
	@echo "   • Requires: Nginx Ingress Controller + cert-manager"
	@echo "   • Authentication is handled by the application (admin username/password)"
	@echo "   • Make sure ports 80 and 443 are open in your firewall/security groups"

# Validate production environment file
validate-prod-env:
	@echo "🔍 Validating production environment configuration..."
	@if [ ! -f .env.prod ]; then \
		echo "❌ .env.prod not found. Create it from .env.example:"; \
		echo "   cp .env.example .env.prod"; \
		echo "   # Edit .env.prod with production values"; \
		exit 1; \
	fi
	@echo "✅ .env.prod exists"
	@echo "🔍 Checking required variables..."
	@if ! grep -q "^ADMIN_PASSWORD=" .env.prod || grep -q "^ADMIN_PASSWORD=$$" .env.prod; then \
		echo "⚠️  ADMIN_PASSWORD not set in .env.prod (will be auto-generated)"; \
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
ci: check test-coverage docker-build
	@echo "✅ All CI checks completed successfully"

# Generate test coverage report
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Security scan with gosec
security-scan:
	@echo "🔒 Running security scan..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	gosec -fmt sarif -out gosec.sarif ./...
	gosec ./...

# Lint Dockerfile with hadolint
lint-docker:
	@echo "🐳 Linting Dockerfile..."
	@if command -v hadolint >/dev/null 2>&1; then \
		hadolint Dockerfile; \
	else \
		docker run --rm -i hadolint/hadolint < Dockerfile; \
	fi

# Build Docker image
docker-build: build-frontend
	@echo "🐳 Building Docker image..."
	docker build -t cloud-manager:latest .

# Build and tag Docker image for GitHub Container Registry
docker-build-ghcr: build-frontend
	@echo "🐳 Building Docker image for GHCR..."
	docker build -t ghcr.io/rusik69/cloud-manager:latest .

# Build multi-architecture Docker image
docker-build-multiarch: build-frontend
	@echo "🐳 Building multi-architecture Docker image..."
	docker buildx create --use --name multiarch-builder || true
	docker buildx build --platform linux/amd64,linux/arm64 -t cloud-manager:latest .
	docker buildx rm multiarch-builder

# Build multi-architecture Docker image and push to registry
docker-build-multiarch-push: build-frontend
	@echo "🐳 Building and pushing multi-architecture Docker image..."
	docker buildx create --use --name multiarch-builder || true
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(IMAGE_TAG) .
	docker buildx rm multiarch-builder

# Push Docker image to GitHub Container Registry
docker-push-ghcr: docker-build-ghcr
	@echo "📤 Pushing Docker image to GHCR..."
	docker push ghcr.io/rusik69/cloud-manager:latest

# Run Docker container locally
docker-run:
	@echo "🐳 Running Docker container locally..."
	docker run -d -p 8080:8080 --name cloud-manager \
		--env-file .env.prod \
		cloud-manager:latest
	@echo "✅ Container started successfully"
	@echo "📍 Application available at: http://localhost:8080"
	@echo "💡 To stop: docker stop cloud-manager"
	@echo "💡 To remove: docker rm cloud-manager"

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
pre-commit: fmt lint test security-scan lint-docker validate-workflows
	@echo "✅ All pre-commit checks passed"

# Create release build
release-build: clean build-prod
	@echo "📦 Creating release artifacts..."
	@mkdir -p dist
	@cp cloud-manager dist/
	@cp -r frontend/dist dist/frontend
	@tar -czf dist/cloud-manager-release.tar.gz -C dist cloud-manager frontend
	@echo "✅ Release build created: dist/cloud-manager-release.tar.gz"