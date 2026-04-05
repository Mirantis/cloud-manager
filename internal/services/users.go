package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// ============================================================================
// USER MANAGEMENT
// ============================================================================

func (s *AWSService) ListUsers(accountID string) ([]models.User, error) {
	cacheKey := fmt.Sprintf("users:%s", accountID)
	
	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.User); ok {
			return users, nil
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		fmt.Printf("[WARNING] Cannot access account %s, skipping user listing: %v\n", accountID, err)
		// Return empty list instead of error for inaccessible accounts
		return []models.User{}, nil
	}

	iamClient := iam.New(sess)
	var users []models.User
	var marker *string

	// Paginate through all users
	for {
		input := &iam.ListUsersInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := iamClient.ListUsers(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		// Process users from this page
		for _, user := range result.Users {
		// Check if user has password
		passwordSet := false
		_, err := iamClient.GetLoginProfile(&iam.GetLoginProfileInput{
			UserName: user.UserName,
		})
		if err == nil {
			passwordSet = true
		}

		// Get password last used date
		var passwordLastUsed *time.Time
		if user.PasswordLastUsed != nil {
			passwordLastUsed = user.PasswordLastUsed
		}

		// Get access keys
		accessKeys, err := s.getAccessKeysForUser(iamClient, user.UserName)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get access keys for user %s: %v\n", *user.UserName, err)
			accessKeys = []models.AccessKey{}
		}

		users = append(users, models.User{
			Username:         *user.UserName,
			UserID:           *user.UserId,
			Arn:              *user.Arn,
			CreateDate:       *user.CreateDate,
			PasswordSet:      passwordSet,
			PasswordLastUsed: passwordLastUsed,
			AccessKeys:       accessKeys,
		})
		}

		// Check if there are more pages
		if !*result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	// Cache the result
	s.cache.Set(cacheKey, users, s.cacheTTL)
	
	return users, nil
}

// ListAllUsers returns all users from all accessible accounts in parallel
func (s *AWSService) ListAllUsers() ([]models.UserWithAccount, error) {
	const cacheKey = "all-users"
	
	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.UserWithAccount); ok {
			return users, nil
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
		return []models.UserWithAccount{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		users     []models.UserWithAccount
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
			
			users, err := s.ListUsers(acc.ID)
			if err != nil {
				resultChan <- accountResult{
					err:       err,
					accountID: acc.ID,
				}
				return
			}

			// Convert to UserWithAccount
			var usersWithAccount []models.UserWithAccount
			for _, user := range users {
				usersWithAccount = append(usersWithAccount, models.UserWithAccount{
					User:        user,
					AccountID:   acc.ID,
					AccountName: acc.Name,
				})
			}

			resultChan <- accountResult{
				users:     usersWithAccount,
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
	var allUsers []models.UserWithAccount
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get users for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allUsers = append(allUsers, result.users...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allUsers, s.cacheTTL)

	return allUsers, nil
}

func (s *AWSService) GetUser(accountID, username string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:%s:%s", accountID, username)
	
	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if user, ok := cached.(*models.User); ok {
			return user, nil
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)
	result, err := iamClient.GetUser(&iam.GetUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}

	user := result.User

	// Check if user has password
	passwordSet := false
	var passwordLastUsed *time.Time
	_, err = iamClient.GetLoginProfile(&iam.GetLoginProfileInput{
		UserName: user.UserName,
	})
	if err == nil {
		passwordSet = true
	}

	// Get password last used date
	if user.PasswordLastUsed != nil {
		passwordLastUsed = user.PasswordLastUsed
	}

	// Get access keys
	accessKeys, err := s.getAccessKeysForUser(iamClient, user.UserName)
	if err != nil {
		fmt.Printf("[WARNING] Failed to get access keys for user %s: %v\n", *user.UserName, err)
		accessKeys = []models.AccessKey{}
	}

	userResponse := &models.User{
		Username:         *user.UserName,
		UserID:           *user.UserId,
		Arn:              *user.Arn,
		CreateDate:       *user.CreateDate,
		PasswordSet:      passwordSet,
		PasswordLastUsed: passwordLastUsed,
		AccessKeys:       accessKeys,
	}

	// Cache the result
	s.cache.Set(cacheKey, userResponse, s.cacheTTL)

	return userResponse, nil
}

func (s *AWSService) CreateAccessKey(accountID, username string) (map[string]any, error) {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)
	result, err := iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}

	// Invalidate related cache entries
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
	s.cache.Delete(fmt.Sprintf("users:%s", accountID))
	// Also invalidate accounts cache in case it includes user/key counts
	s.cache.Delete("accounts")

	key := result.AccessKey
	response := map[string]any{
		"access_key_id":     *key.AccessKeyId,
		"secret_access_key": *key.SecretAccessKey,
		"status":            *key.Status,
		"create_date":       *key.CreateDate,
	}

	return response, nil
}

