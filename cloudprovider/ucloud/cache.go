/*
Copyright 2018 The Kubernetes Authors.

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
	"sync"
)

type asgCache struct {
	//registeredAsgs []*Asg
	registeredAsgs map[string]*Asg
	cacheMutex     sync.Mutex
}

func newAsgCache() *asgCache {
	registry := &asgCache{
		registeredAsgs: make(map[string]*Asg),
		//instanceToAsg:            make(map[string]*Asg),
		//instancesNotInManagedAsg: make(map[string]struct{}),
	}
	return registry
}

// Register registers asg in UCloud Manager.
func (m *asgCache) Register(asg *Asg) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	if m.registeredAsgs == nil {
		m.registeredAsgs = make(map[string]*Asg)
	}
	m.registeredAsgs[asg.Id()] = asg
}

func (m *asgCache) Unregister(asg *Asg) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	if m.registeredAsgs == nil {
		return
	}
	delete(m.registeredAsgs, asg.Id())
}

// FindForInstance returns AsgConfig of the given Instance
func (m *asgCache) FindForInstance(providerId string) (*Asg, error) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	for _, asg := range m.registeredAsgs {
		for _, v := range asg.nodes {
			if v.ProviderId == providerId {
				return asg, nil
			}
		}
	}
	return nil, nil
}
