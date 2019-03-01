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

package machineset

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	MachineSetFinalizer     = "machineset.cluster.k8s.io"
	MIGMachineSetAnnotation = "mig-name"
)

var controllerKind = clusterv1.SchemeGroupVersion.WithKind("MachineSet")

// stateConfirmationTimeout is the amount of time allowed to wait for desired state.
var stateConfirmationTimeout = 10 * time.Second

// stateConfirmationInterval is the amount of time between polling for the desired state.
// The polling is against a local memory cache.
var stateConfirmationInterval = 100 * time.Millisecond

// Add creates a new MachineSet Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.

func AddWithActuator(mgr manager.Manager, actuator Actuator) error {
	r := newReconciler(mgr, actuator)
	return add(mgr, r, r.MachineSetToMachines)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, actuator Actuator) *ReconcileMachineSet {
	return &ReconcileMachineSet{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		actuator: actuator,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, mapFn handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New("machineset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MachineSet
	err = c.Watch(&source.Kind{Type: &clusterv1.MachineSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Map Machine changes to MachineSets using ControllerRef
	err = c.Watch(
		&source.Kind{Type: &clusterv1.Machine{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &clusterv1.MachineSet{}},
	)
	if err != nil {
		return err
	}

	// Map Machine changes to MachineSets by machining labels
	err = c.Watch(
		&source.Kind{Type: &clusterv1.Machine{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMachineSet{}

// ReconcileMachineSet reconciles a MachineSet object
type ReconcileMachineSet struct {
	client.Client
	scheme   *runtime.Scheme
	actuator Actuator
}

func (r *ReconcileMachineSet) MachineSetToMachines(o handler.MapObject) []reconcile.Request {
	result := []reconcile.Request{}
	m := &clusterv1.Machine{}
	key := client.ObjectKey{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()}
	err := r.Client.Get(context.Background(), key, m)
	if err != nil {
		glog.Errorf("Unable to retrieve Machine %v from store: %v", key, err)
		return nil
	}

	for _, ref := range m.ObjectMeta.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return result
		}
	}

	mss := r.getMachineSetsForMachine(m)
	if len(mss) == 0 {
		glog.V(4).Infof("Found no machine set for machine: %v", m.Name)
		return nil
	}

	for _, ms := range mss {
		result = append(result, reconcile.Request{
			NamespacedName: client.ObjectKey{Namespace: ms.Namespace, Name: ms.Name}})
	}

	return result
}

// Reconcile reads that state of the cluster for a MachineSet object and makes changes based on the state read
// and what is in the MachineSet.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=cluster.k8s.io,resources=machinesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.k8s.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileMachineSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()

	// Fetch the MachineSet instance
	machineSet := &clusterv1.MachineSet{}
	err := r.Get(ctx, request.NamespacedName, machineSet)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	name := machineSet.Name
	glog.V(4).Infof("Reconcile machineset %v", name)

	// If object hasn't been deleted and doesn't have a finalizer, add one
	// Add a finalizer to newly created objects.
	if machineSet.ObjectMeta.DeletionTimestamp.IsZero() &&
		!util.Contains(machineSet.ObjectMeta.Finalizers, MachineSetFinalizer) {
		machineSet.Finalizers = append(machineSet.Finalizers, MachineSetFinalizer)
		if err = r.Client.Update(ctx, machineSet); err != nil {
			glog.Infof("failed to add finalizer to MachineSet object %v due to error %v.", name, err)
			return reconcile.Result{}, err
		}
	}

	if !machineSet.ObjectMeta.DeletionTimestamp.IsZero() {
		// no-op if finalizer has been removed.
		if !util.Contains(machineSet.ObjectMeta.Finalizers, MachineSetFinalizer) {
			glog.Infof("reconciling MachineSet object %v causes a no-op as there is no finalizer.", name)
			return reconcile.Result{}, nil
		}

		glog.Infof("reconciling MachineSet object %v triggers delete.", name)
		if err := r.actuator.Delete(ctx, machineSet); err != nil {
			glog.Errorf("Error deleting MachineSet object %v; %v", name, err)
			return reconcile.Result{}, err
		}

		// Remove finalizer on successful deletion.
		glog.Infof("MachineSet object %v deletion successful, removing finalizer.", name)
		machineSet.ObjectMeta.Finalizers = util.Filter(machineSet.ObjectMeta.Finalizers, MachineSetFinalizer)
		if err := r.Client.Update(context.Background(), machineSet); err != nil {
			glog.Errorf("Error removing finalizer from MachineSet object %v; %v", name, err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	allMachines := &clusterv1.MachineList{}
	lo := client.InNamespace(machineSet.Namespace).MatchingLabels(machineSet.Spec.Selector.MatchLabels)
	err = r.Client.List(context.Background(), lo, allMachines)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list machines, %v", err)
	}

	// Filter out irrelevant machines (deleting/mismatch labels) and claim orphaned machines.
	var filteredMachines []*clusterv1.Machine
	for idx := range allMachines.Items {
		machine := &allMachines.Items[idx]
		if shouldExcludeMachine(machineSet, machine) {
			continue
		}
		// Attempt to adopt machine if it meets previous conditions and it has no controller ref.
		if metav1.GetControllerOf(machine) == nil {
			continue
		}
		filteredMachines = append(filteredMachines, machine)
	}

	syncErr := r.syncReplicas(ctx, machineSet, filteredMachines)
	if syncErr != nil {
		glog.Error(syncErr)
	}

	ms := machineSet.DeepCopy()
	newStatus := r.calculateStatus(ms, filteredMachines)

	// Always updates status as machines come up or die.
	updatedMS, err := updateMachineSetStatus(r.Client, ms, newStatus)
	if err != nil {
		if syncErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to sync machines. %v. failed to update machine set status. %v", syncErr, err)
		}
		return reconcile.Result{}, fmt.Errorf("failed to update machine set status. %v", err)
	}

	var replicas int32
	if updatedMS.Spec.Replicas != nil {
		replicas = *updatedMS.Spec.Replicas
	}

	// Resync the MachineSet after MinReadySeconds as a last line of defense to guard against clock-skew.
	// Clock-skew is an issue as it may impact whether an available replica is counted as a ready replica.
	// A replica is available if the amount of time since last transition exceeds MinReadySeconds.
	// If there was a clock skew, checking whether the amount of time since last transition to ready state
	// exceeds MinReadySeconds could be incorrect.
	// To avoid an available replica stuck in the ready state, we force a reconcile after MinReadySeconds,
	// at which point it should confirm any available replica to be available.
	if syncErr == nil && updatedMS.Spec.MinReadySeconds > 0 &&
		updatedMS.Status.ReadyReplicas == replicas &&
		updatedMS.Status.AvailableReplicas != replicas {

		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func (c *ReconcileMachineSet) scaleDown(ctx context.Context, ms *clusterv1.MachineSet, machines []*clusterv1.Machine, n int64) error {
	alreadyDeleting := int64(0)
	if n == 0 {
		return nil
	}
	var runningMachines []*clusterv1.Machine
	for _, m := range machines {
		if !m.DeletionTimestamp.IsZero() {
			alreadyDeleting++
		} else {
			runningMachines = append(runningMachines, m)
		}
	}
	toDelete := n - alreadyDeleting
	for _, m := range runningMachines {
		if toDelete <= 0 {
			// Nothing to do here.
			return nil
		}
		if err := c.Client.Delete(ctx, m); err != nil {
			glog.Errorf("unable to delete a machine = %s, due to %v", m.Name, err)
			return err
		}
		toDelete--
	}
	return nil
}

// syncReplicas essentially scales machine resources up and down.
func (c *ReconcileMachineSet) syncReplicas(ctx context.Context, ms *clusterv1.MachineSet, machines []*clusterv1.Machine) error {
	if ms.Spec.Replicas == nil {
		return fmt.Errorf("the Replicas field in Spec for machineset %v is nil, this should not be allowed.", ms.Name)
	}

	existingMachines := make(map[string]bool)
	for _, m := range machines {
		existingMachines[m.Name] = true
	}

	desired, err := c.actuator.GetSize(ctx, ms)
	if err != nil {
		return err
	}
	current := int64(*ms.Spec.Replicas)
	if desired < current {
		if err := c.scaleDown(ctx, ms, machines, current-desired); err != nil {
			glog.Errorf("Error scaling down machine set %v: %v", ms.Name, err)
			return err
		}
	} else {
		if err := c.actuator.Resize(ctx, ms); err != nil {
			glog.Errorf("Error resizing machine set %v: %v", ms.Name, err)
			return err
		}
	}

	vms, err := c.actuator.ListMachines(ctx, ms)
	if err != nil {
		glog.Errorf("Error listing machines for machine set %v after resize: %v", ms.Name, err)
		return err
	}

	existingVMs := make(map[string]bool)
	for _, vmURL := range vms {
		vm := path.Base(vmURL)
		existingVMs[vm] = true
		if !existingMachines[vm] {
			machine := newMachine(vm, ms)
			err := c.Client.Create(ctx, machine)
			if err != nil {
				glog.Errorf("unable to create a machine = %s, due to %v", machine.Name, err)
			}
		}
	}

	// Cleaning up orphaned machines, the VMs do not exist.
	// What about VMs that are being deleted?
	// todo(maisem): how to handle drains?
	for _, m := range machines {
		if !existingVMs[m.Name] {
			err := c.Client.Delete(ctx, m)
			if err != nil {
				glog.Errorf("unable to delete a machine = %s, due to %v", m.Name, err)
			}
		}
	}

	return nil
}

// newMachine news a machine resource.
// the name of the newly newd resource is going to be newd by the API server, we set the generateName field
func newMachine(machineName string, ms *clusterv1.MachineSet) *clusterv1.Machine {
	gv := clusterv1.SchemeGroupVersion
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: gv.String(),
		},
		ObjectMeta: ms.Spec.Template.ObjectMeta,
		Spec:       ms.Spec.Template.Spec,
	}
	machine.ObjectMeta.Name = machineName
	machine.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(ms, controllerKind)}
	machine.Namespace = ms.Namespace
	if machine.Annotations == nil {
		machine.Annotations = make(map[string]string)
	}
	machine.Annotations[MIGMachineSetAnnotation] = ms.Name
	return machine
}

// shoudExcludeMachine returns true if the machine should be filtered out, false otherwise.
func shouldExcludeMachine(ms *clusterv1.MachineSet, machine *clusterv1.Machine) bool {
	// Ignore inactive machines.
	if metav1.GetControllerOf(machine) != nil && !metav1.IsControlledBy(machine, ms) {
		glog.V(4).Infof("%s not controlled by %v", machine.Name, ms.Name)
		return true
	}
	return false
}
