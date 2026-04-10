package services

import (
	"fmt"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ============================================================================
// SECURITY GROUP MANAGEMENT
// ============================================================================

func (s *AWSService) getSecurityGroupsForAccount(account models.Account) ([]models.SecurityGroup, error) {
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
		return []models.SecurityGroup{}, nil
	}

	// Channel to collect results from region goroutines
	type regionResult struct {
		sgs        []models.SecurityGroup
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
			regionSGs, err := s.getSecurityGroupsForRegion(regionSess, account, regionName)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get security groups in region %s for account %s: %v\n", regionName, account.ID, err)
				regionSGs = []models.SecurityGroup{}
			}

			resultChan <- regionResult{
				sgs:        regionSGs,
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
	var allSGs []models.SecurityGroup
	for result := range resultChan {
		allSGs = append(allSGs, result.sgs...)
	}

	return allSGs, nil
}

func (s *AWSService) getSecurityGroupsForRegion(sess *session.Session, account models.Account, region string) ([]models.SecurityGroup, error) {
	ec2Client := ec2.New(sess)
	var sgs []models.SecurityGroup

	result, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	for _, sg := range result.SecurityGroups {
		// Convert ingress rules
		var ingressRules []models.SecurityGroupRule
		for _, rule := range sg.IpPermissions {
			rules := s.convertEC2RuleToModel(rule)
			ingressRules = append(ingressRules, rules...)
		}

		// Convert egress rules
		var egressRules []models.SecurityGroupRule
		for _, rule := range sg.IpPermissionsEgress {
			rules := s.convertEC2RuleToModel(rule)
			egressRules = append(egressRules, rules...)
		}

		// Check for open ports to internet
		hasOpenPorts, openPortsInfo := s.checkForOpenPorts(ingressRules)

		// Check usage of security group
		usageInfo, err := s.checkSecurityGroupUsage(ec2Client, *sg.GroupId)
		if err != nil {
			fmt.Printf("[WARNING] Failed to check usage for security group %s: %v\n", *sg.GroupId, err)
			usageInfo = models.SecurityGroupUsage{}
		}
		isUnused := usageInfo.TotalAttachments == 0

		securityGroup := models.SecurityGroup{
			GroupID:       *sg.GroupId,
			GroupName:     *sg.GroupName,
			Description:   *sg.Description,
			AccountID:     account.ID,
			AccountName:   account.Name,
			Region:        region,
			VpcID:         aws.StringValue(sg.VpcId),
			IsDefault:     *sg.GroupName == "default",
			IngressRules:  ingressRules,
			EgressRules:   egressRules,
			HasOpenPorts:  hasOpenPorts,
			OpenPortsInfo: openPortsInfo,
			IsUnused:      isUnused,
			UsageInfo:     usageInfo,
		}

		sgs = append(sgs, securityGroup)
	}

	return sgs, nil
}

func (s *AWSService) convertEC2RuleToModel(rule *ec2.IpPermission) []models.SecurityGroupRule {
	var rules []models.SecurityGroupRule

	protocol := aws.StringValue(rule.IpProtocol)
	fromPort := aws.Int64Value(rule.FromPort)
	toPort := aws.Int64Value(rule.ToPort)

	// Handle IPv4 CIDR blocks
	for _, ipRange := range rule.IpRanges {
		rules = append(rules, models.SecurityGroupRule{
			IpProtocol: protocol,
			FromPort:   fromPort,
			ToPort:     toPort,
			CidrIPv4:   aws.StringValue(ipRange.CidrIp),
		})
	}

	// Handle IPv6 CIDR blocks
	for _, ipv6Range := range rule.Ipv6Ranges {
		rules = append(rules, models.SecurityGroupRule{
			IpProtocol: protocol,
			FromPort:   fromPort,
			ToPort:     toPort,
			CidrIPv6:   aws.StringValue(ipv6Range.CidrIpv6),
		})
	}

	// Handle security group references
	for _, sgRef := range rule.UserIdGroupPairs {
		rules = append(rules, models.SecurityGroupRule{
			IpProtocol: protocol,
			FromPort:   fromPort,
			ToPort:     toPort,
			GroupID:    aws.StringValue(sgRef.GroupId),
			GroupOwner: aws.StringValue(sgRef.UserId),
		})
	}

	// If no specific ranges are defined, create a rule without CIDR
	if len(rule.IpRanges) == 0 && len(rule.Ipv6Ranges) == 0 && len(rule.UserIdGroupPairs) == 0 {
		rules = append(rules, models.SecurityGroupRule{
			IpProtocol: protocol,
			FromPort:   fromPort,
			ToPort:     toPort,
		})
	}

	return rules
}

func (s *AWSService) checkForOpenPorts(ingressRules []models.SecurityGroupRule) (bool, []models.OpenPortInfo) {
	var openPortsInfo []models.OpenPortInfo
	hasOpenPorts := false

	for _, rule := range ingressRules {
		isOpenToInternet := false
		source := ""

		// Check if rule allows access from anywhere on the internet
		if rule.CidrIPv4 == "0.0.0.0/0" {
			isOpenToInternet = true
			source = "0.0.0.0/0 (IPv4 Internet)"
		} else if rule.CidrIPv6 == "::/0" {
			isOpenToInternet = true
			source = "::/0 (IPv6 Internet)"
		}

		if isOpenToInternet {
			hasOpenPorts = true

			portRange := ""
			if rule.IpProtocol == "-1" {
				portRange = "All ports"
			} else if rule.FromPort == rule.ToPort {
				portRange = fmt.Sprintf("%d", rule.FromPort)
			} else {
				portRange = fmt.Sprintf("%d-%d", rule.FromPort, rule.ToPort)
			}

			description := ""
			switch rule.IpProtocol {
			case "tcp":
				description = "TCP traffic"
			case "udp":
				description = "UDP traffic"
			case "icmp":
				description = "ICMP traffic"
			case "-1":
				description = "All traffic"
			default:
				description = fmt.Sprintf("Protocol %s", rule.IpProtocol)
			}

			openPortsInfo = append(openPortsInfo, models.OpenPortInfo{
				Protocol:    rule.IpProtocol,
				PortRange:   portRange,
				Source:      source,
				Description: description,
			})
		}
	}

	return hasOpenPorts, openPortsInfo
}

// checkSecurityGroupUsage checks if a security group is being used by any resources
func (s *AWSService) checkSecurityGroupUsage(ec2Client *ec2.EC2, groupID string) (models.SecurityGroupUsage, error) {
	usage := models.SecurityGroupUsage{
		AttachedToInstances:       []string{},
		AttachedToNetworkInterfaces: []string{},
		AttachedToLoadBalancers:   []string{},
		ReferencedBySecurityGroups: []string{},
		TotalAttachments:          0,
	}

	// Check EC2 instances
	instancesResult, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []*string{aws.String(groupID)},
			},
		},
	})
	if err != nil {
		return usage, fmt.Errorf("failed to describe instances: %v", err)
	}

	for _, reservation := range instancesResult.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				usage.AttachedToInstances = append(usage.AttachedToInstances, *instance.InstanceId)
				usage.TotalAttachments++
			}
		}
	}

	// Check network interfaces
	eniResult, err := ec2Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("group-id"),
				Values: []*string{aws.String(groupID)},
			},
		},
	})
	if err != nil {
		return usage, fmt.Errorf("failed to describe network interfaces: %v", err)
	}

	for _, eni := range eniResult.NetworkInterfaces {
		if eni.NetworkInterfaceId != nil {
			usage.AttachedToNetworkInterfaces = append(usage.AttachedToNetworkInterfaces, *eni.NetworkInterfaceId)
			usage.TotalAttachments++
		}
	}

	// Check load balancers (both Classic and Application/Network Load Balancers)
	// Note: For ALB/NLB, we need to use ELBv2 API, but for simplicity, we'll check through ENIs
	// Load balancers are typically attached through ENIs, so they should be caught above

	// Check if this security group is referenced by other security groups
	allSGsResult, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return usage, fmt.Errorf("failed to describe all security groups: %v", err)
	}

	for _, sg := range allSGsResult.SecurityGroups {
		if *sg.GroupId == groupID {
			continue // Skip self
		}

		// Check ingress rules
		for _, rule := range sg.IpPermissions {
			for _, userIdGroupPair := range rule.UserIdGroupPairs {
				if userIdGroupPair.GroupId != nil && *userIdGroupPair.GroupId == groupID {
					usage.ReferencedBySecurityGroups = append(usage.ReferencedBySecurityGroups, *sg.GroupId)
					usage.TotalAttachments++
					break
				}
			}
		}

		// Check egress rules
		for _, rule := range sg.IpPermissionsEgress {
			for _, userIdGroupPair := range rule.UserIdGroupPairs {
				if userIdGroupPair.GroupId != nil && *userIdGroupPair.GroupId == groupID {
					// Only add if not already added from ingress rules
					found := false
					for _, existingRef := range usage.ReferencedBySecurityGroups {
						if existingRef == *sg.GroupId {
							found = true
							break
						}
					}
					if !found {
						usage.ReferencedBySecurityGroups = append(usage.ReferencedBySecurityGroups, *sg.GroupId)
						usage.TotalAttachments++
					}
					break
				}
			}
		}
	}

	return usage, nil
}

