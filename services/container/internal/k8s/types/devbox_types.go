// Package types defines the DevBox Custom Resource types for Kubernetes.
package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevBoxSpec defines the desired state of DevBox
type DevBoxSpec struct {
	// TemplateID is the ID of the template to use for this DevBox
	TemplateID string `json:"templateId"`

	// UserID is the ID of the user who owns this DevBox
	UserID string `json:"userId"`

	// Name is the human-readable name for the DevBox
	// +optional
	Name string `json:"name,omitempty"`

	// Description of the DevBox
	// +optional
	Description string `json:"description,omitempty"`

	// Resources defines the resource allocation for the DevBox
	// +optional
	Resources *ResourceSpec `json:"resources,omitempty"`

	// Runtime configuration
	// +optional
	Runtime *RuntimeSpec `json:"runtime,omitempty"`

	// Environment variables
	// +optional
	Environment []EnvVar `json:"environment,omitempty"`

	// Ports to expose
	// +optional
	Ports []PortSpec `json:"ports,omitempty"`

	// Volumes to mount
	// +optional
	Volumes []VolumeSpec `json:"volumes,omitempty"`

	// SSH configuration
	// +optional
	SSH *SSHSpec `json:"ssh,omitempty"`

	// AutoStop configuration
	// +optional
	AutoStop *AutoStopSpec `json:"autoStop,omitempty"`

	// AutoDelete configuration
	// +optional
	AutoDelete *AutoDeleteSpec `json:"autoDelete,omitempty"`
}

// ResourceSpec defines resource allocation
type ResourceSpec struct {
	// CPU allocation (e.g., "1", "2", "0.5")
	CPU string `json:"cpu,omitempty"`

	// Memory allocation (e.g., "1Gi", "2Gi")
	Memory string `json:"memory,omitempty"`

	// Storage allocation (e.g., "10Gi", "50Gi")
	Storage string `json:"storage,omitempty"`
}

