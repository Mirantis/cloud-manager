package services

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rusik69/aws-iam-manager/internal/models"
)

// ListVPCs returns all VPCs from all accessible accounts
func (s *AWSService) ListVPCs() ([]models.VPC, error) {
	const cacheKey = "all-vpcs"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if vpcs, ok := cached.([]models.VPC); ok {
			return vpcs, nil
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
		return []models.VPC{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		vpcs      []models.VPC
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

			vpcs, err := s.ListVPCsByAccount(acc.ID)
			resultChan <- accountResult{
				vpcs:      vpcs,
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
	var allVPCs []models.VPC
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get VPCs for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allVPCs = append(allVPCs, result.vpcs...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allVPCs, s.cacheTTL)

	return allVPCs, nil
}

// ListVPCsByAccount returns all VPCs for a specific account
func (s *AWSService) ListVPCsByAccount(accountID string) ([]models.VPC, error) {
	cacheKey := fmt.Sprintf("vpcs-%s", accountID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if vpcs, ok := cached.([]models.VPC); ok {
			return vpcs, nil
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

	var allVPCs []models.VPC
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process all regions in parallel
	for _, region := range regionsResult.Regions {
		wg.Add(1)
		go func(regionName string) {
			defer wg.Done()

			regionSess := sess.Copy(&aws.Config{Region: aws.String(regionName)})
			vpcs, err := s.getVPCsInRegion(regionSess, accountID, accountName, regionName)
			if err != nil {
				// Skip logging warnings for disabled regions (AuthFailure) - these are expected
				errStr := err.Error()
				if !strings.Contains(errStr, "AuthFailure") && !strings.Contains(errStr, "not authorized") {
					fmt.Printf("[WARNING] Failed to get VPCs in region %s for account %s: %v\n", regionName, accountID, err)
				}
				return
			}

			mu.Lock()
			allVPCs = append(allVPCs, vpcs...)
			mu.Unlock()
		}(*region.RegionName)
	}

	wg.Wait()

	// Cache the result
	s.cache.Set(cacheKey, allVPCs, s.cacheTTL)

	return allVPCs, nil
}

// getVPCsInRegion fetches VPCs from a specific region
func (s *AWSService) getVPCsInRegion(sess *session.Session, accountID, accountName, region string) ([]models.VPC, error) {
	ec2Client := ec2.New(sess)

	// Get VPCs
	result, err := ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}

	var vpcs []models.VPC
	for _, vpc := range result.Vpcs {
		v := models.VPC{
			VpcID:           aws.StringValue(vpc.VpcId),
			AccountID:       accountID,
			AccountName:     accountName,
			Region:          region,
			CidrBlock:       aws.StringValue(vpc.CidrBlock),
			State:           aws.StringValue(vpc.State),
			IsDefault:       aws.BoolValue(vpc.IsDefault),
			InstanceTenancy: aws.StringValue(vpc.InstanceTenancy),
			DhcpOptionsID:   aws.StringValue(vpc.DhcpOptionsId),
		}

		// Get name from tags
		for _, tag := range vpc.Tags {
			if aws.StringValue(tag.Key) == "Name" {
				v.Name = aws.StringValue(tag.Value)
			}
			v.Tags = append(v.Tags, models.Tag{
				Key:   aws.StringValue(tag.Key),
				Value: aws.StringValue(tag.Value),
			})
		}

		// Get subnet count
		subnets, err := ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId}},
			},
		})
		if err == nil {
			v.SubnetCount = len(subnets.Subnets)
		}

		// Get Internet Gateway
		igws, err := ec2Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("attachment.vpc-id"), Values: []*string{vpc.VpcId}},
			},
		})
		if err == nil && len(igws.InternetGateways) > 0 {
			v.InternetGateway = aws.StringValue(igws.InternetGateways[0].InternetGatewayId)
		}

		// Get NAT Gateway count
		nats, err := ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			Filter: []*ec2.Filter{
				{Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId}},
				{Name: aws.String("state"), Values: []*string{aws.String("available")}},
			},
		})
		if err == nil {
			v.NatGatewayCount = len(nats.NatGateways)
		}

		// Check for flow logs
		flowLogs, err := ec2Client.DescribeFlowLogs(&ec2.DescribeFlowLogsInput{
			Filter: []*ec2.Filter{
				{Name: aws.String("resource-id"), Values: []*string{vpc.VpcId}},
			},
		})
		if err == nil && len(flowLogs.FlowLogs) > 0 {
			v.HasFlowLogs = true
		}

		vpcs = append(vpcs, v)
	}

	return vpcs, nil
}

// DeleteVPC deletes a VPC
func (s *AWSService) DeleteVPC(accountID, region, vpcID string) error {
	sess, err := s.getSessionForAccountAndRegion(accountID, region)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %v", accountID, err)
	}

	ec2Client := ec2.New(sess)

	_, err = ec2Client.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VPC %s: %v", vpcID, err)
	}

	// Invalidate cache
	s.InvalidateVPCsCache()

	return nil
}

// InvalidateVPCsCache invalidates the VPCs cache
func (s *AWSService) InvalidateVPCsCache() {
	s.cache.Delete("all-vpcs")
	// Also invalidate per-account caches
	s.cache.DeletePattern("vpcs-")
}
