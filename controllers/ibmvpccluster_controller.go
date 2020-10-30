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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IBMVPCClusterReconciler reconciles a IBMVPCCluster object
type IBMVPCClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *IBMVPCClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("ibmvpccluster", req.NamespacedName)

	// your logic here
	// Fetch the IBMVPCCluster instance
	ibmCluster := &infrastructurev1alpha3.IBMVPCCluster{}
	err := r.Get(ctx, req.NamespacedName, ibmCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, ibmCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	// Create the scope.
	iamEndpoint := os.Getenv("IAM_ENDPOINT")
	apiKey := os.Getenv("API_KEY")
	svcEndpoint := os.Getenv("SERVICE_ENDPOINT")

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:        r.Client,
		Logger:        log,
		Cluster:       cluster,
		IBMVPCCluster: ibmCluster,
	}, iamEndpoint, apiKey, svcEndpoint)

	if !controllerutil.ContainsFinalizer(clusterScope.IBMVPCCluster, infrastructurev1alpha3.ClusterFinalizer) {
		controllerutil.AddFinalizer(clusterScope.IBMVPCCluster, infrastructurev1alpha3.ClusterFinalizer)
		//_ = r.Update(ctx, clusterScope.IBMVPCCluster)
		return ctrl.Result{}, nil
	}

	// Handle deleted clusters
	if !ibmCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(clusterScope)
	}

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	} else {
		return r.reconcile(ctx, clusterScope)
	}
}

func (r *IBMVPCClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {

	//clusterScope.IBMVPCCluster.ObjectMeta.Finalizers = append(clusterScope.IBMVPCCluster.ObjectMeta.Finalizers, infrastructurev1alpha3.ClusterFinalizer)

	vpc, err := clusterScope.CreateVPC()
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile VPC for IBMVPCCluster %s/%s", clusterScope.IBMVPCCluster.Namespace, clusterScope.IBMVPCCluster.Name)
	}
	if vpc != nil {
		clusterScope.IBMVPCCluster.Status.VPC = infrastructurev1alpha3.VPC{
			ID:   *vpc.ID,
			Name: *vpc.Name,
		}
	}

	// TODO: I didn't find Status().Update in other provider's repo. But it appears in kubebuilder guide
	// https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html#2-list-all-active-jobs-and-update-the-status
	// Is there a way to perform updating of status implicitly?
	if err := r.Status().Update(ctx, clusterScope.IBMVPCCluster); err != nil {
		log.Error(err, "unable to update IBMVPCCluster status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IBMVPCClusterReconciler) reconcileDelete(clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	if err := clusterScope.DeleteVPC(); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to delete VPC")
	} else {
		controllerutil.RemoveFinalizer(clusterScope.IBMVPCCluster, infrastructurev1alpha3.ClusterFinalizer)
		return ctrl.Result{}, nil
	}
}

func (r *IBMVPCClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha3.IBMVPCCluster{}).
		Complete(r)
}
