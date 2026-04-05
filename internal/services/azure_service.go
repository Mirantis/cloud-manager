// Package services provides business logic and external service integrations
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/google/uuid"
	"github.com/rusik69/aws-iam-manager/internal/models"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
)

// AzureService handles Azure AD operations
type AzureService struct {
	client     *msgraphsdk.GraphServiceClient
	credential *azidentity.ClientSecretCredential
	cache      *Cache
	cacheTTL   time.Duration
}

// NewAzureService creates a new Azure service instance
func NewAzureService() (*AzureService, error) {
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

	// Create Microsoft Graph client
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %w", err)
	}

	return &AzureService{
		client:     client,
		credential: cred,
		cache:      NewCache(),
		cacheTTL:   5 * time.Minute,
	}, nil
}

// ListEnterpriseApplications lists all enterprise applications (service principals) in Azure AD
func (s *AzureService) ListEnterpriseApplications(ctx context.Context) ([]models.AzureEnterpriseApplication, error) {
	const cacheKey = "azure-enterprise-apps"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if apps, ok := cached.([]models.AzureEnterpriseApplication); ok {
			return apps, nil
		}
	}

	// First, get all applications to build creation date map (with pagination)
	appCreationDates := make(map[string]time.Time)
	appsTop := int32(999) // Maximum allowed by Azure API
	appsRequestConfig := &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Select: []string{"id", "appId", "createdDateTime"},
			Top:    &appsTop,
		},
	}

	// Fetch applications: first page via SDK, then follow @odata.nextLink with HTTP.
	for once := true; once; once = false {
		appsResult, err := s.client.Applications().Get(ctx, appsRequestConfig)
		if err != nil {
			// If we can't get applications, continue without creation dates
			fmt.Printf("[WARNING] Failed to fetch application creation dates: %v\n", err)
			break
		}

		// Add creation dates to map
		pageApps := appsResult.GetValue()
		if len(pageApps) == 0 {
			break
		}

		for _, app := range pageApps {
			appId := getStringValue(app.GetAppId())
			if appId != "" {
				createdDateTime := getTimeValue(app.GetCreatedDateTime())
				if !createdDateTime.IsZero() {
					appCreationDates[appId] = createdDateTime
				}
			}
		}

		// Check for next page
		nextLink := appsResult.GetOdataNextLink()
		if nextLink == nil || *nextLink == "" {
			break
		}

		// Use nextLink URL to fetch next page via HTTP request
		// Microsoft Graph API doesn't support $skip, so we must use the full nextLink URL
		nextPageAppDates, nextNextLink, err := s.fetchApplicationsPageFromURL(ctx, *nextLink)
		if err != nil {
			fmt.Printf("[WARNING] Failed to fetch next page of applications: %v\n", err)
			break
		}

		// Merge the fetched creation dates
		for appId, createdDateTime := range nextPageAppDates {
			appCreationDates[appId] = createdDateTime
		}

		for nextPtr := nextNextLink; nextPtr != nil && *nextPtr != ""; {
			nextPageAppDates, np, err := s.fetchApplicationsPageFromURL(ctx, *nextPtr)
			if err != nil {
				fmt.Printf("[WARNING] Failed to fetch next page of applications: %v\n", err)
				break
			}
			for appId, createdDateTime := range nextPageAppDates {
				appCreationDates[appId] = createdDateTime
			}
			nextPtr = np
		}
	}

	// Get all service principals (enterprise applications) with pagination
	var apps []models.AzureEnterpriseApplication
	top := int32(999) // Maximum allowed by Azure API
	requestConfig := &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Select: []string{"id", "appId", "displayName", "accountEnabled", "appOwnerOrganizationId", "appRoleAssignmentRequired", "servicePrincipalType", "tags", "homepage", "replyUrls"},
			Top:     &top,
		},
	}

	// Fetch service principals: first page via SDK, then follow @odata.nextLink with HTTP.
	for once := true; once; once = false {
		result, err := s.client.ServicePrincipals().Get(ctx, requestConfig)
		if err != nil {
			// Provide more helpful error messages for common authentication issues
			errMsg := err.Error()
			if strings.Contains(errMsg, "AADSTS700016") || strings.Contains(errMsg, "not found in the directory") {
				return nil, fmt.Errorf("azure app registration not found in tenant. Verify that:\n1. The Client ID matches the Application (client) ID in Azure Portal\n2. The Tenant ID matches the Directory (tenant) ID\n3. The app registration exists in the correct tenant\n4. The app has been consented/admin consented\n\nOriginal error: %w", err)
			}
			if strings.Contains(errMsg, "unauthorized_client") || strings.Contains(errMsg, "AADSTS7000215") {
				return nil, fmt.Errorf("azure authentication failed. Verify that:\n1. The Client Secret is correct and not expired\n2. The app registration has the required API permissions\n3. Admin consent has been granted for the permissions\n\nOriginal error: %w", err)
			}
			return nil, fmt.Errorf("failed to list enterprise applications: %w", err)
		}

		// Process current page results
		pageSPs := result.GetValue()
		if len(pageSPs) == 0 {
			break
		}

		for _, sp := range pageSPs {
			appId := getStringValue(sp.GetAppId())
			createdDateTime := appCreationDates[appId] // Get creation date from application object

			app := models.AzureEnterpriseApplication{
				ID:                        getStringValue(sp.GetId()),
				AppID:                     appId,
				DisplayName:               getStringValue(sp.GetDisplayName()),
				AccountEnabled:            getBoolValue(sp.GetAccountEnabled()),
				AppOwnerOrgID:             getUUIDStringValue(sp.GetAppOwnerOrganizationId()),
				AppRoleAssignmentRequired: getBoolValue(sp.GetAppRoleAssignmentRequired()),
				ServicePrincipalType:      getStringValue(sp.GetServicePrincipalType()),
				Tags:                      sp.GetTags(),
				Homepage:                  getStringValue(sp.GetHomepage()),
				ReplyUrls:                 sp.GetReplyUrls(),
				CreatedDateTime:           createdDateTime,
			}

			apps = append(apps, app)
		}

		// Check for next page
		nextLink := result.GetOdataNextLink()
		if nextLink == nil || *nextLink == "" {
			break
		}

		// Use nextLink URL to fetch next page via HTTP request
		// Microsoft Graph API doesn't support $skip, so we must use the full nextLink URL
		nextPageSPs, nextNextLink, nextPageAppDates, err := s.fetchServicePrincipalsPageFromURL(ctx, *nextLink)
		if err != nil {
			fmt.Printf("[WARNING] Failed to fetch next page of service principals: %v\n", err)
			break
		}

		// Merge creation dates
		for appId, createdDateTime := range nextPageAppDates {
			appCreationDates[appId] = createdDateTime
		}

		// Process the fetched service principals
		for _, sp := range nextPageSPs {
			createdDateTime := appCreationDates[sp.AppID]
			sp.CreatedDateTime = createdDateTime
			apps = append(apps, sp)
		}

		for nextPtr := nextNextLink; nextPtr != nil && *nextPtr != ""; {
			nextPageSPs, np, nextPageAppDates, err := s.fetchServicePrincipalsPageFromURL(ctx, *nextPtr)
			if err != nil {
				fmt.Printf("[WARNING] Failed to fetch next page of service principals: %v\n", err)
				break
			}
			for appId, createdDateTime := range nextPageAppDates {
				appCreationDates[appId] = createdDateTime
			}
			for _, sp := range nextPageSPs {
				createdDateTime := appCreationDates[sp.AppID]
				sp.CreatedDateTime = createdDateTime
				apps = append(apps, sp)
			}
			nextPtr = np
		}
	}

	// Cache the result
	s.cache.Set(cacheKey, apps, s.cacheTTL)

	return apps, nil
}