func (s *AWSService) DeleteAccessKey(accountID, username, keyID string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)
	_, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
		UserName:    aws.String(username),
		AccessKeyId: aws.String(keyID),
	})
	
	if err == nil {
		// Invalidate related cache entries
		s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
		s.cache.Delete(fmt.Sprintf("users:%s", accountID))
		// Also invalidate accounts cache in case it includes user/key counts
		s.cache.Delete("accounts")
	}
	
	return err
}

func (s *AWSService) RotateAccessKey(accountID, username, keyID string) (map[string]any, error) {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// Create new key
	createResult, err := iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}

	// Delete old key
	_, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
		UserName:    aws.String(username),
		AccessKeyId: aws.String(keyID),
	})
	if err != nil {
		// If deletion fails, try to delete the new key we just created
		_, _ = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			UserName:    aws.String(username),
			AccessKeyId: createResult.AccessKey.AccessKeyId,
		})
		return nil, fmt.Errorf("failed to delete old key: %v", err)
	}

	// Invalidate related cache entries
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
	s.cache.Delete(fmt.Sprintf("users:%s", accountID))
	// Also invalidate accounts cache in case it includes user/key counts
	s.cache.Delete("accounts")

	key := createResult.AccessKey
	response := map[string]any{
		"access_key_id":     *key.AccessKeyId,
		"secret_access_key": *key.SecretAccessKey,
		"status":            *key.Status,
		"create_date":       *key.CreateDate,
		"message":           "Access key rotated successfully",
	}

	return response, nil
}

