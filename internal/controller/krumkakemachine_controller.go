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

type KrumkakeMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io.infrastructure.cluster.x-k8s.io,resources=krumkakemachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io.infrastructure.cluster.x-k8s.io,resources=krumkakemachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *KrumkakeMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KrumkakeMachine")

	krumkakeMachine := &infrastructurev1beta1.KrumkakeMachine{}
	if err := r.Get(ctx, req.NamespacedName, krumkakeMachine); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	machine, err := clusterutil.GetOwnerMachine(ctx, r.Client, krumkakeMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Waiting for Machine Controller to set OwnerRef on KrumkakeMachine")
		return ctrl.Result{}, nil
	}

	cluster, err := clusterutil.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("KrumkakeMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	krumkakeCluster := &infrastructurev1beta1.KrumkakeCluster{}
	krumkakeClusterName := client.ObjectKey{
		Namespace: krumkakeMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, krumkakeClusterName, krumkakeCluster); err != nil {
		log.Info("KrumkakeCluster is not available yet.")
		return ctrl.Result{}, nil
	}
	if annotations.IsPaused(cluster, krumkakeCluster) {
		return ctrl.Result{}, nil
	}

	machineCtx := context.MachineContext{
		Context:         ctx,
		Cluster:         cluster,
		KrumkakeCluster: krumkakeCluster,
		Machine:         machine,
		KrumkakeMachine: krumkakeMachine,
		Logger:          ctrl.LoggerFrom(ctx).WithName(req.String()),
	}

	patchHelper, err := patch.NewHelper(krumkakeMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := machineCtx.Patch(patchHelper); err != nil {
			machineCtx.Logger.Error(err, "failed to patch KrumkakeMachine")
		}
	}()

	if !controllerutil.ContainsFinalizer(krumkakeMachine, infrastructurev1beta1.MachineFinalizer) {
		controllerutil.AddFinalizer(krumkakeMachine, infrastructurev1beta1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	if !krumkakeMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(machineCtx)
	} else {
		return r.reconcileNormal(machineCtx)
	}
}

func (r *KrumkakeMachineReconciler) reconcileNormal(ctx context.MachineContext) (ctrl.Result, error) {
	ctx.KrumkakeMachine.Status.Ready = true
	return ctrl.Result{}, nil
}

func (r *KrumkakeMachineReconciler) reconcileDelete(ctx context.MachineContext) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(ctx.KrumkakeMachine, infrastructurev1beta1.MachineFinalizer)
	return ctrl.Result{}, nil
}

func (r *KrumkakeMachineReconciler) KrumkakeClusterToKrumkakeMachines(ctx context.Context, obj client.Object) []ctrl.Request {
	var result []ctrl.Request
	log := ctrl.LoggerFrom(ctx)

	krumkakeCluster := obj.(*infrastructurev1beta1.KrumkakeCluster)
	cluster, err := clusterutil.GetOwnerCluster(ctx, r.Client, krumkakeCluster.ObjectMeta)
	if err != nil || cluster == nil {
		return nil
	}

	labels := map[string]string{clusterv1beta2.ClusterNameLabel: cluster.Name}
	machineList := &clusterv1beta2.MachineList{}
	if err := r.List(ctx, machineList, client.InNamespace(cluster.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "failed to list machines")
		return nil
	}

	for _, machine := range machineList.Items {
		if machine.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: machine.Namespace, Name: machine.Spec.InfrastructureRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

func (r *KrumkakeMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	clusterToKrumkakeMachines, err := clusterutil.ClusterToTypedObjectsMapper(mgr.GetClient(), &infrastructurev1beta1.KrumkakeMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KrumkakeMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1beta2.Machine{},
			handler.EnqueueRequestsFromMapFunc(clusterutil.MachineToInfrastructureMapFunc(infrastructurev1beta1.GroupVersion.WithKind("KrumkakeMachine"))),
		).
		Watches(
			&infrastructurev1beta1.KrumkakeCluster{},
			handler.EnqueueRequestsFromMapFunc(r.KrumkakeClusterToKrumkakeMachines),
		).
		Watches(
			&clusterv1beta2.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToKrumkakeMachines),
			builder.WithPredicates(predicates.ClusterPausedTransitionsOrInfrastructureProvisioned(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}
