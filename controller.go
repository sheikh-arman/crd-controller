/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"context"
	"fmt"
	"github.com/sheikh-arman/crd-controller/pkg/client/informers/externalversions/arman.com"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	myv1alpha1 "github.com/sheikh-arman/crd-controller/pkg/apis/arman.com/v1alpha1"
	myclientset "github.com/sheikh-arman/crd-controller/pkg/client/clientset/versioned"
	myclientscheme "github.com/sheikh-arman/crd-controller/pkg/client/clientset/versioned/scheme"
	myinformers "github.com/sheikh-arman/crd-controller/pkg/client/informers/externalversions/arman.com/v1alpha1"
	mylisters "github.com/sheikh-arman/crd-controller/pkg/client/listers/arman.com/v1alpha1"
)

const controllerAgentName = "my-custom-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a arman is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a arman fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by arman"
	// MessageResourceSynced is the message used for an Event fired when a messi
	// is synced successfully
	MessageResourceSynced = "Arman synced successfully"
)

type DeploymentListerAndSynced struct {
	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
}
type ServiceListerAndSynced struct {
	serviceLister corelisters.ServiceLister
	serviceSynced cache.InformerSynced
}
type ArmanListerAndSynced struct {
	armanLister mylisters.ArmanLister
	armanSynced cache.InformerSynced
}

// Controller is the controller implementation for messi resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset myclientset.Interface

	DeploymentListerAndSynced
	ServiceListerAndSynced
	ArmanListerAndSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func createRecorder(kubeclientset kubernetes.Interface) record.EventRecorder {
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	return recorder
}

// NewCombo returns a new sample controller
func NewCombo(
	kubeclientset kubernetes.Interface,
	sampleclientset myclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	serviceInformer coreinformers.ServiceInformer,
	armanInformer myinformers.ArmanInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(myclientscheme.AddToScheme(scheme.Scheme))

	controller := &Controller{
		kubeclientset:   kubeclientset,
		sampleclientset: sampleclientset,
		DeploymentListerAndSynced: DeploymentListerAndSynced{
			deploymentsLister: deploymentInformer.Lister(),
			deploymentsSynced: deploymentInformer.Informer().HasSynced,
		},
		ServiceListerAndSynced: ServiceListerAndSynced{
			serviceLister: serviceInformer.Lister(),
			serviceSynced: serviceInformer.Informer().HasSynced,
		},
		ArmanListerAndSynced: ArmanListerAndSynced{
			armanLister: armanInformer.Lister(),
			armanSynced: armanInformer.Informer().HasSynced,
		},

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "armans"),
		recorder:  createRecorder(kubeclientset),
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when messi resources change
	armanInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.armanAdderFunction,
		UpdateFunc: func(old, new interface{}) {
			controller.armanAdderFunction(new)
		},
		DeleteFunc: controller.armanDeleteFunction,
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a messi resource then the handler will enqueue that messi resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.deploymentAdderFunction,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.deploymentAdderFunction(new)
		},
		DeleteFunc: controller.deploymentDeleteFunction,
	})

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.serviceAdderFunction,
		UpdateFunc: func(old, new interface{}) {
			newSvc := new.(*corev1.Service)
			oldSvc := old.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				return
			}
			controller.serviceAdderFunction(new)
		},
		DeleteFunc: controller.serviceDeleteFunction,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	fmt.Println("Run is called")
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting arman controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.serviceSynced, c.armanSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process messi resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second*5, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	fmt.Println("processNextWorkItem is called")
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// messi resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the messi resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	fmt.Println("syncHandler is called")
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the messi resource with this namespace/name
	messi, err := c.armanLister.Armans(namespace).Get(name)
	if err != nil {
		// The messi resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("messi '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	deploymentName := arman.Spec.DeploymentName
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	servicename := arman.Spec.ServiceName
	if servicename == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: Service name must be specified", key))
		return nil
	}

	// Get the deployment with the name specified in messi.spec
	deployment, err := c.deploymentsLister.Deployments(messi.Namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(messi.Namespace).Create(context.TODO(), newDeployment(messi), metav1.CreateOptions{})
	}

	// Get the service with the name specified in messi.spec
	svc, err := c.serviceLister.Services(messi.Namespace).Get(servicename)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		svc, err = c.kubeclientset.CoreV1().Services(messi.Namespace).Create(context.TODO(), newService(messi), metav1.CreateOptions{})
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this messi resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(deployment, messi) {
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(messi, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}

	if !metav1.IsControlledBy(svc, messi) {
		msg := fmt.Sprintf(MessageResourceExists, svc.Name)
		c.recorder.Event(messi, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}

	// If this number of the replicas on the messi resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if arman.Spec.Replicas != nil && *messi.Spec.Replicas != *deployment.Spec.Replicas {
		klog.V(4).Infof("messi %s replicas: %d, deployment replicas: %d", name, *messi.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.kubeclientset.AppsV1().Deployments(messi.Namespace).Update(context.TODO(), newDeployment(messi), metav1.UpdateOptions{})
	}

	// this has not been replicated.

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the messi resource to reflect the
	// current state of the world
	err = c.updateArmanStatus(messi, deployment, svc)
	if err != nil {
		return err
	}

	c.recorder.Event(messi, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateArmanStatus(messi *myv1alpha1.Arman, deployment *appsv1.Deployment, svc *corev1.Service) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	fmt.Println("updateMessiStatus is called")
	messiCopy := messi.DeepCopy()
	messiCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the messi resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.sampleclientset.ArmanV1alpha1().Armans(messi.Namespace).Update(context.TODO(), messiCopy, metav1.UpdateOptions{})
	return err
}
