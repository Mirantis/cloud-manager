package services

import (
	"fmt"
	"net"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// ============================================================================
// PUBLIC IP DISCOVERY
// ============================================================================

func (s *AWSService) ListPublicIPs() ([]models.PublicIP, error) {
	const cacheKey = "public-ips"
	
	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if ips, ok := cached.([]models.PublicIP); ok {
			return ips, nil
		}
	}

	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	// Filter accessible accounts
	var accessibleAccounts []models.Account
	for _, account := range accounts {
		if !account.Accessible {
			fmt.Printf("[WARNING] Skipping inaccessible account %s\n", account.ID)
			continue
		}
		accessibleAccounts = append(accessibleAccounts, account)
	}

	if len(accessibleAccounts) == 0 {
		return []models.PublicIP{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		ips []models.PublicIP
		err error
		accountID string
	}

	resultChan := make(chan accountResult, len(accessibleAccounts))
	var wg sync.WaitGroup

	// Process each account in parallel
	for _, account := range accessibleAccounts {
		wg.Add(1)
		go func(acc models.Account) {
			defer wg.Done()
			
			ips, err := s.getPublicIPsForAccount(acc)
			resultChan <- accountResult{
				ips: ips,
				err: err,
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
	var allIPs []models.PublicIP
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get IPs for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allIPs = append(allIPs, result.ips...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allIPs, s.cacheTTL)

	return allIPs, nil
}

func (s *AWSService) getPublicIPsForAccount(account models.Account) ([]models.PublicIP, error) {
	sess, err := s.getSessionForAccount(account.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", account.ID, err)
	}

	// Get all regions
	ec2Client := ec2.New(sess)
	regionsResult, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %v", err)
	}

	if len(regionsResult.Regions) == 0 {
		return []models.PublicIP{}, nil
	}

	// Channel to collect results from region goroutines
	type regionResult struct {
		ips []models.PublicIP
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
			var regionIPs []models.PublicIP
			
			// Get EC2 instances
			ec2IPs, err := s.getEC2PublicIPs(regionSess, account, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get EC2 IPs in region %s for account %s: %v\n", regionName, account.ID, err)
			} else {
				regionIPs = append(regionIPs, ec2IPs...)
			}

			// Get Load Balancers
			elbIPs, err := s.getELBPublicIPs(regionSess, account, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get ELB IPs in region %s for account %s: %v\n", regionName, account.ID, err)
			} else {
				regionIPs = append(regionIPs, elbIPs...)
			}

			// Get NAT Gateways
			natIPs, err := s.getNATPublicIPs(regionSess, account, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get NAT IPs in region %s for account %s: %v\n", regionName, account.ID, err)
			} else {
				regionIPs = append(regionIPs, natIPs...)
			}
			
			resultChan <- regionResult{
				ips: regionIPs,
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
	var allIPs []models.PublicIP
	for result := range resultChan {
		allIPs = append(allIPs, result.ips...)
	}

	return allIPs, nil
}

func (s *AWSService) getEC2PublicIPs(sess *session.Session, account models.Account, region string) ([]models.PublicIP, error) {
	ec2Client := ec2.New(sess)
	var ips []models.PublicIP

	result, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PublicIpAddress != nil && *instance.PublicIpAddress != "" {
				instanceName := ""
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						instanceName = *tag.Value
						break
					}
				}

				ips = append(ips, models.PublicIP{
					IPAddress:    *instance.PublicIpAddress,
					AccountID:    account.ID,
					AccountName:  account.Name,
					Region:       region,
					ResourceType: "EC2",
					ResourceID:   *instance.InstanceId,
					ResourceName: instanceName,
					State:        *instance.State.Name,
				})
			}
		}
	}

	return ips, nil
}

func (s *AWSService) getELBPublicIPs(sess *session.Session, account models.Account, region string) ([]models.PublicIP, error) {
	var ips []models.PublicIP

	// Get ALB/NLB (ELBv2)
	elbv2IPs, err := s.getELBv2PublicIPs(sess, account, region)
	if err != nil {
		fmt.Printf("[WARNING] Failed to get ELBv2 IPs in region %s for account %s: %v\n", region, account.ID, err)
	} else {
		ips = append(ips, elbv2IPs...)
	}

	// Get Classic Load Balancers (ELBv1)
	elbv1IPs, err := s.getClassicELBPublicIPs(sess, account, region)
	if err != nil {
		fmt.Printf("[WARNING] Failed to get Classic ELB IPs in region %s for account %s: %v\n", region, account.ID, err)
	} else {
		ips = append(ips, elbv1IPs...)
	}

	return ips, nil
}

func (s *AWSService) getELBv2PublicIPs(sess *session.Session, account models.Account, region string) ([]models.PublicIP, error) {
	elbClient := elbv2.New(sess)
	var ips []models.PublicIP

	result, err := elbClient.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, err
	}

	for _, lb := range result.LoadBalancers {
		if lb.Scheme != nil && *lb.Scheme == "internet-facing" && lb.DNSName != nil {
			lbType := "ALB"
			if lb.Type != nil {
				lbType = *lb.Type
			}

			// For NLB with static IPs, try to get them directly
			if lb.Type != nil && *lb.Type == "network" {
				for _, az := range lb.AvailabilityZones {
					if az.LoadBalancerAddresses != nil {
						for _, addr := range az.LoadBalancerAddresses {
							if addr.IpAddress != nil && *addr.IpAddress != "" {
								ips = append(ips, models.PublicIP{
									IPAddress:    *addr.IpAddress,
									AccountID:    account.ID,
									AccountName:  account.Name,
									Region:       region,
									ResourceType: "NLB",
									ResourceID:   *lb.LoadBalancerArn,
									ResourceName: *lb.LoadBalancerName,
									State:        *lb.State.Code,
								})
							}
						}
					}
				}
			}

			// For ALB and NLB without static IPs, resolve DNS name
			if len(ips) == 0 || (lb.Type != nil && *lb.Type != "network") {
				resolvedIPs, err := s.resolveDNSToIPs(*lb.DNSName)
				if err != nil {
					fmt.Printf("[WARNING] Failed to resolve DNS %s: %v\n", *lb.DNSName, err)
					continue
				}

				for _, ip := range resolvedIPs {
					ips = append(ips, models.PublicIP{
						IPAddress:    ip,
						AccountID:    account.ID,
						AccountName:  account.Name,
						Region:       region,
						ResourceType: lbType,
						ResourceID:   *lb.LoadBalancerArn,
						ResourceName: *lb.LoadBalancerName,
						State:        *lb.State.Code,
					})
				}
			}
		}
	}

	return ips, nil
}

