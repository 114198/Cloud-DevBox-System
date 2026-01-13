// Package services provides business logic for the container service.
package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// SSHServiceConfig holds configuration for the SSH service
type SSHServiceConfig struct {
	GatewayHost           string
	GatewayPort           int
	GatewayUser           string
	MaxConnectionsPerEnv  int
	KeyRotationDays       int
	DefaultKeyType        models.SSHKeyType
	ConnectionTimeout     time.Duration
	KeepAliveInterval     int
	ServerAliveCountMax   int
}

// DefaultSSHServiceConfig returns default configuration
func DefaultSSHServiceConfig() *SSHServiceConfig {
	return &SSHServiceConfig{
		GatewayHost:          "gateway.devbox.com",
		GatewayPort:          22,
		GatewayUser:          "devbox",
		MaxConnectionsPerEnv: 5,
		KeyRotationDays:      90,
		DefaultKeyType:       models.SSHKeyTypeEd25519,
		ConnectionTimeout:    30 * time.Second,
		KeepAliveInterval:    60,
		ServerAliveCountMax:  3,
	}
}

// SSHService handles SSH key management and connection services
type SSHService struct {
	config *SSHServiceConfig
	logger *zap.Logger

	// In-memory storage (in production, use database)
	keys        map[string]*models.SSHKeyPair
	connections map[string]*models.SSHConnection
	mu          sync.RWMutex

	// Connection tracking per environment
	envConnections map[string][]string // environmentID -> []connectionID
	envMu          sync.RWMutex
}

// NewSSHService creates a new SSH service
func NewSSHService(logger *zap.Logger, config *SSHServiceConfig) *SSHService {
	if config == nil {
		config = DefaultSSHServiceConfig()
	}

	return &SSHService{
		config:         config,
		logger:         logger.Named("ssh-service"),
		keys:           make(map[string]*models.SSHKeyPair),
		connections:    make(map[string]*models.SSHConnection),
		envConnections: make(map[string][]string),
	}
}


// GenerateKeyPair generates a new SSH key pair
func (s *SSHService) GenerateKeyPair(ctx context.Context, userID string, req *models.CreateSSHKeyRequest) (*models.SSHKeyPair, error) {
	logger := s.logger.With(zap.String("userId", userID), zap.String("name", req.Name))
	logger.Info("Generating SSH key pair", zap.String("type", string(req.Type)))

	var publicKey, privateKey string
	var fingerprint string
	var err error

	switch req.Type {
	case models.SSHKeyTypeEd25519:
		publicKey, privateKey, fingerprint, err = s.generateEd25519Key(req.Comment)
	case models.SSHKeyTypeRSA:
		publicKey, privateKey, fingerprint, err = s.generateRSAKey(req.Comment)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", req.Type)
	}

	if err != nil {
		logger.Error("Failed to generate key pair", zap.Error(err))
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	now := time.Now()
	keyPair := &models.SSHKeyPair{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Type:        req.Type,
		PublicKey:   publicKey,
		PrivateKey:  privateKey,
		Fingerprint: fingerprint,
		Comment:     req.Comment,
		CreatedAt:   now,
		IsDefault:   req.SetDefault,
	}

	// Set expiration if specified
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := now.AddDate(0, 0, *req.ExpiresIn)
		keyPair.ExpiresAt = &expiresAt
	}

	// Store the key
	s.mu.Lock()
	// If setting as default, unset other defaults
	if req.SetDefault {
		for _, k := range s.keys {
			if k.UserID == userID {
				k.IsDefault = false
			}
		}
	}
	s.keys[keyPair.ID] = keyPair
	s.mu.Unlock()

	logger.Info("SSH key pair generated", zap.String("keyId", keyPair.ID), zap.String("fingerprint", fingerprint))
	return keyPair, nil
}

// generateEd25519Key generates an Ed25519 SSH key pair
func (s *SSHService) generateEd25519Key(comment string) (publicKey, privateKey, fingerprint string, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Convert to SSH format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Format public key
	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	if comment != "" {
		pubKeyStr = pubKeyStr + " " + comment
	}

	// Format private key in PEM format
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Calculate fingerprint
	fp := ssh.FingerprintSHA256(sshPubKey)

	return pubKeyStr, string(privKeyPEM), fp, nil
}

