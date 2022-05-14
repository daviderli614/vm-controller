package ucloud

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"k8s.io/autoscaler/cluster-autoscaler/config/dynamic"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/ucloud/services/uk8s"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
)

const (
	InternalApi = "http://api.service.ucloud.cn"
	ExternalApi = "http://api.ucloud.cn"

	scaleToZeroSupported = true
)

var (
	refreshInterval                    = 1 * time.Minute
	ResourceGPU     apiv1.ResourceName = "nvidia.com/gpu"
)

type UCloudManager struct {
	cfg                   *cloudConfig
	asgCache              *asgCache
	client                *uk8s.UK8SClient
	interrupt             chan struct{}
	lastRefresh           time.Time
	asgAutoDiscoverySpecs []cloudprovider.LabelAutoDiscoveryConfig
}

func CreateUCloudManager(discoveryOpts cloudprovider.NodeGroupDiscoveryOptions) (*UCloudManager, error) {
	cfg := &cloudConfig{}
	cfg.loadConfigFromSts()
	if cfg.isValid() == false {
		return nil, errors.New("please check whether you have provided correct Access Public Key, Access Private Key,RegionId")
	}
	client := uk8s.NewClient(cfg.Config, cfg.Credential)

	manager := &UCloudManager{
		cfg:       cfg,
		asgCache:  newAsgCache(),
		client:    client,
		interrupt: make(chan struct{}),
	}
	manager.refreshStsToken()

	if discoveryOpts.StaticDiscoverySpecified() {
		klog.V(1).Infoln("UCloud do not support static discovery mode, use auto discovery instead")
	}

	if discoveryOpts.AutoDiscoverySpecified() {
		klog.V(1).Infoln("Found auto discovery specified")
		specs, err := discoveryOpts.ParseLabelAutoDiscoverySpecs()
		if err != nil {
			return nil, err
		}
		manager.asgAutoDiscoverySpecs = specs
		getAutoDiscoverySpecs(manager.asgAutoDiscoverySpecs)
	}

	if err := manager.refresh(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (um *UCloudManager) RegisterAsg(asg *Asg) {
	um.asgCache.Register(asg)
}

func (um *UCloudManager) GetAsgSize(asg *Asg) int {
	return len(asg.nodes)
}

func (um *UCloudManager) Refresh() error {
	if um.lastRefresh.Add(refreshInterval).After(time.Now()) {
		return nil
	}
	return um.refresh()
}

func (um *UCloudManager) refresh() error {
	if err := um.refreshAsgConfig(); err != nil {
		return err
	}
	if len(um.asgCache.registeredAsgs) > 0 {
		if err := um.refreshListNodes(); err != nil {
			return fmt.Errorf("Failed to refresh node list: %v", err)
		}
	}
	return nil
}

func (um *UCloudManager) refreshAsgConfig() error {
	err := um.fetchAutoAsgs()
	if err != nil {
		return fmt.Errorf("Failed to fetch auto autoscaling groups: %v", err)
	}
	um.lastRefresh = time.Now()
	klog.V(2).Infof("Refreshed ASG list, next refresh after %v", um.lastRefresh.Add(refreshInterval))
	return nil
}

func (um *UCloudManager) ScaleUpCluster(delta int, config uk8s.AutoScalingGroup) error {
	klog.V(1).Infof("Start to scale up cluster with %v node", delta)
	err := um.scaleUpClusterV2(delta, config)
	if err != nil {
		return err
	}

	klog.Infoln("Scale up cluster successfully")

	if err := um.refreshListNodes(); err != nil {
		klog.Errorf("Failed to refresh node list: %v", err)
	}
	return nil
}

func (um *UCloudManager) scaleUpClusterV2(delta int, config uk8s.AutoScalingGroup) error {
	params := um.client.NewScaleUpClusterV2Request()
	params.ClusterId = um.cfg.ClusterId
	params.CPU = config.CPU
	params.Mem = config.Mem
	params.Quantity = 1
	params.ChargeType = config.ChargeType
	if params.ChargeType == "" {
		params.ChargeType = "Dynamic"
	} else if params.ChargeType == "Month" {
		params.Quantity = 0
	}
	params.DataDiskType = config.DataDiskType
	params.BootDiskType = config.BootDiskType
	params.DataDiskSize = config.DataDiskSize
	params.Zone = config.Zone
	params.AsgId = config.Id
	params.MachineType = config.MachineType
	params.Labels = config.Labels
	params.Taints = config.Taints
	params.MaxPods = config.MaxPods
	params.Tag = config.Tag
	params.GPU = config.GPU
	params.GpuType = config.GpuType
	params.ImageId = config.ImageId
	params.UserData = config.UserData
	params.InitScript = config.InitScript
	params.MinmalCpuPlatform = config.MinmalCpuPlatform
	params.IsolationGroup = config.IsolationGroup
	if config.Password != "" {
		params.Password = config.Password
	} else {
		params.Password = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s.%s", uk8s.RandPassword(8), uk8s.RandPassword(8))))
	}
	for delta > 10 {
		delta = delta - 10
		params.Count = 10
		err := um.client.ScaleUpClusterV2(params)
		if err != nil {
			return fmt.Errorf("Failed to scale up cluster: %v", err)
		}
	}
	params.Count = delta
	err := um.client.ScaleUpClusterV2(params)
	if err != nil {
		return fmt.Errorf("Failed to scale up cluster: %v", err)
	}
	return nil
}

func (um *UCloudManager) ScaleDownCluster(nodesId []string) error {
	klog.V(1).Infof("Start to scale down cluster with %v nodes, %v", len(nodesId), nodesId)
	if len(nodesId) == 0 {
		klog.Warningf("you don't provide any nodes name to remove")
		return nil
	}
	defer func() {
		if err := um.refreshListNodes(); err != nil {
			klog.Errorf("Failed to refresh node list: %v", err)
		}
	}()

	for _, nodeId := range nodesId {
		params := um.client.NewScaleDownClusterRequest()
		params.ClusterId = um.cfg.ClusterId
		params.NodeId = nodeId
		err := um.client.ScaleDownCluster(params)
		if err != nil {
			return fmt.Errorf("Failed to scale down cluster: %v", err)
		}
	}
	return nil
}

func (um *UCloudManager) GetAsgNodes(asg *Asg) []Node {
	return asg.nodes
}

func (um *UCloudManager) GetAsgForInstance(providerId string) (*Asg, error) {
	return um.asgCache.FindForInstance(providerId)
}

func (um *UCloudManager) GetAsgTemplateNode(asg *Asg) (*asgTemplate, error) {
	if asg.Id() == "" || asg.config.CPU == 0 || asg.config.Mem == 0 {
		return nil, errors.New("Invalid node config")
	}
	return &asgTemplate{
		Id:          asg.Id(),
		CPU:         int64(asg.config.CPU),
		Memory:      int64(asg.config.Mem),
		GPU:         int64(asg.config.GPU),
		Zone:        asg.config.Zone,
		Region:      asg.RegionId,
		Labels:      asg.config.Labels,
		GpuType:     asg.config.GpuType,
		MachineType: asg.config.MachineType,
	}, nil
}

func (um *UCloudManager) buildNodeFromTemplate(asg *Asg, template *asgTemplate) (*apiv1.Node, error) {
	node := apiv1.Node{}
	nodeName := fmt.Sprintf("%s-asg-%d", asg.Id(), rand.Int63())
	node.ObjectMeta = metav1.ObjectMeta{
		Name:     nodeName,
		SelfLink: fmt.Sprintf("/api/v1/nodes/%s", nodeName),
		Labels:   map[string]string{},
	}
	node.Status = apiv1.NodeStatus{
		Capacity:    apiv1.ResourceList{},
		Allocatable: apiv1.ResourceList{},
	}
	var maxPods int64 = 110
	//if template.CPU*8 < 110 {
	//	maxNodes = template.CPU * 8
	//}
	if asg.config.MaxPods > 0 {
		maxPods = int64(asg.config.MaxPods)
	}
	node.Status.Capacity[apiv1.ResourcePods] = *resource.NewQuantity(maxPods, resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceCPU] = *resource.NewQuantity(template.CPU, resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceMemory] = *resource.NewQuantity(template.Memory*MiB, resource.DecimalSI)
	node.Status.Capacity[ResourceGPU] = *resource.NewQuantity(template.GPU, resource.DecimalSI)

	node.Status.Allocatable[apiv1.ResourcePods] = *resource.NewQuantity(maxPods, resource.DecimalSI)
	node.Status.Allocatable[apiv1.ResourceCPU] = *resource.NewMilliQuantity(template.CPU*1000-200, resource.DecimalSI)
	physicalMemory := template.Memory * MiB
	allocatableMem := int64(float64(physicalMemory-CalculateKernelReserved(physicalMemory))*KubeletEvictionHardMemoryRatio - 1000*MiB)
	node.Status.Allocatable[apiv1.ResourceMemory] = *resource.NewQuantity(allocatableMem, resource.DecimalSI)
	node.Status.Allocatable[ResourceGPU] = *resource.NewQuantity(template.GPU, resource.DecimalSI)
	node.Labels = cloudprovider.JoinStringMaps(node.Labels, extractLabelsFromAsg(template.Labels))
	node.Labels = cloudprovider.JoinStringMaps(node.Labels, buildGenericLabels(template))

	node.Status.Conditions = cloudprovider.BuildReadyConditions()
	return &node, nil
}

func buildGenericLabels(template *asgTemplate) map[string]string {
	result := make(map[string]string)
	result[kubeletapis.LabelArch] = cloudprovider.DefaultArch
	result[kubeletapis.LabelOS] = cloudprovider.DefaultOS

	result[apiv1.LabelZoneRegion] = template.Region
	result[apiv1.LabelZoneFailureDomain] = template.Zone
	result["beta.kubernetes.io/arch"] = cloudprovider.DefaultArch
	result["beta.kubernetes.io/os"] = cloudprovider.DefaultOS
	result["node.uk8s.ucloud.cn/machine_type"] = template.MachineType
	result["role.node.kubernetes.io/k8s-node"] = "true"
	result["topology.kubernetes.io/region"] = template.Region
	result["topology.kubernetes.io/zone"] = template.Zone
	result["topology.udisk.csi.ucloud.cn/region"] = template.Region
	result["topology.udisk.csi.ucloud.cn/zone"] = template.Zone

	if strings.ToUpper(template.MachineType) == "G" {
		result["accelerator"] = fmt.Sprintf("nvidia-tesla-%s", strings.ToLower(template.GpuType))
	}

	return result
}

func extractLabelsFromAsg(labelsStr string) map[string]string {
	result := make(map[string]string)
	labels := strings.SplitN(labelsStr, ",", 6)
	for _, label := range labels[:5] {
		kv := strings.Split(label, "=")
		if len(kv) != 2 {
			continue
		}
		result[kv[0]] = kv[1]
	}
	return result
}

type asgTemplate struct {
	Id          string
	Region      string
	Zone        string
	CPU         int64
	Memory      int64
	GPU         int64
	Labels      string
	GpuType     string
	MachineType string
}

func (m *UCloudManager) fetchAutoAsgs() error {
	params := m.client.DescribeUK8SAutoscalerGroupRequest()
	params.ClusterId = m.cfg.ClusterId
	autoscalerGroup, err := m.client.DescribeUK8SAutoscalerGroup(params)
	if err != nil {
		klog.Errorf("describe autoscalergroup error: %v", err)
		return err
	}
	if autoscalerGroup.Autoscaler.Enabled != 1 {
		m.asgCache.registeredAsgs = make(map[string]*Asg)
		return errors.New("autoscaler has been disabled, flush all existing auto scaling groups")
	}
	var lastAsgsName, newAsgsName []string
	//unregister all asgs in case of the same asg_id has different params
	for _, asg := range m.asgCache.registeredAsgs {
		m.asgCache.Unregister(asg)
		lastAsgsName = append(lastAsgsName, asg.Id())
	}
	klog.V(5).Infof("Unregister %v Asgs [%v]", len(lastAsgsName), strings.Join(lastAsgsName, ","))
	if len(m.asgCache.registeredAsgs) > 0 {
		klog.V(1).Infoln("tidy data, flush all")
		m.asgCache.registeredAsgs = make(map[string]*Asg)
	}
	//register new asgs
	for _, v := range autoscalerGroup.AutoScalingGroup {
		asg, err := m.buildAsgFromUCloud(v)
		if err != nil {
			return fmt.Errorf("cannot autodiscover managed instance groups: %v", err)
		}
		m.RegisterAsg(asg)
		newAsgsName = append(newAsgsName, asg.Id())
	}
	klog.V(1).Infof("Register %v Asgs [%v]", len(newAsgsName), strings.Join(newAsgsName, ","))
	return nil
}

func (m *UCloudManager) buildAsgFromUCloud(autoScalingGroup *uk8s.AutoScalingGroup) (*Asg, error) {
	spec := dynamic.NodeGroupSpec{
		Name:               autoScalingGroup.Id,
		MinSize:            autoScalingGroup.Min,
		MaxSize:            autoScalingGroup.Max,
		SupportScaleToZero: scaleToZeroSupported,
	}

	if verr := spec.Validate(); verr != nil {
		return nil, fmt.Errorf("failed to create node group spec: %v", verr)
	}
	return buildAsg(m, autoScalingGroup, ""), nil
}

func (m *UCloudManager) refreshListNodes() error {
	params := m.client.NewListNodeRequest()
	params.ClusterId = m.cfg.ClusterId
	nodeInfos, err := m.client.ListNode(params)
	if err != nil {
		return err
	}

	m.asgCache.cacheMutex.Lock()
	defer m.asgCache.cacheMutex.Unlock()

	for k := range m.asgCache.registeredAsgs {
		nodes := make([]Node, 0)
		//errNodes := make([]string, 0)
		for _, v := range nodeInfos {
			if m.asgCache.registeredAsgs[k].Id() != v.AsgId {
				continue
			}
			if v.NodeStatus == "Deleting" || v.NodeStatus == "Deleted" || v.NodeStatus == "ToBeDeleted" {
				continue
			}
			nodes = append(nodes, Node{ProviderId: "UCloud://" + v.Zone + "//" + v.NodeId, Name: v.InstanceName})
		}
		m.asgCache.registeredAsgs[k].nodes = nodes
	}

	return nil
}

func getAutoDiscoverySpecs(labels []cloudprovider.LabelAutoDiscoveryConfig) {
	if len(labels) == 0 {
		return
	}
	for _, label := range labels {
		if label.Selector == nil {
			continue
		}
		if asgRefreshIntervalLabel, ok := label.Selector["asg_refresh_interval"]; ok {
			asgRefreshInterval, _ := strconv.Atoi(asgRefreshIntervalLabel)
			if asgRefreshInterval < 30 {
				klog.V(1).Infoln("Find asg_refresh_interval < 30 seconds, ignore it.")
				continue
			}
			refreshInterval = time.Second * time.Duration(asgRefreshInterval)
			klog.V(1).Infof("asg_refresh_interval has set to %v seconds, [Tips]: asg_refresh_interval = scan_interval if asg_refresh_interval < scan interval", refreshInterval)
		}
	}
}

func (m *UCloudManager) refreshStsToken() {
	cfg := &cloudConfig{}
	cfg.loadConfigFromSts()
	timer := time.NewTimer(cfg.Credential.Expires.Add(1 * time.Second).Sub(time.Now()))
	go func() {
		for {
			select {
			case <-timer.C:
				cfg = &cloudConfig{}
				cfg.loadConfigFromSts()
				client := uk8s.NewClient(cfg.Config, cfg.Credential)
				m.cfg = cfg
				m.client = client
				timer.Reset(cfg.Credential.Expires.Add(1 * time.Second).Sub(time.Now()))
			}
		}
	}()
}
