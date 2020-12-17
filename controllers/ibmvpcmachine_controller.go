/*


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

package controllers

import (
	"context"
	"os"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/go-logr/logr"
	infrastructurev1alpha3 "github.com/multicloudlab/cluster-api-provider-ibmvpccloud/api/v1alpha3"
	"github.com/multicloudlab/cluster-api-provider-ibmvpccloud/cloud/scope"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IBMVPCMachineReconciler reconciles a IBMVPCMachine object
type IBMVPCMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines/status,verbs=get;update;patch

func (r *IBMVPCMachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	log := r.Log.WithValues("ibmvpcmachine", req.NamespacedName)

	// your logic here
	// Fetch the GCPMachine instance.
	log.Info("zwtzhang debug.......1")
	ibmVpcMachine := &infrastructurev1alpha3.IBMVPCMachine{}
	err := r.Get(ctx, req.NamespacedName, ibmVpcMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("zwtzhang debug.......2")
	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, ibmVpcMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}
	log.Info("zwtzhang debug.......3")
	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, ibmVpcMachine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	ibmCluster := &infrastructurev1alpha3.IBMVPCCluster{}
	log.Info("zwtzhang debug.......4")
	ibmVpcClusterName := client.ObjectKey{
		Namespace: ibmVpcMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, ibmVpcClusterName, ibmCluster); err != nil {
		log.Info("IBMVPCCluster is not available yet")
		return ctrl.Result{}, nil
	}
	log.Info("zwtzhang debug.......5")
	// Create the cluster scope
	iamEndpoint := os.Getenv("IAM_ENDPOINT")
	apiKey := os.Getenv("API_KEY")
	svcEndpoint := os.Getenv("SERVICE_ENDPOINT")

	log.Info("zwtzhang debug.......6")
	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:        r.Client,
		Logger:        log,
		Cluster:       cluster,
		IBMVPCCluster: ibmCluster,
		Machine:       machine,
		IBMVPCMachine: ibmVpcMachine,
	}, iamEndpoint, apiKey, svcEndpoint)
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}
	log.Info("zwtzhang debug.......7")
	// Always close the scope when exiting this function so we can persist any GCPMachine changes.

	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	log.Info("zwtzhang debug.......8")
	// Handle non-deleted machines
	return r.reconcile(ctx, machineScope)
}

func (r *IBMVPCMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha3.IBMVPCMachine{}).
		Complete(r)
}

func (r *IBMVPCMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope) (ctrl.Result, error) {

	//clusterScope.IBMVPCCluster.ObjectMeta.Finalizers = append(clusterScope.IBMVPCCluster.ObjectMeta.Finalizers, infrastructurev1alpha3.ClusterFinalizer)
	log.Info("zwtzhang debug.......reconcile...1")

	instance, err := r.getOrCreate(machineScope)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile VSI for IBMVPCMachine %s/%s", machineScope.IBMVPCMachine.Namespace, machineScope.IBMVPCMachine.Name)
	}
	log.Info("zwtzhang debug.......reconcile...2")

	if instance != nil {
		machineScope.IBMVPCMachine.Status.InstanceID = *instance.ID
		machineScope.IBMVPCMachine.Status.Ready = true
		log.Info(*instance.ID)
	}
	log.Info(*instance.ID)
	log.Info("zwtzhang debug.......reconcile...3")

	return ctrl.Result{}, nil
}

func (r *IBMVPCMachineReconciler) getOrCreate(scope *scope.MachineScope) (*vpcv1.Instance, error) {
	instance, err := scope.CreateMachine()
	return instance, err
}
