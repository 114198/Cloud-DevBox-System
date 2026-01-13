// Package models defines the data models for the container service.
package models

import (
	"time"
)

// SSHKeyType represents the type of SSH key
type SSHKeyType string

const (
	SSHKeyTypeEd25519 SSHKeyType = "ed25519"
	SSHKeyTypeRSA     SSHKeyType = "rsa"
)

// SSHKeyPair represents an SSH key pair
type SSHKeyPair struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	Name         string     `json:"name"`
	Type         SSHKeyType `json:"type"`
	PublicKey    string     `json:"publicKey"`
	PrivateKey   string     `json:"-"` // Never expose in JSON
	Fingerprint  string     `json:"fingerprint"`
	Comment      string     `json:"comment,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	RotatedAt    *time.Time `json:"rotatedAt,omitempty"`
	IsDefault    bool       `json:"isDefault"`
}

// SSHConnection represents an active SSH connection
type SSHConnection struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	UserID        string    `json:"userId"`
	KeyID         string    `json:"keyId"`
	ClientIP      string    `json:"clientIp"`
	ClientPort    int       `json:"clientPort"`
	ServerPort    int       `json:"serverPort"`
	ConnectedAt   time.Time `json:"connectedAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	BytesSent     int64     `json:"bytesSent"`
	BytesReceived int64     `json:"bytesReceived"`
}

// SSHConfig represents SSH configuration for an IDE
type SSHConfig struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	User              string `json:"user"`
	IdentityFile      string `json:"identityFile,omitempty"`
	ProxyCommand      string `json:"proxyCommand,omitempty"`
	StrictHostKeyCheck string `json:"strictHostKeyChecking,omitempty"`
	UserKnownHostsFile string `json:"userKnownHostsFile,omitempty"`
	ServerAliveInterval int   `json:"serverAliveInterval,omitempty"`
	ServerAliveCountMax int   `json:"serverAliveCountMax,omitempty"`
}

// IDEType represents the type of IDE
type IDEType string

const (
	IDETypeVSCode     IDEType = "vscode"
	IDETypeJetBrains  IDEType = "jetbrains"
	IDETypeGeneric    IDEType = "generic"
)

// SSHConfigRequest represents a request to generate SSH config
type SSHConfigRequest struct {
	EnvironmentID string  `json:"environmentId" binding:"required"`
	IDEType       IDEType `json:"ideType" binding:"required"`
	KeyID         string  `json:"keyId,omitempty"`
}

// SSHConfigResponse represents the response with SSH configuration
type SSHConfigResponse struct {
	Config       *SSHConfig `json:"config"`
	ConfigString string     `json:"configString"`
	IDEType      IDEType    `json:"ideType"`
	Instructions string     `json:"instructions,omitempty"`
}

// CreateSSHKeyRequest represents a request to create an SSH key
type CreateSSHKeyRequest struct {
	Name      string     `json:"name" binding:"required,max=255"`
	Type      SSHKeyType `json:"type" binding:"required,oneof=ed25519 rsa"`
	Comment   string     `json:"comment,omitempty"`
	ExpiresIn *int       `json:"expiresIn,omitempty"` // Days until expiration
	SetDefault bool      `json:"setDefault,omitempty"`
}

// RotateSSHKeyRequest represents a request to rotate an SSH key
type RotateSSHKeyRequest struct {
	KeyID string `json:"keyId" binding:"required"`
}

// ListSSHKeysRequest represents a request to list SSH keys
type ListSSHKeysRequest struct {
	UserID   string `form:"userId"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=20"`
}

// ListSSHKeysResponse represents a response containing a list of SSH keys
type ListSSHKeysResponse struct {
	Keys     []SSHKeyPair `json:"keys"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// SSHConnectionStats represents SSH connection statistics
type SSHConnectionStats struct {
	EnvironmentID     string `json:"environmentId"`
	ActiveConnections int    `json:"activeConnections"`
	MaxConnections    int    `json:"maxConnections"`
	TotalConnections  int64  `json:"totalConnections"`
	TotalBytesSent    int64  `json:"totalBytesSent"`
	TotalBytesReceived int64 `json:"totalBytesReceived"`
}

// SSHProxyConfig represents SSH proxy gateway configuration
type SSHProxyConfig struct {
	GatewayHost     string `json:"gatewayHost"`
	GatewayPort     int    `json:"gatewayPort"`
	GatewayUser     string `json:"gatewayUser"`
	TargetHost      string `json:"targetHost"`
	TargetPort      int    `json:"targetPort"`
	ConnectionReuse bool   `json:"connectionReuse"`
	KeepAlive       int    `json:"keepAlive"` // Seconds
}
