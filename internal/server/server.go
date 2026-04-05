package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/config"
	"github.com/rusik69/aws-iam-manager/internal/db"
	"github.com/rusik69/aws-iam-manager/internal/handlers"
	"github.com/rusik69/aws-iam-manager/internal/middleware"
	"github.com/rusik69/aws-iam-manager/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Frontend files are served from filesystem
// TODO: Add embed support for production builds

type Server struct {
	config          config.Config
	handler         *handlers.Handler
	azureHandler    *handlers.AzureHandler
	azureRMHandler  *handlers.AzureRMHandler
	ssoHandler      *handlers.SSOHandler
	ssoInitError    error
}

func NewServer(cfg config.Config) (*Server, error) {
	appDB, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		_ = appDB.Close()
		return nil, fmt.Errorf("hash admin password: %w", err)
	}
	if upsertErr := appDB.UpsertUser(cfg.AdminUsername, string(hash), "admin"); upsertErr != nil {
		_ = appDB.Close()
		return nil, fmt.Errorf("seed admin user: %w", upsertErr)
	}
	log.Printf("[INFO] Application users database: %s", cfg.DBPath)

	awsService := services.NewAWSService(cfg)
	handler := handlers.NewHandler(awsService, cfg, appDB)

	// Initialize Azure handler (optional - will log error if credentials not configured)
	var azureHandler *handlers.AzureHandler
	azureService, err := services.NewAzureService()
	if err != nil {
		log.Printf("[WARNING] Azure service not initialized: %v", err)
		log.Printf("[INFO] Azure endpoints will not be available. Set AZURE_TENANT_ID, AZURE_CLIENT_ID, and AZURE_CLIENT_SECRET to enable Azure features.")
	} else {
		azureHandler = handlers.NewAzureHandler(azureService)
		log.Printf("[INFO] Azure service initialized successfully")
	}

	// Initialize Azure Resource Manager handler (optional)
	var azureRMHandler *handlers.AzureRMHandler
	azureRMService, err := services.NewAzureRMService()
	if err != nil {
		log.Printf("[WARNING] Azure Resource Manager service not initialized: %v", err)
		log.Printf("[INFO] Azure Resource Manager endpoints will not be available. Set AZURE_TENANT_ID, AZURE_CLIENT_ID, and AZURE_CLIENT_SECRET to enable Azure RM features.")
	} else {
		azureRMHandler = handlers.NewAzureRMHandler(azureRMService)
		log.Printf("[INFO] Azure Resource Manager service initialized successfully (works across all subscriptions)")
	}

	// Initialize SSO handler (optional - will log error if Identity Center not configured)
	var ssoHandler *handlers.SSOHandler
	var ssoInitError error
	ssoService, err := services.NewSSOService(awsService)
	if err != nil {
		ssoInitError = err
		log.Printf("[WARNING] SSO service not initialized: %v", err)
		log.Printf("[INFO] SSO endpoints will return error messages. Ensure IAM Identity Center is enabled and you have the required permissions.")
	} else {
		ssoHandler = handlers.NewSSOHandler(ssoService)
		log.Printf("[INFO] SSO service initialized successfully")
	}

	return &Server{
		config:         cfg,
		handler:        handler,
		azureHandler:   azureHandler,
		azureRMHandler: azureRMHandler,
		ssoHandler:     ssoHandler,
		ssoInitError:   ssoInitError,
	}, nil
}

// customLogger is a logging middleware that skips health check endpoints
func customLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health check endpoints
		path := c.Request.URL.Path
		if path == "/ping" || path == "/ready" || path == "/health" {
			c.Next()
			return
		}

		// Start timer
		start := time.Now()
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only if not a health check endpoint
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[%s] %s %s %d %v %s",
			clientIP,
			method,
			path,
			statusCode,
			latency,
			c.Errors.String(),
		)
	}
}