// generateRSAKey generates an RSA SSH key pair (2048 bits)
func (s *SSHService) generateRSAKey(comment string) (publicKey, privateKey, fingerprint string, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Convert to SSH format
	sshPubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Format public key
	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	if comment != "" {
		pubKeyStr = pubKeyStr + " " + comment
	}

	// Format private key in PEM format
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	// Calculate fingerprint
	fp := ssh.FingerprintSHA256(sshPubKey)

	return pubKeyStr, string(privKeyPEM), fp, nil
}

// GetKey retrieves an SSH key by ID
func (s *SSHService) GetKey(ctx context.Context, userID, keyID string) (*models.SSHKeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("SSH key not found: %s", keyID)
	}

	if key.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return key, nil
}

// ListKeys lists SSH keys for a user
func (s *SSHService) ListKeys(ctx context.Context, req *models.ListSSHKeysRequest) (*models.ListSSHKeysResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []models.SSHKeyPair
	for _, key := range s.keys {
		if req.UserID != "" && key.UserID != req.UserID {
			continue
		}
		// Don't include private key in list response
		keyCopy := *key
		keyCopy.PrivateKey = ""
		filtered = append(filtered, keyCopy)
	}

	// Pagination
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return &models.ListSSHKeysResponse{
		Keys:     filtered[start:end],
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteKey deletes an SSH key
func (s *SSHService) DeleteKey(ctx context.Context, userID, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.keys[keyID]
	if !ok {
		return fmt.Errorf("SSH key not found: %s", keyID)
	}

	if key.UserID != userID {
		return fmt.Errorf("access denied")
	}

	delete(s.keys, keyID)
	s.logger.Info("SSH key deleted", zap.String("keyId", keyID), zap.String("userId", userID))
	return nil
}

// RotateKey rotates an SSH key (generates new key pair with same metadata)
func (s *SSHService) RotateKey(ctx context.Context, userID, keyID string) (*models.SSHKeyPair, error) {
	s.mu.Lock()
	oldKey, ok := s.keys[keyID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("SSH key not found: %s", keyID)
	}

	if oldKey.UserID != userID {
		s.mu.Unlock()
		return nil, fmt.Errorf("access denied")
	}
	s.mu.Unlock()

	// Generate new key with same settings
	req := &models.CreateSSHKeyRequest{
		Name:       oldKey.Name,
		Type:       oldKey.Type,
		Comment:    oldKey.Comment,
		SetDefault: oldKey.IsDefault,
	}

	newKey, err := s.GenerateKeyPair(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Mark rotation time
	now := time.Now()
	newKey.RotatedAt = &now

	// Delete old key
	s.mu.Lock()
	delete(s.keys, keyID)
	s.mu.Unlock()

	s.logger.Info("SSH key rotated",
		zap.String("oldKeyId", keyID),
		zap.String("newKeyId", newKey.ID),
		zap.String("userId", userID))

	return newKey, nil
}

// GetDefaultKey gets the default SSH key for a user
func (s *SSHService) GetDefaultKey(ctx context.Context, userID string) (*models.SSHKeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, key := range s.keys {
		if key.UserID == userID && key.IsDefault {
			return key, nil
		}
	}

	return nil, fmt.Errorf("no default SSH key found for user")
}

// SetDefaultKey sets a key as the default for a user
func (s *SSHService) SetDefaultKey(ctx context.Context, userID, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.keys[keyID]
	if !ok {
		return fmt.Errorf("SSH key not found: %s", keyID)
	}

	if key.UserID != userID {
		return fmt.Errorf("access denied")
	}

	// Unset other defaults
	for _, k := range s.keys {
		if k.UserID == userID {
			k.IsDefault = false
		}
	}

	key.IsDefault = true
	return nil
}

// GetExpiredKeys returns keys that have expired or will expire within the given duration
func (s *SSHService) GetExpiredKeys(ctx context.Context, within time.Duration) ([]*models.SSHKeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(within)
	var expired []*models.SSHKeyPair

	for _, key := range s.keys {
		if key.ExpiresAt != nil && key.ExpiresAt.Before(cutoff) {
			expired = append(expired, key)
		}
	}

	return expired, nil
}

// CalculateFingerprint calculates the SHA256 fingerprint of a public key
func (s *SSHService) CalculateFingerprint(publicKey string) (string, error) {
	// Parse the public key
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid public key format")
	}

	keyData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	hash := sha256.Sum256(keyData)
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:]), nil
}


