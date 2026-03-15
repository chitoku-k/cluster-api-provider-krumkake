package controller

import (
	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/chitoku-k/cluster-api-provider-krumkake/context"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type KrumkakeClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakeclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *KrumkakeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KrumkakeCluster")

	krumkakeCluster := &infrastructurev1beta1.KrumkakeCluster{}
	if err := r.Get(ctx, req.NamespacedName, krumkakeCluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if annotations.IsExternallyManaged(krumkakeCluster) {
		return ctrl.Result{}, nil
	}

	cluster, err := clusterutil.GetOwnerCluster(ctx, r.Client, krumkakeCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on KrumkakeCluster")
		return ctrl.Result{}, nil
	}
	if annotations.IsPaused(cluster, krumkakeCluster) {
		return ctrl.Result{}, nil
	}

	clusterCtx := context.ClusterContext{
		Context:         ctx,
		Cluster:         cluster,
		KrumkakeCluster: krumkakeCluster,
		Logger:          ctrl.LoggerFrom(ctx).WithName(req.String()),
	}

	patchHelper, err := patch.NewHelper(krumkakeCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := clusterCtx.Patch(patchHelper); err != nil {
			clusterCtx.Logger.Error(err, "failed to patch KrumkakeCluster")
		}
	}()

	if !controllerutil.ContainsFinalizer(krumkakeCluster, infrastructurev1beta1.ClusterFinalizer) {
		controllerutil.AddFinalizer(krumkakeCluster, infrastructurev1beta1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	if !krumkakeCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(clusterCtx)
	} else {
		return r.reconcileNormal(clusterCtx)
	}
}

func (r *KrumkakeClusterReconciler) reconcileNormal(ctx context.ClusterContext) (ctrl.Result, error) {
	ctx.KrumkakeCluster.Status.Initialization.Provisioned = new(true)
	return ctrl.Result{}, nil
}

func (r *KrumkakeClusterReconciler) reconcileDelete(ctx context.ClusterContext) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(ctx.KrumkakeCluster, infrastructurev1beta1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

func (r *KrumkakeClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KrumkakeCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1beta2.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterutil.ClusterToInfrastructureMapFunc(
				ctx,
				infrastructurev1beta1.GroupVersion.WithKind("KrumkakeCluster"),
				mgr.GetClient(),
				&infrastructurev1beta1.KrumkakeCluster{},
			)),
			builder.WithPredicates(predicates.ClusterUnpaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}
