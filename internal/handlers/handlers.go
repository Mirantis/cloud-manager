// Package handlers provides HTTP request handlers for the AWS IAM manager
package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rusik69/aws-iam-manager/internal/config"
	"github.com/rusik69/aws-iam-manager/internal/db"
	"github.com/rusik69/aws-iam-manager/internal/middleware"
	"github.com/rusik69/aws-iam-manager/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	awsService services.AWSServiceInterface
	config     config.Config
	appDB      *db.DB
}

func NewHandler(awsService services.AWSServiceInterface, cfg config.Config, appDB *db.DB) *Handler {
	return &Handler{
		awsService: awsService,
		config:     cfg,
		appDB:      appDB,
	}
}

func (h *Handler) ListAccounts(c *gin.Context) {
	accounts, err := h.awsService.ListAccounts()
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] ListAccounts failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list AWS accounts. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *Handler) ListUsers(c *gin.Context) {
	accountID := c.Param("accountId")
	users, err := h.awsService.ListUsers(accountID)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] ListUsers failed for account %s: %v\n", accountID, err)
		// Check if it's an access issue
		if containsAccessDenied(err.Error()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied to account",
				"details": fmt.Sprintf("Cannot access account %s. The role may not exist or trust relationship is not configured.", accountID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list users for account %s. Check AWS credentials and permissions.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) ListAllUsers(c *gin.Context) {
	users, err := h.awsService.ListAllUsers()
	if err != nil {
		fmt.Printf("[ERROR] ListAllUsers failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list all users. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, users)
}

// containsAccessDenied checks if an error message indicates access denied
func containsAccessDenied(errMsg string) bool {
	return strings.Contains(errMsg, "AccessDenied") ||
		strings.Contains(errMsg, "assume role") ||
		strings.Contains(errMsg, "not authorized")
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Username and password are required",
		})
		return
	}

	// Validate credentials
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Username and password are required",
		})
		return
	}

	user, err := h.appDB.ValidateCredentials(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"message": "Username or password is incorrect",
		})
		return
	}

	// Create session
	sessionID := middleware.GenerateSessionID()
	middleware.GetSessionStore().SetSession(sessionID, user.Username, user.Role)

	// Set cookie - Secure=false for local development (HTTP), set to true in production (HTTPS)
	// SameSite=Lax allows cookies to be sent with same-site requests
	c.SetCookie("session_id", sessionID, int(24*time.Hour.Seconds()), "/", "", false, false)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Login successful",
		"username":      user.Username,
		"role":          user.Role,
		"authenticated": true,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		middleware.GetSessionStore().DeleteSession(sessionID)
	}

	// Clear cookie - Secure=false for local development
	c.SetCookie("session_id", "", -1, "/", "", false, false)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// CheckAuth checks if the user is authenticated
func (h *Handler) CheckAuth(c *gin.Context) {
	// Check for session cookie directly (this endpoint is public, so no middleware)
	sessionID, err := c.Cookie("session_id")
	if err != nil || sessionID == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
		})
		return
	}

	// Validate session
	session, exists := middleware.GetSessionStore().GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"username":     session.Username,
		"role":         session.Role,
	})
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	username, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	role, _ := middleware.GetCurrentRole(c)

	c.JSON(http.StatusOK, gin.H{
		"username":      username,
		"role":          role,
		"authenticated": true,
	})
}

func validAppRole(role string) bool {
	switch role {
	case "admin", "editor", "viewer":
		return true
	default:
		return false
	}
}

// CreateAppUserRequest is the body for POST /api/admin/users.
type CreateAppUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// CreateAppUser creates a new application user (admin only).
func (h *Handler) CreateAppUser(c *gin.Context) {
	var req CreateAppUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, password, and role are required"})
		return
	}
	if !validAppRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin, editor, or viewer"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := h.appDB.AddUser(req.Username, string(hash), req.Role); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists or could not be created"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User created", "username": req.Username, "role": req.Role})
}