// GenerateSSHConfig generates SSH configuration for connecting to an environment
func (s *SSHService) GenerateSSHConfig(ctx context.Context, env *models.Environment, ideType models.IDEType, keyPath string) (*models.SSHConfigResponse, error) {
	logger := s.logger.With(
		zap.String("environmentId", env.ID),
		zap.String("ideType", string(ideType)),
	)
	logger.Info("Generating SSH config")

	if env.Connection == nil {
		return nil, fmt.Errorf("environment has no connection info")
	}

	// Build SSH config
	config := &models.SSHConfig{
		Host:               fmt.Sprintf("devbox-%s", env.ID[:8]),
		Port:               int(env.Connection.SSHPort),
		User:               "devbox",
		IdentityFile:       keyPath,
		StrictHostKeyCheck: "no",
		UserKnownHostsFile: "/dev/null",
		ServerAliveInterval: s.config.KeepAliveInterval,
		ServerAliveCountMax: s.config.ServerAliveCountMax,
	}

	// Set host based on connection type
	if env.Connection.ExternalIP != "" {
		config.Host = env.Connection.ExternalIP
	} else if env.Connection.SSHHost != "" {
		config.Host = env.Connection.SSHHost
	}

	// Add proxy command if using gateway
	if s.config.GatewayHost != "" {
		config.ProxyCommand = fmt.Sprintf(
			"ssh -W %%h:%%p -p %d %s@%s",
			s.config.GatewayPort,
			s.config.GatewayUser,
			s.config.GatewayHost,
		)
	}

	// Generate config string based on IDE type
	var configString string
	var instructions string

	switch ideType {
	case models.IDETypeVSCode:
		configString = s.generateVSCodeConfig(env, config)
		instructions = s.getVSCodeInstructions(env)
	case models.IDETypeJetBrains:
		configString = s.generateJetBrainsConfig(env, config)
		instructions = s.getJetBrainsInstructions(env)
	default:
		configString = s.generateGenericConfig(env, config)
		instructions = s.getGenericInstructions(env)
	}

	logger.Info("SSH config generated successfully")
	return &models.SSHConfigResponse{
		Config:       config,
		ConfigString: configString,
		IDEType:      ideType,
		Instructions: instructions,
	}, nil
}

// generateVSCodeConfig generates SSH config for VSCode Remote SSH
func (s *SSHService) generateVSCodeConfig(env *models.Environment, config *models.SSHConfig) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# DevBox Environment: %s\n", env.Name))
	sb.WriteString(fmt.Sprintf("Host devbox-%s\n", env.ID[:8]))
	
	if env.Connection.ExternalIP != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.ExternalIP))
	} else if env.Connection.SSHHost != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.SSHHost))
	}
	
	sb.WriteString(fmt.Sprintf("    Port %d\n", config.Port))
	sb.WriteString(fmt.Sprintf("    User %s\n", config.User))
	
	if config.IdentityFile != "" {
		sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", config.IdentityFile))
	}
	
	if config.ProxyCommand != "" {
		sb.WriteString(fmt.Sprintf("    ProxyCommand %s\n", config.ProxyCommand))
	}
	
	sb.WriteString(fmt.Sprintf("    StrictHostKeyChecking %s\n", config.StrictHostKeyCheck))
	sb.WriteString(fmt.Sprintf("    UserKnownHostsFile %s\n", config.UserKnownHostsFile))
	sb.WriteString(fmt.Sprintf("    ServerAliveInterval %d\n", config.ServerAliveInterval))
	sb.WriteString(fmt.Sprintf("    ServerAliveCountMax %d\n", config.ServerAliveCountMax))
	
	// VSCode specific settings
	sb.WriteString("    ForwardAgent yes\n")
	sb.WriteString("    AddKeysToAgent yes\n")

	return sb.String()
}

