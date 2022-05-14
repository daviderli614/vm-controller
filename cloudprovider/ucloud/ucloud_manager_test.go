package ucloud

import "testing"

func TestBuildNodeFromTemplate(t *testing.T) {
	asg := new(Asg)
	template := new(asgTemplate)
	template.CPU = 12
	template.Memory = 64 * 1024
	asg.config.CPU = 12
	asg.config.Mem = 64 * 1024
	um := new(UCloudManager)
	node, _ := um.buildNodeFromTemplate(asg, template)
	t.Logf("capacity cpu %v, memory %v", node.Status.Capacity.Cpu().String(), node.Status.Capacity.Memory().String())
	t.Logf("allocate cpu %v, memory %v", node.Status.Allocatable.Cpu().String(), node.Status.Allocatable.Memory().String())
}
