package services

import (
	"fmt"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53domains"
	"github.com/aws/aws-sdk-go/service/sts"
)

// ============================================================================
// ROUTE53 REGISTERED DOMAIN MANAGEMENT
// ============================================================================

func (s *AWSService) ListRoute53RegisteredDomains() ([]models.Route53RegisteredDomain, error) {
	const cacheKey = "route53-registered-domains"

	if cached, found := s.cache.Get(cacheKey); found {
		if domains, ok := cached.([]models.Route53RegisteredDomain); ok {
			return domains, nil
		}
	}

	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	// Get master account ID so we can query it directly (it can't assume its own cross-account role)
	masterAccountID := ""
	masterAccountName := "Master Account"
	if identity, err := sts.New(s.masterSession).GetCallerIdentity(&sts.GetCallerIdentityInput{}); err == nil {
		masterAccountID = aws.StringValue(identity.Account)
	}
	for _, acc := range accounts {
		if acc.ID == masterAccountID {
			masterAccountName = acc.Name
			break
		}
	}

	var accessibleAccounts []models.Account
	for _, account := range accounts {
		if account.Accessible && account.ID != masterAccountID {
			accessibleAccounts = append(accessibleAccounts, account)
		}
	}

	// Include master account in the goroutine pool using masterSession directly
	masterAccount := models.Account{ID: "", Name: masterAccountName}
	allAccountsToQuery := append([]models.Account{masterAccount}, accessibleAccounts...)

	type accountResult struct {
		domains   []models.Route53RegisteredDomain
		err       error
		accountID string
	}

	resultChan := make(chan accountResult, len(allAccountsToQuery))
	var wg sync.WaitGroup

	for _, account := range allAccountsToQuery {
		wg.Add(1)
		go func(acc models.Account) {
			defer wg.Done()
			queryAcc := acc
			if queryAcc.ID == "" {
				queryAcc.ID = masterAccountID
				queryAcc.Name = masterAccountName
			}
			domains, err := s.getRoute53RegisteredDomainsForAccount(acc)
			resultChan <- accountResult{
				domains:   overrideAccountInfo(domains, queryAcc.ID, queryAcc.Name),
				err:       err,
				accountID: queryAcc.ID,
			}
		}(account)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allDomains []models.Route53RegisteredDomain
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get Route53 registered domains for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allDomains = append(allDomains, result.domains...)
	}

	s.cache.Set(cacheKey, allDomains, s.cacheTTL)
	return allDomains, nil
}

func overrideAccountInfo(domains []models.Route53RegisteredDomain, accountID, accountName string) []models.Route53RegisteredDomain {
	for i := range domains {
		if domains[i].AccountID == "" {
			domains[i].AccountID = accountID
			domains[i].AccountName = accountName
		}
	}
	return domains
}

func (s *AWSService) getRoute53RegisteredDomainsForAccount(account models.Account) ([]models.Route53RegisteredDomain, error) {
	sess, err := s.getSessionForAccount(account.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", account.ID, err)
	}

	// Route53 Domains API is only available in us-east-1
	client := route53domains.New(sess.Copy(&aws.Config{Region: aws.String("us-east-1")}))

	var domains []models.Route53RegisteredDomain
	var marker *string

	for {
		input := &route53domains.ListDomainsInput{}
		if marker != nil {
			input.Marker = marker
		}

		resp, err := client.ListDomains(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list registered domains: %v", err)
		}

		for _, d := range resp.Domains {
			expiry := ""
			if d.Expiry != nil {
				expiry = d.Expiry.Format("2006-01-02")
			}
			domains = append(domains, models.Route53RegisteredDomain{
				DomainName:   aws.StringValue(d.DomainName),
				Expiry:       expiry,
				AutoRenew:    aws.BoolValue(d.AutoRenew),
				TransferLock: aws.BoolValue(d.TransferLock),
				AccountID:    account.ID,
				AccountName:  account.Name,
			})
		}

		if resp.NextPageMarker == nil {
			break
		}
		marker = resp.NextPageMarker
	}

	return domains, nil
}

func (s *AWSService) InvalidateRoute53RegisteredDomainsCache() {
	s.cache.Delete("route53-registered-domains")
}
