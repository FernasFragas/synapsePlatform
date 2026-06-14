package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigPath_DefaultsToLocalConfig(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"synapsePlatform"}

	t.Setenv("SYNAPSE_CONFIG_PATH", "")

	require.Equal(t, "config.yaml", configPath())
}

func TestConfigPath_UsesEnvironmentOverride(t *testing.T) {
	t.Setenv("SYNAPSE_CONFIG_PATH", "config-docker.yaml")

	require.Equal(t, "config-docker.yaml", configPath())
}

func TestConfigPath_UsesFirstArgument(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"synapsePlatform", "custom.yaml"}

	t.Setenv("SYNAPSE_CONFIG_PATH", "")

	require.Equal(t, "custom.yaml", configPath())
}