// ListAppUsers lists all application users (admin only).
func (h *Handler) ListAppUsers(c *gin.Context) {
	users, err := h.appDB.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// DeleteAppUser removes an application user (admin only).
func (h *Handler) DeleteAppUser(c *gin.Context) {
	target := c.Param("username")
	self, _ := middleware.GetCurrentUser(c)
	if target == self {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
		return
	}
	if err := h.appDB.DeleteUser(target); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// UpdateAppUserPasswordRequest resets a user's password (admin only).
type UpdateAppUserPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// UpdateAppUserPassword sets a new password for a user (admin only).
func (h *Handler) UpdateAppUserPassword(c *gin.Context) {
	target := c.Param("username")
	var req UpdateAppUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := h.appDB.UpdatePassword(target, string(hash)); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

func (h *Handler) GetUser(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	user, err := h.awsService.GetUser(accountID, username)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] GetUser failed for user %s in account %s: %v\n", username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to get user %s from account %s. Check AWS credentials and permissions.", username, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) CreateAccessKey(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	response, err := h.awsService.CreateAccessKey(accountID, username)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] CreateAccessKey failed for user %s in account %s: %v\n", username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to create access key for user %s in account %s. Check AWS credentials and permissions.", username, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) DeleteAccessKey(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	keyID := c.Param("keyId")
	err := h.awsService.DeleteAccessKey(accountID, username, keyID)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] DeleteAccessKey failed for key %s, user %s in account %s: %v\n", keyID, username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to delete access key %s for user %s in account %s. Check AWS credentials and permissions.", keyID, username, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Access key deleted successfully"})
}

func (h *Handler) RotateAccessKey(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	keyID := c.Param("keyId")
	response, err := h.awsService.RotateAccessKey(accountID, username, keyID)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] RotateAccessKey failed for key %s, user %s in account %s: %v\n", keyID, username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to rotate access key %s for user %s in account %s. Check AWS credentials and permissions.", keyID, username, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	err := h.awsService.DeleteUser(accountID, username)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] DeleteUser failed for user %s in account %s: %v\n", username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to delete user %s from account %s. Check AWS credentials and permissions.", username, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *Handler) DeleteInactiveUsers(c *gin.Context) {
	accountID := c.Param("accountId")
	deletedUsers, failedUsers, err := h.awsService.DeleteInactiveUsers(accountID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteInactiveUsers failed for account %s: %v\n", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to delete inactive users from account %s. Check AWS credentials and permissions.", accountID),
		})
		return
	}

	response := gin.H{
		"message":       fmt.Sprintf("Deleted %d inactive user(s) successfully", len(deletedUsers)),
		"deleted_users": deletedUsers,
	}

	if len(failedUsers) > 0 {
		response["failed_users"] = failedUsers
		response["message"] = fmt.Sprintf("Deleted %d inactive user(s) successfully. Failed to delete %d user(s)", len(deletedUsers), len(failedUsers))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) DeleteUserPassword(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	err := h.awsService.DeleteUserPassword(accountID, username)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] DeleteUserPassword failed for user %s in account %s: %v\n", username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to delete password for user %s in account %s. %s", username, accountID, err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User password deleted successfully"})
}

func (h *Handler) RotateUserPassword(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	response, err := h.awsService.RotateUserPassword(accountID, username)
	if err != nil {
		// Log the full error for debugging
		fmt.Printf("[ERROR] RotateUserPassword failed for user %s in account %s: %v\n", username, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to rotate password for user %s in account %s. %s", username, accountID, err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListPublicIPs(c *gin.Context) {
	ips, err := h.awsService.ListPublicIPs()
	if err != nil {
		fmt.Printf("[ERROR] ListPublicIPs failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list public IP addresses. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, ips)
}

func (h *Handler) ClearCache(c *gin.Context) {
	h.awsService.ClearCache()
	c.JSON(http.StatusOK, gin.H{"message": "Cache cleared successfully"})
}

func (h *Handler) InvalidateAccountCache(c *gin.Context) {
	accountID := c.Param("accountId")
	h.awsService.InvalidateAccountCache(accountID)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Cache invalidated for account %s", accountID)})
}

func (h *Handler) InvalidateUserCache(c *gin.Context) {
	accountID := c.Param("accountId")
	username := c.Param("username")
	h.awsService.InvalidateUserCache(accountID, username)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Cache invalidated for user %s in account %s", username, accountID)})
}

func (h *Handler) InvalidatePublicIPsCache(c *gin.Context) {
	h.awsService.InvalidatePublicIPsCache()
	c.JSON(http.StatusOK, gin.H{"message": "Public IPs cache invalidated successfully"})
}

func (h *Handler) ListSecurityGroups(c *gin.Context) {
	sgs, err := h.awsService.ListSecurityGroups()
	if err != nil {
		fmt.Printf("[ERROR] ListSecurityGroups failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list security groups. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, sgs)
}

func (h *Handler) ListSecurityGroupsByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	sgs, err := h.awsService.ListSecurityGroupsByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListSecurityGroupsByAccount failed for account %s: %v\n", accountID, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, sgs)
}

func (h *Handler) InvalidateSecurityGroupsCache(c *gin.Context) {
	h.awsService.InvalidateSecurityGroupsCache()
	c.JSON(http.StatusOK, gin.H{"message": "Security groups cache invalidated successfully"})
}

func (h *Handler) InvalidateAccountSecurityGroupsCache(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	h.awsService.InvalidateAccountSecurityGroupsCache(accountID)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Security groups cache invalidated for account %s", accountID)})
}

func (h *Handler) GetSecurityGroup(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	groupID := c.Param("groupId")

	if accountID == "" || region == "" || groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and group ID are required",
		})
		return
	}

	sg, err := h.awsService.GetSecurityGroup(accountID, region, groupID)
	if err != nil {
		fmt.Printf("[ERROR] GetSecurityGroup failed for group %s in account %s, region %s: %v\n", groupID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, sg)
}

func (h *Handler) DeleteSecurityGroup(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	groupID := c.Param("groupId")

	if accountID == "" || region == "" || groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and group ID are required",
		})
		return
	}

	err := h.awsService.DeleteSecurityGroup(accountID, region, groupID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteSecurityGroup failed for group %s in account %s, region %s: %v\n", groupID, accountID, region, err)

		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot delete default") ||
			strings.Contains(err.Error(), "still in use") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Security group %s deleted successfully", groupID),
	})
}