// GetEnterpriseApplication gets a specific enterprise application by ID
func (s *AzureService) GetEnterpriseApplication(ctx context.Context, appID string) (*models.AzureEnterpriseApplication, error) {
	cacheKey := fmt.Sprintf("azure-enterprise-app:%s", appID)

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if app, ok := cached.(*models.AzureEnterpriseApplication); ok {
			return app, nil
		}
	}

	// Get the service principal
	sp, err := s.client.ServicePrincipals().ByServicePrincipalId(appID).Get(ctx, &serviceprincipals.ServicePrincipalItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalItemRequestBuilderGetQueryParameters{
			Select: []string{"id", "appId", "displayName", "accountEnabled", "appOwnerOrganizationId", "appRoleAssignmentRequired", "servicePrincipalType", "tags", "homepage", "replyUrls"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get enterprise application: %w", err)
	}

	// Get the application object to retrieve creation date
	appId := getStringValue(sp.GetAppId())
	var createdDateTime time.Time
	if appId != "" {
		// Try to get the application by appId to get creation date
		filter := fmt.Sprintf("appId eq '%s'", appId)
		appsResult, err := s.client.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{
			QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
				Filter: &filter,
				Select: []string{"createdDateTime"},
			},
		})
		if err == nil && appsResult != nil && len(appsResult.GetValue()) > 0 {
			createdDateTime = getTimeValue(appsResult.GetValue()[0].GetCreatedDateTime())
		}
	}

	app := &models.AzureEnterpriseApplication{
		ID:                        getStringValue(sp.GetId()),
		AppID:                     appId,
		DisplayName:               getStringValue(sp.GetDisplayName()),
		AccountEnabled:            getBoolValue(sp.GetAccountEnabled()),
		AppOwnerOrgID:             getUUIDStringValue(sp.GetAppOwnerOrganizationId()),
		AppRoleAssignmentRequired: getBoolValue(sp.GetAppRoleAssignmentRequired()),
		ServicePrincipalType:      getStringValue(sp.GetServicePrincipalType()),
		Tags:                      sp.GetTags(),
		Homepage:                  getStringValue(sp.GetHomepage()),
		ReplyUrls:                 sp.GetReplyUrls(),
		CreatedDateTime:           createdDateTime,
	}

	// Cache the result
	s.cache.Set(cacheKey, app, s.cacheTTL)

	return app, nil
}

