package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ============================================================================
// SNAPSHOT MANAGEMENT
// ============================================================================

func (s *AWSService) ListSnapshots() ([]models.Snapshot, error) {
	const cacheKey = "snapshots"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if snapshots, ok := cached.([]models.Snapshot); ok {
			return snapshots, nil
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
		return []models.Snapshot{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		snapshots []models.Snapshot
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

			snapshots, err := s.ListSnapshotsByAccount(acc.ID)
			resultChan <- accountResult{
				snapshots: snapshots,
				err:       err,
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
	var allSnapshots []models.Snapshot
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get snapshots for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allSnapshots = append(allSnapshots, result.snapshots...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allSnapshots, s.cacheTTL)

	return allSnapshots, nil
}

// ListSnapshotsByAccount returns all EBS snapshots for a specific account
func (s *AWSService) ListSnapshotsByAccount(accountID string) ([]models.Snapshot, error) {
	cacheKey := fmt.Sprintf("snapshots:%s", accountID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if snapshots, ok := cached.([]models.Snapshot); ok {
			return snapshots, nil
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		fmt.Printf("[WARNING] Cannot access account %s, skipping snapshot listing: %v\n", accountID, err)
		return []models.Snapshot{}, nil
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

	// Get all regions
	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
		"sa-east-1", "ca-central-1",
	}

	var allSnapshots []models.Snapshot
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process each region in parallel
	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()

			regionSess := sess.Copy(&aws.Config{Region: aws.String(r)})
			ec2Client := ec2.New(regionSess)

			// List snapshots owned by this account
			input := &ec2.DescribeSnapshotsInput{
				OwnerIds: []*string{aws.String(accountID)},
			}

			result, err := ec2Client.DescribeSnapshots(input)
			if err != nil {
				fmt.Printf("[WARNING] Failed to list snapshots in %s for account %s: %v\n", r, accountID, err)
				return
			}

			var regionSnapshots []models.Snapshot
			for _, snap := range result.Snapshots {
				snapshot := models.Snapshot{
					SnapshotID:  aws.StringValue(snap.SnapshotId),
					VolumeID:    aws.StringValue(snap.VolumeId),
					VolumeSize:  aws.Int64Value(snap.VolumeSize),
					Description: aws.StringValue(snap.Description),
					State:       aws.StringValue(snap.State),
					Progress:    aws.StringValue(snap.Progress),
					OwnerID:     aws.StringValue(snap.OwnerId),
					Encrypted:   aws.BoolValue(snap.Encrypted),
					AccountID:   accountID,
					AccountName: accountName,
					Region:      r,
				}

				if snap.StartTime != nil {
					snapshot.StartTime = *snap.StartTime
				}

				// Convert tags
				for _, tag := range snap.Tags {
					snapshot.Tags = append(snapshot.Tags, models.Tag{
						Key:   aws.StringValue(tag.Key),
						Value: aws.StringValue(tag.Value),
					})
				}

				regionSnapshots = append(regionSnapshots, snapshot)
			}

			mu.Lock()
			allSnapshots = append(allSnapshots, regionSnapshots...)
			mu.Unlock()
		}(region)
	}

	wg.Wait()

	// Cache the result
	s.cache.Set(cacheKey, allSnapshots, s.cacheTTL)

	return allSnapshots, nil
}

// DeleteSnapshot deletes an EBS snapshot
func (s *AWSService) DeleteSnapshot(accountID, region, snapshotID string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	regionSess := sess.Copy(&aws.Config{Region: aws.String(region)})
	ec2Client := ec2.New(regionSess)

	_, err = ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapshotID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete snapshot %s: %v", snapshotID, err)
	}

	// Update cache - remove the deleted snapshot
	s.updateSnapshotCache(accountID, snapshotID)

	return nil
}

// updateSnapshotCache removes a deleted snapshot from the cache
func (s *AWSService) updateSnapshotCache(accountID, snapshotID string) {
	// Update account-specific cache
	cacheKey := fmt.Sprintf("snapshots:%s", accountID)
	if cached, found := s.cache.Get(cacheKey); found {
		if snapshots, ok := cached.([]models.Snapshot); ok {
			var updated []models.Snapshot
			for _, snap := range snapshots {
				if snap.SnapshotID != snapshotID {
					updated = append(updated, snap)
				}
			}
			s.cache.Set(cacheKey, updated, s.cacheTTL)
		}
	}

	// Update global cache
	if cached, found := s.cache.Get("snapshots"); found {
		if snapshots, ok := cached.([]models.Snapshot); ok {
			var updated []models.Snapshot
			for _, snap := range snapshots {
				if snap.SnapshotID != snapshotID {
					updated = append(updated, snap)
				}
			}
			s.cache.Set("snapshots", updated, s.cacheTTL)
		}
	}
}

// DeleteOldSnapshots deletes all snapshots older than the specified number of months for an account
func (s *AWSService) DeleteOldSnapshots(accountID string, olderThanMonths int) ([]string, error) {
	// Get all snapshots for the account
	snapshots, err := s.ListSnapshotsByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, -olderThanMonths, 0)

	// Filter snapshots older than cutoff
	var oldSnapshots []models.Snapshot
	for _, snap := range snapshots {
		if snap.StartTime.Before(cutoffDate) && snap.State == "completed" {
			oldSnapshots = append(oldSnapshots, snap)
		}
	}

	if len(oldSnapshots) == 0 {
		return []string{}, nil
	}

	// Delete snapshots in parallel
	type deleteResult struct {
		snapshotID string
		err        error
	}

	resultChan := make(chan deleteResult, len(oldSnapshots))
	var wg sync.WaitGroup

	for _, snap := range oldSnapshots {
		wg.Add(1)
		go func(snapshot models.Snapshot) {
			defer wg.Done()
			err := s.DeleteSnapshot(accountID, snapshot.Region, snapshot.SnapshotID)
			resultChan <- deleteResult{
				snapshotID: snapshot.SnapshotID,
				err:        err,
			}
		}(snap)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var deletedSnapshots []string
	var errors []error
	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("failed to delete snapshot %s: %w", result.snapshotID, result.err))
		} else {
			deletedSnapshots = append(deletedSnapshots, result.snapshotID)
		}
	}

	// If there were any errors, return them
	if len(errors) > 0 {
		errMsg := fmt.Sprintf("deleted %d snapshots, but encountered %d errors", len(deletedSnapshots), len(errors))
		for _, e := range errors {
			errMsg += "; " + e.Error()
		}
		return deletedSnapshots, fmt.Errorf("%s", errMsg)
	}

	return deletedSnapshots, nil
}

// InvalidateSnapshotsCache invalidates the snapshots cache
func (s *AWSService) InvalidateSnapshotsCache() {
	s.cache.Delete("snapshots")
	s.cache.DeletePattern("snapshots:")
}

// ListEC2Instances returns all EC2 instances from all accessible accounts
