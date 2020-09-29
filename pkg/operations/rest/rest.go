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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	rresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

func NewRESTEngine(url string) *Engine {
	return &Engine{url: url}
}

type Engine struct {
	url string
}

func (e *Engine) Run(cr rresource.Composite) ([]rresource.Object, error) {
	body, err := json.Marshal(cr)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal composite resource")
	}
	res, err := http.Post(e.url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "cannot make call to the server")
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read response")
	}
	fmt.Print(string(result))
	return nil, nil
}
