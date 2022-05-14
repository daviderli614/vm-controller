package uk8s

import (
	"errors"
	"time"

	"github.com/ucloud/ucloud-sdk-go/ucloud/response"

	"fmt"

	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
)

type ScaleUpClusterV2Request struct {
	request.CommonBase
	ClusterId         string
	Zone              string
	Password          string
	ChargeType        string
	MachineType       string
	MinmalCpuPlatform string
	GpuType           string
	GPU               int
	CPU               int
	Mem               int
	BootDiskType      string
	DataDiskType      string
	DataDiskSize      int
	Labels            string
	Taints            string
	MaxPods           int
	Count             int
	AsgId             string
	ImageId           string
	UserData          string
	InitScript        string
	IsolationGroup    string
	Tag               string
	//特有
	Quantity int
}

type ScaleUpClusterV2Response struct {
	response.CommonBase
}

func (c *UK8SClient) NewScaleUpClusterV2Request() *ScaleUpClusterV2Request {
	req := &ScaleUpClusterV2Request{}
	c.client.SetupRequest(req)
	req.SetRetryable(false)
	return req
}

func (c *UK8SClient) ScaleUpClusterV2(params *ScaleUpClusterV2Request) (err error) {
	var res ScaleUpClusterV2Response
	defer func() {
		if err != nil && res.RetCode == 0 {
			res.RetCode = -1
		}
		latency := time.Since(params.GetRequestTime()).Seconds() * 1000
		go ReportCAMetrics(params.ClusterId, params.GetAction(), params.GetRegion(), res.GetRequestUUID(), float64(res.RetCode), latency)
	}()
	if err := checkScaleUpParamsV2(params); err != nil {
		return err
	}

	if err = c.client.InvokeAction("AddUK8SUHostNode", params, &res); err != nil {
		return err
	}

	if res.RetCode != 0 {
		return fmt.Errorf("scale up node error; ret_code %v; %v", res.RetCode, res.Message)
	}
	return nil
}

func checkScaleUpParamsV2(params *ScaleUpClusterV2Request) error {
	if params.GetRegion() == "" {
		return errors.New("Region is required")
	}
	if params.ClusterId == "" {
		return errors.New("ClusterId is required")
	}
	if params.CPU == 0 {
		return errors.New("CPU is required")
	}
	if params.Mem == 0 {
		return errors.New("Mem is required")
	}
	if params.BootDiskType == "" {
		return errors.New("BootDiskType is required")
	}
	if params.MachineType == "" {
		return errors.New("MachineType is required")
	}
	if params.Count < 1 {
		return errors.New("Count is required")
	}
	if params.Password == "" {
		return errors.New("Password is required")
	}
	if params.AsgId == "" {
		return errors.New("AsgId is required")
	}
	if params.ChargeType != "Year" && params.ChargeType != "Month" && params.ChargeType != "Dynamic" {
		return errors.New("invalid ChargeType")
	}
	return nil
}
