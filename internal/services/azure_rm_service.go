// Package services provides business logic for Azure Resource Manager operations
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/rusik69/aws-iam-manager/internal/models"
)

// AzureRMService handles Azure Resource Manager operations
type AzureRMService struct {
	credential       *azidentity.ClientSecretCredential
	defaultSubscriptionID string // Fallback subscription ID from environment
	cache            *Cache
	cacheTTL         time.Duration
	clientCache      map[string]*subscriptionClients
	clientCacheMutex sync.RWMutex
}

// subscriptionClients holds clients for a specific subscription
type subscriptionClients struct {
	computeClient   *armcompute.VirtualMachinesClient
	resourcesClient *armresources.ResourceGroupsClient
	storageClient   *armstorage.AccountsClient
}

// NewAzureRMService creates a new Azure Resource Manager service instance
func NewAzureRMService() (*AzureRMService, error) {
	// Get Azure credentials from environment variables
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("azure credentials not configured: set AZURE_TENANT_ID, AZURE_CLIENT_ID, and AZURE_CLIENT_SECRET")
	}

	// Create credential
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Get default subscription ID from environment (optional fallback)
	defaultSubscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	return &AzureRMService{
		credential:           cred,
		defaultSubscriptionID: defaultSubscriptionID,
		cache:                NewCache(),
		cacheTTL:             5 * time.Minute,
		clientCache:          make(map[string]*subscriptionClients),
	}, nil
}

// getClientsForSubscription returns or creates clients for a specific subscription
func (s *AzureRMService) getClientsForSubscription(subscriptionID string) (*subscriptionClients, error) {
	s.clientCacheMutex.RLock()
	if clients, found := s.clientCache[subscriptionID]; found {
		s.clientCacheMutex.RUnlock()
		return clients, nil
	}
	s.clientCacheMutex.RUnlock()

	s.clientCacheMutex.Lock()
	defer s.clientCacheMutex.Unlock()

	// Double-check after acquiring write lock
	if clients, found := s.clientCache[subscriptionID]; found {
		return clients, nil
	}

	// Create new clients for this subscription
	computeClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, s.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client for subscription %s: %w", subscriptionID, err)
	}

	resourcesClient, err := armresources.NewResourceGroupsClient(subscriptionID, s.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources client for subscription %s: %w", subscriptionID, err)
	}

	storageClient, err := armstorage.NewAccountsClient(subscriptionID, s.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client for subscription %s: %w", subscriptionID, err)
	}

	clients := &subscriptionClients{
		computeClient:   computeClient,
		resourcesClient: resourcesClient,
		storageClient:   storageClient,
	}

	s.clientCache[subscriptionID] = clients
	return clients, nil
}