// ============================================================================
// SNAPSHOT HANDLERS
// ============================================================================

func (h *Handler) ListSnapshots(c *gin.Context) {
	snapshots, err := h.awsService.ListSnapshots()
	if err != nil {
		fmt.Printf("[ERROR] ListSnapshots failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list snapshots. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

func (h *Handler) ListSnapshotsByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	snapshots, err := h.awsService.ListSnapshotsByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListSnapshotsByAccount failed for account %s: %v\n", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list snapshots for account %s.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

func (h *Handler) DeleteSnapshot(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	snapshotID := c.Param("snapshotId")

	if accountID == "" || region == "" || snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and snapshot ID are required",
		})
		return
	}

	err := h.awsService.DeleteSnapshot(accountID, region, snapshotID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteSnapshot failed for snapshot %s in account %s, region %s: %v\n", snapshotID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Snapshot %s deleted successfully", snapshotID),
	})
}

func (h *Handler) DeleteOldSnapshots(c *gin.Context) {
	accountID := c.Param("accountId")
	olderThanMonthsStr := c.DefaultQuery("older_than_months", "6")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	var olderThanMonths int
	if _, err := fmt.Sscanf(olderThanMonthsStr, "%d", &olderThanMonths); err != nil || olderThanMonths <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "older_than_months must be a positive integer",
		})
		return
	}

	deletedSnapshots, err := h.awsService.DeleteOldSnapshots(accountID, olderThanMonths)
	if err != nil {
		fmt.Printf("[ERROR] DeleteOldSnapshots failed for account %s: %v\n", accountID, err)
		
		// If some snapshots were deleted, return partial success
		if len(deletedSnapshots) > 0 {
			c.JSON(http.StatusPartialContent, gin.H{
				"message":          fmt.Sprintf("Deleted %d snapshots, but encountered errors", len(deletedSnapshots)),
				"deleted_snapshots": deletedSnapshots,
				"error":             err.Error(),
			})
			return
		}

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          fmt.Sprintf("Successfully deleted %d snapshot(s) older than %d months", len(deletedSnapshots), olderThanMonths),
		"deleted_snapshots": deletedSnapshots,
		"count":            len(deletedSnapshots),
	})
}

// ============================================================================
// EC2 INSTANCE HANDLERS
// ============================================================================

func (h *Handler) ListEC2Instances(c *gin.Context) {
	instances, err := h.awsService.ListEC2Instances()
	if err != nil {
		fmt.Printf("[ERROR] ListEC2Instances failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list EC2 instances. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, instances)
}

func (h *Handler) InvalidateEC2InstancesCache(c *gin.Context) {
	h.awsService.InvalidateEC2InstancesCache()
	c.JSON(http.StatusOK, gin.H{"message": "EC2 instances cache invalidated successfully"})
}

func (h *Handler) StopEC2Instance(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	instanceID := c.Param("instanceId")

	if accountID == "" || region == "" || instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and instance ID are required",
		})
		return
	}

	err := h.awsService.StopEC2Instance(accountID, region, instanceID)
	if err != nil {
		fmt.Printf("[ERROR] StopEC2Instance failed for instance %s in account %s, region %s: %v\n", instanceID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Instance %s stopped successfully", instanceID),
	})
}

