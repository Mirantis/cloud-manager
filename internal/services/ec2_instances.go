package services

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ============================================================================
// EC2 INSTANCE MANAGEMENT
// ============================================================================

// calculateEC2InstanceMonthlyCost calculates the monthly cost for an EC2 instance
// Based on AWS EC2 On-Demand Linux pricing (US East N. Virginia region as baseline)
// Pricing varies by region, but this provides a reasonable estimate
// Note: Only running instances incur costs, stopped instances cost $0
func calculateEC2InstanceMonthlyCost(instanceType, state string) float64 {
	// Stopped instances don't incur compute costs
	if state != "running" {
		return 0.0
	}

	// EC2 On-Demand Linux pricing per hour (US East N. Virginia baseline)
	// Prices are approximate and may vary by region
	hourlyPricing := map[string]float64{
		// t3 family (burstable performance)
		"t3.nano":     0.0052,
		"t3.micro":    0.0104,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"t3.xlarge":   0.1664,
		"t3.2xlarge":  0.3328,
		// t3a family
		"t3a.nano":    0.0047,
		"t3a.micro":   0.0094,
		"t3a.small":   0.0188,
		"t3a.medium":  0.0376,
		"t3a.large":   0.0752,
		"t3a.xlarge":  0.1504,
		"t3a.2xlarge": 0.3008,
		// t4g family (ARM-based)
		"t4g.nano":    0.0034,
		"t4g.micro":   0.0068,
		"t4g.small":   0.0136,
		"t4g.medium":  0.0272,
		"t4g.large":   0.0544,
		"t4g.xlarge":  0.1088,
		"t4g.2xlarge": 0.2176,
		// m5 family (general purpose)
		"m5.large":    0.096,
		"m5.xlarge":   0.192,
		"m5.2xlarge":  0.384,
		"m5.4xlarge":  0.768,
		"m5.8xlarge":  1.536,
		"m5.12xlarge": 2.304,
		"m5.16xlarge": 3.072,
		"m5.24xlarge": 4.608,
		// m5a family
		"m5a.large":    0.086,
		"m5a.xlarge":   0.172,
		"m5a.2xlarge":  0.344,
		"m5a.4xlarge":  0.688,
		"m5a.8xlarge":  1.376,
		"m5a.12xlarge": 2.064,
		"m5a.16xlarge": 2.752,
		"m5a.24xlarge": 4.128,
		// m6i family
		"m6i.large":    0.096,
		"m6i.xlarge":   0.192,
		"m6i.2xlarge":  0.384,
		"m6i.4xlarge":  0.768,
		"m6i.8xlarge":  1.536,
		"m6i.12xlarge": 2.304,
		"m6i.16xlarge": 3.072,
		"m6i.24xlarge": 4.608,
		"m6i.32xlarge": 6.144,
		// c5 family (compute optimized)
		"c5.large":    0.085,
		"c5.xlarge":   0.17,
		"c5.2xlarge":  0.34,
		"c5.4xlarge":  0.68,
		"c5.9xlarge":  1.53,
		"c5.12xlarge": 2.04,
		"c5.18xlarge": 3.06,
		"c5.24xlarge": 4.08,
		// c5a family
		"c5a.large":    0.077,
		"c5a.xlarge":   0.154,
		"c5a.2xlarge":  0.308,
		"c5a.4xlarge":  0.616,
		"c5a.8xlarge":  1.232,
		"c5a.12xlarge": 1.848,
		"c5a.16xlarge": 2.464,
		"c5a.24xlarge": 3.696,
		// c6i family
		"c6i.large":    0.085,
		"c6i.xlarge":   0.17,
		"c6i.2xlarge":  0.34,
		"c6i.4xlarge":  0.68,
		"c6i.8xlarge":  1.36,
		"c6i.12xlarge": 2.04,
		"c6i.16xlarge": 2.72,
		"c6i.24xlarge": 4.08,
		"c6i.32xlarge": 5.44,
		// r5 family (memory optimized)
		"r5.large":    0.126,
		"r5.xlarge":   0.252,
		"r5.2xlarge":  0.504,
		"r5.4xlarge":  1.008,
		"r5.8xlarge":  2.016,
		"r5.12xlarge": 3.024,
		"r5.16xlarge": 4.032,
		"r5.24xlarge": 6.048,
		// r5a family
		"r5a.large":    0.113,
		"r5a.xlarge":   0.226,
		"r5a.2xlarge":  0.452,
		"r5a.4xlarge":  0.904,
		"r5a.8xlarge":  1.808,
		"r5a.12xlarge": 2.712,
		"r5a.16xlarge": 3.616,
		"r5a.24xlarge": 5.424,
		// r6i family
		"r6i.large":    0.126,
		"r6i.xlarge":   0.252,
		"r6i.2xlarge":  0.504,
		"r6i.4xlarge":  1.008,
		"r6i.8xlarge":  2.016,
		"r6i.12xlarge": 3.024,
		"r6i.16xlarge": 4.032,
		"r6i.24xlarge": 6.048,
		"r6i.32xlarge": 8.064,
		// i3 family (storage optimized)
		"i3.large":    0.156,
		"i3.xlarge":   0.312,
		"i3.2xlarge":  0.624,
		"i3.4xlarge":  1.248,
		"i3.8xlarge":  2.496,
		"i3.16xlarge": 4.992,
		// i3en family
		"i3en.large":    0.216,
		"i3en.xlarge":   0.432,
		"i3en.2xlarge":  0.864,
		"i3en.3xlarge":  1.296,
		"i3en.6xlarge":  2.592,
		"i3en.12xlarge": 5.184,
		"i3en.24xlarge": 10.368,
		// g4dn family (GPU)
		"g4dn.xlarge":   0.526,
		"g4dn.2xlarge":  0.752,
		"g4dn.4xlarge":  1.204,
		"g4dn.8xlarge":  2.176,
		"g4dn.12xlarge": 3.108,
		"g4dn.16xlarge": 4.352,
		// p3 family (GPU)
		"p3.2xlarge":  3.06,
		"p3.8xlarge":  12.24,
		"p3.16xlarge": 24.48,
		// p4d family (GPU)
		"p4d.24xlarge": 32.77,
	}

	// Normalize instance type (lowercase)
	instanceTypeLower := strings.ToLower(instanceType)

	// Look up pricing
	hourlyRate, found := hourlyPricing[instanceTypeLower]
	if !found {
		// For unknown instance types, try to estimate based on pattern
		// This is a rough heuristic - actual pricing may vary
		if strings.HasPrefix(instanceTypeLower, "t3.") || strings.HasPrefix(instanceTypeLower, "t3a.") {
			hourlyRate = 0.05 // Default for t3 family
		} else if strings.HasPrefix(instanceTypeLower, "t4g.") {
			hourlyRate = 0.04 // Default for t4g family
		} else if strings.HasPrefix(instanceTypeLower, "m5.") || strings.HasPrefix(instanceTypeLower, "m5a.") || strings.HasPrefix(instanceTypeLower, "m6i.") {
			hourlyRate = 0.20 // Default for m5/m6i family
		} else if strings.HasPrefix(instanceTypeLower, "c5.") || strings.HasPrefix(instanceTypeLower, "c5a.") || strings.HasPrefix(instanceTypeLower, "c6i.") {
			hourlyRate = 0.18 // Default for c5/c6i family
		} else if strings.HasPrefix(instanceTypeLower, "r5.") || strings.HasPrefix(instanceTypeLower, "r5a.") || strings.HasPrefix(instanceTypeLower, "r6i.") {
			hourlyRate = 0.25 // Default for r5/r6i family
		} else {
			hourlyRate = 0.10 // Generic default
		}
	}

	// Calculate monthly cost: hourly rate * 24 hours * 30 days
	monthlyCost := hourlyRate * 24 * 30

	return math.Round(monthlyCost*100) / 100 // Round to 2 decimal places
}

