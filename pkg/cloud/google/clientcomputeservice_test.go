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

package google_test

import (
	"context"

	compute "google.golang.org/api/compute/v1"
)

type GCEClientComputeServiceMock struct {
	mockImagesGet           func(ctx context.Context, project string, image string) (*compute.Image, error)
	mockImagesGetFromFamily func(ctx context.Context, project string, family string) (*compute.Image, error)
	mockInstancesDelete     func(ctx context.Context, project string, zone string, targetInstance string) (*compute.Operation, error)
	mockInstancesGet        func(ctx context.Context, project string, zone string, instance string) (*compute.Instance, error)
	mockInstancesInsert     func(ctx context.Context, project string, zone string, instance *compute.Instance) (*compute.Operation, error)
	mockZoneOperationsGet   func(ctx context.Context, project string, zone string, operation string) (*compute.Operation, error)
	mockGlobalOperationsGet func(ctx context.Context, project string, operation string) (*compute.Operation, error)
	mockFirewallsGet        func(ctx context.Context, project string) (*compute.FirewallList, error)
	mockFirewallsInsert     func(ctx context.Context, project string, firewallRule *compute.Firewall) (*compute.Operation, error)
	mockFirewallsDelete     func(ctx context.Context, project string, name string) (*compute.Operation, error)
	mockWaitForOperation    func(ctx context.Context, project string, op *compute.Operation) error
}

func (c *GCEClientComputeServiceMock) ImagesGet(ctx context.Context, project string, image string) (*compute.Image, error) {
	if c.mockImagesGet == nil {
		return nil, nil
	}
	return c.mockImagesGet(ctx, project, image)
}

func (c *GCEClientComputeServiceMock) ImagesGetFromFamily(ctx context.Context, project string, family string) (*compute.Image, error) {
	if c.mockImagesGetFromFamily == nil {
		return nil, nil
	}
	return c.mockImagesGetFromFamily(ctx, project, family)
}

func (c *GCEClientComputeServiceMock) InstancesDelete(ctx context.Context, project string, zone string, targetInstance string) (*compute.Operation, error) {
	if c.mockInstancesDelete == nil {
		return nil, nil
	}
	return c.mockInstancesDelete(ctx, project, zone, targetInstance)
}

func (c *GCEClientComputeServiceMock) InstancesGet(ctx context.Context, project string, zone string, instance string) (*compute.Instance, error) {
	if c.mockInstancesGet == nil {
		return nil, nil
	}
	return c.mockInstancesGet(ctx, project, zone, instance)
}

func (c *GCEClientComputeServiceMock) InstancesInsert(ctx context.Context, project string, zone string, instance *compute.Instance) (*compute.Operation, error) {
	if c.mockInstancesInsert == nil {
		return nil, nil
	}
	return c.mockInstancesInsert(ctx, project, zone, instance)
}

func (c *GCEClientComputeServiceMock) InstancesInsertFromTemplate(ctx context.Context, project, zone, instanceName, instanceTemplate string) (*compute.Operation, error) {
	panic("not implemented")
}

func (c *GCEClientComputeServiceMock) ZoneOperationsGet(ctx context.Context, project string, zone string, operation string) (*compute.Operation, error) {
	if c.mockZoneOperationsGet == nil {
		return nil, nil
	}
	return c.mockZoneOperationsGet(ctx, project, zone, operation)
}

func (c *GCEClientComputeServiceMock) GlobalOperationsGet(ctx context.Context, project string, operation string) (*compute.Operation, error) {
	if c.mockGlobalOperationsGet == nil {
		return nil, nil
	}
	return c.mockGlobalOperationsGet(ctx, project, operation)
}

func (c *GCEClientComputeServiceMock) FirewallsGet(ctx context.Context, project string) (*compute.FirewallList, error) {
	if c.mockFirewallsGet == nil {
		return nil, nil
	}
	return c.mockFirewallsGet(ctx, project)
}

func (c *GCEClientComputeServiceMock) FirewallsInsert(ctx context.Context, project string, firewallRule *compute.Firewall) (*compute.Operation, error) {
	if c.mockFirewallsInsert == nil {
		return nil, nil
	}
	return c.mockFirewallsInsert(ctx, project, firewallRule)
}

func (c *GCEClientComputeServiceMock) FirewallsDelete(ctx context.Context, project string, name string) (*compute.Operation, error) {
	if c.mockFirewallsDelete == nil {
		return nil, nil
	}
	return c.mockFirewallsDelete(ctx, project, name)
}

func (c *GCEClientComputeServiceMock) WaitForOperation(ctx context.Context, project string, op *compute.Operation) error {
	if c.mockWaitForOperation == nil {
		return nil
	}
	return c.mockWaitForOperation(ctx, project, op)
}