// RuntimeSpec defines runtime configuration
type RuntimeSpec struct {
	// Image is the container image to use
	Image string `json:"image,omitempty"`

	// Command to run in the container
	Command []string `json:"command,omitempty"`

	// Args for the command
	Args []string `json:"args,omitempty"`

	// WorkingDir in the container
	WorkingDir string `json:"workingDir,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PortSpec defines a port mapping
type PortSpec struct {
	// Name of the port
	Name string `json:"name,omitempty"`

	// ContainerPort is the port inside the container
	ContainerPort int32 `json:"containerPort"`

	// Protocol (TCP or UDP)
	Protocol string `json:"protocol,omitempty"`
}

// VolumeSpec defines a volume mount
type VolumeSpec struct {
	// Name of the volume
	Name string `json:"name"`

	// MountPath in the container
	MountPath string `json:"mountPath"`

	// SubPath within the volume
	SubPath string `json:"subPath,omitempty"`

	// ReadOnly mount
	ReadOnly bool `json:"readOnly,omitempty"`
}

// SSHSpec defines SSH configuration
type SSHSpec struct {
	// Enabled indicates if SSH is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Port for SSH (default 22)
	Port int32 `json:"port,omitempty"`

	// PublicKey for SSH authentication
	PublicKey string `json:"publicKey,omitempty"`
}

// AutoStopSpec defines auto-stop configuration
type AutoStopSpec struct {
	// Enabled indicates if auto-stop is enabled
	Enabled bool `json:"enabled,omitempty"`

	// InactivityTimeout duration before auto-stop (e.g., "72h")
	InactivityTimeout string `json:"inactivityTimeout,omitempty"`
}

// AutoDeleteSpec defines auto-delete configuration
type AutoDeleteSpec struct {
	// Enabled indicates if auto-delete is enabled
	Enabled bool `json:"enabled,omitempty"`

	// StoppedTimeout duration after stop before auto-delete (e.g., "168h")
	StoppedTimeout string `json:"stoppedTimeout,omitempty"`
}

// DevBoxPhase represents the current phase of a DevBox
type DevBoxPhase string

const (
	// DevBoxPhaseCreating indicates the DevBox is being created
	DevBoxPhaseCreating DevBoxPhase = "Creating"

	// DevBoxPhaseRunning indicates the DevBox is running
	DevBoxPhaseRunning DevBoxPhase = "Running"

	// DevBoxPhaseStopped indicates the DevBox is stopped
	DevBoxPhaseStopped DevBoxPhase = "Stopped"

	// DevBoxPhaseFailed indicates the DevBox has failed
	DevBoxPhaseFailed DevBoxPhase = "Failed"

	// DevBoxPhaseSuspended indicates the DevBox is suspended
	DevBoxPhaseSuspended DevBoxPhase = "Suspended"

	// DevBoxPhaseDeleting indicates the DevBox is being deleted
	DevBoxPhaseDeleting DevBoxPhase = "Deleting"
)

// DevBoxStatus defines the observed state of DevBox
type DevBoxStatus struct {
	// Phase is the current phase of the DevBox
	Phase DevBoxPhase `json:"phase,omitempty"`

	// Message is a human-readable message about the current status
	Message string `json:"message,omitempty"`

	// PodName is the name of the underlying Pod
	PodName string `json:"podName,omitempty"`

	// PodIP is the internal IP address of the Pod
	PodIP string `json:"podIP,omitempty"`

	// ExternalIP is the external IP address for access
	ExternalIP string `json:"externalIP,omitempty"`

	// SSHPort is the external SSH port
	SSHPort int32 `json:"sshPort,omitempty"`

	// WebPorts are the exposed web ports
	WebPorts []WebPortStatus `json:"webPorts,omitempty"`

	// Conditions represent the latest available observations
	Conditions []DevBoxCondition `json:"conditions,omitempty"`

	// ResourceUsage shows current resource usage
	ResourceUsage *ResourceUsageStatus `json:"resourceUsage,omitempty"`

	// LastActivityTime is the last time the DevBox was accessed
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// CreatedAt is when the DevBox was created
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// StartedAt is when the DevBox started running
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// StoppedAt is when the DevBox was stopped
	StoppedAt *metav1.Time `json:"stoppedAt,omitempty"`
}

// WebPortStatus represents an exposed web port
type WebPortStatus struct {
	Name         string `json:"name,omitempty"`
	Port         int32  `json:"port,omitempty"`
	ExternalPort int32  `json:"externalPort,omitempty"`
	URL          string `json:"url,omitempty"`
}

// DevBoxConditionType represents a condition type
type DevBoxConditionType string

const (
	// DevBoxConditionReady indicates the DevBox is ready
	DevBoxConditionReady DevBoxConditionType = "Ready"

	// DevBoxConditionPodReady indicates the Pod is ready
	DevBoxConditionPodReady DevBoxConditionType = "PodReady"

	// DevBoxConditionNetworkReady indicates networking is ready
	DevBoxConditionNetworkReady DevBoxConditionType = "NetworkReady"

	// DevBoxConditionStorageReady indicates storage is ready
	DevBoxConditionStorageReady DevBoxConditionType = "StorageReady"

	// DevBoxConditionSSHReady indicates SSH is ready
	DevBoxConditionSSHReady DevBoxConditionType = "SSHReady"
)

// DevBoxCondition represents a condition of a DevBox
type DevBoxCondition struct {
	// Type of the condition
	Type DevBoxConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine-readable reason for the condition
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable message
	Message string `json:"message,omitempty"`
}

// ResourceUsageStatus shows current resource usage
type ResourceUsageStatus struct {
	CPU     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="User",type=string,JSONPath=`.spec.userId`
// +kubebuilder:printcolumn:name="Template",type=string,JSONPath=`.spec.templateId`
// +kubebuilder:printcolumn:name="Pod IP",type=string,JSONPath=`.status.podIP`
// +kubebuilder:printcolumn:name="SSH Port",type=integer,JSONPath=`.status.sshPort`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DevBox is the Schema for the devboxes API
type DevBox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevBoxSpec   `json:"spec,omitempty"`
	Status DevBoxStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DevBoxList contains a list of DevBox
type DevBoxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevBox `json:"items"`
}

// DeepCopyInto copies the receiver into out
func (in *DevBox) DeepCopyInto(out *DevBox) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a deep copy of DevBox
func (in *DevBox) DeepCopy() *DevBox {
	if in == nil {
		return nil
	}
	out := new(DevBox)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *DevBox) DeepCopyObject() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto copies the receiver into out
func (in *DevBoxList) DeepCopyInto(out *DevBoxList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DevBox, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of DevBoxList
func (in *DevBoxList) DeepCopy() *DevBoxList {
	if in == nil {
		return nil
	}
	out := new(DevBoxList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *DevBoxList) DeepCopyObject() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto copies DevBoxSpec
func (in *DevBoxSpec) DeepCopyInto(out *DevBoxSpec) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(ResourceSpec)
		**out = **in
	}
	if in.Runtime != nil {
		in, out := &in.Runtime, &out.Runtime
		*out = new(RuntimeSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Environment != nil {
		in, out := &in.Environment, &out.Environment
		*out = make([]EnvVar, len(*in))
		copy(*out, *in)
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortSpec, len(*in))
		copy(*out, *in)
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]VolumeSpec, len(*in))
		copy(*out, *in)
	}
	if in.SSH != nil {
		in, out := &in.SSH, &out.SSH
		*out = new(SSHSpec)
		**out = **in
	}
	if in.AutoStop != nil {
		in, out := &in.AutoStop, &out.AutoStop
		*out = new(AutoStopSpec)
		**out = **in
	}
	if in.AutoDelete != nil {
		in, out := &in.AutoDelete, &out.AutoDelete
		*out = new(AutoDeleteSpec)
		**out = **in
	}
}

// DeepCopyInto copies RuntimeSpec
func (in *RuntimeSpec) DeepCopyInto(out *RuntimeSpec) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto copies DevBoxStatus
func (in *DevBoxStatus) DeepCopyInto(out *DevBoxStatus) {
	*out = *in
	if in.WebPorts != nil {
		in, out := &in.WebPorts, &out.WebPorts
		*out = make([]WebPortStatus, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]DevBoxCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ResourceUsage != nil {
		in, out := &in.ResourceUsage, &out.ResourceUsage
		*out = new(ResourceUsageStatus)
		**out = **in
	}
	if in.LastActivityTime != nil {
		in, out := &in.LastActivityTime, &out.LastActivityTime
		*out = (*in).DeepCopy()
	}
	if in.CreatedAt != nil {
		in, out := &in.CreatedAt, &out.CreatedAt
		*out = (*in).DeepCopy()
	}
	if in.StartedAt != nil {
		in, out := &in.StartedAt, &out.StartedAt
		*out = (*in).DeepCopy()
	}
	if in.StoppedAt != nil {
		in, out := &in.StoppedAt, &out.StoppedAt
		*out = (*in).DeepCopy()
	}
}

// DeepCopyInto copies DevBoxCondition
func (in *DevBoxCondition) DeepCopyInto(out *DevBoxCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}
