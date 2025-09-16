// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"github.com/abklabs/pulumi-runner/pkg/runner"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	csharpGen "github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
	goGen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	nodejsGen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pythonGen "github.com/pulumi/pulumi/pkg/v3/codegen/python"
)

const Name string = "runner"

func Provider() p.Provider {
	// We tell the provider what resources it needs to support.
	// In this case, a single custom resource.
	return infer.Provider(infer.Options{
		Metadata: schema.Metadata{
			DisplayName: "runner",
			Description: "An alternative way to run scripts locally and remotely for pulumi",
			LanguageMap: map[string]any{
				"go": goGen.GoPackageInfo{
					ImportBasePath:  "github.com/abklabs/pulumi-runner/sdk/go",
					RootPackageName: "runner",
				},
				"nodejs": nodejsGen.NodePackageInfo{
					PackageName: "@svmkit/pulumi-runner",
				},
				"python": pythonGen.PackageInfo{
					PackageName: "pulumi_runner",
				},
				"csharp": csharpGen.CSharpPackageInfo{
					RootNamespace: "ABKLabs",
				},
			},
			Keywords: []string{
				"pulumi",
				"runner",
			},
			Homepage:          "https://abklabs.com",
			License:           "GPL-3.0-only",
			Repository:        "https://github.com/abklabs/pulumi-runner",
			Publisher:         "ABK Labs",
			PluginDownloadURL: "github://api.github.com/abklabs",
		},
		Resources: []infer.InferredResource{
			infer.Resource[runner.SSHDeployer](),
		},
		Functions: []infer.InferredFunction{
			infer.Function[runner.LocalFile](),
			infer.Function[runner.StringFile](),
		},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{},
	})
}
