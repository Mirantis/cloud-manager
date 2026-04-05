// Package services provides business logic and external service integrations
package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	identitystoretypes "github.com/aws/aws-sdk-go-v2/service/identitystore/types"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	ssoadmintypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
)

// SSOService handles AWS SSO (IAM Identity Center) operations
type SSOService struct {
	cfg              aws.Config
	identityStoreID  string
	instanceARN      string
	identityClient   *identitystore.Client
	ssoAdminClient   *ssoadmin.Client
	cache            *Cache
	cacheTTL         time.Duration
	accountsService  *AWSService // For getting account names
}

// NewSSOService creates a new SSO service instance
func NewSSOService(accountsService *AWSService) (*SSOService, error) {
	// Use separate region for SSO management (IAM Identity Center)
	// Defaults to eu-west-2 (London) as specified
	region := os.Getenv("AWS_SSO_REGION")
	if region == "" {
		region = "eu-west-2" // Default to London as specified
	}

	// Load AWS config using SDK v2
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	service := &SSOService{
		cfg:             cfg,
		identityClient:  identitystore.NewFromConfig(cfg),
		ssoAdminClient:  ssoadmin.NewFromConfig(cfg),
		cache:           NewCache(),
		cacheTTL:        5 * time.Minute,
		accountsService: accountsService,
	}

	// Discover Identity Center instance
	instance, err := service.GetIdentityCenterInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to discover Identity Center instance: %w", err)
	}

	service.instanceARN = instance.InstanceARN
	service.identityStoreID = instance.IdentityStoreID

	return service, nil
}

// GetIdentityCenterInstance discovers the Identity Center instance ARN and Identity Store ID
func (s *SSOService) GetIdentityCenterInstance() (*models.SSOIdentityCenterInstance, error) {
	const cacheKey = "sso-instance"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if instance, ok := cached.(*models.SSOIdentityCenterInstance); ok {
			return instance, nil
		}
	}

	ctx := context.TODO()

	// List instances (should only be one per region)
	result, err := s.ssoAdminClient.ListInstances(ctx, &ssoadmin.ListInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Identity Center instances: %w", err)
	}

	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no Identity Center instance found")
	}

	instance := result.Instances[0]
	identityCenterInstance := &models.SSOIdentityCenterInstance{
		InstanceARN:      *instance.InstanceArn,
		IdentityStoreID: *instance.IdentityStoreId,
	}

	// Cache the result
	s.cache.Set(cacheKey, identityCenterInstance, s.cacheTTL)

	return identityCenterInstance, nil
}

// ListSSOUsers lists all SSO users
func (s *SSOService) ListSSOUsers() ([]models.SSOUser, error) {
	const cacheKey = "sso-users"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.SSOUser); ok {
			return users, nil
		}
	}

	ctx := context.TODO()
	var allUsers []models.SSOUser
	var nextToken *string

	for {
		input := &identitystore.ListUsersInput{
			IdentityStoreId: aws.String(s.identityStoreID),
			MaxResults:      aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.identityClient.ListUsers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list SSO users: %w", err)
		}

		for _, user := range result.Users {
			ssoUser := models.SSOUser{
				UserID:      *user.UserId,
				UserName:    *user.UserName,
				DisplayName: getStringValue(user.DisplayName),
				Active:      true, // Users returned by ListUsers are active
			}

			// Extract emails
			if user.Emails != nil {
				for _, email := range user.Emails {
					if email.Value != nil {
						ssoUser.Emails = append(ssoUser.Emails, *email.Value)
					}
				}
			}

			allUsers = append(allUsers, ssoUser)
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, allUsers, s.cacheTTL)

	return allUsers, nil
}

// ListSSOGroups lists all SSO groups
func (s *SSOService) ListSSOGroups() ([]models.SSOGroup, error) {
	const cacheKey = "sso-groups"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if groups, ok := cached.([]models.SSOGroup); ok {
			return groups, nil
		}
	}

	ctx := context.TODO()
	var allGroups []models.SSOGroup
	var nextToken *string

	for {
		input := &identitystore.ListGroupsInput{
			IdentityStoreId: aws.String(s.identityStoreID),
			MaxResults:      aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.identityClient.ListGroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list SSO groups: %w", err)
		}

		for _, group := range result.Groups {
			ssoGroup := models.SSOGroup{
				GroupID:     *group.GroupId,
				DisplayName: *group.DisplayName,
				Description: getStringValue(group.Description),
			}

			allGroups = append(allGroups, ssoGroup)
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, allGroups, s.cacheTTL)

	return allGroups, nil
}

