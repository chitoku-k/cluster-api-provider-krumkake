package context

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MachineContext struct {
	context.Context
	Cluster               *clusterv1beta2.Cluster
	KrumkakeCluster       *infrastructurev1beta1.KrumkakeCluster
	Machine               *clusterv1beta2.Machine
	KrumkakeMachine       *infrastructurev1beta1.KrumkakeMachine
	Node                  *corev1.Node
	WorkloadClusterClient client.Client
	Logger                logr.Logger
}

func (m *MachineContext) Patch(patchHelper *patch.Helper) error {
	return patchHelper.Patch(
		m.Context,
		m.KrumkakeMachine,
		patch.WithOwnedConditions{
			Conditions: []string{clusterv1beta2.ReadyCondition},
		},
	)
}

func (m *MachineContext) String() string {
	return fmt.Sprintf(
		"%s %s/%s",
		m.KrumkakeMachine.GroupVersionKind(),
		m.KrumkakeMachine.Namespace,
		m.KrumkakeMachine.Name,
	)
}
