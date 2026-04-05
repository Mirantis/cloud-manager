package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Test default values
	require.NoError(t, os.Unsetenv("PORT"))
	require.NoError(t, os.Unsetenv("AWS_REGION"))
	require.NoError(t, os.Unsetenv("IAM_ORG_ROLE_NAME"))

	cfg := LoadConfig()
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "us-east-1", cfg.AWSRegion)
	assert.Equal(t, "IAMManagerCrossAccountRole", cfg.RoleName)
	assert.Equal(t, "./iam-manager.db", cfg.DBPath)

	require.NoError(t, os.Setenv("DB_PATH", "/data/app.db"))
	cfg = LoadConfig()
	assert.Equal(t, "/data/app.db", cfg.DBPath)
	require.NoError(t, os.Unsetenv("DB_PATH"))

	// Test custom values
	require.NoError(t, os.Setenv("PORT", "9000"))
	require.NoError(t, os.Setenv("AWS_REGION", "us-west-2"))
	require.NoError(t, os.Setenv("IAM_ORG_ROLE_NAME", "CustomRole"))

	cfg = LoadConfig()
	assert.Equal(t, "9000", cfg.Port)
	assert.Equal(t, "us-west-2", cfg.AWSRegion)
	assert.Equal(t, "CustomRole", cfg.RoleName)

	require.NoError(t, os.Unsetenv("PORT"))
	require.NoError(t, os.Unsetenv("AWS_REGION"))
	require.NoError(t, os.Unsetenv("IAM_ORG_ROLE_NAME"))
}
