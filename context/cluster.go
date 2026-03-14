package context

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/go-logr/logr"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/patch"
)

type ClusterContext struct {
	context.Context
	Cluster         *clusterv1beta2.Cluster
	KrumkakeCluster *infrastructurev1beta1.KrumkakeCluster
	Logger          logr.Logger
}

func (m *ClusterContext) Patch(patchHelper *patch.Helper) error {
	return patchHelper.Patch(
		m.Context,
		m.KrumkakeCluster,
		patch.WithOwnedConditions{
			Conditions: []string{clusterv1beta2.ReadyCondition},
		},
	)
}

func (m *ClusterContext) String() string {
	return fmt.Sprintf(
		"%s %s/%s",
		m.KrumkakeCluster.GroupVersionKind(),
		m.KrumkakeCluster.Namespace,
		m.KrumkakeCluster.Name,
	)
}