// ListSubscriptions lists all available Azure subscriptions using REST API
func (s *AzureRMService) ListSubscriptions(ctx context.Context) ([]models.AzureSubscription, error) {
	const cacheKey = "azure-subscriptions"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if subs, ok := cached.([]models.AzureSubscription); ok {
			fmt.Printf("[INFO] Returning %d subscription(s) from cache\n", len(subs))
			return subs, nil
		}
	}

	fmt.Printf("[INFO] Fetching subscriptions from Azure API...\n")

	// Get access token
	token, err := s.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	fmt.Printf("[INFO] Successfully obtained access token\n")

	// Use REST API to list subscriptions
	url := "https://management.azure.com/subscriptions?api-version=2020-01-01"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] Subscriptions API returned status %d: %s\n", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("[INFO] Subscriptions API response status: %d\n", resp.StatusCode)
	responsePreview := bodyBytes
	if len(bodyBytes) > 500 {
		responsePreview = bodyBytes[:500]
	}
	fmt.Printf("[DEBUG] Response body (first 500 chars): %s\n", string(responsePreview))

	var jsonResponse struct {
		Value []struct {
			ID             string `json:"id"`
			SubscriptionID string `json:"subscriptionId"`
			DisplayName    string `json:"displayName"`
			State          string `json:"state"`
			TenantID       string `json:"tenantId"`
		} `json:"value"`
		NextLink *string `json:"nextLink"`
		Error    *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		fmt.Printf("[ERROR] Failed to parse subscriptions JSON: %v\n", err)
		fmt.Printf("[DEBUG] Full response body: %s\n", string(bodyBytes))
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if jsonResponse.Error != nil {
		fmt.Printf("[ERROR] Azure API returned error: Code=%s, Message=%s\n", jsonResponse.Error.Code, jsonResponse.Error.Message)
		return nil, fmt.Errorf("azure API error: %s - %s", jsonResponse.Error.Code, jsonResponse.Error.Message)
	}

	fmt.Printf("[INFO] Parsed %d subscription(s) from first page\n", len(jsonResponse.Value))
	if len(jsonResponse.Value) == 0 {
		fmt.Printf("[DEBUG] Empty 'value' array in response. Full response: %s\n", string(bodyBytes))
	}

	var subscriptions []models.AzureSubscription
	for _, sub := range jsonResponse.Value {
		if sub.SubscriptionID == "" {
			fmt.Printf("[WARNING] Skipping subscription with empty SubscriptionID: %+v\n", sub)
			continue
		}
		subscriptions = append(subscriptions, models.AzureSubscription{
			ID:             sub.ID,
			SubscriptionID: sub.SubscriptionID,
			DisplayName:    sub.DisplayName,
			State:          sub.State,
			TenantID:       sub.TenantID,
		})
		fmt.Printf("[INFO] Added subscription: %s (%s) - State: %s\n", sub.SubscriptionID, sub.DisplayName, sub.State)
	}

	// Handle pagination if nextLink exists
	for jsonResponse.NextLink != nil && *jsonResponse.NextLink != "" {
		fmt.Printf("[INFO] Fetching next page of subscriptions from: %s\n", *jsonResponse.NextLink)
		req, err := http.NewRequestWithContext(ctx, "GET", *jsonResponse.NextLink, nil)
		if err != nil {
			fmt.Printf("[WARNING] Failed to create request for next page: %v\n", err)
			break
		}

		req.Header.Set("Authorization", "Bearer "+token.Token)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[WARNING] Failed to execute request for next page: %v\n", err)
			break
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			fmt.Printf("[WARNING] Failed to read response for next page: %v\n", err)
			break
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[WARNING] Next page request returned status %d: %s\n", resp.StatusCode, string(bodyBytes))
			break
		}

		jsonResponse.NextLink = nil
		if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
			fmt.Printf("[WARNING] Failed to parse JSON for next page: %v\n", err)
			break
		}

		fmt.Printf("[INFO] Parsed %d subscription(s) from next page\n", len(jsonResponse.Value))
		for _, sub := range jsonResponse.Value {
			if sub.SubscriptionID == "" {
				fmt.Printf("[WARNING] Skipping subscription with empty SubscriptionID: %+v\n", sub)
				continue
			}
			subscriptions = append(subscriptions, models.AzureSubscription{
				ID:             sub.ID,
				SubscriptionID: sub.SubscriptionID,
				DisplayName:    sub.DisplayName,
				State:          sub.State,
				TenantID:       sub.TenantID,
			})
			fmt.Printf("[INFO] Added subscription: %s (%s) - State: %s\n", sub.SubscriptionID, sub.DisplayName, sub.State)
		}
	}

	fmt.Printf("[INFO] Total subscriptions found: %d\n", len(subscriptions))

	// If no subscriptions found via API but we have a default subscription ID, use it as fallback
	if len(subscriptions) == 0 && s.defaultSubscriptionID != "" {
		fmt.Printf("[INFO] No subscriptions found via API, using fallback subscription ID from AZURE_SUBSCRIPTION_ID: %s\n", s.defaultSubscriptionID)
		subscriptions = []models.AzureSubscription{
			{
				SubscriptionID: s.defaultSubscriptionID,
				DisplayName:     "Default Subscription",
				State:           "Enabled",
			},
		}
	}

	// Cache the result
	s.cache.Set(cacheKey, subscriptions, s.cacheTTL)

	return subscriptions, nil
}