func (s *Server) SetupRoutes() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(customLogger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoints
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Public auth routes (no authentication required)
	api := r.Group("/api")
	{
		// Authentication routes (public)
		api.POST("/auth/login", s.handler.Login)
		api.POST("/auth/logout", s.handler.Logout)
		api.GET("/auth/check", s.handler.CheckAuth)
	}

	// Admin-only application user management
	adminAPI := r.Group("/api/admin")
	adminAPI.Use(middleware.AuthMiddleware())
	adminAPI.Use(middleware.AdminMiddleware())
	{
		adminAPI.POST("/users", s.handler.CreateAppUser)
		adminAPI.GET("/users", s.handler.ListAppUsers)
		adminAPI.DELETE("/users/:username", s.handler.DeleteAppUser)
		adminAPI.PUT("/users/:username/password", s.handler.UpdateAppUserPassword)
	}

	// Protected API routes (require authentication)
	apiProtected := r.Group("/api")
	apiProtected.Use(middleware.AuthMiddleware())
	apiProtected.Use(middleware.WriteAccessMiddleware())
	{
		// Authentication info route
		apiProtected.GET("/auth/user", s.handler.GetCurrentUser)

		// Account and user management routes
		apiProtected.GET("/accounts", s.handler.ListAccounts)
		apiProtected.GET("/users", s.handler.ListAllUsers) // All users from all accounts in parallel
		apiProtected.GET("/accounts/:accountId/users", s.handler.ListUsers)
		apiProtected.GET("/accounts/:accountId/users/:username", s.handler.GetUser)
		apiProtected.DELETE("/accounts/:accountId/users/:username", s.handler.DeleteUser)
		apiProtected.DELETE("/accounts/:accountId/users/:username/password", s.handler.DeleteUserPassword)
		apiProtected.POST("/accounts/:accountId/users/inactive/delete", s.handler.DeleteInactiveUsers)
		apiProtected.POST("/accounts/:accountId/users/:username/password/rotate", s.handler.RotateUserPassword)
		apiProtected.POST("/accounts/:accountId/users/:username/keys", s.handler.CreateAccessKey)
		apiProtected.DELETE("/accounts/:accountId/users/:username/keys/:keyId", s.handler.DeleteAccessKey)
		apiProtected.PUT("/accounts/:accountId/users/:username/keys/:keyId/rotate", s.handler.RotateAccessKey)

		// IP management routes
		apiProtected.GET("/public-ips", s.handler.ListPublicIPs)

		// Security groups routes
		apiProtected.GET("/security-groups", s.handler.ListSecurityGroups)
		apiProtected.GET("/accounts/:accountId/security-groups", s.handler.ListSecurityGroupsByAccount)
		apiProtected.GET("/accounts/:accountId/regions/:region/security-groups/:groupId", s.handler.GetSecurityGroup)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/security-groups/:groupId", s.handler.DeleteSecurityGroup)

		// Snapshots routes
		apiProtected.GET("/snapshots", s.handler.ListSnapshots)
		apiProtected.GET("/accounts/:accountId/snapshots", s.handler.ListSnapshotsByAccount)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/snapshots/:snapshotId", s.handler.DeleteSnapshot)
		apiProtected.DELETE("/accounts/:accountId/snapshots/old", s.handler.DeleteOldSnapshots)

		// EC2 instances routes
		apiProtected.GET("/ec2-instances", s.handler.ListEC2Instances)
		apiProtected.POST("/accounts/:accountId/regions/:region/instances/:instanceId/stop", s.handler.StopEC2Instance)
		apiProtected.POST("/accounts/:accountId/regions/:region/instances/:instanceId/terminate", s.handler.TerminateEC2Instance)

		// EBS volumes routes
		apiProtected.GET("/ebs-volumes", s.handler.ListEBSVolumes)
		apiProtected.GET("/accounts/:accountId/ebs-volumes", s.handler.ListEBSVolumesByAccount)
		apiProtected.POST("/accounts/:accountId/regions/:region/volumes/:volumeId/detach", s.handler.DetachEBSVolume)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/volumes/:volumeId", s.handler.DeleteEBSVolume)

		// S3 buckets routes
		apiProtected.GET("/s3-buckets", s.handler.ListS3Buckets)
		apiProtected.GET("/accounts/:accountId/s3-buckets", s.handler.ListS3BucketsByAccount)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/buckets/:bucketName", s.handler.DeleteS3Bucket)

		// IAM roles routes
		apiProtected.GET("/roles", s.handler.ListAllRoles)
		apiProtected.GET("/accounts/:accountId/roles", s.handler.ListRoles)
		apiProtected.GET("/accounts/:accountId/roles/:roleName", s.handler.GetRole)
		apiProtected.DELETE("/accounts/:accountId/roles/:roleName", s.handler.DeleteRole)

		// Load balancer routes
		apiProtected.GET("/load-balancers", s.handler.ListAllLoadBalancers)
		apiProtected.GET("/accounts/:accountId/load-balancers", s.handler.ListLoadBalancersByAccount)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/load-balancers", s.handler.DeleteLoadBalancer)

		// VPC routes
		apiProtected.GET("/vpcs", s.handler.ListVPCs)
		apiProtected.GET("/accounts/:accountId/vpcs", s.handler.ListVPCsByAccount)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/vpcs/:vpcId", s.handler.DeleteVPC)

		// NAT Gateway routes
		apiProtected.GET("/nat-gateways", s.handler.ListNATGateways)
		apiProtected.GET("/accounts/:accountId/nat-gateways", s.handler.ListNATGatewaysByAccount)
		apiProtected.DELETE("/accounts/:accountId/regions/:region/nat-gateways/:natGatewayId", s.handler.DeleteNATGateway)

		// Route53 domain routes
		apiProtected.GET("/route53-domains", s.handler.ListRoute53Domains)
		apiProtected.GET("/accounts/:accountId/route53-domains/:hostedZoneId/records", s.handler.ListRoute53Records)

		// Azure routes (combine Azure AD and Azure RM routes in a single /azure group)
		if s.azureHandler != nil || s.azureRMHandler != nil {
			azure := apiProtected.Group("/azure")
			
			// Azure AD routes (if Azure AD handler is configured)
			if s.azureHandler != nil {
				azure.GET("/enterprise-applications", s.azureHandler.ListEnterpriseApplications)
				azure.GET("/enterprise-applications/:appId", s.azureHandler.GetEnterpriseApplication)
				azure.DELETE("/enterprise-applications/:appId", s.azureHandler.DeleteEnterpriseApplication)

				// Azure cache management routes
				azure.POST("/cache/clear", s.azureHandler.ClearAzureCache)
				azure.POST("/cache/enterprise-applications/invalidate", s.azureHandler.InvalidateEnterpriseApplicationsCache)
				azure.POST("/cache/enterprise-applications/:appId/invalidate", s.azureHandler.InvalidateEnterpriseApplicationCache)
			}

			// Azure Resource Manager routes (if Azure RM handler is configured)
			if s.azureRMHandler != nil {
				// Subscription routes
				azure.GET("/subscriptions", s.azureRMHandler.ListSubscriptions)

				// VM routes
				azure.GET("/vms", s.azureRMHandler.ListVMs)
				azure.GET("/subscriptions/:subscriptionId/vms/:resourceGroup/:vmName", s.azureRMHandler.GetVM)
				azure.POST("/subscriptions/:subscriptionId/vms/:resourceGroup/:vmName/start", s.azureRMHandler.StartVM)
				azure.POST("/subscriptions/:subscriptionId/vms/:resourceGroup/:vmName/stop", s.azureRMHandler.StopVM)
				azure.DELETE("/subscriptions/:subscriptionId/vms/:resourceGroup/:vmName", s.azureRMHandler.DeleteVM)

				// Storage account routes
				azure.GET("/storage-accounts", s.azureRMHandler.ListStorageAccounts)
				azure.GET("/subscriptions/:subscriptionId/storage-accounts/:resourceGroup/:name", s.azureRMHandler.GetStorageAccount)
				azure.DELETE("/subscriptions/:subscriptionId/storage-accounts/:resourceGroup/:name", s.azureRMHandler.DeleteStorageAccount)

				// Azure RM cache management routes
				azure.POST("/rm/cache/clear", s.azureRMHandler.ClearAzureRMCache)
				azure.POST("/rm/cache/vms/invalidate", s.azureRMHandler.InvalidateVMsCache)
				azure.POST("/rm/cache/storage/invalidate", s.azureRMHandler.InvalidateStorageCache)
			}
		}

		// SSO routes (always register, but return error if not initialized)
		sso := apiProtected.Group("/sso")
		
		// Helper function to handle SSO not initialized
		handleSSONotInitialized := func(c *gin.Context) {
			errorMsg := "SSO service is not available"
			details := "Ensure IAM Identity Center is enabled and the service has the required permissions."
			if s.ssoInitError != nil {
				errorMsg = s.ssoInitError.Error()
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   errorMsg,
				"details": details,
			})
		}
		
		// Diagnostic endpoint to check SSO status
		sso.GET("/status", func(c *gin.Context) {
			if s.ssoHandler != nil {
				c.JSON(http.StatusOK, gin.H{
					"status": "available",
					"message": "SSO service is initialized and ready",
				})
			} else {
				errorMsg := "SSO service is not available"
				if s.ssoInitError != nil {
					errorMsg = s.ssoInitError.Error()
				}
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":  "unavailable",
					"error":   errorMsg,
					"details": "Ensure IAM Identity Center is enabled and the service has the required permissions. Check server logs for more details.",
				})
			}
		})
		
		// Identity Center instance
		if s.ssoHandler != nil {
			sso.GET("/instance", s.ssoHandler.GetIdentityCenterInstance)
		} else {
			sso.GET("/instance", handleSSONotInitialized)
		}

		// User routes
		if s.ssoHandler != nil {
			sso.GET("/users", s.ssoHandler.ListSSOUsers)
			sso.GET("/users/:userId", s.ssoHandler.GetSSOUser)
			sso.GET("/users/:userId/assignments", s.ssoHandler.ListUserAccountAssignments)
		} else {
			sso.GET("/users", handleSSONotInitialized)
			sso.GET("/users/:userId", handleSSONotInitialized)
			sso.GET("/users/:userId/assignments", handleSSONotInitialized)
		}

		// Group routes
		if s.ssoHandler != nil {
			sso.GET("/groups", s.ssoHandler.ListSSOGroups)
			sso.GET("/groups/:groupId", s.ssoHandler.GetSSOGroup)
			sso.GET("/groups/:groupId/members", s.ssoHandler.ListGroupMembers)
			sso.GET("/groups/:groupId/assignments", s.ssoHandler.ListGroupAccountAssignments)
		} else {
			sso.GET("/groups", handleSSONotInitialized)
			sso.GET("/groups/:groupId", handleSSONotInitialized)
			sso.GET("/groups/:groupId/members", handleSSONotInitialized)
			sso.GET("/groups/:groupId/assignments", handleSSONotInitialized)
		}

		// Account assignment routes
		if s.ssoHandler != nil {
			sso.GET("/accounts/:accountId/assignments", s.ssoHandler.ListAccountAssignments)
		} else {
			sso.GET("/accounts/:accountId/assignments", handleSSONotInitialized)
		}

		// Aggregated assignment routes
		if s.ssoHandler != nil {
			sso.GET("/user-assignments", s.ssoHandler.ListAllUserAssignments)
			sso.GET("/group-assignments", s.ssoHandler.ListAllGroupAssignments)
			sso.GET("/account-assignments", s.ssoHandler.ListAllAccountAssignments)
		} else {
			sso.GET("/user-assignments", handleSSONotInitialized)
			sso.GET("/group-assignments", handleSSONotInitialized)
			sso.GET("/account-assignments", handleSSONotInitialized)
		}

		// SSO cache management routes
		if s.ssoHandler != nil {
			sso.POST("/cache/clear", s.ssoHandler.ClearSSOCache)
			sso.POST("/cache/users/invalidate", s.ssoHandler.InvalidateSSOUsersCache)
			sso.POST("/cache/groups/invalidate", s.ssoHandler.InvalidateSSOGroupsCache)
			sso.POST("/cache/users/:userId/invalidate", s.ssoHandler.InvalidateSSOUserCache)
			sso.POST("/cache/groups/:groupId/invalidate", s.ssoHandler.InvalidateSSOGroupCache)
			sso.POST("/cache/accounts/:accountId/assignments/invalidate", s.ssoHandler.InvalidateAccountAssignmentsCache)
		} else {
			sso.POST("/cache/clear", handleSSONotInitialized)
			sso.POST("/cache/users/invalidate", handleSSONotInitialized)
			sso.POST("/cache/groups/invalidate", handleSSONotInitialized)
			sso.POST("/cache/users/:userId/invalidate", handleSSONotInitialized)
			sso.POST("/cache/groups/:groupId/invalidate", handleSSONotInitialized)
			sso.POST("/cache/accounts/:accountId/assignments/invalidate", handleSSONotInitialized)
		}

		// Cache management routes
		apiProtected.POST("/cache/clear", s.handler.ClearCache)
		apiProtected.POST("/cache/accounts/:accountId/invalidate", s.handler.InvalidateAccountCache)
		apiProtected.POST("/cache/accounts/:accountId/users/:username/invalidate", s.handler.InvalidateUserCache)
		apiProtected.POST("/cache/public-ips/invalidate", s.handler.InvalidatePublicIPsCache)
		apiProtected.POST("/cache/security-groups/invalidate", s.handler.InvalidateSecurityGroupsCache)
		apiProtected.POST("/cache/accounts/:accountId/security-groups/invalidate", s.handler.InvalidateAccountSecurityGroupsCache)
		apiProtected.POST("/cache/ec2-instances/invalidate", s.handler.InvalidateEC2InstancesCache)
		apiProtected.POST("/cache/ebs-volumes/invalidate", s.handler.InvalidateEBSVolumesCache)
		apiProtected.POST("/cache/s3-buckets/invalidate", s.handler.InvalidateS3BucketsCache)
		apiProtected.POST("/cache/roles/invalidate", s.handler.InvalidateRolesCache)
		apiProtected.POST("/cache/accounts/:accountId/roles/invalidate", s.handler.InvalidateAccountRolesCache)
		apiProtected.POST("/cache/load-balancers/invalidate", s.handler.InvalidateAllLoadBalancersCache)
		apiProtected.POST("/cache/accounts/:accountId/load-balancers/invalidate", s.handler.InvalidateLoadBalancersCache)
		apiProtected.POST("/cache/vpcs/invalidate", s.handler.InvalidateVPCsCache)
		apiProtected.POST("/cache/nat-gateways/invalidate", s.handler.InvalidateNATGatewaysCache)
		apiProtected.POST("/cache/route53-domains/invalidate", s.handler.InvalidateRoute53DomainsCache)
	}

	// Serve frontend
	s.setupFrontendRoutes(r)

	return r
}

func (s *Server) setupFrontendRoutes(r *gin.Engine) {
	log.Printf("[INFO] Serving frontend files from ./frontend/dist/")

	// Serve static assets
	r.Static("/assets", "./frontend/dist/assets")

	// Serve index.html for root and SPA routes
	indexHandler := func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	}

	r.GET("/", indexHandler)
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}
		indexHandler(c)
	})
}

func (s *Server) Start() error {
	r := s.SetupRoutes()

	log.Printf("[INFO] Server starting on port %s", s.config.Port)
	fmt.Printf("Server starting on port %s\n", s.config.Port)
	return r.Run(":" + s.config.Port)
}
