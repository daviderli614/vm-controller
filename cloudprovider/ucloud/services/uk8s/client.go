package uk8s

import (
	"github.com/ucloud/ucloud-sdk-go/services/uk8s"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"
)

type UK8SClient struct {
	client *uk8s.UK8SClient
}

func NewClient(config *ucloud.Config, credential *auth.Credential) *UK8SClient {
	client := uk8s.NewClient(config, credential)
	return &UK8SClient{
		client: client,
	}
}