func (s *AWSService) DeleteUser(accountID, username string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// First, delete all access keys for the user
	keysResult, err := iamClient.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list access keys: %v", err)
	}

	for _, key := range keysResult.AccessKeyMetadata {
		_, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			UserName:    aws.String(username),
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete access key %s: %v", *key.AccessKeyId, err)
		}
	}

	// Remove user from all groups
	groupsResult, err := iamClient.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list groups for user: %v", err)
	}

	for _, group := range groupsResult.Groups {
		_, err = iamClient.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
			UserName:  aws.String(username),
			GroupName: group.GroupName,
		})
		if err != nil {
			return fmt.Errorf("failed to remove user from group %s: %v", *group.GroupName, err)
		}
	}

	// Detach all managed policies from the user
	attachedPolicies, err := iamClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %v", err)
	}

	for _, policy := range attachedPolicies.AttachedPolicies {
		_, err = iamClient.DetachUserPolicy(&iam.DetachUserPolicyInput{
			UserName:  aws.String(username),
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			return fmt.Errorf("failed to detach policy %s: %v", *policy.PolicyName, err)
		}
	}

	// Delete all inline policies from the user
	inlinePolicies, err := iamClient.ListUserPolicies(&iam.ListUserPoliciesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list inline policies: %v", err)
	}

	for _, policyName := range inlinePolicies.PolicyNames {
		_, err = iamClient.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
			UserName:   aws.String(username),
			PolicyName: policyName,
		})
		if err != nil {
			return fmt.Errorf("failed to delete inline policy %s: %v", *policyName, err)
		}
	}

	// Deactivate and delete all MFA devices
	mfaDevices, err := iamClient.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list MFA devices: %v", err)
	}

	for _, device := range mfaDevices.MFADevices {
		// Deactivate the MFA device first
		_, err = iamClient.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
			UserName:     aws.String(username),
			SerialNumber: device.SerialNumber,
		})
		if err != nil {
			return fmt.Errorf("failed to deactivate MFA device %s: %v", *device.SerialNumber, err)
		}
		// Delete virtual MFA device if it's a virtual one (ARN contains "mfa/")
		if device.SerialNumber != nil && strings.Contains(*device.SerialNumber, ":mfa/") {
			_, _ = iamClient.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
				SerialNumber: device.SerialNumber,
			})
			// Ignore errors for virtual MFA deletion as it may not be virtual
		}
	}

	// Delete all SSH public keys
	sshKeys, err := iamClient.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list SSH public keys: %v", err)
	}

	for _, sshKey := range sshKeys.SSHPublicKeys {
		_, err = iamClient.DeleteSSHPublicKey(&iam.DeleteSSHPublicKeyInput{
			UserName:       aws.String(username),
			SSHPublicKeyId: sshKey.SSHPublicKeyId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete SSH public key %s: %v", *sshKey.SSHPublicKeyId, err)
		}
	}

	// Delete all signing certificates
	signingCerts, err := iamClient.ListSigningCertificates(&iam.ListSigningCertificatesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list signing certificates: %v", err)
	}

	for _, cert := range signingCerts.Certificates {
		_, err = iamClient.DeleteSigningCertificate(&iam.DeleteSigningCertificateInput{
			UserName:      aws.String(username),
			CertificateId: cert.CertificateId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete signing certificate %s: %v", *cert.CertificateId, err)
		}
	}

	// Delete all service-specific credentials
	serviceCredentials, err := iamClient.ListServiceSpecificCredentials(&iam.ListServiceSpecificCredentialsInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to list service-specific credentials: %v", err)
	}

	for _, cred := range serviceCredentials.ServiceSpecificCredentials {
		_, err = iamClient.DeleteServiceSpecificCredential(&iam.DeleteServiceSpecificCredentialInput{
			UserName:                    aws.String(username),
			ServiceSpecificCredentialId: cred.ServiceSpecificCredentialId,
		})
		if err != nil {
			return fmt.Errorf("failed to delete service-specific credential %s: %v", *cred.ServiceSpecificCredentialId, err)
		}
	}

	// Delete the login profile if it exists (ignore error if missing)
	_, _ = iamClient.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
		UserName: aws.String(username),
	})

	// Finally, delete the user
	_, err = iamClient.DeleteUser(&iam.DeleteUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	// Update the users cache by removing the deleted user instead of invalidating
	cacheKey := fmt.Sprintf("users:%s", accountID)
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.User); ok {
			// Filter out the deleted user
			var updatedUsers []models.User
			for _, user := range users {
				if user.Username != username {
					updatedUsers = append(updatedUsers, user)
				}
			}
			// Update the cache with the filtered list
			s.cache.Set(cacheKey, updatedUsers, s.cacheTTL)
		}
	}
	
	// Also update the all-users cache
	if cached, found := s.cache.Get("all-users"); found {
		if allUsers, ok := cached.([]models.UserWithAccount); ok {
			var updatedAllUsers []models.UserWithAccount
			for _, user := range allUsers {
				if user.AccountID != accountID || user.Username != username {
					updatedAllUsers = append(updatedAllUsers, user)
				}
			}
			s.cache.Set("all-users", updatedAllUsers, s.cacheTTL)
		}
	}
	
	// Delete the individual user cache entry
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))

	return nil
}

func (s *AWSService) DeleteUserPassword(accountID, username string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// Delete the login profile (console password)
	_, err = iamClient.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
		UserName: aws.String(username),
	})
	if err != nil {
		// Check if it's because the login profile doesn't exist
		if awsErr, ok := err.(interface{ Code() string }); ok && awsErr.Code() == "NoSuchEntity" {
			return fmt.Errorf("user %s does not have a console password", username)
		}
		return fmt.Errorf("failed to delete user password: %v", err)
	}

	// Invalidate related cache entries
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
	s.cache.Delete(fmt.Sprintf("users:%s", accountID))
	// Also invalidate accounts cache in case it includes user/password counts
	s.cache.Delete("accounts")

	return nil
}

