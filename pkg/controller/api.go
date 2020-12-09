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
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"
)

// NopEngine is a no-op templating engine.
type NopEngine struct{}

// Run does nothing.
func (n *NopEngine) Run(_ rresource.Composite) ([]rresource.Object, error) {
	return nil, nil
}

// NewOwnerReferenceAdder returns a new *OwnerReferenceAdder
func NewOwnerReferenceAdder() OwnerReferenceAdder {
	return OwnerReferenceAdder{}
}

// OwnerReferenceAdder adds owner reference of rresource.Composite to all rresource.Objects
// except the Providers since their deletion should be delayed until all resources
// refer to them are deleted.
type OwnerReferenceAdder struct{}

// Patch patches the child resources with information in rresource.Composite.
func (lo OwnerReferenceAdder) Patch(cr rresource.Composite, list []rresource.Object) ([]rresource.Object, error) {
	ref := meta.AsController(meta.TypedReferenceTo(cr, cr.GetObjectKind().GroupVersionKind()))
	trueVal := true
	ref.BlockOwnerDeletion = &trueVal
	for _, o := range list {
		meta.AddOwnerReference(o, ref)
	}
	return list, nil
}
