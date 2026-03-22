package controller

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/chitoku-k/cluster-api-provider-krumkake/context"
	projectcalicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/utils/ptr"
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
	Scheme          *runtime.Scheme
	InstanceService govultr.InstanceService
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakemachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=krumkakemachines/status,verbs=get;update;patch
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
	if !ptr.Deref(ctx.Cluster.Status.Initialization.InfrastructureProvisioned, false) {
		return ctrl.Result{}, nil
	}
	if ctx.Machine.Spec.Bootstrap.DataSecretName == nil {
		return ctrl.Result{}, nil
	}

	if ctx.KrumkakeMachine.Spec.Vultr.Region != "" {
		return r.reconcileNormalVultr(ctx)
	}

	ctx.KrumkakeMachine.Status.Initialization.Provisioned = new(true)
	return ctrl.Result{}, nil
}

func (r *KrumkakeMachineReconciler) reconcileNormalVultr(ctx context.MachineContext) (ctrl.Result, error) {
	var instance *govultr.Instance
	var res *http.Response
	var err error
	if ctx.KrumkakeMachine.Spec.ProviderID == "" {
		dataSecretName := types.NamespacedName{Namespace: ctx.KrumkakeMachine.Namespace, Name: *ctx.Machine.Spec.Bootstrap.DataSecretName}
		dataSecret := &corev1.Secret{}
		if err := r.Get(ctx, dataSecretName, dataSecret); err != nil {
			return ctrl.Result{}, err
		}

		dataSecretValue, ok := dataSecret.Data["value"]
		if !ok {
			return ctrl.Result{}, fmt.Errorf("no value found in the secret")
		}

		krumkakeImageName := types.NamespacedName{Namespace: ctx.KrumkakeMachine.Namespace, Name: ctx.KrumkakeMachine.Spec.ImageName}
		krumkakeImage := &infrastructurev1beta1.KrumkakeImage{}
		if err := r.Get(ctx, krumkakeImageName, krumkakeImage); err != nil {
			return ctrl.Result{}, err
		}

		if ptr.Deref(krumkakeImage.Status.Vultr.SnapshotState, infrastructurev1beta1.SnapshotStateNone) != infrastructurev1beta1.SnapshotStateComplete {
			return ctrl.Result{}, nil
		}

		var attachVPC []string
		if vpcID := ctx.KrumkakeMachine.Spec.Vultr.VPCID; vpcID != "" {
			attachVPC = append(attachVPC, vpcID)
		}

		instance, res, err = r.InstanceService.Create(ctx, &govultr.InstanceCreateReq{
			Region: ctx.KrumkakeMachine.Spec.Vultr.Region,
			Plan:   ctx.KrumkakeMachine.Spec.Vultr.PlanID,
			Label:  ctx.KrumkakeMachine.Name,
			Tags: []string{
				fmt.Sprintf("clusterUID:%s", ctx.KrumkakeCluster.UID),
				fmt.Sprintf("clusterName:%s", ctx.KrumkakeCluster.Name),
				fmt.Sprintf("machineUID:%s", ctx.KrumkakeMachine.UID),
				fmt.Sprintf("machineName:%s", ctx.KrumkakeMachine.Name),
				fmt.Sprintf("namespace:%s", ctx.KrumkakeMachine.Namespace),
			},
			FirewallGroupID: ctx.KrumkakeMachine.Spec.Vultr.FirewallGroupID,
			SnapshotID:      krumkakeImage.Status.Vultr.SnapshotID,
			Hostname:        ctx.KrumkakeMachine.Name,
			EnableIPv6:      new(true),
			AttachVPC:       attachVPC,
			VPCOnly:         &ctx.KrumkakeMachine.Spec.Vultr.VPCOnly,
			SSHKeys:         ctx.KrumkakeMachine.Spec.Vultr.SSHKeys,
			UserData:        base64.StdEncoding.EncodeToString(dataSecretValue),
		})
	} else if instanceID, ok := strings.CutPrefix(ctx.KrumkakeMachine.Spec.ProviderID, "vultr://"); ok {
		instance, res, err = r.InstanceService.Get(ctx, instanceID)
	} else {
		return ctrl.Result{}, err
	}

	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			ctx.KrumkakeMachine.Spec.ProviderID = ""
			ctx.KrumkakeMachine.Status.Vultr = infrastructurev1beta1.KrumkakeMachineVultrStatus{}
			return ctrl.Result{}, nil
		}
		ctx.KrumkakeMachine.Status.Vultr.ServerState = new(infrastructurev1beta1.ServerStateError)
		return ctrl.Result{}, err
	}

	ctx.KrumkakeMachine.Spec.ProviderID = fmt.Sprintf("vultr://%s", instance.ID)

	ctx.KrumkakeMachine.Status.Addresses = []clusterv1beta2.MachineAddress{
		{
			Type:    clusterv1beta2.MachineExternalIP,
			Address: instance.MainIP,
		},
	}

	// TODO: Use instance.InternalIP when all migrations are complete.
	if instance.V6Network != "" {
		externalIPv6Address := instance.V6Network + "1"
		externalIPv6AddressSHA256 := sha256.Sum256([]byte(externalIPv6Address + "/128"))
		v, err := strconv.ParseUint(hex.EncodeToString(externalIPv6AddressSHA256[:])[:4], 16, 64)
		if err != nil {
			ctx.Logger.Error(err, "failed to parse sha256 of external IPv6 address")
		}
		v = (v >> 7) + 1
		internalIPv4Address := fmt.Sprintf("192.168.%d.%d", 34+(v/256), v%256)

		ctx.KrumkakeMachine.Status.Addresses = append(ctx.KrumkakeMachine.Status.Addresses,
			clusterv1beta2.MachineAddress{
				Type:    clusterv1beta2.MachineExternalIP,
				Address: externalIPv6Address,
			},
			clusterv1beta2.MachineAddress{
				Type:    clusterv1beta2.MachineInternalIP,
				Address: internalIPv4Address,
			},
		)
	}

	ctx.KrumkakeMachine.Status.CPU = instance.VCPUCount
	ctx.KrumkakeMachine.Status.RAM = instance.RAM
	ctx.KrumkakeMachine.Status.Storage = instance.Disk

	switch instance.PowerStatus {
	case "starting":
		ctx.KrumkakeMachine.Status.Vultr.PowerStatus = new(infrastructurev1beta1.PowerStatusStarting)
	case "stopped":
		ctx.KrumkakeMachine.Status.Vultr.PowerStatus = new(infrastructurev1beta1.PowerStatusStopped)
	case "running":
		ctx.KrumkakeMachine.Status.Vultr.PowerStatus = new(infrastructurev1beta1.PowerStatusRunning)
	}

	switch instance.Status {
	case "active":
		ctx.KrumkakeMachine.Status.Vultr.SubscriptionStatus = new(infrastructurev1beta1.SubscriptionStatusActive)
		ctx.KrumkakeMachine.Status.Initialization.Provisioned = new(true)
		return r.reconcileNode(ctx)
	case "suspended":
		ctx.KrumkakeMachine.Status.Vultr.SubscriptionStatus = new(infrastructurev1beta1.SubscriptionStatusSuspended)
		return ctrl.Result{}, nil
	case "closed":
		ctx.KrumkakeMachine.Status.Vultr.SubscriptionStatus = new(infrastructurev1beta1.SubscriptionStatusClosed)
		return ctrl.Result{}, nil
	case "pending":
		ctx.KrumkakeMachine.Status.Vultr.SubscriptionStatus = new(infrastructurev1beta1.SubscriptionStatusPending)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	default:
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

func (r *KrumkakeMachineReconciler) reconcileNode(ctx context.MachineContext) (ctrl.Result, error) {
	if ctx.Machine.Status.NodeRef.Name == "" {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	kubeconfigSecretName := types.NamespacedName{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name + "-kubeconfig"}
	kubeconfigSecret := &corev1.Secret{}
	if err := r.Get(ctx, kubeconfigSecretName, kubeconfigSecret); err != nil {
		return ctrl.Result{}, err
	}

	kubeconfigSecretValue, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return ctrl.Result{}, fmt.Errorf("no value found in the kubeconfig secret")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigSecretValue)
	if err != nil {
		return ctrl.Result{}, err
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := projectcalicov3.AddToScheme(scheme); err != nil {
		return ctrl.Result{}, err
	}

	ctx.WorkloadClusterClient, err = client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return ctrl.Result{}, err
	}

	nodeName := types.NamespacedName{Name: ctx.Machine.Status.NodeRef.Name}
	node := &corev1.Node{}
	if err := ctx.WorkloadClusterClient.Get(ctx, nodeName, node); err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	ctx.Node = node.DeepCopy()
	ctx.Node.Status.Addresses = make([]corev1.NodeAddress, 0, len(ctx.KrumkakeMachine.Status.Addresses))
	for _, address := range ctx.KrumkakeMachine.Status.Addresses {
		ctx.Node.Status.Addresses = append(ctx.Node.Status.Addresses, corev1.NodeAddress{
			Type:    corev1.NodeAddressType(address.Type),
			Address: address.Address,
		})
	}
	if err := ctx.WorkloadClusterClient.Status().Patch(ctx, ctx.Node, client.MergeFrom(node)); err != nil {
		return ctrl.Result{}, err
	}

	ctx.Node.Spec.Taints = slices.DeleteFunc(node.Spec.Taints, func(taint corev1.Taint) bool {
		return taint.MatchTaint(&corev1.Taint{Key: cloudproviderapi.TaintExternalCloudProvider, Effect: corev1.TaintEffectNoSchedule})
	})
	if err := ctx.WorkloadClusterClient.Patch(ctx, ctx.Node, client.MergeFrom(node)); err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileIPPool(ctx)
}

func (r *KrumkakeMachineReconciler) reconcileIPPool(ctx context.MachineContext) (ctrl.Result, error) {
	var cidr netip.Prefix
	for _, address := range ctx.Node.Status.Addresses {
		if address.Type != corev1.NodeExternalIP {
			continue
		}
		addr, _ := netip.ParseAddr(address.Address)
		if !addr.Is6() {
			continue
		}
		cidr, _ = addr.Prefix(64)
	}
	if !cidr.IsValid() {
		return ctrl.Result{}, nil
	}

	ipPoolName := types.NamespacedName{Name: ctx.Node.Name}
	ipPool := &projectcalicov3.IPPool{}
	if err := ctx.WorkloadClusterClient.Get(ctx, ipPoolName, ipPool); apierrors.IsNotFound(err) {
		ipPool := &projectcalicov3.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: ctx.Node.Name,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(ctx.Node, corev1.SchemeGroupVersion.WithKind("Node")),
				},
			},
			Spec: projectcalicov3.IPPoolSpec{
				CIDR:             cidr.String(),
				VXLANMode:        projectcalicov3.VXLANModeNever,
				IPIPMode:         projectcalicov3.IPIPModeNever,
				NATOutgoing:      false,
				DisableBGPExport: true,
				NodeSelector:     fmt.Sprintf("%s == %q", corev1.LabelHostname, ctx.Machine.Labels[corev1.LabelHostname]),
			},
		}
		if err := ctx.WorkloadClusterClient.Create(ctx, ipPool); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KrumkakeMachineReconciler) reconcileDelete(ctx context.MachineContext) (ctrl.Result, error) {
	if instanceID, ok := strings.CutPrefix(ctx.KrumkakeMachine.Spec.ProviderID, "vultr://"); ok {
		instance, res, err := r.InstanceService.Get(ctx, instanceID)
		if err != nil {
			if res != nil && res.StatusCode == http.StatusNotFound {
				ctx.KrumkakeMachine.Spec.ProviderID = ""
				ctx.KrumkakeMachine.Status.Vultr = infrastructurev1beta1.KrumkakeMachineVultrStatus{}
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		if err := r.InstanceService.Delete(ctx, instance.ID); err != nil {
			return ctrl.Result{}, err
		}
	}

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
