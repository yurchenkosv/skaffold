/*
Copyright 2021 The Skaffold Authors

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

package inspect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	pathToCfg1 = "path/to/cfg1"
	pathToCfg2 = "path/to/cfg2"
)

func TestAddGcbBuildEnv(t *testing.T) {
	tests := []struct {
		description     string
		profile         string
		modules         []string
		buildEnvOpts    inspect.BuildEnvOptions
		expectedConfigs []string
		err             error
		expectedErrMsg  string
	}{
		{
			description:  "add to default pipeline",
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project1", DiskSizeGb: 2, MachineType: "machine1", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  googleCloudBuild:
    projectId: project1
    diskSizeGb: 2
    machineType: machine1
    timeout: "128"
    concurrency: 2
profiles:
- name: p1
  build:
    cluster: {}
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project1
    diskSizeGb: 2
    machineType: machine1
    timeout: "128"
    concurrency: 2
profiles:
- name: p1
  build:
    cluster: {}
`, ``,
			},
		},
		{
			description:  "add to existing profile",
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project1", DiskSizeGb: 2, MachineType: "machine1", Timeout: "128", Concurrency: 2, Profile: "p1"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p1
    activatedBy:
    - p1
build:
  local: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  googleCloudBuild: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`,
			},
		},
		{
			description:  "add to new profile",
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project1", DiskSizeGb: 2, MachineType: "machine1", Timeout: "128", Concurrency: 2, Profile: "p2"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p2
    activatedBy:
    - p2
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  googleCloudBuild: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`,
			},
		},
		{
			description:  "add to new profile in selected modules",
			modules:      []string{"cfg1_1"},
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project1", DiskSizeGb: 2, MachineType: "machine1", Timeout: "128", Concurrency: 2, Profile: "p2"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p2
    activatedBy:
    - p2
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  googleCloudBuild: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`, "",
			},
		},
		{
			description:  "add to new profile in nested module",
			modules:      []string{"cfg2"},
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project1", DiskSizeGb: 2, MachineType: "machine1", Timeout: "128", Concurrency: 2, Profile: "p2"},
			expectedConfigs: []string{"",
				`apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  googleCloudBuild: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    googleCloudBuild:
      projectId: project1
      diskSizeGb: 2
      machineType: machine1
      timeout: "128"
      concurrency: 2
`,
			},
		},
		{
			description:    "actionable error",
			err:            sErrors.MainConfigFileNotFoundErr("path/to/skaffold.yaml", fmt.Errorf("failed to read file : %q", "skaffold.yaml")),
			expectedErrMsg: `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/skaffold.yaml\": failed to read file : \"skaffold.yaml\". Check that the specified configuration file exists at \"path/to/skaffold.yaml\"."}` + "\n",
		},
		{
			description:    "generic error",
			err:            errors.New("some error occurred"),
			expectedErrMsg: `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata: v1.Metadata{Name: "cfg1_0"},
					Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{LocalBuild: &v1.LocalBuild{}}}},
					Profiles: []v1.Profile{
						{Name: "p1", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{Cluster: &v1.ClusterDetails{}}}}},
					}}, SourceFile: pathToCfg1, IsRootConfig: true, SourceIndex: 0},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata:     v1.Metadata{Name: "cfg1_1"},
					Dependencies: []v1.ConfigDependency{{Path: pathToCfg2}},
					Pipeline:     v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{LocalBuild: &v1.LocalBuild{}}}},
					Profiles: []v1.Profile{
						{Name: "p1", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{Cluster: &v1.ClusterDetails{}}}}},
					}}, SourceFile: pathToCfg1, IsRootConfig: true, SourceIndex: 1},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata: v1.Metadata{Name: "cfg2"},
					Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{GoogleCloudBuild: &v1.GoogleCloudBuild{}}}},
					Profiles: []v1.Profile{
						{Name: "p1", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{LocalBuild: &v1.LocalBuild{}}}}},
					}}, SourceFile: pathToCfg2, SourceIndex: 0},
			}
			t.Override(&inspect.ConfigSetFunc, func(opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				if test.err != nil {
					return nil, test.err
				}
				var sets parser.SkaffoldConfigSet
				if len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, "cfg2") || util.StrSliceContains(opts.ConfigurationFilter, "cfg1_1") {
					sets = append(sets, configSet[2])
				}
				if len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, "cfg1_0") {
					sets = append(sets, configSet[0])
				}
				if len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, "cfg1_1") {
					sets = append(sets, configSet[1])
				}
				return sets, nil
			})
			t.Override(&inspect.ReadFileFunc, func(filename string) ([]byte, error) {
				if filename == pathToCfg1 {
					return yaml.MarshalWithSeparator([]*v1.SkaffoldConfig{configSet[0].SkaffoldConfig, configSet[1].SkaffoldConfig})
				} else if filename == pathToCfg2 {
					return yaml.MarshalWithSeparator([]*v1.SkaffoldConfig{configSet[2].SkaffoldConfig})
				}
				t.FailNow()
				return nil, nil
			})
			var actualCfg1, actualCfg2 string
			t.Override(&inspect.WriteFileFunc, func(filename string, data []byte) error {
				switch filename {
				case pathToCfg1:
					actualCfg1 = string(data)
				case pathToCfg2:
					actualCfg2 = string(data)
				default:
					t.FailNow()
				}
				return nil
			})

			var buf bytes.Buffer
			err := AddGcbBuildEnv(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.modules, BuildEnvOptions: test.buildEnvOpts})
			t.CheckNoError(err)
			if test.err == nil {
				t.CheckDeepEqual(test.expectedConfigs[0], actualCfg1)
				t.CheckDeepEqual(test.expectedConfigs[1], actualCfg2)
			} else {
				t.CheckDeepEqual(test.expectedErrMsg, buf.String())
			}
		})
	}
}