// generateJetBrainsConfig generates SSH config for JetBrains Gateway
func (s *SSHService) generateJetBrainsConfig(env *models.Environment, config *models.SSHConfig) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# JetBrains Gateway Configuration for DevBox: %s\n", env.Name))
	sb.WriteString("# Add this to your SSH config file (~/.ssh/config)\n\n")
	
	sb.WriteString(fmt.Sprintf("Host devbox-%s\n", env.ID[:8]))
	
	if env.Connection.ExternalIP != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.ExternalIP))
	} else if env.Connection.SSHHost != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.SSHHost))
	}
	
	sb.WriteString(fmt.Sprintf("    Port %d\n", config.Port))
	sb.WriteString(fmt.Sprintf("    User %s\n", config.User))
	
	if config.IdentityFile != "" {
		sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", config.IdentityFile))
	}
	
	if config.ProxyCommand != "" {
		sb.WriteString(fmt.Sprintf("    ProxyCommand %s\n", config.ProxyCommand))
	}
	
	sb.WriteString(fmt.Sprintf("    StrictHostKeyChecking %s\n", config.StrictHostKeyCheck))
	sb.WriteString(fmt.Sprintf("    UserKnownHostsFile %s\n", config.UserKnownHostsFile))
	sb.WriteString(fmt.Sprintf("    ServerAliveInterval %d\n", config.ServerAliveInterval))
	sb.WriteString(fmt.Sprintf("    ServerAliveCountMax %d\n", config.ServerAliveCountMax))
	
	// JetBrains specific settings
	sb.WriteString("    Compression yes\n")
	sb.WriteString("    TCPKeepAlive yes\n")
	sb.WriteString("    ControlMaster auto\n")
	sb.WriteString("    ControlPath ~/.ssh/sockets/%r@%h-%p\n")
	sb.WriteString("    ControlPersist 600\n")

	return sb.String()
}

// generateGenericConfig generates generic SSH config
func (s *SSHService) generateGenericConfig(env *models.Environment, config *models.SSHConfig) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# SSH Configuration for DevBox: %s\n", env.Name))
	sb.WriteString(fmt.Sprintf("Host devbox-%s\n", env.ID[:8]))
	
	if env.Connection.ExternalIP != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.ExternalIP))
	} else if env.Connection.SSHHost != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", env.Connection.SSHHost))
	}
	
	sb.WriteString(fmt.Sprintf("    Port %d\n", config.Port))
	sb.WriteString(fmt.Sprintf("    User %s\n", config.User))
	
	if config.IdentityFile != "" {
		sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", config.IdentityFile))
	}
	
	if config.ProxyCommand != "" {
		sb.WriteString(fmt.Sprintf("    ProxyCommand %s\n", config.ProxyCommand))
	}
	
	sb.WriteString(fmt.Sprintf("    StrictHostKeyChecking %s\n", config.StrictHostKeyCheck))
	sb.WriteString(fmt.Sprintf("    UserKnownHostsFile %s\n", config.UserKnownHostsFile))
	sb.WriteString(fmt.Sprintf("    ServerAliveInterval %d\n", config.ServerAliveInterval))
	sb.WriteString(fmt.Sprintf("    ServerAliveCountMax %d\n", config.ServerAliveCountMax))

	return sb.String()
}

// getVSCodeInstructions returns instructions for VSCode Remote SSH
func (s *SSHService) getVSCodeInstructions(env *models.Environment) string {
	return fmt.Sprintf(`## VSCode Remote SSH Setup

1. Install the "Remote - SSH" extension in VSCode
2. Copy the SSH config above to your ~/.ssh/config file
3. Press Ctrl+Shift+P (Cmd+Shift+P on Mac) and select "Remote-SSH: Connect to Host"
4. Select "devbox-%s" from the list
5. VSCode will connect to your DevBox environment

### Quick Connect Command
ssh devbox-%s

### Alternative: Direct Connection
ssh -p %d devbox@%s
`, env.ID[:8], env.ID[:8], env.Connection.SSHPort, env.Connection.ExternalIP)
}

// getJetBrainsInstructions returns instructions for JetBrains Gateway
func (s *SSHService) getJetBrainsInstructions(env *models.Environment) string {
	return fmt.Sprintf(`## JetBrains Gateway Setup

1. Download and install JetBrains Gateway from https://www.jetbrains.com/remote-development/gateway/
2. Copy the SSH config above to your ~/.ssh/config file
3. Create the sockets directory: mkdir -p ~/.ssh/sockets
4. Open JetBrains Gateway
5. Click "Connect via SSH"
6. Select "devbox-%s" from the SSH configurations
7. Choose your preferred IDE (IntelliJ IDEA, PyCharm, WebStorm, etc.)
8. Gateway will install the IDE backend and connect

### Quick Connect Command
ssh devbox-%s

### Connection Multiplexing
The configuration enables connection multiplexing for faster reconnections.
`, env.ID[:8], env.ID[:8])
}

