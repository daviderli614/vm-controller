package uk8s

import (
	"errors"
	"fmt"
	"time"

	"github.com/ucloud/ucloud-sdk-go/ucloud/request"
	"github.com/ucloud/ucloud-sdk-go/ucloud/response"
)

type ScaleDownClusterRequest struct {
	request.CommonBase
	ClusterId string
	NodeId    string
}

type ScaleDownClusterResponse struct {
	response.CommonBase
}

func (c *UK8SClient) NewScaleDownClusterRequest() *ScaleDownClusterRequest {
	req := &ScaleDownClusterRequest{}
	c.client.SetupRequest(req)
	req.SetRetryable(false)
	return req
}

func (c *UK8SClient) ScaleDownCluster(params *ScaleDownClusterRequest) (err error) {
	var res ScaleDownClusterResponse
	defer func() {
		if err != nil && res.RetCode == 0 {
			res.RetCode = -1
		}
		latency := time.Since(params.GetRequestTime()).Seconds() * 1000
		go ReportCAMetrics(params.ClusterId, params.GetAction(), params.GetRegion(), res.GetRequestUUID(), float64(res.RetCode), latency)
	}()
	if err := checkScaleDownParams(params); err != nil {
		return err
	}
	if err = c.client.InvokeAction("DelUK8SClusterNodeV2", params, &res); err != nil {
		return err
	}
	if res.RetCode != 0 {
		return fmt.Errorf("scale down node error; ret_code %v; %v", res.RetCode, res.Message)
	}
	return nil
}

func checkScaleDownParams(params *ScaleDownClusterRequest) error {
	if params.GetRegion() == "" {
		return errors.New("Region is required")
	}
	if params.ClusterId == "" {
		return errors.New("ClusterId is required")
	}
	if params.NodeId == "" {
		return errors.New("NodeId is required")
	}
	return nil
}
