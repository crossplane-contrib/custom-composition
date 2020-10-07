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

package rest

import (
	"bufio"
	"bytes"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	jsonutil "k8s.io/apimachinery/pkg/util/json"

	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"

	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
)

var serializer = json.NewSerializerWithOptions(
	json.DefaultMetaFactory,
	unstructuredscheme.NewUnstructuredCreator(),
	unstructuredscheme.NewUnstructuredObjectTyper(),
	json.SerializerOptions{Yaml: true},
)

func NewRESTEngine(url string, kube client.Client) *Engine {
	return &Engine{url: url, kube: kube}
}

type Engine struct {
	kube client.Client
	url  string
}

func (e *Engine) Run(cr rresource.Composite) ([]rresource.Object, error) {
	// TODO(muvaf): cdk8s server accepts json but returns YAML. So, we can't
	// use the same YAML serializer for both encode/decode for now.
	body, err := jsonutil.Marshal(cr)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal composite")
	}
	response, err := http.Post(e.url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "cannot make call to the server")
	}
	yr := yaml.NewYAMLReader(bufio.NewReader(response.Body))
	var result []rresource.Object
	for {
		object, err := yr.Read()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			break
		}
		if len(object) == 0 {
			continue
		}
		o, _, err := serializer.Decode(object, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "cannot decode object")
		}
		result = append(result, o.(rresource.Object))
	}
	return result, nil
}