// getGenericInstructions returns generic SSH instructions
func (s *SSHService) getGenericInstructions(env *models.Environment) string {
	return fmt.Sprintf(`## SSH Connection Instructions

1. Copy the SSH config above to your ~/.ssh/config file
2. Connect using: ssh devbox-%s

### Direct Connection (without config)
ssh -p %d -i /path/to/your/key devbox@%s

### Port Forwarding Example
ssh -L 8080:localhost:8080 devbox-%s

### File Transfer (SCP)
scp -P %d local_file devbox@%s:/path/to/destination
`, env.ID[:8], env.Connection.SSHPort, env.Connection.ExternalIP, env.ID[:8], env.Connection.SSHPort, env.Connection.ExternalIP)
}

// GenerateSSHCommand generates a direct SSH command for an environment
func (s *SSHService) GenerateSSHCommand(env *models.Environment, keyPath string) string {
	if env.Connection == nil {
		return ""
	}

	host := env.Connection.ExternalIP
	if host == "" {
		host = env.Connection.SSHHost
	}

	cmd := fmt.Sprintf("ssh -p %d", env.Connection.SSHPort)
	
	if keyPath != "" {
		cmd += fmt.Sprintf(" -i %s", keyPath)
	}
	
	cmd += fmt.Sprintf(" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null")
	cmd += fmt.Sprintf(" devbox@%s", host)

	return cmd
}


// GetProxyConfig returns the proxy gateway configuration for an environment
func (s *SSHService) GetProxyConfig(env *models.Environment) *models.SSHProxyConfig {
	if env.Connection == nil {
		return nil
	}

	targetHost := env.Connection.PodIP
	if targetHost == "" {
		targetHost = env.Connection.ExternalIP
	}

	return &models.SSHProxyConfig{
		GatewayHost:     s.config.GatewayHost,
		GatewayPort:     s.config.GatewayPort,
		GatewayUser:     s.config.GatewayUser,
		TargetHost:      targetHost,
		TargetPort:      int(env.Connection.SSHPort),
		ConnectionReuse: true,
		KeepAlive:       s.config.KeepAliveInterval,
	}
}

// RegisterConnection registers a new SSH connection
func (s *SSHService) RegisterConnection(ctx context.Context, conn *models.SSHConnection) error {
	s.envMu.Lock()
	defer s.envMu.Unlock()

	// Check connection limit
	currentConns := s.envConnections[conn.EnvironmentID]
	if len(currentConns) >= s.config.MaxConnectionsPerEnv {
		return &ConnectionLimitError{
			EnvironmentID:     conn.EnvironmentID,
			CurrentConnections: len(currentConns),
			MaxConnections:    s.config.MaxConnectionsPerEnv,
		}
	}

	// Generate connection ID if not set
	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	conn.ConnectedAt = now
	conn.LastActivityAt = now

	// Store connection
	s.mu.Lock()
	s.connections[conn.ID] = conn
	s.mu.Unlock()

	// Track by environment
	s.envConnections[conn.EnvironmentID] = append(s.envConnections[conn.EnvironmentID], conn.ID)

	s.logger.Info("SSH connection registered",
		zap.String("connectionId", conn.ID),
		zap.String("environmentId", conn.EnvironmentID),
		zap.String("clientIp", conn.ClientIP),
		zap.Int("activeConnections", len(s.envConnections[conn.EnvironmentID])))

	return nil
}

// UnregisterConnection removes an SSH connection
func (s *SSHService) UnregisterConnection(ctx context.Context, connectionID string) error {
	s.mu.Lock()
	conn, ok := s.connections[connectionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("connection not found: %s", connectionID)
	}
	delete(s.connections, connectionID)
	s.mu.Unlock()

	// Remove from environment tracking
	s.envMu.Lock()
	envConns := s.envConnections[conn.EnvironmentID]
	for i, id := range envConns {
		if id == connectionID {
			s.envConnections[conn.EnvironmentID] = append(envConns[:i], envConns[i+1:]...)
			break
		}
	}
	s.envMu.Unlock()

	s.logger.Info("SSH connection unregistered",
		zap.String("connectionId", connectionID),
		zap.String("environmentId", conn.EnvironmentID))

	return nil
}

