// Package services provides business logic and external service integrations
package services

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/config"
	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// AWSService provides AWS service integration
type AWSService struct {
	masterSession *session.Session
	config        config.Config
	cache         *Cache
	cacheTTL      time.Duration
}

// NewAWSService creates a new AWS service instance
func NewAWSService(cfg config.Config) *AWSService {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	// Create AWS config
	awsConfig := &aws.Config{
		Region:     aws.String(region),
		MaxRetries: aws.Int(1),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Only use static credentials if both access key and secret key are provided
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKey != "" && secretKey != "" {
		// Use static credentials
		awsConfig.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
	}
	// Otherwise, let the SDK use the default credential chain

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create AWS session: %v", err))
	}

	return &AWSService{
		masterSession: sess,
		config:        cfg,
		cache:         NewCache(),
		cacheTTL:      5 * time.Minute, // Default 5 minute TTL
	}
}

// getSessionForAccount gets a session for a specific AWS account by assuming a role
func (s *AWSService) getSessionForAccount(accountID string) (*session.Session, error) {
	if accountID == "" {
		return s.masterSession, nil
	}

	stsClient := sts.New(s.masterSession)
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, s.config.RoleName)

	// Generate ExternalId as per CloudFormation template: {accountId}-iam-manager
	externalId := fmt.Sprintf("%s-iam-manager", accountID)

	result, err := stsClient.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String("IAMManager"),
		DurationSeconds: aws.Int64(3600),
		ExternalId:      aws.String(externalId),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %v", err)
	}

	creds := result.Credentials
	sess, err := session.NewSession(&aws.Config{
		Region: s.masterSession.Config.Region,
		Credentials: credentials.NewStaticCredentials(
			*creds.AccessKeyId,
			*creds.SecretAccessKey,
			*creds.SessionToken,
		),
		MaxRetries: aws.Int(1),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session with assumed role: %v", err)
	}

	return sess, nil
}

// ============================================================================
// CACHE MANAGEMENT
// ============================================================================

// ClearCache clears all cached data
func (s *AWSService) ClearCache() {
	s.cache.Clear()
}

// InvalidateUserCache removes a specific user from the cache without invalidating the entire account cache
func (s *AWSService) InvalidateUserCache(accountID, username string) {
	// Update the users cache by removing the specific user
	cacheKey := fmt.Sprintf("users:%s", accountID)
	if cached, found := s.cache.Get(cacheKey); found {
		if users, ok := cached.([]models.User); ok {
			var updatedUsers []models.User
			for _, user := range users {
				if user.Username != username {
					updatedUsers = append(updatedUsers, user)
				}
			}
			s.cache.Set(cacheKey, updatedUsers, s.cacheTTL)
		}
	}
	// Delete the individual user cache entry
	s.cache.Delete(fmt.Sprintf("user:%s:%s", accountID, username))
}

// InvalidatePublicIPsCache invalidates the public IPs cache
func (s *AWSService) InvalidatePublicIPsCache() {
	s.cache.Delete("public-ips")
}

// InvalidateSecurityGroupsCache invalidates the security groups cache
func (s *AWSService) InvalidateSecurityGroupsCache() {
	s.cache.Delete("security-groups")
	// Also invalidate all account-specific security group caches
	s.cache.DeletePattern("security-groups:")
	// Also invalidate all individual security group caches
	s.cache.DeletePattern("security-group:")
}

// InvalidateAccountSecurityGroupsCache invalidates security groups cache for a specific account
func (s *AWSService) InvalidateAccountSecurityGroupsCache(accountID string) {
	s.cache.Delete(fmt.Sprintf("security-groups:%s", accountID))
	// Also invalidate individual security group caches for this account
	s.cache.DeletePattern(fmt.Sprintf("security-group:%s:", accountID))
	// Also invalidate the global cache since it contains this account's data
	s.cache.Delete("security-groups")
}

// InvalidateSecurityGroupCache invalidates cache for a specific security group
func (s *AWSService) InvalidateSecurityGroupCache(accountID, region, groupID string) {
	s.cache.Delete(fmt.Sprintf("security-group:%s:%s:%s", accountID, region, groupID))
	// Also invalidate broader caches that contain this security group
	s.cache.Delete(fmt.Sprintf("security-groups:%s", accountID))
	s.cache.Delete("security-groups")
}
