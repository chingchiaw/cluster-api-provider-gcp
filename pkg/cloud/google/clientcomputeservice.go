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

package google

import (
	"context"

	compute "google.golang.org/api/compute/v1"
)

type GCEClientComputeService interface {
	ImagesGet(ctx context.Context, project, image string) (*compute.Image, error)
	ImagesGetFromFamily(ctx context.Context, project, family string) (*compute.Image, error)

	InstancesDelete(ctx context.Context, project, zone, targetInstance string) (*compute.Operation, error)
	InstancesGet(ctx context.Context, project, zone string, instance string) (*compute.Instance, error)
	InstancesInsert(ctx context.Context, project, zone string, instance *compute.Instance) (*compute.Operation, error)
	InstancesInsertFromTemplate(ctx context.Context, project, zone, instanceName, instanceTemplate string) (*compute.Operation, error)

	ZoneOperationsGet(ctx context.Context, project, zone, operation string) (*compute.Operation, error)
	GlobalOperationsGet(ctx context.Context, project, operation string) (*compute.Operation, error)
	WaitForOperation(ctx context.Context, project string, op *compute.Operation) error

	FirewallsGet(ctx context.Context, project string) (*compute.FirewallList, error)
	FirewallsInsert(ctx context.Context, project string, firewallRule *compute.Firewall) (*compute.Operation, error)
	FirewallsDelete(ctx context.Context, project, name string) (*compute.Operation, error)

	InstanceGroupManagersGet(ctx context.Context, project, zone, igm string) (*compute.InstanceGroupManager, error)
	InstanceGroupManagersInsert(ctx context.Context, project, zone string, igm *compute.InstanceGroupManager) (*compute.Operation, error)
	InstanceGroupManagersDelete(ctx context.Context, project, zone, igm string) (*compute.Operation, error)
	InstanceGroupManagersResize(ctx context.Context, project, zone, igm string, size int64) (*compute.Operation, error)
	InstanceGroupManagersListInstances(ctx context.Context, project, zone, igm string) (*compute.InstanceGroupManagersListManagedInstancesResponse, error)
}