// DeleteEnterpriseApplication deletes an enterprise application (service principal) by ID
func (s *AzureService) DeleteEnterpriseApplication(ctx context.Context, appID string) error {
	// Delete the service principal
	err := s.client.ServicePrincipals().ByServicePrincipalId(appID).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete enterprise application: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("azure-enterprise-app:%s", appID))
	s.cache.Delete("azure-enterprise-apps")

	return nil
}

// ClearCache clears all cached data
func (s *AzureService) ClearCache() {
	s.cache.Clear()
}

// InvalidateEnterpriseApplicationsCache invalidates the enterprise applications cache
func (s *AzureService) InvalidateEnterpriseApplicationsCache() {
	s.cache.Delete("azure-enterprise-apps")
}

// InvalidateEnterpriseApplicationCache invalidates cache for a specific enterprise application
func (s *AzureService) InvalidateEnterpriseApplicationCache(appID string) {
	s.cache.Delete(fmt.Sprintf("azure-enterprise-app:%s", appID))
	s.cache.Delete("azure-enterprise-apps")
}

// Helper functions to safely get values from pointers
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getBoolValue(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func getUUIDStringValue(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func getTimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// fetchApplicationsPageFromURL fetches applications from a nextLink URL using HTTP request
// Returns app data as map[string]time.Time (appId -> createdDateTime) and nextLink
func (s *AzureService) fetchApplicationsPageFromURL(ctx context.Context, url string) (map[string]time.Time, *string, error) {
	// Get access token - use the same scopes as the main client
	token, err := s.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Make HTTP request to nextLink URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var jsonResponse struct {
		Value    []struct {
			AppID          *string `json:"appId"`
			CreatedDateTime *string `json:"createdDateTime"`
		} `json:"value"`
		NextLink *string `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract appId -> createdDateTime mapping
	appDates := make(map[string]time.Time)
	for _, app := range jsonResponse.Value {
		if app.AppID == nil || *app.AppID == "" {
			continue
		}
		if app.CreatedDateTime != nil && *app.CreatedDateTime != "" {
			if createdTime, err := time.Parse(time.RFC3339, *app.CreatedDateTime); err == nil {
				appDates[*app.AppID] = createdTime
			}
		}
	}

	return appDates, jsonResponse.NextLink, nil
}

// fetchServicePrincipalsPageFromURL fetches service principals from a nextLink URL using HTTP request
// Returns service principal data and nextLink - we'll parse and convert to our models
func (s *AzureService) fetchServicePrincipalsPageFromURL(ctx context.Context, url string) ([]models.AzureEnterpriseApplication, *string, map[string]time.Time, error) {
	// Get access token - use the same scopes as the main client
	token, err := s.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Make HTTP request to nextLink URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var jsonResponse struct {
		Value    []json.RawMessage `json:"value"`
		NextLink *string            `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Parse each service principal manually
	sps := make([]models.AzureEnterpriseApplication, 0, len(jsonResponse.Value))
	appDates := make(map[string]time.Time)
	for _, rawSP := range jsonResponse.Value {
		var spData struct {
			ID                        *string   `json:"id"`
			AppID                     *string   `json:"appId"`
			DisplayName               *string   `json:"displayName"`
			AccountEnabled            *bool     `json:"accountEnabled"`
			AppOwnerOrgID             *string   `json:"appOwnerOrganizationId"`
			AppRoleAssignmentRequired *bool     `json:"appRoleAssignmentRequired"`
			ServicePrincipalType      *string   `json:"servicePrincipalType"`
			Tags                      []string  `json:"tags"`
			Homepage                  *string   `json:"homepage"`
			ReplyUrls                 []string  `json:"replyUrls"`
		}
		if err := json.Unmarshal(rawSP, &spData); err != nil {
			continue
		}

		app := models.AzureEnterpriseApplication{
			ID:                        getStringValue(spData.ID),
			AppID:                     getStringValue(spData.AppID),
			DisplayName:               getStringValue(spData.DisplayName),
			AccountEnabled:            getBoolValue(spData.AccountEnabled),
			AppOwnerOrgID:             getUUIDStringValue(nil), // Will be set from appCreationDates
			AppRoleAssignmentRequired: getBoolValue(spData.AppRoleAssignmentRequired),
			ServicePrincipalType:      getStringValue(spData.ServicePrincipalType),
			Tags:                      spData.Tags,
			Homepage:                  getStringValue(spData.Homepage),
			ReplyUrls:                 spData.ReplyUrls,
			CreatedDateTime:           time.Time{}, // Will be set from appCreationDates
		}
		sps = append(sps, app)
	}

	return sps, jsonResponse.NextLink, appDates, nil
}