func (h *Handler) TerminateEC2Instance(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	instanceID := c.Param("instanceId")

	if accountID == "" || region == "" || instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and instance ID are required",
		})
		return
	}

	err := h.awsService.TerminateEC2Instance(accountID, region, instanceID)
	if err != nil {
		fmt.Printf("[ERROR] TerminateEC2Instance failed for instance %s in account %s, region %s: %v\n", instanceID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Instance %s terminated successfully", instanceID),
	})
}

// ============================================================================
// EBS VOLUME HANDLERS
// ============================================================================

func (h *Handler) ListEBSVolumes(c *gin.Context) {
	volumes, err := h.awsService.ListEBSVolumes()
	if err != nil {
		fmt.Printf("[ERROR] ListEBSVolumes failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list EBS volumes. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, volumes)
}

func (h *Handler) ListEBSVolumesByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	volumes, err := h.awsService.ListEBSVolumesByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListEBSVolumesByAccount failed for account %s: %v\n", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list EBS volumes for account %s.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, volumes)
}

func (h *Handler) DetachEBSVolume(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	volumeID := c.Param("volumeId")

	if accountID == "" || region == "" || volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and volume ID are required",
		})
		return
	}

	err := h.awsService.DetachEBSVolume(accountID, region, volumeID)
	if err != nil {
		fmt.Printf("[ERROR] DetachEBSVolume failed for volume %s in account %s, region %s: %v\n", volumeID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "not attached") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Volume %s detached successfully", volumeID),
	})
}

func (h *Handler) DeleteEBSVolume(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	volumeID := c.Param("volumeId")

	if accountID == "" || region == "" || volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and volume ID are required",
		})
		return
	}

	err := h.awsService.DeleteEBSVolume(accountID, region, volumeID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteEBSVolume failed for volume %s in account %s, region %s: %v\n", volumeID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "still attached") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Volume %s deleted successfully", volumeID),
	})
}

func (h *Handler) InvalidateEBSVolumesCache(c *gin.Context) {
	h.awsService.InvalidateEBSVolumesCache()
	c.JSON(http.StatusOK, gin.H{"message": "EBS volumes cache invalidated successfully"})
}

func (h *Handler) ListS3Buckets(c *gin.Context) {
	buckets, err := h.awsService.ListS3Buckets()
	if err != nil {
		fmt.Printf("[ERROR] ListS3Buckets failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list S3 buckets. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, buckets)
}

func (h *Handler) ListS3BucketsByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	buckets, err := h.awsService.ListS3BucketsByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListS3BucketsByAccount failed for account %s: %v\n", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list S3 buckets for account %s.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, buckets)
}

func (h *Handler) DeleteS3Bucket(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	bucketName := c.Param("bucketName")

	if accountID == "" || region == "" || bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and bucket name are required",
		})
		return
	}

	err := h.awsService.DeleteS3Bucket(accountID, region, bucketName)
	if err != nil {
		fmt.Printf("[ERROR] DeleteS3Bucket failed for bucket %s in account %s, region %s: %v\n", bucketName, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "must be empty") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Bucket %s deleted successfully", bucketName),
	})
}

func (h *Handler) InvalidateS3BucketsCache(c *gin.Context) {
	h.awsService.InvalidateS3BucketsCache()
	c.JSON(http.StatusOK, gin.H{"message": "S3 buckets cache invalidated successfully"})
}

// ============================================================================
// ROUTE53 DOMAIN HANDLERS
// ============================================================================