// GetConnection retrieves a connection by ID
func (s *SSHService) GetConnection(ctx context.Context, connectionID string) (*models.SSHConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.connections[connectionID]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", connectionID)
	}

	return conn, nil
}

// GetEnvironmentConnections returns all connections for an environment
func (s *SSHService) GetEnvironmentConnections(ctx context.Context, environmentID string) ([]*models.SSHConnection, error) {
	s.envMu.RLock()
	connIDs := s.envConnections[environmentID]
	s.envMu.RUnlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var connections []*models.SSHConnection
	for _, id := range connIDs {
		if conn, ok := s.connections[id]; ok {
			connections = append(connections, conn)
		}
	}

	return connections, nil
}

// GetConnectionStats returns connection statistics for an environment
func (s *SSHService) GetConnectionStats(ctx context.Context, environmentID string) (*models.SSHConnectionStats, error) {
	conns, err := s.GetEnvironmentConnections(ctx, environmentID)
	if err != nil {
		return nil, err
	}

	stats := &models.SSHConnectionStats{
		EnvironmentID:     environmentID,
		ActiveConnections: len(conns),
		MaxConnections:    s.config.MaxConnectionsPerEnv,
	}

	for _, conn := range conns {
		stats.TotalBytesSent += conn.BytesSent
		stats.TotalBytesReceived += conn.BytesReceived
	}

	return stats, nil
}

// UpdateConnectionActivity updates the last activity time for a connection
func (s *SSHService) UpdateConnectionActivity(ctx context.Context, connectionID string, bytesSent, bytesReceived int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.connections[connectionID]
	if !ok {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	conn.LastActivityAt = time.Now()
	conn.BytesSent += bytesSent
	conn.BytesReceived += bytesReceived

	return nil
}

// CanConnect checks if a new connection can be established to an environment
func (s *SSHService) CanConnect(ctx context.Context, environmentID string) (bool, int, int) {
	s.envMu.RLock()
	defer s.envMu.RUnlock()

	currentConns := len(s.envConnections[environmentID])
	return currentConns < s.config.MaxConnectionsPerEnv, currentConns, s.config.MaxConnectionsPerEnv
}

// CleanupStaleConnections removes connections that haven't had activity within the timeout
func (s *SSHService) CleanupStaleConnections(ctx context.Context, timeout time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	var staleIDs []string

	for id, conn := range s.connections {
		if conn.LastActivityAt.Before(cutoff) {
			staleIDs = append(staleIDs, id)
		}
	}

	// Remove stale connections
	for _, id := range staleIDs {
		conn := s.connections[id]
		delete(s.connections, id)

		// Remove from environment tracking
		s.envMu.Lock()
		envConns := s.envConnections[conn.EnvironmentID]
		for i, connID := range envConns {
			if connID == id {
				s.envConnections[conn.EnvironmentID] = append(envConns[:i], envConns[i+1:]...)
				break
			}
		}
		s.envMu.Unlock()
	}

	if len(staleIDs) > 0 {
		s.logger.Info("Cleaned up stale SSH connections", zap.Int("count", len(staleIDs)))
	}

	return len(staleIDs), nil
}

// ConnectionLimitError represents a connection limit exceeded error
type ConnectionLimitError struct {
	EnvironmentID      string
	CurrentConnections int
	MaxConnections     int
}

func (e *ConnectionLimitError) Error() string {
	return fmt.Sprintf("connection limit exceeded for environment %s: %d/%d connections",
		e.EnvironmentID, e.CurrentConnections, e.MaxConnections)
}

// IsConnectionLimitError checks if an error is a connection limit error
func IsConnectionLimitError(err error) bool {
	_, ok := err.(*ConnectionLimitError)
	return ok
}

// GetMaxConnectionsPerEnv returns the maximum connections allowed per environment
func (s *SSHService) GetMaxConnectionsPerEnv() int {
	return s.config.MaxConnectionsPerEnv
}

// GetConfig returns the SSH service configuration
func (s *SSHService) GetConfig() *SSHServiceConfig {
	return s.config
}
