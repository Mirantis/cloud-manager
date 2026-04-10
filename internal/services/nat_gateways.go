package services

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rusik69/aws-iam-manager/internal/models"
)

// ListNATGateways returns all NAT Gateways from all accessible accounts
func (s *AWSService) ListNATGateways() ([]models.NATGateway, error) {
	const cacheKey = "all-nat-gateways"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if nats, ok := cached.([]models.NATGateway); ok {
			return nats, nil
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
		return []models.NATGateway{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		nats      []models.NATGateway
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

			nats, err := s.ListNATGatewaysByAccount(acc.ID)
			resultChan <- accountResult{
				nats:      nats,
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
	var allNATs []models.NATGateway
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get NAT Gateways for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allNATs = append(allNATs, result.nats...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allNATs, s.cacheTTL)

	return allNATs, nil
}

// ListNATGatewaysByAccount returns all NAT Gateways for a specific account
func (s *AWSService) ListNATGatewaysByAccount(accountID string) ([]models.NATGateway, error) {
	cacheKey := fmt.Sprintf("nat-gateways-%s", accountID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if nats, ok := cached.([]models.NATGateway); ok {
			return nats, nil
		}
	}

	// Get account info
	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("cannot list accounts: %v", err)
	}

	var accountName string
	for _, acc := range accounts {
		if acc.ID == accountID {
			accountName = acc.Name
			break
		}
	}

	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %v", accountID, err)
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

	var allNATs []models.NATGateway
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process all regions in parallel
	for _, region := range regionsResult.Regions {
		wg.Add(1)
		go func(regionName string) {
			defer wg.Done()

			regionSess := sess.Copy(&aws.Config{Region: aws.String(regionName)})
			nats, err := s.getNATGatewaysInRegion(regionSess, accountID, accountName, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get NAT Gateways in region %s for account %s: %v\n", regionName, accountID, err)
				return
			}

			mu.Lock()
			allNATs = append(allNATs, nats...)
			mu.Unlock()
		}(*region.RegionName)
	}

	wg.Wait()

	// Cache the result
	s.cache.Set(cacheKey, allNATs, s.cacheTTL)

	return allNATs, nil
}

// getNATGatewaysInRegion fetches NAT Gateways from a specific region
func (s *AWSService) getNATGatewaysInRegion(sess *session.Session, accountID, accountName, region string) ([]models.NATGateway, error) {
	ec2Client := ec2.New(sess)

	result, err := ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{})
	if err != nil {
		return nil, err
	}

	var nats []models.NATGateway
	for _, nat := range result.NatGateways {
		n := models.NATGateway{
			NatGatewayID:     aws.StringValue(nat.NatGatewayId),
			AccountID:        accountID,
			AccountName:      accountName,
			Region:           region,
			VpcID:            aws.StringValue(nat.VpcId),
			SubnetID:         aws.StringValue(nat.SubnetId),
			State:            aws.StringValue(nat.State),
			ConnectivityType: aws.StringValue(nat.ConnectivityType),
		}

		if nat.CreateTime != nil {
			n.CreateTime = nat.CreateTime
		}

		// Get public and private IPs from addresses
		for _, addr := range nat.NatGatewayAddresses {
			if addr.PublicIp != nil {
				n.PublicIP = aws.StringValue(addr.PublicIp)
			}
			if addr.PrivateIp != nil {
				n.PrivateIP = aws.StringValue(addr.PrivateIp)
			}
		}

		// Get name from tags
		for _, tag := range nat.Tags {
			if aws.StringValue(tag.Key) == "Name" {
				n.Name = aws.StringValue(tag.Value)
			}
			n.Tags = append(n.Tags, models.Tag{
				Key:   aws.StringValue(tag.Key),
				Value: aws.StringValue(tag.Value),
			})
		}

		nats = append(nats, n)
	}

	return nats, nil
}

// DeleteNATGateway deletes a NAT Gateway
func (s *AWSService) DeleteNATGateway(accountID, region, natGatewayID string) error {
	sess, err := s.getSessionForAccountAndRegion(accountID, region)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %v", accountID, err)
	}

	ec2Client := ec2.New(sess)

	_, err = ec2Client.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(natGatewayID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete NAT Gateway %s: %v", natGatewayID, err)
	}

	// Invalidate cache
	s.InvalidateNATGatewaysCache()

	return nil
}

// InvalidateNATGatewaysCache invalidates the NAT Gateways cache
func (s *AWSService) InvalidateNATGatewaysCache() {
	s.cache.Delete("all-nat-gateways")
	// Also invalidate per-account caches
	s.cache.DeletePattern("nat-gateways-")
}
