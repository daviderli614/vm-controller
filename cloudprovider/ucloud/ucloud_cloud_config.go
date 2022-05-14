package ucloud

import (
	"os"

	"github.com/ucloud/ucloud-sdk-go/external"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"k8s.io/klog"
)

const (
	accessPubKey  = "UCLOUD_ACCESS_PUBKEY"
	accessPriKey  = "UCLOUD_ACCESS_PRIKEY"
	regionId      = "UCLOUD_REGION_ID"
	clusterId     = "UCLOUD_UK8S_CLUSTER_ID"
	projectId     = "UCLOUD_PROJECT_ID"
	CharacterName = "Uk8sServiceCharacter"
)

type cloudConfig struct {
	ClusterId string
	*auth.Credential
	*ucloud.Config
}

func (cc *cloudConfig) isValid() bool {
	if cc.Region == "" || cc.ClusterId == "" || cc.ProjectId == "" {
		klog.V(1).Infof("Failed to get UCLOUD_REGION_ID:%s,UCLOUD_PROJECT_ID:%v,UCLOUD_UK8S_CLUSTER_ID:%v from cloudConfig and Env\n", cc.Region, cc.ProjectId, cc.ClusterId)
		return false
	}
	return true
}

func (cc *cloudConfig) loadConfigFromSts() {
	c, err := external.LoadSTSConfig(external.AssumeRoleRequest{RoleName: CharacterName})
	if err != nil {
		klog.Fatalf("Cannot load api config from STS service, %v\n", err)
	}
	cc.Credential = c.Credential()
	cc.Config = c.Config()
	if cc.ClusterId == "" {
		cc.ClusterId = os.Getenv(clusterId)
	}
}
