package uk8s

import (
	"errors"
	"time"

	"fmt"

	"github.com/ucloud/ucloud-sdk-go/ucloud/response"

	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
)

const (
	DescribeUK8SAutoscalerGroupAction = "DescribeUK8SAutoscalerGroup"
)

type DescribeUK8SAutoscalerGroupRequest struct {
	request.CommonBase
	ClusterId string
}

type DescribeUK8SAutoscalerGroupResponse struct {
	response.CommonBase
	AutoscalerGroup
}

type AutoscalerGroup struct {
	Autoscaler       *Autoscaler
	AutoScalingGroup []*AutoScalingGroup
}

type Autoscaler struct {
	Enabled int
}

type AutoScalingGroup struct {
	Id             string
	Name           string
	CPU            int
	Mem            int
	DataDiskType   string
	BootDiskType   string
	DataDiskSize   int
	Zone           string
	Min            int
	Max            int
	CurrentNodeNum int
	CreateTime     int64
	UpdateTime     int64
	Password       string
	//新增
	MinmalCpuPlatform string
	MaxPods           int
	MachineType       string
	GPU               int
	GpuType           string
	Labels            string
	Taints            string
	ImageId           string
	UserData          string
	InitScript        string
	ChargeType        string
	IsolationGroup    string
	Tag               string

	//兼容旧伸缩组
	NodeCPU          int
	NodeMem          int
	NodeUHostType    string
	NodeDataDiskType string
	NodeBootDiskType string
	NodeDataDiskSize int
}

func (c *UK8SClient) DescribeUK8SAutoscalerGroupRequest() *DescribeUK8SAutoscalerGroupRequest {
	req := &DescribeUK8SAutoscalerGroupRequest{}
	c.client.SetupRequest(req)
	req.SetRetryable(false)
	return req
}

func (c *UK8SClient) DescribeUK8SAutoscalerGroup(params *DescribeUK8SAutoscalerGroupRequest) (*AutoscalerGroup, error) {
	var res DescribeUK8SAutoscalerGroupResponse
	var err error
	defer func() {
		if err != nil && res.RetCode == 0 {
			res.RetCode = -1
		}
		latency := time.Since(params.GetRequestTime()).Seconds() * 1000
		go ReportCAMetrics(params.ClusterId, params.GetAction(), params.GetRegion(), res.GetRequestUUID(), float64(res.RetCode), latency)
	}()
	if err := checkDescribeUK8SAutoscalerGroupParams(params); err != nil {
		return nil, err
	}
	if err = c.client.InvokeAction(DescribeUK8SAutoscalerGroupAction, params, &res); err != nil {
		return nil, err
	}
	if res.RetCode != 0 {
		return nil, fmt.Errorf("describe AutoscalerGroup error; ret_code %v; %v", res.RetCode, res.Message)
	}
	return &AutoscalerGroup{res.Autoscaler, res.AutoScalingGroup}, nil
}

func checkDescribeUK8SAutoscalerGroupParams(params *DescribeUK8SAutoscalerGroupRequest) error {
	if params.GetRegion() == "" {
		return errors.New("Region is required")
	}
	if params.ClusterId == "" {
		return errors.New("ClusterId is required")
	}
	return nil
}