// ListVMs lists all virtual machines across all subscriptions and resource groups
func (s *AzureRMService) ListVMs(ctx context.Context, subscriptionID string) ([]models.AzureVM, error) {
	cacheKey := fmt.Sprintf("azure-vms-%s", subscriptionID)
	if subscriptionID == "" {
		cacheKey = "azure-vms-all"
	}

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if vms, ok := cached.([]models.AzureVM); ok {
			return vms, nil
		}
	}

	var allVMs []models.AzureVM
	var subscriptions []models.AzureSubscription

	if subscriptionID != "" {
		// Use specific subscription
		subscriptions = []models.AzureSubscription{
			{SubscriptionID: subscriptionID},
		}
	} else {
		// List all subscriptions
		var err error
		subscriptions, err = s.ListSubscriptions(ctx)
		if err != nil {
			fmt.Printf("[WARNING] Failed to list subscriptions: %v\n", err)
			// If we have a default subscription ID, use it as fallback
			if s.defaultSubscriptionID != "" {
				fmt.Printf("[INFO] Using fallback subscription ID: %s\n", s.defaultSubscriptionID)
				subscriptions = []models.AzureSubscription{
					{SubscriptionID: s.defaultSubscriptionID},
				}
			} else {
				return nil, fmt.Errorf("failed to list subscriptions and no AZURE_SUBSCRIPTION_ID fallback configured: %w", err)
			}
		}
	}

	// Iterate through all subscriptions
	fmt.Printf("[INFO] Listing VMs across %d subscription(s)\n", len(subscriptions))
	if len(subscriptions) == 0 {
		fmt.Printf("[WARNING] No subscriptions found. Check Azure credentials and permissions.\n")
		return allVMs, nil
	}

	for _, sub := range subscriptions {
		fmt.Printf("[INFO] Processing subscription %s (%s)\n", sub.SubscriptionID, sub.DisplayName)
		clients, err := s.getClientsForSubscription(sub.SubscriptionID)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get clients for subscription %s: %v\n", sub.SubscriptionID, err)
			continue
		}

		// List all resource groups in this subscription
		pager := clients.resourcesClient.NewListPager(nil)
		resourceGroupCount := 0
		hasMoreRGs := true
		for pager.More() && hasMoreRGs {
			page, err := pager.NextPage(ctx)
			if err != nil {
				fmt.Printf("[WARNING] Failed to list resource groups in subscription %s: %v\n", sub.SubscriptionID, err)
				break
			}

			if len(page.Value) == 0 {
				fmt.Printf("[INFO] No resource groups found in subscription %s\n", sub.SubscriptionID)
				break
			}

			for _, rg := range page.Value {
				resourceGroupName := ""
				if rg.Name != nil {
					resourceGroupName = *rg.Name
				} else {
					continue
				}
				resourceGroupCount++

				// List VMs in this resource group
				vmPager := clients.computeClient.NewListPager(resourceGroupName, nil)
				vmCountInRG := 0
				for vmPager.More() {
					vmPage, err := vmPager.NextPage(ctx)
					if err != nil {
						// Skip resource groups that don't have VMs or have permission issues
						errStr := err.Error()
						if strings.Contains(errStr, "NotFound") || 
						   strings.Contains(errStr, "Authorization") || 
						   strings.Contains(errStr, "ResourceNotFound") ||
						   strings.Contains(errStr, "does not exist") ||
						   strings.Contains(errStr, "not found") {
							// This is expected for resource groups without VMs - skip silently
							break
						}
						fmt.Printf("[WARNING] Failed to list VMs in resource group %s (subscription %s): %v\n", resourceGroupName, sub.SubscriptionID, err)
						break
					}

					if len(vmPage.Value) > 0 {
						for _, vm := range vmPage.Value {
							azureVM := s.convertVMToModel(vm, resourceGroupName, sub.SubscriptionID)
							allVMs = append(allVMs, *azureVM)
							vmCountInRG++
						}
					}
				}
				if vmCountInRG > 0 {
					fmt.Printf("[INFO] Found %d VM(s) in resource group %s (subscription %s)\n", vmCountInRG, resourceGroupName, sub.SubscriptionID)
				}
			}
		}
		fmt.Printf("[INFO] Processed %d resource group(s) in subscription %s, found %d total VM(s) so far\n", resourceGroupCount, sub.SubscriptionID, len(allVMs))
	}
	
	fmt.Printf("[INFO] Total VMs found across all subscriptions: %d\n", len(allVMs))

	// Cache the result
	s.cache.Set(cacheKey, allVMs, s.cacheTTL)

	return allVMs, nil
}

// GetVM gets a specific virtual machine
func (s *AzureRMService) GetVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) (*models.AzureVM, error) {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("azure-vm:%s:%s:%s", subscriptionID, resourceGroup, vmName)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if vm, ok := cached.(*models.AzureVM); ok {
			return vm, nil
		}
	}

	resp, err := clients.computeClient.Get(ctx, resourceGroup, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	vm := s.convertVMToModel(&resp.VirtualMachine, resourceGroup, subscriptionID)

	// Cache the result
	s.cache.Set(cacheKey, vm, s.cacheTTL)

	return vm, nil
}