// GetSSOUser gets a specific SSO user
func (s *SSOService) GetSSOUser(userID string) (*models.SSOUser, error) {
	cacheKey := fmt.Sprintf("sso-user:%s", userID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if user, ok := cached.(*models.SSOUser); ok {
			return user, nil
		}
	}

	ctx := context.TODO()

	result, err := s.identityClient.DescribeUser(ctx, &identitystore.DescribeUserInput{
		IdentityStoreId: aws.String(s.identityStoreID),
		UserId:          aws.String(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO user: %w", err)
	}

	ssoUser := &models.SSOUser{
		UserID:      getStringValue(result.UserId),
		UserName:    getStringValue(result.UserName),
		DisplayName: getStringValue(result.DisplayName),
		Active:      true,
	}

	// Extract emails
	if result.Emails != nil {
		for _, email := range result.Emails {
			if email.Value != nil {
				ssoUser.Emails = append(ssoUser.Emails, *email.Value)
			}
		}
	}

	// Cache the result
	s.cache.Set(cacheKey, ssoUser, s.cacheTTL)

	return ssoUser, nil
}

// GetSSOGroup gets a specific SSO group
func (s *SSOService) GetSSOGroup(groupID string) (*models.SSOGroup, error) {
	cacheKey := fmt.Sprintf("sso-group:%s", groupID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if group, ok := cached.(*models.SSOGroup); ok {
			return group, nil
		}
	}

	ctx := context.TODO()

	result, err := s.identityClient.DescribeGroup(ctx, &identitystore.DescribeGroupInput{
		IdentityStoreId: aws.String(s.identityStoreID),
		GroupId:         aws.String(groupID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO group: %w", err)
	}

	ssoGroup := &models.SSOGroup{
		GroupID:     getStringValue(result.GroupId),
		DisplayName: getStringValue(result.DisplayName),
		Description: getStringValue(result.Description),
	}

	// Cache the result
	s.cache.Set(cacheKey, ssoGroup, s.cacheTTL)

	return ssoGroup, nil
}

// ListGroupMembers lists members of a group
func (s *SSOService) ListGroupMembers(groupID string) ([]models.SSOGroupMember, error) {
	cacheKey := fmt.Sprintf("sso-group-members:%s", groupID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if members, ok := cached.([]models.SSOGroupMember); ok {
			return members, nil
		}
	}

	ctx := context.TODO()
	var allMembers []models.SSOGroupMember
	var nextToken *string

	for {
		input := &identitystore.ListGroupMembershipsInput{
			IdentityStoreId: aws.String(s.identityStoreID),
			GroupId:         aws.String(groupID),
			MaxResults:      aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.identityClient.ListGroupMemberships(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list group members: %w", err)
		}

		for _, membership := range result.GroupMemberships {
			// MemberId is an interface - check if it's a UserId type (pointer type)
			memberIdUserId, ok := membership.MemberId.(*identitystoretypes.MemberIdMemberUserId)
			if !ok || memberIdUserId == nil {
				continue
			}

			userID := memberIdUserId.Value
			member := models.SSOGroupMember{
				MemberType: "USER",
				MemberID:   userID,
			}

			// Get user details for display name
			user, err := s.GetSSOUser(userID)
			if err == nil {
				member.UserName = user.UserName
				member.DisplayName = user.DisplayName
			}

			allMembers = append(allMembers, member)
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, allMembers, s.cacheTTL)

	return allMembers, nil
}

// ListAccountAssignments lists account assignments for a specific account
func (s *SSOService) ListAccountAssignments(accountID string) ([]models.SSOAccountAssignment, error) {
	cacheKey := fmt.Sprintf("sso-account-assignments:%s", accountID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if assignments, ok := cached.([]models.SSOAccountAssignment); ok {
			return assignments, nil
		}
	}

	ctx := context.TODO()

	// First, get all permission sets
	permissionSets, err := s.listPermissionSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list permission sets: %w", err)
	}

	var allAssignments []models.SSOAccountAssignment

	// Get account name
	accountName := s.getAccountName(accountID)

	// For each permission set, get account assignments
	for _, psArn := range permissionSets {
		var nextToken *string

		for {
			input := &ssoadmin.ListAccountAssignmentsInput{
				InstanceArn:      aws.String(s.instanceARN),
				AccountId:        aws.String(accountID),
				PermissionSetArn: aws.String(psArn),
				MaxResults:       aws.Int32(100),
			}
			if nextToken != nil {
				input.NextToken = nextToken
			}

			result, err := s.ssoAdminClient.ListAccountAssignments(ctx, input)
			if err != nil {
				// Continue with next permission set if this one fails
				fmt.Printf("[WARNING] Failed to list account assignments for permission set %s: %v\n", psArn, err)
				break
			}

			for _, assignment := range result.AccountAssignments {
				psName := s.getPermissionSetName(psArn)
				principalID := getStringValue(assignment.PrincipalId)
				principalName := s.getPrincipalName(principalID, string(assignment.PrincipalType))

				allAssignments = append(allAssignments, models.SSOAccountAssignment{
					AccountID:        accountID,
					AccountName:      accountName,
					PrincipalID:      *assignment.PrincipalId,
					PrincipalType:    string(assignment.PrincipalType),
					PrincipalName:    principalName,
					PermissionSetArn: psArn,
					PermissionSetName: psName,
				})
			}

			if result.NextToken == nil {
				break
			}
			nextToken = result.NextToken
		}
	}

	// Cache the result
	s.cache.Set(cacheKey, allAssignments, s.cacheTTL)

	return allAssignments, nil
}

// ListAccountAssignmentsForUser lists account assignments for a specific user
func (s *SSOService) ListAccountAssignmentsForUser(userID string) ([]models.SSOAccountAssignment, error) {
	cacheKey := fmt.Sprintf("sso-user-assignments:%s", userID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if assignments, ok := cached.([]models.SSOAccountAssignment); ok {
			return assignments, nil
		}
	}

	ctx := context.TODO()

	var allAssignments []models.SSOAccountAssignment
	var nextToken *string

	for {
		input := &ssoadmin.ListAccountAssignmentsForPrincipalInput{
			InstanceArn:   aws.String(s.instanceARN),
			PrincipalId:   aws.String(userID),
			PrincipalType: ssoadmintypes.PrincipalTypeUser,
			MaxResults:    aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ssoAdminClient.ListAccountAssignmentsForPrincipal(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list account assignments for user: %w", err)
		}

		for _, assignment := range result.AccountAssignments {
			accountName := s.getAccountName(*assignment.AccountId)
			psName := s.getPermissionSetName(*assignment.PermissionSetArn)
			user, _ := s.GetSSOUser(userID)

			principalName := user.UserName
			if user.DisplayName != "" {
				principalName = user.DisplayName
			}
			allAssignments = append(allAssignments, models.SSOAccountAssignment{
				AccountID:        *assignment.AccountId,
				AccountName:      accountName,
				PrincipalID:      userID,
				PrincipalType:    "USER",
				PrincipalName:    principalName,
				PermissionSetArn: *assignment.PermissionSetArn,
				PermissionSetName: psName,
			})
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, allAssignments, s.cacheTTL)

	return allAssignments, nil
}

// ListAccountAssignmentsForGroup lists account assignments for a specific group
func (s *SSOService) ListAccountAssignmentsForGroup(groupID string) ([]models.SSOAccountAssignment, error) {
	cacheKey := fmt.Sprintf("sso-group-assignments:%s", groupID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if assignments, ok := cached.([]models.SSOAccountAssignment); ok {
			return assignments, nil
		}
	}

	ctx := context.TODO()

	var allAssignments []models.SSOAccountAssignment
	var nextToken *string

	for {
		input := &ssoadmin.ListAccountAssignmentsForPrincipalInput{
			InstanceArn:   aws.String(s.instanceARN),
			PrincipalId:   aws.String(groupID),
			PrincipalType: ssoadmintypes.PrincipalTypeGroup,
			MaxResults:    aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ssoAdminClient.ListAccountAssignmentsForPrincipal(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list account assignments for group: %w", err)
		}

		for _, assignment := range result.AccountAssignments {
			accountName := s.getAccountName(*assignment.AccountId)
			psName := s.getPermissionSetName(*assignment.PermissionSetArn)
			group, _ := s.GetSSOGroup(groupID)

			allAssignments = append(allAssignments, models.SSOAccountAssignment{
				AccountID:        *assignment.AccountId,
				AccountName:      accountName,
				PrincipalID:      groupID,
				PrincipalType:    "GROUP",
				PrincipalName:    group.DisplayName,
				PermissionSetArn: *assignment.PermissionSetArn,
				PermissionSetName: psName,
			})
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, allAssignments, s.cacheTTL)

	return allAssignments, nil
}

// ListAllUserAssignments lists all users with their account assignments
func (s *SSOService) ListAllUserAssignments() ([]models.SSOUserWithAssignments, error) {
	const cacheKey = "sso-all-user-assignments"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.SSOUserWithAssignments); ok {
			return users, nil
		}
	}

	users, err := s.ListSSOUsers()
	if err != nil {
		return nil, err
	}

	var usersWithAssignments []models.SSOUserWithAssignments

	for _, user := range users {
		assignments, err := s.ListAccountAssignmentsForUser(user.UserID)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get assignments for user %s: %v\n", user.UserID, err)
			assignments = []models.SSOAccountAssignment{}
		}

		// Get group memberships
		groups, _ := s.ListSSOGroups()
		var groupMemberships []string
		for _, group := range groups {
			members, err := s.ListGroupMembers(group.GroupID)
			if err == nil {
				for _, member := range members {
					if member.MemberType == "USER" && member.MemberID == user.UserID {
						groupMemberships = append(groupMemberships, group.GroupID)
						break
					}
				}
			}
		}

		usersWithAssignments = append(usersWithAssignments, models.SSOUserWithAssignments{
			SSOUser:            user,
			AccountAssignments: assignments,
			GroupMemberships:   groupMemberships,
		})
	}

	// Cache the result
	s.cache.Set(cacheKey, usersWithAssignments, s.cacheTTL)

	return usersWithAssignments, nil
}

// ListAllGroupAssignments lists all groups with their account assignments
func (s *SSOService) ListAllGroupAssignments() ([]models.SSOGroupWithAssignments, error) {
	const cacheKey = "sso-all-group-assignments"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if groups, ok := cached.([]models.SSOGroupWithAssignments); ok {
			return groups, nil
		}
	}

	groups, err := s.ListSSOGroups()
	if err != nil {
		return nil, err
	}

	var groupsWithAssignments []models.SSOGroupWithAssignments

	for _, group := range groups {
		assignments, err := s.ListAccountAssignmentsForGroup(group.GroupID)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get assignments for group %s: %v\n", group.GroupID, err)
			assignments = []models.SSOAccountAssignment{}
		}

		members, err := s.ListGroupMembers(group.GroupID)
		if err != nil {
			members = []models.SSOGroupMember{}
		}

		groupsWithAssignments = append(groupsWithAssignments, models.SSOGroupWithAssignments{
			SSOGroup:           group,
			AccountAssignments: assignments,
			MemberCount:        len(members),
			Members:            members,
		})
	}

	// Cache the result
	s.cache.Set(cacheKey, groupsWithAssignments, s.cacheTTL)

	return groupsWithAssignments, nil
}

// ListAllAccountAssignments lists all accounts with their SSO assignments
func (s *SSOService) ListAllAccountAssignments() ([]models.SSOAccountWithAssignments, error) {
	const cacheKey = "sso-all-account-assignments"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if accounts, ok := cached.([]models.SSOAccountWithAssignments); ok {
			return accounts, nil
		}
	}

	// Get all accounts from the accounts service
	accounts, err := s.accountsService.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accountsWithAssignments []models.SSOAccountWithAssignments

	for _, account := range accounts {
		assignments, err := s.ListAccountAssignments(account.ID)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get assignments for account %s: %v\n", account.ID, err)
			assignments = []models.SSOAccountAssignment{}
		}

		accountsWithAssignments = append(accountsWithAssignments, models.SSOAccountWithAssignments{
			AccountID:   account.ID,
			AccountName: account.Name,
			Assignments: assignments,
		})
	}

	// Cache the result
	s.cache.Set(cacheKey, accountsWithAssignments, s.cacheTTL)

	return accountsWithAssignments, nil
}

// Helper functions

func (s *SSOService) listPermissionSets(ctx context.Context) ([]string, error) {
	cacheKey := "sso-permission-sets"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if ps, ok := cached.([]string); ok {
			return ps, nil
		}
	}

	var permissionSets []string
	var nextToken *string

	for {
		input := &ssoadmin.ListPermissionSetsInput{
			InstanceArn: aws.String(s.instanceARN),
			MaxResults:  aws.Int32(100),
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := s.ssoAdminClient.ListPermissionSets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list permission sets: %w", err)
		}

		permissionSets = append(permissionSets, result.PermissionSets...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Cache the result
	s.cache.Set(cacheKey, permissionSets, s.cacheTTL)

	return permissionSets, nil
}

func (s *SSOService) getPermissionSetName(arn string) string {
	cacheKey := fmt.Sprintf("sso-ps-name:%s", arn)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if name, ok := cached.(string); ok {
			return name
		}
	}

	ctx := context.TODO()

	result, err := s.ssoAdminClient.DescribePermissionSet(ctx, &ssoadmin.DescribePermissionSetInput{
		InstanceArn:      aws.String(s.instanceARN),
		PermissionSetArn: aws.String(arn),
	})
	if err != nil {
		// Extract name from ARN as fallback
		return extractNameFromARN(arn)
	}

	name := extractNameFromARN(arn)
	if result.PermissionSet.Name != nil {
		name = *result.PermissionSet.Name
	}

	// Cache the result
	s.cache.Set(cacheKey, name, s.cacheTTL)

	return name
}

func (s *SSOService) getPrincipalName(principalID, principalType string) string {
	switch principalType {
	case "USER":
		user, err := s.GetSSOUser(principalID)
		if err == nil {
			if user.DisplayName != "" {
				return user.DisplayName
			}
			return user.UserName
		}
	case "GROUP":
		group, err := s.GetSSOGroup(principalID)
		if err == nil {
			return group.DisplayName
		}
	}
	return principalID
}

func (s *SSOService) getAccountName(accountID string) string {
	if s.accountsService == nil {
		return ""
	}

	accounts, err := s.accountsService.ListAccounts()
	if err != nil {
		return ""
	}

	for _, account := range accounts {
		if account.ID == accountID {
			return account.Name
		}
	}

	return ""
}

// Cache management methods

func (s *SSOService) ClearCache() {
	s.cache.Clear()
}

func (s *SSOService) InvalidateSSOUsersCache() {
	s.cache.Delete("sso-users")
	s.cache.Delete("sso-all-user-assignments")
	s.cache.DeletePattern("sso-user:")
}

func (s *SSOService) InvalidateSSOGroupsCache() {
	s.cache.Delete("sso-groups")
	s.cache.Delete("sso-all-group-assignments")
	s.cache.DeletePattern("sso-group:")
	s.cache.DeletePattern("sso-group-members:")
}

func (s *SSOService) InvalidateSSOUserCache(userID string) {
	s.cache.Delete(fmt.Sprintf("sso-user:%s", userID))
	s.cache.Delete(fmt.Sprintf("sso-user-assignments:%s", userID))
	s.cache.Delete("sso-users")
	s.cache.Delete("sso-all-user-assignments")
}

func (s *SSOService) InvalidateSSOGroupCache(groupID string) {
	s.cache.Delete(fmt.Sprintf("sso-group:%s", groupID))
	s.cache.Delete(fmt.Sprintf("sso-group-assignments:%s", groupID))
	s.cache.Delete(fmt.Sprintf("sso-group-members:%s", groupID))
	s.cache.Delete("sso-groups")
	s.cache.Delete("sso-all-group-assignments")
}

func (s *SSOService) InvalidateAccountAssignmentsCache(accountID string) {
	s.cache.Delete(fmt.Sprintf("sso-account-assignments:%s", accountID))
	s.cache.Delete("sso-all-user-assignments")
	s.cache.Delete("sso-all-group-assignments")
	s.cache.DeletePattern("sso-user-assignments:")
	s.cache.DeletePattern("sso-group-assignments:")
}

// Utility functions
// Note: getStringValue is already defined in azure_service.go and can be used here

func extractNameFromARN(arn string) string {
	// Extract name from ARN like: arn:aws:sso:::permissionSet/ssoins-xxx/ps-xxx
	parts := splitARN(arn)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return arn
}

func splitARN(arn string) []string {
	var parts []string
	var current string
	for _, char := range arn {
		if char == '/' || char == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
