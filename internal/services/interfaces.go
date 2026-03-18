package services

import (
	"context"

	"github.com/rusik69/aws-iam-manager/internal/models"
)

type AWSServiceInterface interface {
	ListAccounts() ([]models.Account, error)
	ListUsers(accountID string) ([]models.User, error)
	ListAllUsers() ([]models.UserWithAccount, error)
	GetUser(accountID, username string) (*models.User, error)
	CreateAccessKey(accountID, username string) (map[string]any, error)
	DeleteAccessKey(accountID, username, keyID string) error
	RotateAccessKey(accountID, username, keyID string) (map[string]any, error)
	DeleteUser(accountID, username string) error
	DeleteUserPassword(accountID, username string) error
	RotateUserPassword(accountID, username string) (map[string]any, error)
	DeleteInactiveUsers(accountID string) ([]string, []string, error)
	ListPublicIPs() ([]models.PublicIP, error)
	ListSecurityGroups() ([]models.SecurityGroup, error)
	ListSecurityGroupsByAccount(accountID string) ([]models.SecurityGroup, error)
	GetSecurityGroup(accountID, region, groupID string) (*models.SecurityGroup, error)
	DeleteSecurityGroup(accountID, region, groupID string) error
	// Snapshot management
	ListSnapshots() ([]models.Snapshot, error)
	ListSnapshotsByAccount(accountID string) ([]models.Snapshot, error)
	DeleteSnapshot(accountID, region, snapshotID string) error
	DeleteOldSnapshots(accountID string, olderThanMonths int) ([]string, error)
	// EC2 instance management
	ListEC2Instances() ([]models.EC2Instance, error)
	StopEC2Instance(accountID, region, instanceID string) error
	TerminateEC2Instance(accountID, region, instanceID string) error
	InvalidateEC2InstancesCache()
	// EBS volume management
	ListEBSVolumes() ([]models.EBSVolume, error)
	ListEBSVolumesByAccount(accountID string) ([]models.EBSVolume, error)
	DetachEBSVolume(accountID, region, volumeID string) error
	DeleteEBSVolume(accountID, region, volumeID string) error
	InvalidateEBSVolumesCache()
	// S3 bucket management
	ListS3Buckets() ([]models.S3Bucket, error)
	ListS3BucketsByAccount(accountID string) ([]models.S3Bucket, error)
	DeleteS3Bucket(accountID, region, bucketName string) error
	InvalidateS3BucketsCache()
	// IAM role management
	ListRoles(accountID string) ([]models.IAMRole, error)
	ListAllRoles() ([]models.RoleWithAccount, error)
	GetRole(accountID, roleName string) (*models.IAMRole, error)
	DeleteRole(accountID, roleName string) error
	InvalidateRolesCache()
	InvalidateAccountRolesCache(accountID string)
	// Load balancer management
	ListAllLoadBalancers() ([]models.LoadBalancer, error)
	ListLoadBalancersByAccount(accountID string) ([]models.LoadBalancer, error)
	DeleteLoadBalancer(accountID, region, loadBalancerArnOrName, lbType string) error
	InvalidateLoadBalancersCache(accountID string)
	InvalidateAllLoadBalancersCache()
	// VPC management
	ListVPCs() ([]models.VPC, error)
	ListVPCsByAccount(accountID string) ([]models.VPC, error)
	DeleteVPC(accountID, region, vpcID string) error
	InvalidateVPCsCache()
	// NAT Gateway management
	ListNATGateways() ([]models.NATGateway, error)
	ListNATGatewaysByAccount(accountID string) ([]models.NATGateway, error)
	DeleteNATGateway(accountID, region, natGatewayID string) error
	InvalidateNATGatewaysCache()
	// Route53 domain management
	ListRoute53Domains() ([]models.Route53Domain, error)
	ListRoute53Records(accountID, hostedZoneID string) ([]models.Route53Record, error)
	InvalidateRoute53DomainsCache()
	// Cache management methods
	ClearCache()
	InvalidateAccountCache(accountID string)
	InvalidateUserCache(accountID, username string)
	InvalidatePublicIPsCache()
	InvalidateSecurityGroupsCache()
	InvalidateAccountSecurityGroupsCache(accountID string)
}

type AzureServiceInterface interface {
	ListEnterpriseApplications(ctx context.Context) ([]models.AzureEnterpriseApplication, error)
	GetEnterpriseApplication(ctx context.Context, appID string) (*models.AzureEnterpriseApplication, error)
	DeleteEnterpriseApplication(ctx context.Context, appID string) error
	// Cache management methods
	ClearCache()
	InvalidateEnterpriseApplicationsCache()
	InvalidateEnterpriseApplicationCache(appID string)
}

type AzureRMServiceInterface interface {
	ListSubscriptions(ctx context.Context) ([]models.AzureSubscription, error)
	ListVMs(ctx context.Context, subscriptionID string) ([]models.AzureVM, error)
	GetVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) (*models.AzureVM, error)
	StartVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error
	StopVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error
	DeleteVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error
	ListStorageAccounts(ctx context.Context, subscriptionID string) ([]models.AzureStorageAccount, error)
	GetStorageAccount(ctx context.Context, subscriptionID, resourceGroup, name string) (*models.AzureStorageAccount, error)
	DeleteStorageAccount(ctx context.Context, subscriptionID, resourceGroup, name string) error
	// Cache methods
	ClearCache()
	InvalidateVMsCache()
	InvalidateStorageCache()
}

type SSOServiceInterface interface {
	GetIdentityCenterInstance() (*models.SSOIdentityCenterInstance, error)
	ListSSOUsers() ([]models.SSOUser, error)
	ListSSOGroups() ([]models.SSOGroup, error)
	GetSSOUser(userID string) (*models.SSOUser, error)
	GetSSOGroup(groupID string) (*models.SSOGroup, error)
	ListGroupMembers(groupID string) ([]models.SSOGroupMember, error)
	ListAccountAssignments(accountID string) ([]models.SSOAccountAssignment, error)
	ListAccountAssignmentsForUser(userID string) ([]models.SSOAccountAssignment, error)
	ListAccountAssignmentsForGroup(groupID string) ([]models.SSOAccountAssignment, error)
	ListAllUserAssignments() ([]models.SSOUserWithAssignments, error)
	ListAllGroupAssignments() ([]models.SSOGroupWithAssignments, error)
	ListAllAccountAssignments() ([]models.SSOAccountWithAssignments, error)
	// Cache management methods
	ClearCache()
	InvalidateSSOUsersCache()
	InvalidateSSOGroupsCache()
	InvalidateSSOUserCache(userID string)
	InvalidateSSOGroupCache(groupID string)
	InvalidateAccountAssignmentsCache(accountID string)
}