func (h *Handler) ListRoute53Domains(c *gin.Context) {
	domains, err := h.awsService.ListRoute53Domains()
	if err != nil {
		fmt.Printf("[ERROR] ListRoute53Domains failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list Route53 domains. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, domains)
}

func (h *Handler) InvalidateRoute53DomainsCache(c *gin.Context) {
	h.awsService.InvalidateRoute53DomainsCache()
	c.JSON(http.StatusOK, gin.H{"message": "Route53 domains cache invalidated successfully"})
}

func (h *Handler) ListRoute53Records(c *gin.Context) {
	accountID := c.Param("accountId")
	hostedZoneID := c.Param("hostedZoneId")

	if accountID == "" || hostedZoneID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID and hosted zone ID are required",
		})
		return
	}

	records, err := h.awsService.ListRoute53Records(accountID, hostedZoneID)
	if err != nil {
		fmt.Printf("[ERROR] ListRoute53Records failed for zone %s in account %s: %v\n", hostedZoneID, accountID, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, records)
}

// ============================================================================
// IAM ROLE HANDLERS
// ============================================================================

func (h *Handler) ListRoles(c *gin.Context) {
	accountID := c.Param("accountId")
	roles, err := h.awsService.ListRoles(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListRoles failed for account %s: %v\n", accountID, err)
		if containsAccessDenied(err.Error()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied to account",
				"details": fmt.Sprintf("Cannot access account %s. The role may not exist or trust relationship is not configured.", accountID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list roles for account %s. Check AWS credentials and permissions.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, roles)
}

func (h *Handler) ListAllRoles(c *gin.Context) {
	roles, err := h.awsService.ListAllRoles()
	if err != nil {
		fmt.Printf("[ERROR] ListAllRoles failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list all roles. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, roles)
}

func (h *Handler) GetRole(c *gin.Context) {
	accountID := c.Param("accountId")
	roleName := c.Param("roleName")
	role, err := h.awsService.GetRole(accountID, roleName)
	if err != nil {
		fmt.Printf("[ERROR] GetRole failed for role %s in account %s: %v\n", roleName, accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to get role %s from account %s. Check AWS credentials and permissions.", roleName, accountID),
		})
		return
	}
	c.JSON(http.StatusOK, role)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	accountID := c.Param("accountId")
	roleName := c.Param("roleName")
	err := h.awsService.DeleteRole(accountID, roleName)
	if err != nil {
		fmt.Printf("[ERROR] DeleteRole failed for role %s in account %s: %v\n", roleName, accountID, err)
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Role %s deleted successfully", roleName),
	})
}

func (h *Handler) InvalidateRolesCache(c *gin.Context) {
	h.awsService.InvalidateRolesCache()
	c.JSON(http.StatusOK, gin.H{"message": "Roles cache invalidated successfully"})
}

func (h *Handler) InvalidateAccountRolesCache(c *gin.Context) {
	accountID := c.Param("accountId")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}
	h.awsService.InvalidateAccountRolesCache(accountID)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Roles cache invalidated for account %s", accountID)})
}

// ============================================================================
// LOAD BALANCER HANDLERS (Account-Specific)
// ============================================================================

func (h *Handler) ListAllLoadBalancers(c *gin.Context) {
	loadBalancers, err := h.awsService.ListAllLoadBalancers()
	if err != nil {
		fmt.Printf("[ERROR] ListAllLoadBalancers failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list load balancers. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, loadBalancers)
}

func (h *Handler) ListLoadBalancersByAccount(c *gin.Context) {
	accountID := c.Param("accountId")
	loadBalancers, err := h.awsService.ListLoadBalancersByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListLoadBalancersByAccount failed for account %s: %v\n", accountID, err)
		if containsAccessDenied(err.Error()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied to account",
				"details": fmt.Sprintf("Cannot access account %s. The role may not exist or trust relationship is not configured.", accountID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list load balancers for account %s. Check AWS credentials and permissions.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, loadBalancers)
}

func (h *Handler) DeleteLoadBalancer(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	loadBalancerArnOrName := c.Query("id") // ARN for ALB/NLB, name for Classic ELB
	lbType := c.Query("type")              // "application", "network", or "classic"

	if accountID == "" || region == "" || loadBalancerArnOrName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and load balancer id (ARN/name) are required",
		})
		return
	}

	if lbType == "" {
		// Try to infer from ARN format
		if strings.HasPrefix(loadBalancerArnOrName, "arn:aws:elasticloadbalancing") {
			if strings.Contains(loadBalancerArnOrName, ":loadbalancer/app/") {
				lbType = "application"
			} else if strings.Contains(loadBalancerArnOrName, ":loadbalancer/net/") {
				lbType = "network"
			} else if strings.Contains(loadBalancerArnOrName, ":loadbalancer/") {
				lbType = "classic"
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Cannot determine load balancer type. Please specify 'type' query parameter (application, network, or classic)",
				})
				return
			}
		} else {
			// Assume Classic ELB if it's just a name
			lbType = "classic"
		}
	}

	err := h.awsService.DeleteLoadBalancer(accountID, region, loadBalancerArnOrName, lbType)
	if err != nil {
		fmt.Printf("[ERROR] DeleteLoadBalancer failed for %s in account %s, region %s: %v\n", loadBalancerArnOrName, accountID, region, err)
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Load balancer %s deleted successfully", loadBalancerArnOrName),
	})
}

