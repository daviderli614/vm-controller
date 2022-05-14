/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ucloud

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/ucloud/services/uk8s"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/klog"
)

const (
	// GPULabel is the label added to nodes with GPU resource.
	GPULabel = "accelerator"
)

var (
	availableGPUTypes = map[string]struct{}{
		"nvidia-tesla-k80":  {},
		"nvidia-tesla-p40":  {},
		"nvidia-tesla-v100": {},
	}
)

// UCloudCloudProvider implements CloudProvider interface.
type UCloudCloudProvider struct {
	ucloudManager *UCloudManager
	//asgs          []*Asg
	// This resource limiter is used if resource limits are not defined through cloud API.
	resourceLimiter *cloudprovider.ResourceLimiter
}

// BuildUCloudCloudProvider builds CloudProvider implementation for UCloud.
func BuildUCloudCloudProvider(manager *UCloudManager, resourceLimiter *cloudprovider.ResourceLimiter) *UCloudCloudProvider {
	return &UCloudCloudProvider{
		ucloudManager:   manager,
		resourceLimiter: resourceLimiter,
	}
}

func BuildUCloud(opts config.AutoscalingOptions, do cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	manager, err := CreateUCloudManager(do)
	if err != nil {
		klog.Fatalf("Failed to create ucloud Manager: %v", err)
	}

	return BuildUCloudCloudProvider(manager, rl)
}

//func buildAsgFromSpec(value string, ucloudManager *UCloudManager) (*Asg, error) {
//	spec, err := dynamic.SpecFromString(value, true)
//	if err != nil {
//		return nil, fmt.Errorf("failed to parse node group spec: %v", err)
//	}
//	asg := buildAsg(ucloudManager, spec.MinSize, spec.MaxSize, spec.Name, ucloudManager.cfg.RegionId)
//	return asg, nil
//}

func buildAsg(ucloudManager *UCloudManager, autoScalingGroup *uk8s.AutoScalingGroup, regionId string) *Asg {
	return &Asg{
		ucloudManager: ucloudManager,
		config:        *autoScalingGroup,
		UCloudRef: UCloudRef{
			RegionId: regionId,
			Id:       autoScalingGroup.Id,
		},
	}
}

// add node group defined in string spec. Format:
// minNodes:maxNodes:asgName
//func (ucloud *UCloudCloudProvider) addNodeGroup(spec string) error {
//	asg, err := buildAsgFromSpec(spec, ucloud.ucloudManager)
//	if err != nil {
//		klog.Errorf("failed to build ASG from spec,because of %s", err.Error())
//		return err
//	}
//	ucloud.addAsg(asg)
//	return nil
//}

// add and register an asg to this cloud provider
//func (ucloud *UCloudCloudProvider) addAsg(asg *Asg) {
//	//ucloud.asgs = append(ucloud.asgs, asg)
//	ucloud.ucloudManager.RegisterAsg(asg)
//}

// Name returns name of the cloud provider.
func (ucloud *UCloudCloudProvider) Name() string {
	return cloudprovider.UCloudProviderName
}

// NodeGroups returns all node groups configured for this cloud provider.
func (ucloud *UCloudCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	result := make([]cloudprovider.NodeGroup, 0, len(ucloud.ucloudManager.asgCache.registeredAsgs))
	for _, asg := range ucloud.ucloudManager.asgCache.registeredAsgs {
		result = append(result, asg)
	}
	return result
}

// NodeGroupForNode returns the node group for the given node, nil if the node
// should not be processed by cluster autoscaler, or non-nil error if such
// occurred. Must be implemented.
func (ucloud *UCloudCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	if node.Spec.ProviderID == "" {
		return nil, nil //不需要被处理
	}
	//splitted := strings.Split(node.Spec.ProviderID, "//")
	//if len(splitted) != 3 {
	//	klog.V(1).Infof("parse ProviderID failed: %v", node.Spec.ProviderID)
	//	return nil, nil
	//}
	//nodeName := splitted[2] //node-id
	//return ucloud.ucloudManager.GetAsgForInstance(providerId)
	return ucloud.ucloudManager.GetAsgForInstance(node.Spec.ProviderID)
}

// Pricing returns pricing model for this cloud provider or error if not available.
// Implementation optional.
func (ucloud *UCloudCloudProvider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetAvailableMachineTypes get all machine types that can be requested from the cloud provider.
// Implementation optional.
func (ucloud *UCloudCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return []string{}, nil
}

// NewNodeGroup builds a theoretical node group based on the node definition provided. The node group is not automatically
// created on the cloud provider side. The node group is not returned by NodeGroups() until it is created.
// Implementation optional.
func (ucloud *UCloudCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string, taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetResourceLimiter returns struct containing limits (max, min) for resources (cores, memory etc.).
func (ucloud *UCloudCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return ucloud.resourceLimiter, nil
}

// GPULabel returns the label added to nodes with GPU resource.
func (ucloud *UCloudCloudProvider) GPULabel() string {
	return GPULabel
}

// GetAvailableGPUTypes return all available GPU types cloud provider supports.
func (ucloud *UCloudCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return availableGPUTypes
}

// Cleanup cleans up open resources before the cloud provider is destroyed, i.e. go routines etc.
func (ucloud *UCloudCloudProvider) Cleanup() error {
	return nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
func (ucloud *UCloudCloudProvider) Refresh() error {
	return ucloud.ucloudManager.Refresh()
}