func (s *AWSService) RotateUserPassword(accountID, username string) (map[string]any, error) {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	iamClient := iam.New(sess)

	// Check if user already has a login profile
	hasLoginProfile := true
	_, err = iamClient.GetLoginProfile(&iam.GetLoginProfileInput{
		UserName: aws.String(username),
	})
	if err != nil {
		if awsErr, ok := err.(interface{ Code() string }); ok && awsErr.Code() == "NoSuchEntity" {
			hasLoginProfile = false
		} else {
			return nil, fmt.Errorf("failed to check login profile: %v", err)
		}
	}

	// Generate a new password
	newPassword := s.generatePassword()

	if hasLoginProfile {
		// Update existing login profile
		_, err = iamClient.UpdateLoginProfile(&iam.UpdateLoginProfileInput{
			UserName: aws.String(username),
			Password: aws.String(newPassword),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update user password: %v", err)
		}
	} else {
		// Create new login profile
		_, err = iamClient.CreateLoginProfile(&iam.CreateLoginProfileInput{
			UserName: aws.String(username),
			Password: aws.String(newPassword),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user password: %v", err)
		}
	}

	// Invalidate related cache entries
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
	s.cache.Delete(fmt.Sprintf("users:%s", accountID))
	// Also invalidate accounts cache in case it includes user/password counts
	s.cache.Delete("accounts")

	response := map[string]any{
		"message":      "User password rotated successfully",
		"new_password": newPassword,
		"username":     username,
	}

	return response, nil
}

func (s *AWSService) getAccessKeysForUser(iamClient *iam.IAM, userName *string) ([]models.AccessKey, error) {
	keysResult, err := iamClient.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: userName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list access keys: %v", err)
	}

	var accessKeys []models.AccessKey
	for _, key := range keysResult.AccessKeyMetadata {
		accessKey := models.AccessKey{
			AccessKeyID: *key.AccessKeyId,
			Status:      *key.Status,
			CreateDate:  *key.CreateDate,
		}

		// Get last used information
		lastUsedResult, err := iamClient.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			fmt.Printf("[WARNING] Failed to get last used info for key %s: %v\n", *key.AccessKeyId, err)
		} else if lastUsedResult.AccessKeyLastUsed != nil && lastUsedResult.AccessKeyLastUsed.LastUsedDate != nil {
			accessKey.LastUsedDate = lastUsedResult.AccessKeyLastUsed.LastUsedDate
			if lastUsedResult.AccessKeyLastUsed.ServiceName != nil {
				accessKey.LastUsedService = *lastUsedResult.AccessKeyLastUsed.ServiceName
			}
			if lastUsedResult.AccessKeyLastUsed.Region != nil {
				accessKey.LastUsedRegion = *lastUsedResult.AccessKeyLastUsed.Region
			}
		}

		accessKeys = append(accessKeys, accessKey)
	}

	return accessKeys, nil
}

// DeleteInactiveUsers deletes users that have been inactive for 6+ months
// A user is considered inactive if:
// - PasswordLastUsed is nil or older than 6 months, AND
// - All active access keys have LastUsedDate nil or older than 6 months
func (s *AWSService) DeleteInactiveUsers(accountID string) ([]string, []string, error) {
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	
	users, err := s.ListUsers(accountID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list users: %w", err)
	}

	var deletedUsers []string
	var failedUsers []string

	for _, user := range users {
		// Check if user is inactive
		isInactive := true

		// Check password last used - if password was used recently, user is active
		if user.PasswordSet && user.PasswordLastUsed != nil {
			if user.PasswordLastUsed.After(sixMonthsAgo) {
				continue
			}
		}

		// Check access keys last used - if any active key was used recently, user is active
		if len(user.AccessKeys) > 0 {
			for _, key := range user.AccessKeys {
				if key.Status == "Active" {
					// If key was used recently, user is active
					if key.LastUsedDate != nil && key.LastUsedDate.After(sixMonthsAgo) {
						isInactive = false
						break
					}
				}
			}
			// If user has active keys but none were used recently (or never used), they're still inactive
		}

		// If user is inactive, delete them
		if isInactive {
			err := s.DeleteUser(accountID, user.Username)
			if err != nil {
				failedUsers = append(failedUsers, fmt.Sprintf("%s: %v", user.Username, err))
			} else {
				deletedUsers = append(deletedUsers, user.Username)
			}
		}
	}

	return deletedUsers, failedUsers, nil
}
