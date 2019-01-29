package machineset

import (
	"context"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type Actuator interface {
	Delete(context.Context, *clusterv1.MachineSet) error
	Resize(context.Context, *clusterv1.MachineSet) error
	ListMachines(context.Context, *clusterv1.MachineSet) ([]string, error)
}
