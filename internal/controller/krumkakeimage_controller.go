package controller

import (
	"time"

	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/chitoku-k/cluster-api-provider-krumkake/context"
	"github.com/vultr/govultr/v3"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type KrumkakeImageReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	SnapshotService govultr.SnapshotService
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakeimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakeimages/status,verbs=get;update;patch

func (r *KrumkakeImageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KrumkakeImage")

	krumkakeImage := &infrastructurev1beta1.KrumkakeImage{}
	if err := r.Get(ctx, req.NamespacedName, krumkakeImage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	imageCtx := context.ImageContext{
		Context:       ctx,
		KrumkakeImage: krumkakeImage,
		Logger:        ctrl.LoggerFrom(ctx).WithName(req.String()),
	}

	patchHelper, err := patch.NewHelper(krumkakeImage, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := imageCtx.Patch(patchHelper); err != nil {
			imageCtx.Logger.Error(err, "failed to patch KrumkakeImage")
		}
	}()

	if !controllerutil.ContainsFinalizer(krumkakeImage, infrastructurev1beta1.ImageFinalizer) {
		controllerutil.AddFinalizer(krumkakeImage, infrastructurev1beta1.ImageFinalizer)
		return ctrl.Result{}, nil
	}

	if !krumkakeImage.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(imageCtx)
	} else {
		return r.reconcileNormal(imageCtx)
	}
}

func (r *KrumkakeImageReconciler) reconcileNormal(ctx context.ImageContext) (ctrl.Result, error) {
	switch ctx.KrumkakeImage.Status.Vultr.GetSnapshotState() {
	case infrastructurev1beta1.SnapshotStateNone:
		krumkakeMachineList := &infrastructurev1beta1.KrumkakeMachineList{}
		selector := fields.AndSelectors(
			fields.OneTermEqualSelector("spec.imageName", ctx.KrumkakeImage.Name),
			fields.OneTermNotEqualSelector("spec.vultr.region", ""),
		)
		if err := r.List(ctx, krumkakeMachineList, client.MatchingFieldsSelector{Selector: selector}); err != nil {
			return ctrl.Result{}, err
		}
		if len(krumkakeMachineList.Items) == 0 {
			return ctrl.Result{}, nil
		}

		snapshot, _, err := r.SnapshotService.CreateFromURL(ctx, &govultr.SnapshotURLReq{URL: ctx.KrumkakeImage.Spec.URL, UEFI: new(ctx.KrumkakeImage.Spec.UEFI)})
		if err != nil {
			ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStateError)
			return ctrl.Result{}, err
		}

		switch snapshot.Status {
		case "pending":
			ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStatePending)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

		default:
			ctx.Logger.Info("unknown snapshot status", "id", snapshot.ID, "status", snapshot.Status)
			ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStateError)
			return ctrl.Result{}, nil
		}

	case infrastructurev1beta1.SnapshotStatePending:
		snapshot, _, err := r.SnapshotService.Get(ctx, ctx.KrumkakeImage.Status.Vultr.GetSnapshotID())
		if err != nil {
			return ctrl.Result{}, err
		}

		switch snapshot.Status {
		case "complete":
			ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStateComplete)

		case "pending":
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

		default:
			ctx.Logger.Info("unknown snapshot status", "id", snapshot.ID, "status", snapshot.Status)
			ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStateError)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *KrumkakeImageReconciler) reconcileDelete(ctx context.ImageContext) (ctrl.Result, error) {
	if snapshotID := ctx.KrumkakeImage.Status.Vultr.GetSnapshotID(); snapshotID != "" {
		ctx.KrumkakeImage.Status.Vultr.SnapshotState = new(infrastructurev1beta1.SnapshotStateDeleted)

		if err := r.SnapshotService.Delete(ctx, snapshotID); err != nil {
			return ctrl.Result{}, err
		}
	}

	controllerutil.RemoveFinalizer(ctx.KrumkakeImage, infrastructurev1beta1.ImageFinalizer)
	return ctrl.Result{}, nil
}

func (r *KrumkakeImageReconciler) KrumkakeMachineToKrumkakeImages(ctx context.Context, obj client.Object) []ctrl.Request {
	krumkakeMachine := obj.(*infrastructurev1beta1.KrumkakeMachine)
	if krumkakeMachine.Spec.ImageName == "" || krumkakeMachine.Spec.Vultr.Region == "" {
		return nil
	}

	name := client.ObjectKey{Namespace: krumkakeMachine.Namespace, Name: krumkakeMachine.Spec.ImageName}
	return []ctrl.Request{
		{NamespacedName: name},
	}
}

func (r *KrumkakeImageReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KrumkakeImage{}).
		Watches(
			&infrastructurev1beta1.KrumkakeMachine{},
			handler.EnqueueRequestsFromMapFunc(r.KrumkakeMachineToKrumkakeImages),
		).
		Complete(r)
}