// DeleteSecurityGroup deletes a security group
func (s *AWSService) DeleteSecurityGroup(accountID, region, groupID string) error {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	// Create session for the specific region
	regionSess := sess.Copy(&aws.Config{Region: aws.String(region)})
	ec2Client := ec2.New(regionSess)

	// First, check if the security group exists and get its details
	sgResult, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{aws.String(groupID)},
	})
	if err != nil {
		return fmt.Errorf("failed to describe security group %s: %v", groupID, err)
	}

	if len(sgResult.SecurityGroups) == 0 {
		return fmt.Errorf("security group %s not found", groupID)
	}

	sg := sgResult.SecurityGroups[0]

	// Prevent deletion of default security groups
	if *sg.GroupName == "default" {
		return fmt.Errorf("cannot delete default security group")
	}

	// Check if the security group is in use before attempting deletion
	usage, err := s.checkSecurityGroupUsage(ec2Client, groupID)
	if err != nil {
		// Log warning but proceed with deletion attempt
		fmt.Printf("[WARNING] Failed to check usage for security group %s: %v\n", groupID, err)
	} else if usage.TotalAttachments > 0 {
		return fmt.Errorf("security group %s is still in use (attached to %d resources)", groupID, usage.TotalAttachments)
	}

	// Attempt to delete the security group
	_, err = ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(groupID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete security group %s: %v", groupID, err)
	}

	// Invalidate the specific security group cache and related caches
	s.InvalidateSecurityGroupCache(accountID, region, groupID)

	return nil
}