func (s *AWSService) ListEC2Instances() ([]models.EC2Instance, error) {
	const cacheKey = "ec2-instances"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if instances, ok := cached.([]models.EC2Instance); ok {
			return instances, nil
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
		return []models.EC2Instance{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		instances []models.EC2Instance
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

			instances, err := s.getEC2InstancesForAccount(acc)
			resultChan <- accountResult{
				instances: instances,
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
	var allInstances []models.EC2Instance
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get EC2 instances for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allInstances = append(allInstances, result.instances...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allInstances, s.cacheTTL)

	return allInstances, nil
}

// getEC2InstancesForAccount returns all EC2 instances for a specific account across all regions
func (s *AWSService) getEC2InstancesForAccount(account models.Account) ([]models.EC2Instance, error) {
	sess, err := s.getSessionForAccount(account.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", account.ID, err)
	}

	// Get all regions (exclude opt-in regions to avoid AuthFailure in disabled regions)
	ec2Client := ec2.New(sess)
	regionsResult, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("opt-in-status"),
				Values: []*string{aws.String("opt-in-not-required"), aws.String("opted-in")},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %v", err)
	}

	if len(regionsResult.Regions) == 0 {
		return []models.EC2Instance{}, nil
	}

	// Channel to collect results from region goroutines
	type regionResult struct {
		instances  []models.EC2Instance
		regionName string
	}

	resultChan := make(chan regionResult, len(regionsResult.Regions))
	var wg sync.WaitGroup

	// Process each region in parallel
	for _, region := range regionsResult.Regions {
		wg.Add(1)
		go func(regionName string) {
			defer wg.Done()

			// Create session for this region
			regionSess := sess.Copy(&aws.Config{Region: aws.String(regionName)})
			regionInstances, err := s.getEC2InstancesForRegion(regionSess, account, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get instances in region %s for account %s: %v\n", regionName, account.ID, err)
				regionInstances = []models.EC2Instance{}
			}

			resultChan <- regionResult{
				instances:  regionInstances,
				regionName: regionName,
			}
		}(*region.RegionName)
	}

	// Wait for all goroutines to complete and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allInstances []models.EC2Instance
	for result := range resultChan {
		allInstances = append(allInstances, result.instances...)
	}

	return allInstances, nil
}

// getEC2InstancesForRegion returns all EC2 instances for a specific region
func (s *AWSService) getEC2InstancesForRegion(sess *session.Session, account models.Account, region string) ([]models.EC2Instance, error) {
	ec2Client := ec2.New(sess)
	var instances []models.EC2Instance
	var nextToken *string

	// Paginate through all instances (handle large instance counts)
	for {
		input := &ec2.DescribeInstancesInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := ec2Client.DescribeInstances(input)
		if err != nil {
			return nil, err
		}

		// Process instances from this page
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				// Extract instance name from tags
				instanceName := ""
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						instanceName = *tag.Value
						break
					}
				}

				// Convert tags
				var tags []models.Tag
				for _, tag := range instance.Tags {
					tags = append(tags, models.Tag{
						Key:   aws.StringValue(tag.Key),
						Value: aws.StringValue(tag.Value),
					})
				}

				instanceType := aws.StringValue(instance.InstanceType)
				instanceState := aws.StringValue(instance.State.Name)
				
				// Calculate monthly cost
				monthlyCost := calculateEC2InstanceMonthlyCost(instanceType, instanceState)

				ec2Instance := models.EC2Instance{
					InstanceID:   aws.StringValue(instance.InstanceId),
					Name:         instanceName,
					AccountID:    account.ID,
					AccountName:  account.Name,
					Region:       region,
					InstanceType: instanceType,
					State:        instanceState,
					MonthlyCost:  monthlyCost,
					Tags:         tags,
				}

				if instance.LaunchTime != nil {
					ec2Instance.LaunchTime = *instance.LaunchTime
				}

				instances = append(instances, ec2Instance)
			}
		}

		// Check if there are more pages
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return instances, nil
}

