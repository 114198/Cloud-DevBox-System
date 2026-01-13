// Package services provides business logic for the container service.
package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

func newTestSSHService() *SSHService {
	logger, _ := zap.NewDevelopment()
	return NewSSHService(logger, DefaultSSHServiceConfig())
}

// TestGenerateEd25519Key tests Ed25519 key generation
func TestGenerateEd25519Key(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	req := &models.CreateSSHKeyRequest{
		Name:    "test-key",
		Type:    models.SSHKeyTypeEd25519,
		Comment: "test@devbox",
	}

	keyPair, err := svc.GenerateKeyPair(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	// Verify key properties
	if keyPair.ID == "" {
		t.Error("Key ID should not be empty")
	}
	if keyPair.Type != models.SSHKeyTypeEd25519 {
		t.Errorf("Expected key type ed25519, got %s", keyPair.Type)
	}
	if !strings.HasPrefix(keyPair.PublicKey, "ssh-ed25519") {
		t.Error("Public key should start with ssh-ed25519")
	}
	if !strings.Contains(keyPair.PublicKey, "test@devbox") {
		t.Error("Public key should contain comment")
	}
	if !strings.Contains(keyPair.PrivateKey, "PRIVATE KEY") {
		t.Error("Private key should be in PEM format")
	}
	if !strings.HasPrefix(keyPair.Fingerprint, "SHA256:") {
		t.Error("Fingerprint should start with SHA256:")
	}
}

// TestGenerateRSAKey tests RSA key generation
func TestGenerateRSAKey(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	req := &models.CreateSSHKeyRequest{
		Name:    "test-rsa-key",
		Type:    models.SSHKeyTypeRSA,
		Comment: "rsa@devbox",
	}

	keyPair, err := svc.GenerateKeyPair(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	if keyPair.Type != models.SSHKeyTypeRSA {
		t.Errorf("Expected key type rsa, got %s", keyPair.Type)
	}
	if !strings.HasPrefix(keyPair.PublicKey, "ssh-rsa") {
		t.Error("Public key should start with ssh-rsa")
	}
	if !strings.Contains(keyPair.PrivateKey, "RSA PRIVATE KEY") {
		t.Error("Private key should be RSA PEM format")
	}
}

// TestKeyExpiration tests key expiration setting
func TestKeyExpiration(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	expiresIn := 30 // 30 days
	req := &models.CreateSSHKeyRequest{
		Name:      "expiring-key",
		Type:      models.SSHKeyTypeEd25519,
		ExpiresIn: &expiresIn,
	}

	keyPair, err := svc.GenerateKeyPair(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if keyPair.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}

	expectedExpiry := time.Now().AddDate(0, 0, 30)
	diff := keyPair.ExpiresAt.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("Expiry time is not within expected range")
	}
}

// TestDefaultKey tests setting and getting default key
func TestDefaultKey(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	userID := "user-default-test"

	// Create first key as default
	req1 := &models.CreateSSHKeyRequest{
		Name:       "key-1",
		Type:       models.SSHKeyTypeEd25519,
		SetDefault: true,
	}
	key1, _ := svc.GenerateKeyPair(ctx, userID, req1)

	// Verify it's the default
	defaultKey, err := svc.GetDefaultKey(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get default key: %v", err)
	}
	if defaultKey.ID != key1.ID {
		t.Error("First key should be default")
	}

	// Create second key as default
	req2 := &models.CreateSSHKeyRequest{
		Name:       "key-2",
		Type:       models.SSHKeyTypeEd25519,
		SetDefault: true,
	}
	key2, _ := svc.GenerateKeyPair(ctx, userID, req2)

	// Verify second key is now default
	defaultKey, _ = svc.GetDefaultKey(ctx, userID)
	if defaultKey.ID != key2.ID {
		t.Error("Second key should now be default")
	}

	// Verify first key is no longer default
	key1Updated, _ := svc.GetKey(ctx, userID, key1.ID)
	if key1Updated.IsDefault {
		t.Error("First key should no longer be default")
	}
}

// TestKeyRotation tests key rotation
func TestKeyRotation(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	userID := "user-rotate-test"

	// Create initial key
	req := &models.CreateSSHKeyRequest{
		Name:    "rotate-key",
		Type:    models.SSHKeyTypeEd25519,
		Comment: "original",
	}
	oldKey, _ := svc.GenerateKeyPair(ctx, userID, req)
	oldID := oldKey.ID
	oldFingerprint := oldKey.Fingerprint

	// Rotate the key
	newKey, err := svc.RotateKey(ctx, userID, oldID)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// Verify new key has different ID and fingerprint
	if newKey.ID == oldID {
		t.Error("New key should have different ID")
	}
	if newKey.Fingerprint == oldFingerprint {
		t.Error("New key should have different fingerprint")
	}
	if newKey.RotatedAt == nil {
		t.Error("RotatedAt should be set")
	}

	// Verify old key is deleted
	_, err = svc.GetKey(ctx, userID, oldID)
	if err == nil {
		t.Error("Old key should be deleted")
	}
}

// TestListKeys tests listing keys
func TestListKeys(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	userID := "user-list-test"

	// Create multiple keys
	for i := 0; i < 5; i++ {
		req := &models.CreateSSHKeyRequest{
			Name: "list-key-" + string(rune('a'+i)),
			Type: models.SSHKeyTypeEd25519,
		}
		svc.GenerateKeyPair(ctx, userID, req)
	}

	// List all keys
	resp, err := svc.ListKeys(ctx, &models.ListSSHKeysRequest{
		UserID:   userID,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("Expected 5 keys, got %d", resp.Total)
	}

	// Verify private keys are not included
	for _, key := range resp.Keys {
		if key.PrivateKey != "" {
			t.Error("Private key should not be included in list response")
		}
	}
}

// TestDeleteKey tests key deletion
func TestDeleteKey(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	userID := "user-delete-test"

	req := &models.CreateSSHKeyRequest{
		Name: "delete-key",
		Type: models.SSHKeyTypeEd25519,
	}
	key, _ := svc.GenerateKeyPair(ctx, userID, req)

	// Delete the key
	err := svc.DeleteKey(ctx, userID, key.ID)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Verify key is deleted
	_, err = svc.GetKey(ctx, userID, key.ID)
	if err == nil {
		t.Error("Key should be deleted")
	}
}

// TestAccessControl tests that users can only access their own keys
func TestAccessControl(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	// Create key for user1
	req := &models.CreateSSHKeyRequest{
		Name: "user1-key",
		Type: models.SSHKeyTypeEd25519,
	}
	key, _ := svc.GenerateKeyPair(ctx, "user1", req)

	// Try to access with user2
	_, err := svc.GetKey(ctx, "user2", key.ID)
	if err == nil {
		t.Error("User2 should not be able to access user1's key")
	}

	// Try to delete with user2
	err = svc.DeleteKey(ctx, "user2", key.ID)
	if err == nil {
		t.Error("User2 should not be able to delete user1's key")
	}
}


// TestSSHConfigGeneration tests SSH config generation
func TestSSHConfigGeneration(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	env := &models.Environment{
		ID:     "test-env-12345678",
		Name:   "Test Environment",
		UserID: "user-1",
		Connection: &models.ConnectionInfo{
			ExternalIP: "192.168.1.100",
			SSHPort:    30022,
			SSHHost:    "devbox.example.com",
		},
	}

	// Test VSCode config
	config, err := svc.GenerateSSHConfig(ctx, env, models.IDETypeVSCode, "~/.ssh/devbox_key")
	if err != nil {
		t.Fatalf("Failed to generate VSCode config: %v", err)
	}

	if config.IDEType != models.IDETypeVSCode {
		t.Errorf("Expected IDE type vscode, got %s", config.IDEType)
	}
	if !strings.Contains(config.ConfigString, "Host devbox-test-env") {
		t.Error("Config should contain host alias")
	}
	if !strings.Contains(config.ConfigString, "Port 30022") {
		t.Error("Config should contain port")
	}
	if !strings.Contains(config.ConfigString, "ForwardAgent yes") {
		t.Error("VSCode config should have ForwardAgent")
	}
}

// TestJetBrainsConfigGeneration tests JetBrains Gateway config generation
func TestJetBrainsConfigGeneration(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	env := &models.Environment{
		ID:     "jetbrains-env-123",
		Name:   "JetBrains Test",
		UserID: "user-1",
		Connection: &models.ConnectionInfo{
			ExternalIP: "10.0.0.50",
			SSHPort:    30123,
		},
	}

	config, err := svc.GenerateSSHConfig(ctx, env, models.IDETypeJetBrains, "~/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("Failed to generate JetBrains config: %v", err)
	}

	if config.IDEType != models.IDETypeJetBrains {
		t.Errorf("Expected IDE type jetbrains, got %s", config.IDEType)
	}
	if !strings.Contains(config.ConfigString, "ControlMaster auto") {
		t.Error("JetBrains config should have ControlMaster")
	}
	if !strings.Contains(config.ConfigString, "ControlPersist") {
		t.Error("JetBrains config should have ControlPersist")
	}
}

// TestConnectionLimit tests concurrent connection limits
func TestConnectionLimit(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	envID := "env-conn-limit"

	// Register max connections
	maxConns := svc.GetMaxConnectionsPerEnv()
	for i := 0; i < maxConns; i++ {
		conn := &models.SSHConnection{
			EnvironmentID: envID,
			UserID:        "user-1",
			ClientIP:      "192.168.1.1",
			ClientPort:    50000 + i,
			ServerPort:    22,
		}
		err := svc.RegisterConnection(ctx, conn)
		if err != nil {
			t.Fatalf("Failed to register connection %d: %v", i, err)
		}
	}

	// Try to register one more - should fail
	conn := &models.SSHConnection{
		EnvironmentID: envID,
		UserID:        "user-1",
		ClientIP:      "192.168.1.1",
		ClientPort:    60000,
		ServerPort:    22,
	}
	err := svc.RegisterConnection(ctx, conn)
	if err == nil {
		t.Error("Should not allow more than max connections")
	}
	if !IsConnectionLimitError(err) {
		t.Errorf("Expected ConnectionLimitError, got %T", err)
	}
}

// TestCanConnect tests connection availability check
func TestCanConnect(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	envID := "env-can-connect"

	// Initially should be able to connect
	canConnect, current, max := svc.CanConnect(ctx, envID)
	if !canConnect {
		t.Error("Should be able to connect initially")
	}
	if current != 0 {
		t.Errorf("Expected 0 current connections, got %d", current)
	}
	if max != svc.GetMaxConnectionsPerEnv() {
		t.Errorf("Expected max %d, got %d", svc.GetMaxConnectionsPerEnv(), max)
	}

	// Register a connection
	conn := &models.SSHConnection{
		EnvironmentID: envID,
		UserID:        "user-1",
		ClientIP:      "192.168.1.1",
		ClientPort:    50000,
		ServerPort:    22,
	}
	svc.RegisterConnection(ctx, conn)

	// Check again
	canConnect, current, _ = svc.CanConnect(ctx, envID)
	if !canConnect {
		t.Error("Should still be able to connect")
	}
	if current != 1 {
		t.Errorf("Expected 1 current connection, got %d", current)
	}
}

// TestUnregisterConnection tests connection unregistration
func TestUnregisterConnection(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	envID := "env-unregister"

	// Register a connection
	conn := &models.SSHConnection{
		EnvironmentID: envID,
		UserID:        "user-1",
		ClientIP:      "192.168.1.1",
		ClientPort:    50000,
		ServerPort:    22,
	}
	svc.RegisterConnection(ctx, conn)

	// Verify connection exists
	_, current, _ := svc.CanConnect(ctx, envID)
	if current != 1 {
		t.Errorf("Expected 1 connection, got %d", current)
	}

	// Unregister
	err := svc.UnregisterConnection(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Failed to unregister connection: %v", err)
	}

	// Verify connection is removed
	_, current, _ = svc.CanConnect(ctx, envID)
	if current != 0 {
		t.Errorf("Expected 0 connections after unregister, got %d", current)
	}
}

// TestConnectionStats tests connection statistics
func TestConnectionStats(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	envID := "env-stats"

	// Register connections with traffic
	for i := 0; i < 3; i++ {
		conn := &models.SSHConnection{
			EnvironmentID: envID,
			UserID:        "user-1",
			ClientIP:      "192.168.1.1",
			ClientPort:    50000 + i,
			ServerPort:    22,
			BytesSent:     int64(1000 * (i + 1)),
			BytesReceived: int64(2000 * (i + 1)),
		}
		svc.RegisterConnection(ctx, conn)
	}

	stats, err := svc.GetConnectionStats(ctx, envID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.ActiveConnections != 3 {
		t.Errorf("Expected 3 active connections, got %d", stats.ActiveConnections)
	}
	if stats.TotalBytesSent != 6000 { // 1000 + 2000 + 3000
		t.Errorf("Expected 6000 bytes sent, got %d", stats.TotalBytesSent)
	}
	if stats.TotalBytesReceived != 12000 { // 2000 + 4000 + 6000
		t.Errorf("Expected 12000 bytes received, got %d", stats.TotalBytesReceived)
	}
}

// TestProxyConfig tests proxy configuration generation
func TestProxyConfig(t *testing.T) {
	svc := newTestSSHService()

	env := &models.Environment{
		ID:     "proxy-test-env",
		Name:   "Proxy Test",
		UserID: "user-1",
		Connection: &models.ConnectionInfo{
			PodIP:      "10.244.0.5",
			ExternalIP: "192.168.1.100",
			SSHPort:    30022,
		},
	}

	proxyConfig := svc.GetProxyConfig(env)
	if proxyConfig == nil {
		t.Fatal("Proxy config should not be nil")
	}

	if proxyConfig.GatewayHost != svc.config.GatewayHost {
		t.Errorf("Expected gateway host %s, got %s", svc.config.GatewayHost, proxyConfig.GatewayHost)
	}
	if proxyConfig.TargetHost != "10.244.0.5" {
		t.Errorf("Expected target host 10.244.0.5, got %s", proxyConfig.TargetHost)
	}
	if proxyConfig.TargetPort != 30022 {
		t.Errorf("Expected target port 30022, got %d", proxyConfig.TargetPort)
	}
}

// TestCleanupStaleConnections tests stale connection cleanup
func TestCleanupStaleConnections(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()
	envID := "env-cleanup"

	// Register a connection
	conn := &models.SSHConnection{
		EnvironmentID: envID,
		UserID:        "user-1",
		ClientIP:      "192.168.1.1",
		ClientPort:    50000,
		ServerPort:    22,
	}
	svc.RegisterConnection(ctx, conn)

	// Manually set last activity to past
	svc.mu.Lock()
	svc.connections[conn.ID].LastActivityAt = time.Now().Add(-2 * time.Hour)
	svc.mu.Unlock()

	// Cleanup with 1 hour timeout
	cleaned, err := svc.CleanupStaleConnections(ctx, time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("Expected 1 cleaned connection, got %d", cleaned)
	}

	// Verify connection is removed
	_, current, _ := svc.CanConnect(ctx, envID)
	if current != 0 {
		t.Errorf("Expected 0 connections after cleanup, got %d", current)
	}
}

// **Property 5: SSH 配置生成 < 5秒**
// **Validates: Requirements 2.2**
// TestSSHConfigGenerationPerformance tests that SSH config generation completes within 5 seconds
func TestSSHConfigGenerationPerformance(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	// Create test environments with various configurations
	testCases := []struct {
		name    string
		ideType models.IDEType
	}{
		{"VSCode", models.IDETypeVSCode},
		{"JetBrains", models.IDETypeJetBrains},
		{"Generic", models.IDETypeGeneric},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := &models.Environment{
				ID:     "perf-test-" + string(tc.ideType),
				Name:   "Performance Test " + tc.name,
				UserID: "user-perf",
				Connection: &models.ConnectionInfo{
					ExternalIP: "192.168.1.100",
					SSHPort:    30022,
					SSHHost:    "devbox.example.com",
					PodIP:      "10.244.0.5",
				},
			}

			// Run multiple iterations to ensure consistent performance
			iterations := 100
			maxDuration := 5 * time.Second

			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, err := svc.GenerateSSHConfig(ctx, env, tc.ideType, "~/.ssh/devbox_key")
				duration := time.Since(start)

				if err != nil {
					t.Fatalf("Failed to generate config: %v", err)
				}

				if duration > maxDuration {
					t.Errorf("SSH config generation took %v, expected < %v", duration, maxDuration)
				}
			}
		})
	}
}

