# Cloud Manager

A comprehensive web application for managing cloud resources across AWS and Azure. Features multi-cloud resource management, automated deployments, cross-account role management, and a modern web interface. Built with Go backend and Vue.js frontend with Podman/OCI image support.

## ✨ Features

### Core Functionality
- **🏢 Multi-Account Management**: Switch between AWS accounts using cross-account roles
- **👥 User Management**: View, create, and manage IAM users across organization accounts
- **🔑 Access Key Management**: Create, rotate, and delete access keys securely
- **🔒 Password Management**: Check and manage console password status
- **📊 User Details**: Comprehensive user information including ARN, creation date, and permissions

### Advanced Features
- **📦 StackSet Deployment**: Deploy IAM roles to all organization accounts with one click
- **✅ Permission Validation**: Automatically validate required AWS permissions before deployment
- **📈 Real-time Monitoring**: Track StackSet deployment progress across all accounts
- **🌙 Dark/Light Theme**: Modern responsive UI with theme switching
- **🛡️ Security**: External ID protection and least privilege permissions
- **🔐 AWS SSO (IAM Identity Center) Management**: View SSO users, groups, and account assignments
- **☁️ Azure AD Integration**: Manage Azure AD Enterprise Applications (optional)

## 🏗️ Architecture

- **Backend**: Go 1.21+ with Gin framework and AWS SDK v2
- **Frontend**: Vue.js 3 with Vite, compiled and served by Go (no Node.js runtime dependency)
- **Deployment**: Podman Compose for easy local deployment; production image `ghcr.io/mirantis/cloud-manager`
- **Authentication**: AWS IAM roles for cross-account access
- **Security**: External ID validation and proper credential management

## 📋 Prerequisites

