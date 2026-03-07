package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetServerFlags resets the package-level server flag variables to their
// cobra default values so tests don't leak state.
func resetServerFlags() {
	enableHub = false
	enableRuntimeBroker = false
	enableWeb = false
	enableDevAuth = false
	enableDebug = false
	serverAutoProvide = false
	serverStartForeground = false
	productionMode = false
	hubHost = "0.0.0.0"
	hubPort = 9810
	runtimeBrokerPort = 9800
	webPort = 8080
	storageBucket = ""
	storageDir = ""
	serverConfigPath = ""
	dbURL = ""
}

func TestWorkstationModeDefaults(t *testing.T) {
	// Reset flags after test to avoid leaking into other tests
	t.Cleanup(resetServerFlags)

	// Parse with no flags — simulates bare "scion server start"
	resetServerFlags()
	require.NoError(t, serverStartCmd.ParseFlags([]string{}))

	// Simulate the workstation defaults logic from runServerStartOrDaemon
	if !productionMode {
		if !serverStartCmd.Flags().Changed("enable-hub") {
			enableHub = true
		}
		if !serverStartCmd.Flags().Changed("enable-runtime-broker") {
			enableRuntimeBroker = true
		}
		if !serverStartCmd.Flags().Changed("enable-web") {
			enableWeb = true
		}
		if !serverStartCmd.Flags().Changed("dev-auth") {
			enableDevAuth = true
		}
		if !serverStartCmd.Flags().Changed("auto-provide") {
			serverAutoProvide = true
		}
		if !serverStartCmd.Flags().Changed("host") {
			hubHost = "127.0.0.1"
		}
	}

	assert.True(t, enableHub, "hub should be enabled in workstation mode")
	assert.True(t, enableRuntimeBroker, "runtime broker should be enabled in workstation mode")
	assert.True(t, enableWeb, "web should be enabled in workstation mode")
	assert.True(t, enableDevAuth, "dev-auth should be enabled in workstation mode")
	assert.True(t, serverAutoProvide, "auto-provide should be enabled in workstation mode")
	assert.Equal(t, "127.0.0.1", hubHost, "host should default to loopback in workstation mode")
}

func TestProductionModeNoDefaults(t *testing.T) {
	t.Cleanup(resetServerFlags)

	resetServerFlags()
	require.NoError(t, serverStartCmd.ParseFlags([]string{"--production"}))

	// In production mode, no defaults are applied
	assert.True(t, productionMode, "production flag should be set")
	assert.False(t, enableHub, "hub should not be enabled by default in production mode")
	assert.False(t, enableRuntimeBroker, "runtime broker should not be enabled by default in production mode")
	assert.False(t, enableWeb, "web should not be enabled by default in production mode")
	assert.False(t, enableDevAuth, "dev-auth should not be enabled by default in production mode")
	assert.False(t, serverAutoProvide, "auto-provide should not be enabled by default in production mode")
	assert.Equal(t, "0.0.0.0", hubHost, "host should default to 0.0.0.0 in production mode")
}

func TestWorkstationModeExplicitOverrides(t *testing.T) {
	t.Cleanup(resetServerFlags)

	// Explicitly disable web and bind to all interfaces in workstation mode
	resetServerFlags()
	require.NoError(t, serverStartCmd.ParseFlags([]string{"--enable-web=false", "--host=0.0.0.0"}))

	if !productionMode {
		if !serverStartCmd.Flags().Changed("enable-hub") {
			enableHub = true
		}
		if !serverStartCmd.Flags().Changed("enable-runtime-broker") {
			enableRuntimeBroker = true
		}
		if !serverStartCmd.Flags().Changed("enable-web") {
			enableWeb = true
		}
		if !serverStartCmd.Flags().Changed("dev-auth") {
			enableDevAuth = true
		}
		if !serverStartCmd.Flags().Changed("auto-provide") {
			serverAutoProvide = true
		}
		if !serverStartCmd.Flags().Changed("host") {
			hubHost = "127.0.0.1"
		}
	}

	assert.True(t, enableHub, "hub should be enabled (workstation default)")
	assert.True(t, enableRuntimeBroker, "runtime broker should be enabled (workstation default)")
	assert.False(t, enableWeb, "web should be disabled (explicit override)")
	assert.True(t, enableDevAuth, "dev-auth should be enabled (workstation default)")
	assert.Equal(t, "0.0.0.0", hubHost, "host should be 0.0.0.0 (explicit override)")
}

func TestProductionModeWithExplicitFlags(t *testing.T) {
	t.Cleanup(resetServerFlags)

	resetServerFlags()
	require.NoError(t, serverStartCmd.ParseFlags([]string{
		"--production",
		"--enable-hub",
		"--enable-web",
		"--dev-auth",
	}))

	assert.True(t, productionMode, "production flag should be set")
	assert.True(t, enableHub, "hub should be enabled (explicit)")
	assert.False(t, enableRuntimeBroker, "runtime broker should not be enabled (not explicitly set)")
	assert.True(t, enableWeb, "web should be enabled (explicit)")
	assert.True(t, enableDevAuth, "dev-auth should be enabled (explicit)")
	assert.Equal(t, "0.0.0.0", hubHost, "host should default to 0.0.0.0 in production mode")
}

func TestBrokerDelegationUsesProductionMode(t *testing.T) {
	t.Cleanup(resetServerFlags)

	// Simulate what broker start does: parse --production --enable-runtime-broker
	resetServerFlags()
	require.NoError(t, serverStartCmd.ParseFlags([]string{
		"--production",
		"--enable-runtime-broker",
	}))

	assert.True(t, productionMode, "production flag should be set")
	assert.True(t, enableRuntimeBroker, "runtime broker should be enabled")
	assert.False(t, enableHub, "hub should NOT be enabled (broker-only)")
	assert.False(t, enableWeb, "web should NOT be enabled (broker-only)")
}
