package services

import (
	"fmt"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// ============================================================================
// ROUTE53 DOMAIN MANAGEMENT
// ============================================================================

func (s *AWSService) ListRoute53Domains() ([]models.Route53Domain, error) {
	const cacheKey = "route53-domains"

	if cached, found := s.cache.Get(cacheKey); found {
		if domains, ok := cached.([]models.Route53Domain); ok {
			return domains, nil
		}
	}

	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	var accessibleAccounts []models.Account
	for _, account := range accounts {
		if account.Accessible {
			accessibleAccounts = append(accessibleAccounts, account)
		}
	}

	if len(accessibleAccounts) == 0 {
		return []models.Route53Domain{}, nil
	}

	type accountResult struct {
		domains   []models.Route53Domain
		err       error
		accountID string
	}

	resultChan := make(chan accountResult, len(accessibleAccounts))
	var wg sync.WaitGroup

	for _, account := range accessibleAccounts {
		wg.Add(1)
		go func(acc models.Account) {
			defer wg.Done()
			domains, err := s.getRoute53DomainsForAccount(acc)
			resultChan <- accountResult{
				domains:   domains,
				err:       err,
				accountID: acc.ID,
			}
		}(account)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allDomains []models.Route53Domain
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get Route53 domains for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allDomains = append(allDomains, result.domains...)
	}

	s.cache.Set(cacheKey, allDomains, s.cacheTTL)
	return allDomains, nil
}

func (s *AWSService) getRoute53DomainsForAccount(account models.Account) ([]models.Route53Domain, error) {
	sess, err := s.getSessionForAccount(account.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", account.ID, err)
	}

	client := route53.New(sess.Copy(&aws.Config{Region: aws.String("us-east-1")}))

	var domains []models.Route53Domain
	var marker *string

	for {
		input := &route53.ListHostedZonesInput{}
		if marker != nil {
			input.Marker = marker
		}

		resp, err := client.ListHostedZones(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list hosted zones: %v", err)
		}

		for _, hz := range resp.HostedZones {
			recordCount := int64(0)
			if hz.ResourceRecordSetCount != nil {
				recordCount = *hz.ResourceRecordSetCount
			}
			isPrivate := false
			comment := ""
			if hz.Config != nil {
				if hz.Config.PrivateZone != nil {
					isPrivate = *hz.Config.PrivateZone
				}
				if hz.Config.Comment != nil {
					comment = *hz.Config.Comment
				}
			}

			domains = append(domains, models.Route53Domain{
				HostedZoneID:   aws.StringValue(hz.Id),
				Name:           aws.StringValue(hz.Name),
				RecordSetCount: recordCount,
				IsPrivate:      isPrivate,
				Comment:        comment,
				AccountID:      account.ID,
				AccountName:    account.Name,
			})
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}
		marker = resp.NextMarker
	}

	return domains, nil
}

func (s *AWSService) InvalidateRoute53DomainsCache() {
	s.cache.Delete("route53-domains")
}

func (s *AWSService) ListRoute53Records(accountID, hostedZoneID string) ([]models.Route53Record, error) {
	sess, err := s.getSessionForAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", accountID, err)
	}

	client := route53.New(sess.Copy(&aws.Config{Region: aws.String("us-east-1")}))

	var records []models.Route53Record
	var startName *string
	var startType *string

	for {
		input := &route53.ListResourceRecordSetsInput{
			HostedZoneId: aws.String(hostedZoneID),
		}
		if startName != nil {
			input.StartRecordName = startName
		}
		if startType != nil {
			input.StartRecordType = startType
		}

		resp, err := client.ListResourceRecordSets(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource record sets: %v", err)
		}

		for _, rr := range resp.ResourceRecordSets {
			ttl := int64(0)
			if rr.TTL != nil {
				ttl = *rr.TTL
			}
			var values []string
			if rr.ResourceRecords != nil {
				for _, r := range rr.ResourceRecords {
					if r.Value != nil {
						values = append(values, *r.Value)
					}
				}
			}
			if rr.AliasTarget != nil && rr.AliasTarget.DNSName != nil {
				values = append(values, *rr.AliasTarget.DNSName)
			}
			if len(values) == 0 {
				values = []string{}
			}

			records = append(records, models.Route53Record{
				Name:   aws.StringValue(rr.Name),
				Type:   aws.StringValue(rr.Type),
				TTL:    ttl,
				Values: values,
			})
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}
		startName = resp.NextRecordName
		startType = resp.NextRecordType
	}

	return records, nil
}
