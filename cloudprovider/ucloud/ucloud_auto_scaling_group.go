package ucloud

import (
	"fmt"
	"strings"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/ucloud/services/uk8s"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

/*
node group code
*/

type UCloudRef struct {
	RegionId string
	Id       string
}

//type asgInformation struct {
//	asg      *Asg
//	basename string
//}

// Asg implements NodeGroup interface.
type Asg struct {
	ucloudManager *UCloudManager

	config uk8s.AutoScalingGroup
	nodes  []Node
	UCloudRef
}

type Node struct {
	ProviderId string
	Name       string
}

// MaxSize returns maximum size of the node group.
func (asg *Asg) MaxSize() int {
	return asg.config.Max
}

// MinSize returns minimum size of the node group.
func (asg *Asg) MinSize() int {
	return asg.config.Min
}

// TargetSize returns the current TARGET size of the node group. It is possible that the
// number is different from the number of nodes registered in Kubernetes.
func (asg *Asg) TargetSize() (int, error) {
	return asg.ucloudManager.GetAsgSize(asg), nil
}

// IncreaseSize increases Asg size
func (asg *Asg) IncreaseSize(delta int) error {
	if delta <= 0 {
		return fmt.Errorf("size increase must be positive")
	}
	size := asg.ucloudManager.GetAsgSize(asg)
	if int(size)+delta > asg.MaxSize() {
		return fmt.Errorf("size increase too large - desired:%d max:%d", int(size)+delta, asg.MaxSize())
	}
	return asg.ucloudManager.ScaleUpCluster(delta, asg.config)
}

// DecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the
// request for new nodes that have not been yet fulfilled. Delta should be negative.
func (asg *Asg) DecreaseTargetSize(delta int) error {
	return cloudprovider.ErrNotImplemented
}

// DeleteNodes deletes the nodes from the group.
func (asg *Asg) DeleteNodes(nodes []*apiv1.Node) error {
	size := asg.ucloudManager.GetAsgSize(asg)
	if int(size) <= asg.MinSize() {
		return fmt.Errorf("min size reached, nodes will not be deleted")
	}
	nodesId := make([]string, 0)
	for _, node := range nodes {
		splitted := strings.Split(node.Spec.ProviderID, "//")
		if len(splitted) != 3 {
			return fmt.Errorf("Not expected name: %s\n", node.Spec.ProviderID)
		}
		nodesId = append(nodesId, splitted[2])
	}
	return asg.ucloudManager.ScaleDownCluster(nodesId)
}

// Id returns asg url.
func (asg *Asg) Id() string {
	return asg.UCloudRef.Id
}

// Debug returns a debug string for the Asg.
func (asg *Asg) Debug() string {
	return fmt.Sprintf("%s (%d:%d)", asg.Id(), asg.MinSize(), asg.MaxSize())
}

// Nodes returns a list of all nodes that belong to this node group.
func (asg *Asg) Nodes() ([]cloudprovider.Instance, error) {
	asgNodes := asg.ucloudManager.GetAsgNodes(asg)
	instances := make([]cloudprovider.Instance, len(asgNodes))

	for i, asgNode := range asgNodes {
		instances[i] = cloudprovider.Instance{Id: asgNode.ProviderId}
	}
	return instances, nil
}

// Exist checks if the node group really exists on the cloud provider side.
func (asg *Asg) Exist() bool {
	return true
}

// Create creates the node group on the cloud provider side.
func (asg *Asg) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// Delete deletes the node group on the cloud provider side.
func (asg *Asg) Delete() error {
	return cloudprovider.ErrNotImplemented
}

// Autoprovisioned returns true if the node group is autoprovisioned.
func (asg *Asg) Autoprovisioned() bool {
	return false
}

// TemplateNodeInfo returns a node template for this node group.
func (asg *Asg) TemplateNodeInfo() (*schedulernodeinfo.NodeInfo, error) {
	template, err := asg.ucloudManager.GetAsgTemplateNode(asg)
	if err != nil {
		return nil, err
	}
	node, err := asg.ucloudManager.buildNodeFromTemplate(asg, template)
	if err != nil {
		return nil, err
	}
	nodeInfo := schedulernodeinfo.NewNodeInfo()
	nodeInfo.SetNode(node)
	return nodeInfo, nil
}