func (s *AWSService) getClassicELBPublicIPs(sess *session.Session, account models.Account, region string) ([]models.PublicIP, error) {
	// Import classic ELB package
	elbClient := elb.New(sess)
	var ips []models.PublicIP

	result, err := elbClient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, err
	}

	for _, lb := range result.LoadBalancerDescriptions {
		if lb.Scheme != nil && *lb.Scheme == "internet-facing" && lb.DNSName != nil {
			// Resolve DNS name to get IP addresses
			resolvedIPs, err := s.resolveDNSToIPs(*lb.DNSName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to resolve DNS %s: %v\n", *lb.DNSName, err)
				continue
			}

			for _, ip := range resolvedIPs {
				ips = append(ips, models.PublicIP{
					IPAddress:    ip,
					AccountID:    account.ID,
					AccountName:  account.Name,
					Region:       region,
					ResourceType: "CLB",
					ResourceID:   *lb.LoadBalancerName, // Classic ELB doesn't have ARN
					ResourceName: *lb.LoadBalancerName,
					State:        "active", // Classic ELB doesn't have detailed state
				})
			}
		}
	}

	return ips, nil
}

func (s *AWSService) resolveDNSToIPs(dnsName string) ([]string, error) {
	ips, err := net.LookupIP(dnsName)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			result = append(result, ipv4.String())
		}
	}

	return result, nil
}

func (s *AWSService) getNATPublicIPs(sess *session.Session, account models.Account, region string) ([]models.PublicIP, error) {
	ec2Client := ec2.New(sess)
	var ips []models.PublicIP

	result, err := ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{})
	if err != nil {
		return nil, err
	}

	for _, nat := range result.NatGateways {
		for _, addr := range nat.NatGatewayAddresses {
			if addr.PublicIp != nil && *addr.PublicIp != "" {
				natName := ""
				for _, tag := range nat.Tags {
					if *tag.Key == "Name" {
						natName = *tag.Value
						break
					}
				}

				ips = append(ips, models.PublicIP{
					IPAddress:    *addr.PublicIp,
					AccountID:    account.ID,
					AccountName:  account.Name,
					Region:       region,
					ResourceType: "NAT",
					ResourceID:   *nat.NatGatewayId,
					ResourceName: natName,
					State:        *nat.State,
				})
			}
		}
	}

	return ips, nil
}

// generatePassword generates a secure random password
