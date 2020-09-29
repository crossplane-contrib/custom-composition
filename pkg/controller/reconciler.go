/*
Copyright 2019 The Crossplane Authors.

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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/composite"
)

const (
	reconcileTimeout = 1 * time.Minute
	defaultShortWait = 30 * time.Second
	defaultLongWait  = 1 * time.Minute

	errUpdateResourceStatus  = "could not update status of the parent resource"
	errGetResource           = "could not get the parent resource"
	errTemplatingOperation   = "templating operation failed"
	errChildResourcePatchers = "child resource patchers failed"
	errApply                 = "apply failed"
)

// ReconcilerOption is used to provide necessary changes to templating
// reconciler configuration.
type ReconcilerOption func(*Reconciler)

// WithShortWait returns a ReconcilerOption that changes the wait
// duration that determines after how much time another reconcile should be triggered
// after an error pass.
func WithShortWait(d time.Duration) ReconcilerOption {
	return func(reconciler *Reconciler) {
		reconciler.shortWait = d
	}
}

// WithLongWait returns a ReconcilerOption that changes the wait
// duration that determines after how much time another reconcile should be triggered
// after a successful pass.
func WithLongWait(d time.Duration) ReconcilerOption {
	return func(reconciler *Reconciler) {
		reconciler.longWait = d
	}
}

// WithLogger returns a ReconcilerOption that changes the logger.
func WithLogger(l logging.Logger) ReconcilerOption {
	return func(reconciler *Reconciler) {
		reconciler.log = l
	}
}

// WithEngine returns a ReconcilerOption that changes the
// engine.
func WithEngine(eng Engine) ReconcilerOption {
	return func(reconciler *Reconciler) {
		reconciler.external = eng
	}
}

// NewReconciler returns a new templating reconciler that will reconcile
// given GroupVersionKind.
func NewReconciler(m manager.Manager, of schema.GroupVersionKind, options ...ReconcilerOption) *Reconciler {
	nr := func() rresource.Composite { return composite.New(composite.WithGroupVersionKind(of)) }
	uclient := unstructured.NewClient(m.GetClient())
	r := &Reconciler{
		client: rresource.ClientApplicator{
			Client:     uclient,
			Applicator: rresource.NewAPIPatchingApplicator(uclient),
		},
		newComposite: nr,
		shortWait:    defaultShortWait,
		longWait:     defaultLongWait,
		log:          logging.NewNopLogger(),
		external:     &NopEngine{},
		composed:     NewOwnerReferenceAdder(),
	}

	for _, opt := range options {
		opt(r)
	}
	return r
}

// Reconciler is used to reconcile an arbitrary CRD whose GroupVersionKind
// is supplied.
type Reconciler struct {
	client       rresource.ClientApplicator
	newComposite func() rresource.Composite
	shortWait    time.Duration
	longWait     time.Duration
	log          logging.Logger

	external Engine
	composed ChildResourcePatcher
}

// Reconcile is called by controller-runtime for reconciliation.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) { // nolint:gocyclo
	// NOTE(muvaf): This method is well over our cyclomatic complexity goal.
	// Be wary of adding additional complexity.

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	log := r.log.WithValues("parent-resource", req)

	cr := r.newComposite()
	if err := r.client.Get(ctx, req.NamespacedName, cr); err != nil {
		// There's no need to requeue if the resource no longer exists. Otherwise
		// we'll be requeued implicitly because we return an error.
		log.Info("Cannot get the requested resource", "error", err)
		return reconcile.Result{Requeue: false}, errors.Wrap(client.IgnoreNotFound(err), errGetResource)
	}

	composedResources, err := r.external.Run(cr)
	if err != nil {
		log.Info("Cannot run templating operation", "error", err)
		cr.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, errTemplatingOperation)))
		return ctrl.Result{RequeueAfter: r.shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateResourceStatus)
	}

	composedResources, err = r.composed.Patch(cr, composedResources)
	if err != nil {
		log.Info("Cannot run patchers on the child resources", "error", err)
		cr.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, errChildResourcePatchers)))
		return ctrl.Result{RequeueAfter: r.shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateResourceStatus)
	}

	for _, o := range composedResources {
		if err := r.client.Apply(ctx, o, rresource.MustBeControllableBy(cr.GetUID())); err != nil {
			log.Info("Cannot apply the changes to the child resources", "error", err)
			cr.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, fmt.Sprintf("%s: %s/%s of type %s", errApply, o.GetName(), o.GetNamespace(), o.GetObjectKind().GroupVersionKind().String()))))
			return ctrl.Result{RequeueAfter: r.shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateResourceStatus)
		}
	}
	log.Debug("Reconciliation finished with success")
	cr.SetConditions(v1alpha1.ReconcileSuccess())
	return ctrl.Result{RequeueAfter: r.longWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateResourceStatus)
}