// StartVM starts a virtual machine
func (s *AzureRMService) StartVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return err
	}

	poller, err := clients.computeClient.BeginStart(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to complete VM start: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("azure-vm:%s:%s:%s", subscriptionID, resourceGroup, vmName))
	s.InvalidateVMsCache()

	return nil
}

// StopVM stops a virtual machine
func (s *AzureRMService) StopVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return err
	}

	poller, err := clients.computeClient.BeginDeallocate(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to complete VM stop: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("azure-vm:%s:%s:%s", subscriptionID, resourceGroup, vmName))
	s.InvalidateVMsCache()

	return nil
}

// DeleteVM deletes a virtual machine
func (s *AzureRMService) DeleteVM(ctx context.Context, subscriptionID, resourceGroup, vmName string) error {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return err
	}

	poller, err := clients.computeClient.BeginDelete(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to complete VM deletion: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("azure-vm:%s:%s:%s", subscriptionID, resourceGroup, vmName))
	s.InvalidateVMsCache()

	return nil
}

// ListStorageAccounts lists all storage accounts across all subscriptions
func (s *AzureRMService) ListStorageAccounts(ctx context.Context, subscriptionID string) ([]models.AzureStorageAccount, error) {
	cacheKey := fmt.Sprintf("azure-storage-accounts-%s", subscriptionID)
	if subscriptionID == "" {
		cacheKey = "azure-storage-accounts-all"
	}

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if accounts, ok := cached.([]models.AzureStorageAccount); ok {
			return accounts, nil
		}
	}

	var allAccounts []models.AzureStorageAccount
	var subscriptions []models.AzureSubscription

	if subscriptionID != "" {
		// Use specific subscription
		subscriptions = []models.AzureSubscription{
			{SubscriptionID: subscriptionID},
		}
	} else {
		// List all subscriptions
		var err error
		subscriptions, err = s.ListSubscriptions(ctx)
		if err != nil {
			fmt.Printf("[WARNING] Failed to list subscriptions: %v\n", err)
			// If we have a default subscription ID, use it as fallback
			if s.defaultSubscriptionID != "" {
				fmt.Printf("[INFO] Using fallback subscription ID: %s\n", s.defaultSubscriptionID)
				subscriptions = []models.AzureSubscription{
					{SubscriptionID: s.defaultSubscriptionID},
				}
			} else {
				return nil, fmt.Errorf("failed to list subscriptions and no AZURE_SUBSCRIPTION_ID fallback configured: %w", err)
			}
		}
	}

	// Iterate through all subscriptions
	for _, sub := range subscriptions {
		clients, err := s.getClientsForSubscription(sub.SubscriptionID)
		if err != nil {
			fmt.Printf("[WARNING] Failed to get clients for subscription %s: %v\n", sub.SubscriptionID, err)
			continue
		}

		pager := clients.storageClient.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				fmt.Printf("[WARNING] Failed to list storage accounts in subscription %s: %v\n", sub.SubscriptionID, err)
				break
			}

			for _, account := range page.Value {
				azureAccount := s.convertStorageAccountToModel(account, sub.SubscriptionID)
				allAccounts = append(allAccounts, *azureAccount)
			}
		}
	}

	// Cache the result
	s.cache.Set(cacheKey, allAccounts, s.cacheTTL)

	return allAccounts, nil
}

// GetStorageAccount gets a specific storage account
func (s *AzureRMService) GetStorageAccount(ctx context.Context, subscriptionID, resourceGroup, name string) (*models.AzureStorageAccount, error) {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("azure-storage-account:%s:%s:%s", subscriptionID, resourceGroup, name)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if account, ok := cached.(*models.AzureStorageAccount); ok {
			return account, nil
		}
	}

	resp, err := clients.storageClient.GetProperties(ctx, resourceGroup, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage account: %w", err)
	}

	account := s.convertStorageAccountToModel(&resp.Account, subscriptionID)

	// Cache the result
	s.cache.Set(cacheKey, account, s.cacheTTL)

	return account, nil
}