func (h *Handler) InvalidateLoadBalancersCache(c *gin.Context) {
	accountID := c.Param("accountId")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}
	h.awsService.InvalidateLoadBalancersCache(accountID)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Load balancers cache invalidated for account %s", accountID)})
}

func (h *Handler) InvalidateAllLoadBalancersCache(c *gin.Context) {
	h.awsService.InvalidateAllLoadBalancersCache()
	c.JSON(http.StatusOK, gin.H{"message": "All load balancers cache invalidated"})
}

// ============================================================================
// VPC HANDLERS
// ============================================================================

func (h *Handler) ListVPCs(c *gin.Context) {
	vpcs, err := h.awsService.ListVPCs()
	if err != nil {
		fmt.Printf("[ERROR] ListVPCs failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list VPCs. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, vpcs)
}

func (h *Handler) ListVPCsByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	vpcs, err := h.awsService.ListVPCsByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListVPCsByAccount failed for account %s: %v\n", accountID, err)
		if containsAccessDenied(err.Error()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied to account",
				"details": fmt.Sprintf("Cannot access account %s.", accountID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list VPCs for account %s.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, vpcs)
}

func (h *Handler) DeleteVPC(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	vpcID := c.Param("vpcId")

	if accountID == "" || region == "" || vpcID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and VPC ID are required",
		})
		return
	}

	err := h.awsService.DeleteVPC(accountID, region, vpcID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteVPC failed for VPC %s in account %s, region %s: %v\n", vpcID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "has dependencies") || strings.Contains(err.Error(), "DependencyViolation") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("VPC %s deleted successfully", vpcID),
	})
}

func (h *Handler) InvalidateVPCsCache(c *gin.Context) {
	h.awsService.InvalidateVPCsCache()
	c.JSON(http.StatusOK, gin.H{"message": "VPCs cache invalidated successfully"})
}

// ============================================================================
// NAT GATEWAY HANDLERS
// ============================================================================

func (h *Handler) ListNATGateways(c *gin.Context) {
	nats, err := h.awsService.ListNATGateways()
	if err != nil {
		fmt.Printf("[ERROR] ListNATGateways failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": "Failed to list NAT Gateways. Check AWS credentials and permissions.",
		})
		return
	}
	c.JSON(http.StatusOK, nats)
}

func (h *Handler) ListNATGatewaysByAccount(c *gin.Context) {
	accountID := c.Param("accountId")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID is required",
		})
		return
	}

	nats, err := h.awsService.ListNATGatewaysByAccount(accountID)
	if err != nil {
		fmt.Printf("[ERROR] ListNATGatewaysByAccount failed for account %s: %v\n", accountID, err)
		if containsAccessDenied(err.Error()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied to account",
				"details": fmt.Sprintf("Cannot access account %s.", accountID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"details": fmt.Sprintf("Failed to list NAT Gateways for account %s.", accountID),
		})
		return
	}
	c.JSON(http.StatusOK, nats)
}

func (h *Handler) DeleteNATGateway(c *gin.Context) {
	accountID := c.Param("accountId")
	region := c.Param("region")
	natGatewayID := c.Param("natGatewayId")

	if accountID == "" || region == "" || natGatewayID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account ID, region, and NAT Gateway ID are required",
		})
		return
	}

	err := h.awsService.DeleteNATGateway(accountID, region, natGatewayID)
	if err != nil {
		fmt.Printf("[ERROR] DeleteNATGateway failed for NAT Gateway %s in account %s, region %s: %v\n", natGatewayID, accountID, region, err)

		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "cannot access account") {
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("NAT Gateway %s deleted successfully", natGatewayID),
	})
}

func (h *Handler) InvalidateNATGatewaysCache(c *gin.Context) {
	h.awsService.InvalidateNATGatewaysCache()
	c.JSON(http.StatusOK, gin.H{"message": "NAT Gateways cache invalidated successfully"})
}
