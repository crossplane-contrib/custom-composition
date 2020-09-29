/*
Copyright 2020 The Crossplane Authors.

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
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane/apis/apiextensions/v1alpha1"

	"github.com/crossplane/custom-composition/pkg/controller"
	restoperation "github.com/crossplane/custom-composition/pkg/operations/rest"
)

var (
	scheme = runtime.NewScheme()
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Custom composition controller for Crossplane.").DefaultEnvars()
	xrdName := app.Flag("xrd-name", "Name of the Composite Resource Definition to reconcile its custom resources.").Required().String()
	ctx := context.Background()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	kingpin.FatalIfError(v1alpha1.AddToScheme(scheme), "cannot add Crossplane apiextensions type to the scheme")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	kingpin.FatalIfError(err, "unable to create a manager")

	xrd := &v1alpha1.CompositeResourceDefinition{}
	nn := types.NamespacedName{Name: *xrdName}
	kingpin.FatalIfError(getXRD(ctx, nn, xrd), fmt.Sprintf("cannot get composite resource definition with name: %s", *xrdName))

	r := controller.NewReconciler(mgr, xrd.GetCompositeGroupVersionKind(), controller.WithEngine(restoperation.NewRESTEngine("http://127.0.0.1:8080")))
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(xrd.GetCompositeGroupVersionKind())
	// TODO(muvaf): let's assume that only this controller reconciles that XRD.
	kingpin.FatalIfError(ctrl.NewControllerManagedBy(mgr).For(u).Complete(r), "cannot create controller")

	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "unable to start the manager")
}

func getXRD(ctx context.Context, nn types.NamespacedName, xrd *v1alpha1.CompositeResourceDefinition) error {
	config := ctrl.GetConfigOrDie()
	config.ContentConfig.GroupVersion = &v1alpha1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	client, err := rest.RESTClientFor(config)
	if err != nil {
		return err
	}
	return client.Get().Name(nn.Name).Resource("compositeresourcedefinitions").Do(ctx).Into(xrd)
}