1. **AWS Organization** with multiple accounts
2. **IAM user** in master account with permissions to:
   - List organization accounts (`organizations:ListAccounts`)
   - Assume roles in target accounts (`sts:AssumeRole`)
   - CloudFormation StackSets operations (for automated role deployment)
   - Organizations service access
   - IAM Identity Center (SSO) management (if using SSO features):
     - `sso:ListInstances`
     - `identitystore:ListUsers`, `identitystore:DescribeUser`
     - `identitystore:ListGroups`, `identitystore:DescribeGroup`
     - `identitystore:ListGroupMemberships`
     - `sso:ListPermissionSets`, `sso:DescribePermissionSet`
     - `sso:ListAccountAssignments`, `sso:ListAccountAssignmentsForPrincipal`
     - `sso:ListManagedPoliciesInPermissionSet`, `sso:GetInlinePolicyForPermissionSet`
   
   **Note**: SSO permissions must be attached directly to the IAM user in the master account (not just the cross-account role). See [SSO Setup](#sso-setup) section below.
3. **Target account roles** (`IAMManagerCrossAccountRole`) **OR** use our automated StackSet deployment
4. **Podman** (and **Podman Compose**, for containerized local deployment)
5. **Go 1.21+** (for CLI and local development)

## 🚀 Quick Start

### Option 1: Automated Setup (Recommended)

Use our modern Go CLI with Makefile targets for the best experience:

```bash
# 1. Clone the repository
git clone https://github.com/Mirantis/aws-iam-manager.git
cd aws-iam-manager

# 2. Check AWS configuration
make check-aws-config

# 3. Configure AWS credentials (if needed)
aws configure
# OR set environment variables:
# export AWS_ACCESS_KEY_ID=your_key
# export AWS_SECRET_ACCESS_KEY=your_secret
# export AWS_REGION=us-east-1

# 4. Deploy IAM user with required permissions
make deploy-user

# 5. Deploy roles to all organization accounts
make deploy-stackset

# 6. Check deployment status
make status-stackset

# 7. Start the web application
make dev
```

Access the application at `http://localhost:8080`

### Option 2: Manual Podman setup

```bash
# 1. Clone and configure
git clone https://github.com/Mirantis/aws-iam-manager.git
cd aws-iam-manager

# 2. Configure environment
cp .env.example .env
# Edit .env with your AWS credentials

# 3. Build and run (requires a compose file, or use make podman-run after make podman-build)
podman compose up --build
```

### Kubernetes (shared ingress, e.g. Mirantis)

The pod serves HTTP on **8080**; **nginx ingress** exposes **80/443** on the cluster load balancer. Other applications can keep using port 80 on the same ingress as long as they use a **different hostname**; this app’s Ingress only matches the hostname you pass as **DOMAIN**.

`make deploy` uses **SSH/SCP** to the machine where you run `kubectl` (`HOST` + optional `USER`). That can be a **bare IP** (e.g. control plane or jump host).

```bash
# Typical: SSH to ubuntu@172.19.112.251, public site on a DNS name (TLS + Ingress):
make deploy HOST=172.19.112.251 USER=ubuntu DOMAIN=iammanager.it.eu-cloud.mirantis.net

# Equivalent SSH target in one variable (no separate USER=):
make deploy HOST=ubuntu@172.19.112.251 DOMAIN=iammanager.it.eu-cloud.mirantis.net

# Other bastions (hostname instead of IP):
make deploy HOST=your-bastion.example.com DOMAIN=iammanager.it.eu-cloud.mirantis.net USER=ubuntu
```

Requirements: **passwordless SSH** or an **SSH agent** (for repeated `scp`/`ssh`), and the SSH target must have **`kubectl`** on disk and a working **kubeconfig** (`kubectl get nodes` succeeds). Non-interactive SSH often has a minimal `PATH`; the Makefile adds `/usr/local/bin` and `/snap/bin`. If you still see `kubectl: command not found`, install kubectl on that host or run deploy with an explicit binary, e.g. `KUBECTL=/snap/bin/kubectl make deploy HOST=...`.

Create a DNS **A** (or **AAAA**) record for **DOMAIN** pointing at the ingress controller’s external address (often the same as your existing app). TLS and HTTP-01 challenges are documented in [k8s/README-SSL.md](k8s/README-SSL.md). Application users are stored in SQLite on a **PersistentVolumeClaim** (`/data/iam-manager.db` via `k8s/configmap.yaml`).

On **k0s / bare metal**, if the site shows **connection refused** on `:443`, install **ingress-nginx** and use **hostNetwork** so the node listens on 80/443:  
`make install-ingress-nginx HOST=… USER=ubuntu INGRESS_HOSTNETWORK=1`  
(then point DNS at that node’s IP). See **“Connection refused”** in [k8s/README-SSL.md](k8s/README-SSL.md).

## ⚙️ Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Required AWS credentials
AWS_ACCESS_KEY_ID=your_access_key_here
AWS_SECRET_ACCESS_KEY=your_secret_key_here
AWS_REGION=us-east-1
AWS_SSO_REGION=eu-west-2  # Region for IAM Identity Center (SSO) management (defaults to eu-west-2)

# Optional IAM configuration
IAM_ORG_ROLE_NAME=IAMManagerCrossAccountRole
IAM_USER_NAME=iam-manager
IAM_POLICY_NAME=IAMManagerCustomPolicy
STACK_SET_NAME=IAMManagerRoleStackSet

# Optional Azure AD credentials (for Azure Enterprise Applications feature)
AZURE_TENANT_ID=your_tenant_id_here
AZURE_CLIENT_ID=your_client_id_here
AZURE_CLIENT_SECRET=your_client_secret_here

# Optional Azure Resource Manager credentials (for Azure VM and Storage management)
AZURE_SUBSCRIPTION_ID=your_subscription_id_here

# Application settings
PORT=8080

# Web UI authentication (SQLite user store; persistent across restarts)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me-in-production
# Optional: path to SQLite file (default ./iam-manager.db). Mount a volume in Podman/K8s for persistence.
DB_PATH=./iam-manager.db
```

Roles: **admin** (user management + all actions), **editor** (modify AWS/Azure resources), **viewer** (read-only). Admins create additional users via **Admin → User Management** in the UI (`/admin/users`).

### Azure AD Setup (Optional)

To enable Azure AD Enterprise Applications management:

1. **Register an Azure AD Application**:
   - Go to [Azure Portal](https://portal.azure.com) → Azure Active Directory → App registrations
   - Click "New registration"
   - Name: `cloud-manager` (or your preferred name)
   - Supported account types: Single tenant or Multi-tenant
   - Click "Register"

2. **Create a Client Secret**:
   - In your app registration, go to "Certificates & secrets"
   - Click "New client secret"
   - Add description and expiration
   - **Copy the secret value immediately** (it won't be shown again)

3. **Grant API Permissions**:
   - Go to "API permissions"
   - Click "Add a permission" → "Microsoft Graph" → "Application permissions"
   - Add the following permissions:
     - `Application.Read.All` (to list and read enterprise applications)
     - `Application.ReadWrite.All` (to delete enterprise applications)
   - Click "Add permissions"
   - **Important**: Click "Grant admin consent" for your organization

4. **Get Required Values**:
   - **Tenant ID**: Found in Azure AD → Overview → Tenant ID
   - **Client ID**: Found in your app registration → Overview → Application (client) ID
   - **Client Secret**: The secret value you copied in step 2

5. **Configure Environment Variables**:
   ```bash
   export AZURE_TENANT_ID=your_tenant_id
   export AZURE_CLIENT_ID=your_client_id
   export AZURE_CLIENT_SECRET=your_client_secret
   ```
   
   Or add them to your `.env.prod` file (for `make dev`) or Kubernetes secrets (for production).

6. **Verify Setup**:
   - Start the application: `make dev`
   - Check logs for: `[INFO] Azure service initialized successfully`
   - Navigate to "Azure Apps" tab in the web interface

**Note**: Azure features are optional. The application will work without Azure credentials, but Azure endpoints will not be available.

### SSO Setup (IAM Identity Center)

To enable AWS SSO (IAM Identity Center) management features:

1. **Ensure IAM Identity Center is enabled** in your AWS Organization:
   - Go to AWS Console → IAM Identity Center
   - If not enabled, follow the setup wizard to enable it

2. **Attach SSO permissions to your IAM user** in the master account:
   
   The IAM user (e.g., `iam-manager`) needs SSO permissions directly attached, as SSO operations are performed in the master account context.
   
   **Option A: Using AWS CLI** (Recommended):
   ```bash
   # Attach the SSO policy to your IAM user
   aws iam put-user-policy \
     --user-name iam-manager \
     --policy-name SSOIdentityCenterManagement \
     --policy-document file://cloudformation/iam-manager-user-sso-policy.json
   ```
   
   **Option B: Using AWS Console**:
   - Go to IAM → Users → `iam-manager` → Permissions
   - Click "Add permissions" → "Create inline policy"
   - Choose JSON tab
   - Copy the contents from `cloudformation/iam-manager-user-sso-policy.json`
   - Review and create the policy
   
   **Option C: Using CloudFormation** (if you prefer IaC):
   ```bash
   # Create a stack that attaches the policy
   aws cloudformation create-stack \
     --stack-name iam-manager-user-sso-policy \
     --template-body file://cloudformation/iam-manager-user-sso-policy.yaml \
     --parameters ParameterKey=UserName,ParameterValue=iam-manager
   ```

3. **Set the SSO region** (if different from default):
   ```bash
   export AWS_SSO_REGION=eu-west-2  # Default is eu-west-2 (London)
   ```
   
   Or add to your `.env.prod` file or Kubernetes secrets.

4. **Verify Setup**:
   - Restart the application: `make dev`
   - Check logs for: `[INFO] SSO service initialized successfully`
   - Navigate to "SSO Users" or "SSO Groups" in the web interface
   - If you see an error, check the `/api/sso/status` endpoint for details

**Note**: SSO features are optional. The application will work without SSO permissions, but SSO endpoints will return error messages indicating the service is unavailable.

### Azure Resource Manager Setup (Optional)

To enable Azure VM and Storage Account management:

1. **Follow Azure AD Setup** (steps 1-4 above) to create the service principal

2. **Grant Subscription Listing Permissions**:

   The service principal needs permission to list subscriptions. You have two options:

   **Option A: Grant Reader Role at Management Group Level (Recommended)**
   
   This allows the service principal to list all subscriptions under a management group:
   
   ```bash
   # Get your Management Group ID (or use "Tenant Root Group")
   MANAGEMENT_GROUP_ID="your-management-group-id"  # or use "/" for Tenant Root Group
   SERVICE_PRINCIPAL_ID="your-service-principal-object-id"  # Found in Azure AD → Enterprise applications → Your app → Object ID
   
   # Grant Reader role at Management Group level
   az role assignment create \
     --assignee $SERVICE_PRINCIPAL_ID \
     --role "Reader" \
     --scope "/providers/Microsoft.Management/managementGroups/$MANAGEMENT_GROUP_ID"
   ```
   
   **Via Azure Portal:**
   - Go to [Azure Portal](https://portal.azure.com) → Management groups
   - Select your management group (or "Tenant Root Group")
   - Click "Access control (IAM)"
   - Click "Add" → "Add role assignment"
   - Role: **Reader**
   - Assign access to: **User, group, or service principal**
   - Select your service principal (search by app name)
   - Click "Save"

   **Option B: Grant Reader Role on Each Subscription**
   
   If you prefer to grant access per subscription:
   
   ```bash
   SUBSCRIPTION_ID="your-subscription-id"
   SERVICE_PRINCIPAL_ID="your-service-principal-object-id"
   
   # Grant Reader role on subscription
   az role assignment create \
     --assignee $SERVICE_PRINCIPAL_ID \
     --role "Reader" \
     --scope "/subscriptions/$SUBSCRIPTION_ID"
   ```
   
   **Via Azure Portal:**
   - Go to [Azure Portal](https://portal.azure.com) → Subscriptions
   - Select your subscription
   - Click "Access control (IAM)"
   - Click "Add" → "Add role assignment"
   - Role: **Reader**
   - Assign access to: **User, group, or service principal**
   - Select your service principal
   - Click "Save"
   - Repeat for each subscription you want to access

3. **Grant VM and Storage Permissions**:

   For VM management, grant **Virtual Machine Contributor** role:
   
   ```bash
   SUBSCRIPTION_ID="your-subscription-id"
   SERVICE_PRINCIPAL_ID="your-service-principal-object-id"
   
   # Grant VM Contributor role
   az role assignment create \
     --assignee $SERVICE_PRINCIPAL_ID \
     --role "Virtual Machine Contributor" \
     --scope "/subscriptions/$SUBSCRIPTION_ID"
   
   # Grant Storage Account Contributor role
   az role assignment create \
     --assignee $SERVICE_PRINCIPAL_ID \
     --role "Storage Account Contributor" \
     --scope "/subscriptions/$SUBSCRIPTION_ID"
   ```
   
   **Via Azure Portal:**
   - Go to Subscription → Access control (IAM)
   - Add role assignment:
     - **Virtual Machine Contributor** (for VM start/stop/delete)
     - **Storage Account Contributor** (for storage account management)
   - Assign to your service principal

4. **Configure Environment Variables**:
   
   ```bash
   export AZURE_TENANT_ID=your_tenant_id
   export AZURE_CLIENT_ID=your_client_id
   export AZURE_CLIENT_SECRET=your_client_secret
   # Optional: Set AZURE_SUBSCRIPTION_ID as fallback if subscription listing fails
   export AZURE_SUBSCRIPTION_ID=your_subscription_id
   ```

5. **Verify Setup**:
   - Start the application: `make dev`
   - Check logs for: `[INFO] Azure Resource Manager service initialized successfully`
   - Navigate to "Azure VMs" or "Azure Storage" tabs
   - If subscriptions list is empty, the system will use `AZURE_SUBSCRIPTION_ID` as fallback

**Troubleshooting:**

- **Empty subscriptions list**: If you see `[WARNING] No subscriptions found`, the service principal doesn't have permission to list subscriptions. Either:
  - Grant Reader role at Management Group/Tenant Root level (Option A above), OR
  - Set `AZURE_SUBSCRIPTION_ID` environment variable as a fallback
- **Permission denied errors**: Ensure the service principal has the required roles (Reader, Virtual Machine Contributor, Storage Account Contributor) on the subscriptions you want to access


## 🛠️ Makefile Targets

### Build & Development
- `make help` - Show all available targets
- `make build-frontend` - Build Vue.js frontend
- `make build-backend` - Build Go backend server
- `make build-cli` - Build Go CLI application
- `make dev` - Build with Podman and deploy to local Kubernetes

### AWS IAM Management
- `make deploy-user` - Deploy IAM user and resources
- `make remove-user` - Remove IAM user and resources
- `make create-role` - Create IAM role for cross-account access
- `make remove-role` - Remove IAM role and resources
- `make deploy-stackset` - Deploy StackSet for organization setup
- `make status-stackset` - Show StackSet deployment status
- `make delete-stackset` - Delete StackSet and all instances
- `make cli-status` - Show current deployment status

### Setup & Configuration
- `make check-aws-config` - Verify AWS credentials and configuration

### Testing & Quality
- `make test` - Run all tests
- `make lint` - Lint all code
- `make check` - Run all checks (fmt + lint + test)

## 🔐 IAM Permissions

### Master Account User

The CLI deployment script creates an IAM user with these managed policies:
- `IAMFullAccess` - For IAM user management
- `CloudFormationFullAccess` - For StackSet operations  
- `AWSOrganizationsReadOnlyAccess` - For organization account discovery

### Custom Policy (Minimum Permissions)

If you prefer minimal permissions, create a custom policy:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "OrganizationsAccess",
            "Effect": "Allow",
            "Action": [
                "organizations:ListAccounts",
                "organizations:DescribeOrganization"
            ],
            "Resource": "*"
        },
        {
            "Sid": "AssumeRolePermissions", 
            "Effect": "Allow",
            "Action": [
                "sts:AssumeRole",
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        },
        {
            "Sid": "StackSetPermissions",
            "Effect": "Allow", 
            "Action": [
                "cloudformation:CreateStackSet",
                "cloudformation:UpdateStackSet", 
                "cloudformation:DeleteStackSet",
                "cloudformation:DescribeStackSet",
                "cloudformation:ListStackSets",
                "cloudformation:CreateStackInstances",
                "cloudformation:DeleteStackInstances",
                "cloudformation:DescribeStackInstance", 
                "cloudformation:ListStackInstances",
                "cloudformation:DescribeStackSetOperation",
                "cloudformation:ListStackSetOperations"
            ],
            "Resource": "*"
        }
    ]
}
```

## 📦 StackSet Deployment

### Web Interface (Recommended)

1. **Start the application**: `make dev`
2. **Open browser**: `http://localhost:8080`
3. **Navigate to StackSet tab**
4. **Validate permissions** and **deploy with one click**

### What Gets Deployed

The StackSet creates in each target account:

- **IAM Role**: `IAMManagerCrossAccountRole`
- **Permissions**: Full IAM access for user management operations
- **Trust Policy**: Allows your master account user to assume the role
- **External ID**: Account-specific external ID for security (`{AccountId}-iam-manager`)
- **Security**: Built-in protection against confused deputy attacks

### CLI Commands

```bash
# Deploy StackSet to all organization accounts
make deploy-stackset

# Check deployment status
make status-stackset

# Get detailed status information
make cli-status

# Delete StackSet and all instances
make delete-stackset
```

## 🌐 API Endpoints

### IAM Management
- `GET /api/accounts` - List organization accounts
- `GET /api/accounts/:accountId/users` - List users in account
- `GET /api/accounts/:accountId/users/:username` - Get user details
- `POST /api/accounts/:accountId/users/:username/keys` - Create access key
- `DELETE /api/accounts/:accountId/users/:username/keys/:keyId` - Delete access key
- `PUT /api/accounts/:accountId/users/:username/keys/:keyId/rotate` - Rotate access key

### Security Groups Management
- `GET /api/security-groups` - List security groups across all accounts
- `GET /api/accounts/:accountId/security-groups` - List security groups by account
- `GET /api/accounts/:accountId/regions/:region/security-groups/:groupId` - Get security group details
- `DELETE /api/accounts/:accountId/regions/:region/security-groups/:groupId` - Delete security group

### StackSet Management
- `GET /api/stackset/status` - Get StackSet deployment status
- `POST /api/stackset/deploy` - Deploy/update StackSet to all accounts
- `GET /api/stackset/deployment/:operationId` - Get deployment operation status
- `DELETE /api/stackset/` - Delete StackSet and all instances
- `GET /api/stackset/validate` - Validate StackSet deployment permissions

### Azure AD Management (Optional - requires Azure credentials)
- `GET /api/azure/enterprise-applications` - List all Azure AD enterprise applications
- `GET /api/azure/enterprise-applications/:appId` - Get enterprise application details
- `DELETE /api/azure/enterprise-applications/:appId` - Delete enterprise application
- `POST /api/azure/cache/clear` - Clear all Azure cache
- `POST /api/azure/cache/enterprise-applications/invalidate` - Invalidate enterprise applications cache

### Azure Resource Manager Management (Optional - requires Azure RM credentials)
- `GET /api/azure/vms` - List all Azure virtual machines
- `GET /api/azure/vms/:resourceGroup/:vmName` - Get VM details
- `POST /api/azure/vms/:resourceGroup/:vmName/start` - Start VM
- `POST /api/azure/vms/:resourceGroup/:vmName/stop` - Stop VM
- `DELETE /api/azure/vms/:resourceGroup/:vmName` - Delete VM
- `GET /api/azure/storage-accounts` - List all storage accounts
- `GET /api/azure/storage-accounts/:resourceGroup/:name` - Get storage account details
- `DELETE /api/azure/storage-accounts/:resourceGroup/:name` - Delete storage account
- `POST /api/azure/rm/cache/clear` - Clear all Azure RM cache
- `POST /api/azure/rm/cache/vms/invalidate` - Invalidate VMs cache
- `POST /api/azure/rm/cache/storage/invalidate` - Invalidate storage cache

## 🏗️ Development

### Prerequisites
- Go 1.21+
- Node.js 18+ (for frontend development)
- AWS CLI configured
- Podman (and Podman Compose if you use compose workflows)

### Backend Development
```bash
make build-backend
cd backend && go run ./cmd/server
```

### Frontend Development
```bash
make dev-frontend
# Runs on http://localhost:5173 with API proxy to :8080
```

### Full Development Environment
```bash
make dev
# Builds the image with Podman and runs in local Kubernetes (or use make dev-compose with Podman Compose)
```

### Testing
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Run all checks
make check
```

## 🔒 Security Considerations

- **Access Keys**: Only displayed once when created/rotated
- **IAM Permissions**: All operations require proper AWS permissions
- **Temporary Credentials**: Uses temporary credentials when assuming roles
- **External ID**: Prevents confused deputy attacks
- **Audit Trail**: All operations logged in CloudTrail
- **Frontend Security**: Served by Go eliminates Node.js attack surface
- **HTTPS Ready**: Built-in support for TLS termination

## 🛠️ Troubleshooting

### Common Issues

**AWS Credentials Not Configured**
```bash
make check-aws-config
```

**Permission Denied**
```bash
# Check if your user has required permissions
aws sts get-caller-identity
aws organizations list-accounts
```

**StackSet Deployment Fails**
```bash
# Check StackSet status
make status-stackset

# View detailed logs
make logs
```

**Container Won't Start**
```bash
# Check container logs (compose) or Podman logs
podman compose logs cloud-manager
# or: podman logs cloud-manager

# Rebuild image and rerun
make podman-build
make podman-run
```

**Azure Features Not Available**
```bash
# Check if Azure credentials are set
grep -E "AZURE_TENANT_ID|AZURE_CLIENT_ID|AZURE_CLIENT_SECRET" .env.prod

# If missing, add them to .env.prod:
# AZURE_TENANT_ID=your_tenant_id
# AZURE_CLIENT_ID=your_client_id
# AZURE_CLIENT_SECRET=your_client_secret

# Restart the application
make dev-stop
make dev

# Check logs for Azure initialization
make dev-logs | grep -i azure
```

**Azure Error: "Application not found in the directory" (AADSTS700016)**
This error means the app registration doesn't exist in the specified tenant. To fix:

1. **Verify Tenant ID**:
   - Go to Azure Portal → Azure Active Directory → Overview
   - Copy the "Tenant ID" (should match `AZURE_TENANT_ID`)

2. **Verify Client ID**:
   - Go to Azure Portal → Azure Active Directory → App registrations
   - Find your app registration
   - Copy the "Application (client) ID" (should match `AZURE_CLIENT_ID`)
   - **Important**: Make sure the app registration exists in the same tenant as the Tenant ID

3. **Check Tenant Match**:
   - The Tenant ID and Client ID must be from the same Azure AD tenant
   - If you have multiple tenants, ensure you're using the correct pair

4. **Verify App Registration**:
   - Ensure the app registration exists in the tenant
   - Check that it hasn't been deleted or moved to another tenant

5. **Update Credentials**:
   ```bash
   # Edit .env.prod with correct values
   nano .env.prod
   
   # Restart the application
   make dev-stop
   make dev
   ```

**Azure Error: "unauthorized_client" or Authentication Failed**
This usually means:
- Client Secret is incorrect or expired (create a new one)
- API permissions not granted (grant admin consent)
- App registration doesn't have required permissions

## 📚 Additional Documentation

- [CloudFormation Template](cloudformation/iam-manager-role.yaml) - StackSet deployment template
- [Contributing Guidelines](CONTRIBUTING.md) - Development and contribution guide
- [Environment Variables](.env.example) - Configuration options

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `make check`
4. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/Mirantis/aws-iam-manager/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Mirantis/aws-iam-manager/discussions)
- **Documentation**: This README and inline code documentation

---

**Made with ❤️ for Cloud Management** - Simplifying multi-cloud resource management