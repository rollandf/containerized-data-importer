/*
Copyright 2018 The CDI Authors.

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

package image

import (
	"strings"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"kubevirt.io/containerized-data-importer/pkg/system"
)

const dataTmpDir string = "/data_tmp"

// SkopeoOperations defines the interface for executing skopeo subprocesses
type SkopeoOperations interface {
	CopyImage(string, string) error
	ExtractTar(string, string) error
}

type skopeoOperations struct{}

var skopeoExecFunction = system.ExecWithLimits

var processLimits = &system.ProcessLimitValues{AddressSpaceLimit: maxMemory, CPUTimeLimit: maxCPUSecs}

var skopeoInterface = NewSkopeoOperations()

// NewSkopeoOperations returns the default implementation of QEMUOperations
func NewSkopeoOperations() SkopeoOperations {
	return &skopeoOperations{}
}

func (o *skopeoOperations) CopyImage(url, dest string) error {
	_, err := skopeoExecFunction(processLimits, "skopeo", "copy", url, dest)
	if err != nil {
		//os.Remove(dest)
		return errors.Wrap(err, "could not copy image")
	}

	return nil
}

func (o *skopeoOperations) ExtractTar(file, dest string) error {
	_, err := skopeoExecFunction(processLimits, "tar", "-xf", file, "-C", dest)
	if err != nil {
		//os.Remove(file)
		return errors.Wrap(err, "could not extract image")
	}

	return nil
}

// CopyImage
func CopyImage(url, dest string) error {
	return skopeoInterface.CopyImage(url, dest + dataTmpDir)
}

type manifest struct {
	Layers []layer `json:"layers"`
}
type layer struct {
	Digest string `json:"digest"`
}

func ExtractImageLayers(dest string) error {
	manifest := getImageManifest(dest + dataTmpDir)
	glog.V(1).Infof("manifest:" + manifest.Layers[0].Digest)
	for _, m := range manifest.Layers {
		layer := strings.TrimPrefix(m.Digest, "sha256:")		
		file := fmt.Sprintf("%s%s/%s", dest, dataTmpDir, layer)
		skopeoInterface.ExtractTar(file, dest)
	}
	// Clean temp folder
	os.RemoveAll(dest + dataTmpDir)
	return nil
}

func getImageManifest(dest string) manifest {
	// Open Manifest.json
	manifestFile, err := ioutil.ReadFile(dest + "/manifest.json")
	if err != nil {
		fmt.Println(err)
	}

	// Parse json file
	var manifestObj manifest
	json.Unmarshal(manifestFile, &manifestObj)
	return manifestObj
}