// TestSSHCommandGeneration tests SSH command generation
func TestSSHCommandGeneration(t *testing.T) {
	svc := newTestSSHService()

	env := &models.Environment{
		ID:     "cmd-test-env",
		Name:   "Command Test",
		UserID: "user-1",
		Connection: &models.ConnectionInfo{
			ExternalIP: "192.168.1.100",
			SSHPort:    30022,
		},
	}

	cmd := svc.GenerateSSHCommand(env, "~/.ssh/id_ed25519")

	if !strings.Contains(cmd, "-p 30022") {
		t.Error("Command should contain port")
	}
	if !strings.Contains(cmd, "-i ~/.ssh/id_ed25519") {
		t.Error("Command should contain identity file")
	}
	if !strings.Contains(cmd, "devbox@192.168.1.100") {
		t.Error("Command should contain user@host")
	}
	if !strings.Contains(cmd, "StrictHostKeyChecking=no") {
		t.Error("Command should disable strict host key checking")
	}
}

// TestNoConnectionInfo tests handling of environments without connection info
func TestNoConnectionInfo(t *testing.T) {
	svc := newTestSSHService()
	ctx := context.Background()

	env := &models.Environment{
		ID:         "no-conn-env",
		Name:       "No Connection",
		UserID:     "user-1",
		Connection: nil,
	}

	_, err := svc.GenerateSSHConfig(ctx, env, models.IDETypeVSCode, "~/.ssh/key")
	if err == nil {
		t.Error("Should return error for environment without connection info")
	}

	proxyConfig := svc.GetProxyConfig(env)
	if proxyConfig != nil {
		t.Error("Proxy config should be nil for environment without connection info")
	}

	cmd := svc.GenerateSSHCommand(env, "~/.ssh/key")
	if cmd != "" {
		t.Error("SSH command should be empty for environment without connection info")
	}
}
