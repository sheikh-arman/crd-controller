/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	armancomv1alpha1 "github.com/sheikh-arman/crd-controller/pkg/apis/arman.com/v1alpha1"
	versioned "github.com/sheikh-arman/crd-controller/pkg/client/clientset/versioned"
	internalinterfaces "github.com/sheikh-arman/crd-controller/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/sheikh-arman/crd-controller/pkg/client/listers/arman.com/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ArmanInformer provides access to a shared informer and lister for
// Armans.
type ArmanInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ArmanLister
}

type armanInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewArmanInformer constructs a new informer for Arman type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewArmanInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredArmanInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredArmanInformer constructs a new informer for Arman type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredArmanInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ArmanV1alpha1().Armans(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ArmanV1alpha1().Armans(namespace).Watch(context.TODO(), options)
			},
		},
		&armancomv1alpha1.Arman{},
		resyncPeriod,
		indexers,
	)
}

func (f *armanInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredArmanInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *armanInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&armancomv1alpha1.Arman{}, f.defaultInformer)
}

func (f *armanInformer) Lister() v1alpha1.ArmanLister {
	return v1alpha1.NewArmanLister(f.Informer().GetIndexer())
}
