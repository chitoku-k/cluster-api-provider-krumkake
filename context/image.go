package context

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/chitoku-k/cluster-api-provider-krumkake/api/v1beta1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/util/patch"
)

type ImageContext struct {
	context.Context
	KrumkakeImage *infrastructurev1beta1.KrumkakeImage
	Logger        logr.Logger
}

func (i *ImageContext) Patch(patchHelper *patch.Helper) error {
	return patchHelper.Patch(i.Context, i.KrumkakeImage)
}

func (i *ImageContext) String() string {
	return fmt.Sprintf(
		"%s %s/%s",
		i.KrumkakeImage.GroupVersionKind(),
		i.KrumkakeImage.Namespace,
		i.KrumkakeImage.Name,
	)
}
