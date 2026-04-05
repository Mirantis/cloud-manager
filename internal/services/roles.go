package services

import (
	"fmt"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// ============================================================================
// IAM ROLE MANAGEMENT
// ============================================================================

func (s *AWSService) ListRoles(accountID string) ([]models.IAMRole, error) {
	cacheKey := fmt.Sprintf("roles:%s", accountID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if roles, ok := cached.([]models.IAMRole); ok {
			return roles, nil
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		fmt.Printf("[WARNING] Cannot access account %s, skipping role listing: %v\n", accountID, err)
		return []models.IAMRole{}, nil
	}

	iamClient := iam.New(sess)
	var roles []models.IAMRole
	var marker *string

	// Get account name
	accounts, _ := s.ListAccounts()
	accountName := accountID
	for _, acc := range accounts {
		if acc.ID == accountID {
			accountName = acc.Name
			break
		}
	}

	// Paginate through all roles
	for {
		input := &iam.ListRolesInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := iamClient.ListRoles(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}

		// Process roles from this page
		for _, role := range result.Roles {
			roleDetail, err := s.getRoleDetails(iamClient, role, accountID, accountName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get details for role %s: %v\n", *role.RoleName, err)
				continue
			}
			roles = append(roles, roleDetail)
		}

		// Check if there are more pages
		if !*result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	// Cache the result
	s.cache.Set(cacheKey, roles, s.cacheTTL)

	return roles, nil
}

// ListAllRoles returns all IAM roles from all accessible accounts in parallel
func (s *AWSService) ListAllRoles() ([]models.RoleWithAccount, error) {
	const cacheKey = "all-roles"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if roles, ok := cached.([]models.RoleWithAccount); ok {
			return roles, nil
		}
	}

	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	// Filter accessible accounts
	var accessibleAccounts []models.Account
	for _, account := range accounts {
		if account.Accessible {
			accessibleAccounts = append(accessibleAccounts, account)
		}
	}

	if len(accessibleAccounts) == 0 {
		return []models.RoleWithAccount{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		roles     []models.RoleWithAccount
		err       error
		accountID string
	}

	resultChan := make(chan accountResult, len(accessibleAccounts))
	var wg sync.WaitGroup

	// Process each account in parallel
	for _, account := range accessibleAccounts {
		wg.Add(1)
		go func(acc models.Account) {
			defer wg.Done()

			roles, err := s.ListRoles(acc.ID)
			if err != nil {
				resultChan <- accountResult{
					err:       err,
					accountID: acc.ID,
				}
				return
			}

			// Convert to RoleWithAccount
			var rolesWithAccount []models.RoleWithAccount
			for _, role := range roles {
				rolesWithAccount = append(rolesWithAccount, models.RoleWithAccount{
					IAMRole:    role,
				})
			}

			resultChan <- accountResult{
				roles:     rolesWithAccount,
				accountID: acc.ID,
			}
		}(account)
	}

	// Wait for all goroutines to complete and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allRoles []models.RoleWithAccount
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get roles for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allRoles = append(allRoles, result.roles...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allRoles, s.cacheTTL)

	return allRoles, nil
}

// GetRole returns details for a specific IAM role
func (s *AWSService) GetRole(accountID, roleName string) (*models.IAMRole, error) {
	cacheKey := fmt.Sprintf("role:%s:%s", accountID, roleName)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if role, ok := cached.(*models.IAMRole); ok {
			return role, nil
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// Get role
	result, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Get account name
	accounts, _ := s.ListAccounts()
	accountName := accountID
	for _, acc := range accounts {
		if acc.ID == accountID {
			accountName = acc.Name
			break
		}
	}

	roleDetail, err := s.getRoleDetails(iamClient, result.Role, accountID, accountName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, &roleDetail, s.cacheTTL)

	return &roleDetail, nil
}

// getRoleDetails gets detailed information about a specific IAM role
func (s *AWSService) getRoleDetails(iamClient *iam.IAM, role *iam.Role, accountID, accountName string) (models.IAMRole, error) {
	roleName := *role.RoleName

	roleModel := models.IAMRole{
		RoleName:     roleName,
		RoleID:       *role.RoleId,
		Arn:          *role.Arn,
		AccountID:    accountID,
		AccountName:  accountName,
		CreateDate:   *role.CreateDate,
		Path:         *role.Path,
	}

	if role.Description != nil {
		roleModel.Description = *role.Description
	}
	if role.MaxSessionDuration != nil {
		roleModel.MaxSessionDuration = role.MaxSessionDuration
	}

	// Get assume role policy document
	if role.AssumeRolePolicyDocument != nil {
		roleModel.AssumeRolePolicyDocument = *role.AssumeRolePolicyDocument
	}

	// Get attached managed policies
	attachedPolicies, err := iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policy := range attachedPolicies.AttachedPolicies {
			attachedPolicy := models.AttachedPolicy{
				PolicyArn:  *policy.PolicyArn,
				PolicyName: *policy.PolicyName,
			}
			roleModel.AttachedManagedPolicies = append(roleModel.AttachedManagedPolicies, attachedPolicy)
		}
	}

	// Get inline policies
	inlinePolicies, err := iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policyName := range inlinePolicies.PolicyNames {
			// Get policy document
			policyResult, policyErr := iamClient.GetRolePolicy(&iam.GetRolePolicyInput{
				RoleName:   aws.String(roleName),
				PolicyName: policyName,
			})
			if policyErr == nil && policyResult.PolicyDocument != nil {
				roleModel.InlinePolicies = append(roleModel.InlinePolicies, models.InlinePolicy{
					PolicyName:     *policyName,
					PolicyDocument: *policyResult.PolicyDocument,
				})
			}
		}
	}

	// Get instance profiles
	instanceProfiles, err := iamClient.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, profile := range instanceProfiles.InstanceProfiles {
			roleModel.InstanceProfiles = append(roleModel.InstanceProfiles, *profile.InstanceProfileName)
		}
	}

	// Get tags
	tagsResult, err := iamClient.ListRoleTags(&iam.ListRoleTagsInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, tag := range tagsResult.Tags {
			roleModel.Tags = append(roleModel.Tags, models.Tag{
				Key:   *tag.Key,
				Value: *tag.Value,
			})
		}
	}

	// Get last used information
	lastUsedResult, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err == nil && lastUsedResult.Role.RoleLastUsed != nil {
		if lastUsedResult.Role.RoleLastUsed.LastUsedDate != nil {
			roleModel.LastUsedDate = lastUsedResult.Role.RoleLastUsed.LastUsedDate
		}
		if lastUsedResult.Role.RoleLastUsed.Region != nil {
			roleModel.LastUsedRegion = *lastUsedResult.Role.RoleLastUsed.Region
		}
	}

	return roleModel, nil
}

// DeleteRole deletes an IAM role
func (s *AWSService) DeleteRole(accountID, roleName string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// First, detach all managed policies
	attachedPolicies, err := iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policy := range attachedPolicies.AttachedPolicies {
			_, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				return fmt.Errorf("failed to detach policy %s: %v", *policy.PolicyArn, err)
			}
		}
	}

	// Delete all inline policies
	inlinePolicies, err := iamClient.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policyName := range inlinePolicies.PolicyNames {
			_, err = iamClient.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				RoleName:   aws.String(roleName),
				PolicyName: policyName,
			})
			if err != nil {
				return fmt.Errorf("failed to delete inline policy %s: %v", *policyName, err)
			}
		}
	}

	// Remove role from instance profiles
	instanceProfiles, err := iamClient.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, profile := range instanceProfiles.InstanceProfiles {
			_, err = iamClient.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: profile.InstanceProfileName,
				RoleName:            aws.String(roleName),
			})
			if err != nil {
				return fmt.Errorf("failed to remove role from instance profile %s: %v", *profile.InstanceProfileName, err)
			}
		}
	}

	// Finally, delete the role
	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	// Invalidate cache
	s.InvalidateAccountRolesCache(accountID)
	s.InvalidateRolesCache()

	return nil
}

// InvalidateRolesCache invalidates the roles cache
func (s *AWSService) InvalidateRolesCache() {
	s.cache.Delete("all-roles")
	s.cache.DeletePattern("roles:")
	s.cache.DeletePattern("role:")
}

// InvalidateAccountRolesCache invalidates roles cache for a specific account
func (s *AWSService) InvalidateAccountRolesCache(accountID string) {
	s.cache.Delete(fmt.Sprintf("roles:%s", accountID))
	s.cache.DeletePattern(fmt.Sprintf("role:%s:", accountID))
	s.cache.Delete("all-roles")
}