// InvalidateEC2InstancesCache invalidates the EC2 instances cache
func (s *AWSService) InvalidateEC2InstancesCache() {
	s.cache.Delete("ec2-instances")
}

// updateEC2InstanceStateInCache updates a specific instance's state in the cache
func (s *AWSService) updateEC2InstanceStateInCache(instanceID, newState string) {
	const cacheKey = "ec2-instances"

	if cached, found := s.cache.Get(cacheKey); found {
		if instances, ok := cached.([]models.EC2Instance); ok {
			// Update the specific instance's state
			for i := range instances {
				if instances[i].InstanceID == instanceID {
					instances[i].State = newState
					break
				}
			}
			// Update the cache with the modified list
			s.cache.Set(cacheKey, instances, s.cacheTTL)
		}
	}
}

// StopEC2Instance stops an EC2 instance
func (s *AWSService) StopEC2Instance(accountID, region, instanceID string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	// Create session for the specific region
	regionSess := sess.Copy(&aws.Config{Region: aws.String(region)})
	ec2Client := ec2.New(regionSess)

	// Stop the instance
	_, err = ec2Client.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance %s: %v", instanceID, err)
	}

	// Update the instance state in cache instead of invalidating
	s.updateEC2InstanceStateInCache(instanceID, "stopping")

	return nil
}

// TerminateEC2Instance terminates an EC2 instance
func (s *AWSService) TerminateEC2Instance(accountID, region, instanceID string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	// Create session for the specific region
	regionSess := sess.Copy(&aws.Config{Region: aws.String(region)})
	ec2Client := ec2.New(regionSess)

	// Terminate the instance
	_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance %s: %v", instanceID, err)
	}

	// Update the instance state in cache instead of invalidating
	s.updateEC2InstanceStateInCache(instanceID, "terminating")

	return nil
}

// ============================================================================
// EBS VOLUME MANAGEMENT
// ============================================================================

// ListEBSVolumes returns all EBS volumes from all accessible accounts