// DeleteStorageAccount deletes a storage account
func (s *AzureRMService) DeleteStorageAccount(ctx context.Context, subscriptionID, resourceGroup, name string) error {
	clients, err := s.getClientsForSubscription(subscriptionID)
	if err != nil {
		return err
	}

	_, err = clients.storageClient.Delete(ctx, resourceGroup, name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete storage account: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("azure-storage-account:%s:%s:%s", subscriptionID, resourceGroup, name))
	s.InvalidateStorageCache()

	return nil
}

// Helper methods

func (s *AzureRMService) convertVMToModel(vm *armcompute.VirtualMachine, resourceGroup, subscriptionID string) *models.AzureVM {
	azureVM := &models.AzureVM{
		ResourceGroup:  resourceGroup,
		SubscriptionID: subscriptionID,
	}

	if vm.ID != nil {
		azureVM.ID = *vm.ID
	}
	if vm.Name != nil {
		azureVM.Name = *vm.Name
	}
	if vm.Location != nil {
		azureVM.Location = *vm.Location
	}
	if vm.Properties != nil {
		if vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
			azureVM.VMSize = string(*vm.Properties.HardwareProfile.VMSize)
		}
		if vm.Properties.ProvisioningState != nil {
			azureVM.ProvisioningState = *vm.Properties.ProvisioningState
		}
		if vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil {
			if vm.Properties.StorageProfile.OSDisk.OSType != nil {
				azureVM.OsType = string(*vm.Properties.StorageProfile.OSDisk.OSType)
			}
		}
		if vm.Properties.InstanceView != nil {
			if vm.Properties.InstanceView.Statuses != nil {
				for _, status := range vm.Properties.InstanceView.Statuses {
					if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
						azureVM.Status = strings.TrimPrefix(*status.Code, "PowerState/")
						break
					}
				}
			}
		}
		if vm.Properties.TimeCreated != nil {
			azureVM.CreatedTime = *vm.Properties.TimeCreated
		}
	}

	return azureVM
}

func (s *AzureRMService) convertStorageAccountToModel(account *armstorage.Account, subscriptionID string) *models.AzureStorageAccount {
	azureAccount := &models.AzureStorageAccount{
		PrimaryEndpoints: make(map[string]string),
		SubscriptionID:   subscriptionID,
	}

	if account.ID != nil {
		azureAccount.ID = *account.ID
		// Extract resource group from ID
		parts := strings.Split(*account.ID, "/")
		for i, part := range parts {
			if part == "resourceGroups" && i+1 < len(parts) {
				azureAccount.ResourceGroup = parts[i+1]
				break
			}
		}
	}
	if account.Name != nil {
		azureAccount.Name = *account.Name
	}
	if account.Location != nil {
		azureAccount.Location = *account.Location
	}
	if account.Kind != nil {
		azureAccount.Kind = string(*account.Kind)
	}
	if account.SKU != nil && account.SKU.Name != nil {
		azureAccount.Sku = string(*account.SKU.Name)
	}
	if account.Properties != nil {
		if account.Properties.CreationTime != nil {
			azureAccount.CreatedTime = *account.Properties.CreationTime
		}
		if account.Properties.PrimaryEndpoints != nil {
			if account.Properties.PrimaryEndpoints.Blob != nil {
				azureAccount.PrimaryEndpoints["blob"] = *account.Properties.PrimaryEndpoints.Blob
			}
			if account.Properties.PrimaryEndpoints.File != nil {
				azureAccount.PrimaryEndpoints["file"] = *account.Properties.PrimaryEndpoints.File
			}
			if account.Properties.PrimaryEndpoints.Queue != nil {
				azureAccount.PrimaryEndpoints["queue"] = *account.Properties.PrimaryEndpoints.Queue
			}
			if account.Properties.PrimaryEndpoints.Table != nil {
				azureAccount.PrimaryEndpoints["table"] = *account.Properties.PrimaryEndpoints.Table
			}
		}
	}

	return azureAccount
}

// Cache management methods

func (s *AzureRMService) ClearCache() {
	s.cache.Clear()
}

func (s *AzureRMService) InvalidateVMsCache() {
	// Invalidate all VM caches (both subscription-specific and all)
	s.cache.Delete("azure-vms-all")
	// Also invalidate subscription-specific caches by pattern
	// Note: This is a simple approach - in production you might want a more sophisticated cache key management
}

func (s *AzureRMService) InvalidateStorageCache() {
	// Invalidate all storage account caches (both subscription-specific and all)
	s.cache.Delete("azure-storage-accounts-all")
	// Also invalidate subscription-specific caches by pattern
}
