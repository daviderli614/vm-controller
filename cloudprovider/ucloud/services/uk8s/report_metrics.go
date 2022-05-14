package uk8s

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/klog"

	"github.com/google/uuid"
)

const (
	TelemetryMetricCAAPIStatus  = "uk8s.ca.uapistatus"
	TelemetryMetricCAAPILatency = "uk8s.ca.uapilatency"

	TelemetryAPIEndpoint = "http://umon.transfer.service.ucloud.cn/api/update"
	TelemetryToken       = "07d985806dd54f109329b074ce876e9d"
)

type MetricValues struct {
	Metric     string  `json:"metric"`
	Endpoint   string  `json:"endpoint"`
	Tags       string  `json:"tags"`
	Value      float64 `json:"value"`
	Timestamp  int64   `json:"timestamp"`
	MetricType string  `json:"metrictype"`
}

type Report struct {
	SessionID    string         `json:"sessionid"`
	MetricValues []MetricValues `json:"metricvalues"`
	Token        string         `json:"token"`
}

type ReportResponse struct {
	Total     int64
	Invalid   int64
	Message   string
	SessionId string
}

func TelemetryReport(rp Report) error {
	content, err := json.Marshal(rp)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", TelemetryAPIEndpoint, bytes.NewBuffer(content))
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req.Header.Set("X-SessionId", rp.SessionID)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telemetry report[request id:%v] response status %v", rp.SessionID, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var rr ReportResponse
	err = json.Unmarshal(body, &rr)
	if err != nil {
		return err
	}
	if rr.Invalid > 0 {
		return errors.New(rr.Message)
	}
	return nil
}

func buildBaseTags(action, region string) string {
	var tags []string
	if action != "" {
		tags = append(tags, fmt.Sprintf("Action=%v", action))
	}
	if region != "" {
		tags = append(tags, fmt.Sprintf("Region=%v", region))
	}
	if version, ok := os.LookupEnv("VERSION"); ok {
		tags = append(tags, fmt.Sprintf("Version=%v", version))
	}
	return strings.Join(tags, ",")
}

func ReportCAMetrics(clusterId string, action, region, reqUuid string, code, latency float64) {
	if reportErr := reportCAMetrics(clusterId, action, region, reqUuid, code, latency); reportErr != nil {
		klog.Warningf("report ca metrics error: %v", reportErr)
	}
}

func reportCAMetrics(clusterId string, action, region, reqUuid string, code, latency float64) error {
	var mvs []MetricValues
	sessionID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	statusMV := MetricValues{
		Metric:     TelemetryMetricCAAPIStatus,
		Endpoint:   clusterId,
		MetricType: "gauge",
		Value:      code,
		Tags:       buildBaseTags(action, region),
		Timestamp:  time.Now().Unix(),
	}
	latencyMV := MetricValues{
		Metric:     TelemetryMetricCAAPILatency,
		Endpoint:   clusterId,
		MetricType: "gauge",
		Value:      latency,
		Tags:       buildBaseTags(action, region),
		Timestamp:  time.Now().Unix(),
	}
	mvs = append(mvs, statusMV, latencyMV)
	r := Report{
		SessionID:    sessionID.String(),
		Token:        TelemetryToken,
		MetricValues: mvs,
	}
	return TelemetryReport(r)
}
