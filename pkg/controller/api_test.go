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
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/crossplane-runtime/pkg/test"
)

const (
	name = "fakename"
	uid  = "my-uid"
)

var (
	_ ChildResourcePatcher = OwnerReferenceAdder{}
)

type args struct {
	cr   rresource.Composite
	list []rresource.Object
}

type want struct {
	result []rresource.Object
	err    error
}

func TestOwnerReferenceAdder(t *testing.T) {
	cr := &fake.Composite{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			UID:  uid,
		},
	}
	cases := map[string]struct {
		args
		want
	}{
		"Add": {
			args: args{
				cr: cr,
				list: []rresource.Object{
					&fake.Composed{},
					&fake.Composed{},
				},
			},
			want: want{
				result: []rresource.Object{
					&fake.Composed{
						ObjectMeta: v1.ObjectMeta{
							OwnerReferences: []v1.OwnerReference{meta.AsController(meta.TypedReferenceTo(cr, cr.GetObjectKind().GroupVersionKind()))},
						},
					},
					&fake.Composed{
						ObjectMeta: v1.ObjectMeta{
							OwnerReferences: []v1.OwnerReference{meta.AsController(meta.TypedReferenceTo(cr, cr.GetObjectKind().GroupVersionKind()))},
						},
					},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := NewOwnerReferenceAdder()
			got, err := p.Patch(tc.args.cr, tc.args.list)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Patch(...): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("Patch(...): -want, +got:\n%s", diff)
			}
		})
	}
}