// ============================================================================
// SNAPSHOT MANAGEMENT
// ============================================================================

// ListSnapshots returns all EBS snapshots from all accessible accounts
func (s *AWSService) ListSecurityGroups() ([]models.SecurityGroup, error) {
	const cacheKey = "security-groups"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if sgs, ok := cached.([]models.SecurityGroup); ok {
			return sgs, nil
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
		return []models.SecurityGroup{}, nil
	}

	// Channel to collect results from goroutines
	type accountResult struct {
		sgs       []models.SecurityGroup
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

			sgs, err := s.getSecurityGroupsForAccount(acc)
			resultChan <- accountResult{
				sgs:       sgs,
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
	var allSGs []models.SecurityGroup
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get security groups for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allSGs = append(allSGs, result.sgs...)
	}

	// Cache the result
	s.cache.Set(cacheKey, allSGs, s.cacheTTL)

	return allSGs, nil
}

func (s *AWSService) ListSecurityGroupsByAccount(accountID string) ([]models.SecurityGroup, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("security-groups:%s", accountID)
	if cached, found := s.cache.Get(cacheKey); found {
		if sgs, ok := cached.([]models.SecurityGroup); ok {
			return sgs, nil
		}
	}

	// Get account info
	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	var targetAccount models.Account
	found := false
	for _, account := range accounts {
		if account.ID == accountID {
			targetAccount = account
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	if !targetAccount.Accessible {
		return nil, fmt.Errorf("cannot access account %s", accountID)
	}

	// Get security groups for the account
	sgs, err := s.getSecurityGroupsForAccount(targetAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to get security groups for account %s: %v", accountID, err)
	}

	// Cache the result
	s.cache.Set(cacheKey, sgs, s.cacheTTL)

	return sgs, nil
}
func (s *AWSService) GetSecurityGroup(accountID, region, groupID string) (*models.SecurityGroup, error) {
	// Check individual cache first
	cacheKey := fmt.Sprintf("security-group:%s:%s:%s", accountID, region, groupID)
	if cached, found := s.cache.Get(cacheKey); found {
		if sg, ok := cached.(models.SecurityGroup); ok {
			return &sg, nil
		}
	}

	// Fetch directly from AWS using specific security group ID
	sg, err := s.fetchSecurityGroupDirect(accountID, region, groupID)
	if err != nil {
		return nil, err
	}

	// Cache the individual security group
	s.cache.Set(cacheKey, *sg, s.cacheTTL)

	return sg, nil
}

func (s *AWSService) fetchSecurityGroupDirect(accountID, region, groupID string) (*models.SecurityGroup, error) {
	// Get session for the account
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	// We'll use the accountID as the account name if we can't get it from cache
	// This avoids the expensive ListAccounts() call
	accountName := accountID

	// Try to get account name from cache first
	if cached, found := s.cache.Get("accounts"); found {
		if accounts, ok := cached.([]models.Account); ok {
			for _, acc := range accounts {
				if acc.ID == accountID {
					accountName = acc.Name
					break
				}
			}
		}
	}

	// Create EC2 client for the specific region
	ec2Client := ec2.New(sess.Copy(&aws.Config{Region: aws.String(region)}))

	// Fetch the specific security group by ID
	result, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{aws.String(groupID)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe security group %s: %v", groupID, err)
	}

	if len(result.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group %s not found in account %s, region %s", groupID, accountID, region)
	}

	sg := result.SecurityGroups[0]

	// Convert ingress rules
	var ingressRules []models.SecurityGroupRule
	for _, rule := range sg.IpPermissions {
		rules := s.convertEC2RuleToModel(rule)
		ingressRules = append(ingressRules, rules...)
	}

	// Convert egress rules
	var egressRules []models.SecurityGroupRule
	for _, rule := range sg.IpPermissionsEgress {
		rules := s.convertEC2RuleToModel(rule)
		egressRules = append(egressRules, rules...)
	}

	// Check for open ports to internet
	hasOpenPorts, openPortsInfo := s.checkForOpenPorts(ingressRules)

	// Check usage of security group
	usageInfo, err := s.checkSecurityGroupUsage(ec2Client, *sg.GroupId)
	if err != nil {
		fmt.Printf("[WARNING] Failed to check usage for security group %s: %v\n", *sg.GroupId, err)
		usageInfo = models.SecurityGroupUsage{}
	}
	isUnused := usageInfo.TotalAttachments == 0

	securityGroup := &models.SecurityGroup{
		GroupID:       *sg.GroupId,
		GroupName:     *sg.GroupName,
		Description:   *sg.Description,
		AccountID:     accountID,
		AccountName:   accountName,
		Region:        region,
		VpcID:         aws.StringValue(sg.VpcId),
		IsDefault:     *sg.GroupName == "default",
		IngressRules:  ingressRules,
		EgressRules:   egressRules,
		HasOpenPorts:  hasOpenPorts,
		OpenPortsInfo: openPortsInfo,
		IsUnused:      isUnused,
		UsageInfo:     usageInfo,
	}

	return securityGroup, nil
}

