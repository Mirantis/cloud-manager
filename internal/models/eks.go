package models

import "time"

// EKSCluster represents an AWS EKS cluster
type EKSCluster struct {
	ClusterName    string     `json:"cluster_name"`
	ClusterARN     string     `json:"cluster_arn"`
	AccountID      string     `json:"account_id"`
	AccountName    string     `json:"account_name"`
	Region         string     `json:"region"`
	Status         string     `json:"status"`
	PlatformVersion string    `json:"platform_version"`
	Arn             string    `json:"arn"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	Endpoint        string     `json:"endpoint,omitempty"`
	AuthMode        string     `json:"auth_mode,omitempty"`
	Tags            []Tag      `json:"tags,omitempty"`
	NodeGroups     []NodeGroup `json:"node_groups,omitempty"`
}

// NodeGroup represents an EKS node group
type NodeGroup struct {
	NodeGroupName  string     `json:"node_group_name"`
	NodeGroupARN   string     `json:"node_group_arn"`
	Status         string     `json:"status"`
	InstanceTypes  []string   `json:"instance_types"`
	ScalingConfig  NodeScaling `json:"scaling_config"`
	ReleaseVersion string     `json:"release_version,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Tags           []Tag      `json:"tags,omitempty"`
}

// NodeScaling represents EKS node group scaling configuration
type NodeScaling struct {
	DesiredSize int64 `json:"desired_size"`
	MinSize     int64 `json:"min_size"`
	MaxSize     int64 `json:"max_size"`
}

// EKSClusterWithAccount wraps an EKSCluster with account info
type EKSClusterWithAccount struct {
	EKSCluster
}
