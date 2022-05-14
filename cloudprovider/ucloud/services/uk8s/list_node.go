package uk8s

import (
	"errors"
	"time"

	"github.com/ucloud/ucloud-sdk-go/ucloud/response"

	"fmt"

	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
)

const (
	ListNodeAction = "ListUK8SClusterNodeV2"
)

type ListNodeRequest struct {
	request.CommonBase
	ClusterId string
}

type NodeInfo struct {
	Zone          string
	NodeId        string
	NodeRole      string
	NodeStatus    string
	InstanceType  string
	InstanceName  string
	InstanceId    string
	MachineType   string
	OsType        string
	OsName        string
	CPU           int
	Memory        int
	GPU           int
	CreateTime    int
	ExpireTime    int
	AsgId         string
	Unschedulable bool
}

type ListNodeResponse struct {
	response.CommonBase
	TotalCount int
	NodeSet    []*NodeInfo
}

func (c *UK8SClient) NewListNodeRequest() *ListNodeRequest {
	req := &ListNodeRequest{}
	c.client.SetupRequest(req)
	req.SetRetryable(false)
	return req
}

func (c *UK8SClient) ListNode(params *ListNodeRequest) ([]*NodeInfo, error) {
	var res ListNodeResponse
	var err error
	defer func() {
		if err != nil && res.RetCode == 0 {
			res.RetCode = -1
		}
		latency := time.Since(params.GetRequestTime()).Seconds() * 1000
		go ReportCAMetrics(params.ClusterId, params.GetAction(), params.GetRegion(), res.GetRequestUUID(), float64(res.RetCode), latency)
	}()
	if err := checkListNodeParams(params); err != nil {
		return nil, err
	}
	if err = c.client.InvokeAction(ListNodeAction, params, &res); err != nil {
		return nil, err
	}

	if res.RetCode != 0 {
		return nil, fmt.Errorf("list node error; ret_code %v; %v", res.RetCode, res.Message)
	}
	return res.NodeSet, nil
}

func checkListNodeParams(params *ListNodeRequest) error {
	if params.GetRegion() == "" {
		return errors.New("Region is required")
	}
	if params.ClusterId == "" {
		return errors.New("ClusterId is required")
	}
	return nil
}
