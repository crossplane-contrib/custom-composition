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
	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Engine is used as main generation engine by the templating reconciler.
// Its input is typically a Custom Resource instance and output is various
// Kubernetes objects generated by the given implementation of the Engine.
type Engine interface {
	Run(composite rresource.Composite) ([]rresource.Object, error)
}

// EngineFunc used for supplying only one function as templating engine.
type EngineFunc func(rresource.Composite) ([]rresource.Object, error)

// Run calls the EngineFunc function.
func (t EngineFunc) Run(cr rresource.Composite) ([]rresource.Object, error) {
	return t(cr)
}

// ChildResourcePatcher operates on the resources rendered by the templating
// engine.
type ChildResourcePatcher interface {
	Patch(rresource.Composite, []rresource.Object) ([]rresource.Object, error)
}

// ChildResourcePatcherFunc makes it easier to provide only a function as
// ChildResourcePatcher
type ChildResourcePatcherFunc func(rresource.Composite, []rresource.Object) ([]rresource.Object, error)

// Patch calls the ChildResourcePatcherFunc function.
func (pre ChildResourcePatcherFunc) Patch(cr rresource.Composite, list []rresource.Object) ([]rresource.Object, error) {
	return pre(cr, list)
}

// ChildResourcePatcherChain makes it easier to provide a list of ChildResourcePatcher
// to be called in order.
type ChildResourcePatcherChain []ChildResourcePatcher

// Patch calls the ChildResourcePatcherChain functions in order.
func (pre ChildResourcePatcherChain) Patch(cr rresource.Composite, list []rresource.Object) ([]rresource.Object, error) {
	currentList := list
	var err error
	for _, f := range pre {
		currentList, err = f.Patch(cr, currentList)
		if err != nil {
			return nil, err
		}
	}
	return currentList, nil
}
