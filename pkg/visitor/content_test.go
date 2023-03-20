// Copyright 2023 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package visitor

import (
	"bytes"
	"context"
	"testing"

	"github.com/apigee/registry/cmd/registry/compress"
	"github.com/apigee/registry/pkg/connection/grpctest"
	"github.com/apigee/registry/pkg/names"

	"github.com/apigee/registry/rpc"
	"github.com/apigee/registry/server/registry/test/seeder"
)

func TestContentHelpers(t *testing.T) {
	specName, err := names.ParseSpec("projects/content-test/locations/global/apis/a/versions/v/specs/s")
	if err != nil {
		t.Fatalf("failed to parse spec name %s", specName)
	}
	compressedSpecName, err := names.ParseSpec("projects/content-test/locations/global/apis/a/versions/v/specs/s-gzip")
	if err != nil {
		t.Fatalf("failed to parse spec name %s", compressedSpecName)
	}
	specContents := "hello"
	compressedSpecContents, err := compress.GZippedBytes([]byte(specContents))
	if err != nil {
		t.Fatalf("failed to compress contents for %s", compressedSpecName)
	}
	ctx := context.Background()
	registryClient, adminClient := grpctest.SetupRegistry(ctx, t, "content-test", []seeder.RegistryResource{
		&rpc.ApiSpec{
			Name:     specName.String(),
			MimeType: "text/plain",
			Contents: []byte(specContents)},
		&rpc.ApiSpec{
			Name:     compressedSpecName.String(),
			MimeType: "text/plain+gzip",
			Contents: compressedSpecContents},
	})
	t.Cleanup(func() {
		if err := adminClient.DeleteProject(ctx, &rpc.DeleteProjectRequest{
			Name:  "projects/content-test",
			Force: true,
		}); err != nil {
			t.Fatalf("failed to delete test project: %s", err)
		}
	})
	t.Run("fetch-spec-contents", func(t *testing.T) {
		bytes, err := GetBytesForSpec(ctx, registryClient, &rpc.ApiSpec{Name: specName.String()})
		if err != nil {
			t.Fatalf("GetBytesForSpec failed to read spec contents")
		}
		if string(bytes) != specContents {
			t.Fatalf("GetBytesForSpec returned incorrect spec contents (%q expected %q)", string(bytes), specContents)
		}
	})
	t.Run("fetch-compressed-spec-contents", func(t *testing.T) {
		bytes, err := GetBytesForSpec(ctx, registryClient, &rpc.ApiSpec{Name: compressedSpecName.String()})
		if err != nil {
			t.Fatalf("GetBytesForSpec failed to read spec contents")
		}
		if string(bytes) != specContents {
			t.Fatalf("GetBytesForSpec returned incorrect spec contents (%q expected %q)", string(bytes), specContents)
		}
	})
	t.Run("fetch-spec-contents-invalid", func(t *testing.T) {
		_, err := GetBytesForSpec(ctx, registryClient, &rpc.ApiSpec{Name: specName.String() + "-invalid"})
		if err == nil {
			t.Fatalf("GetBytesForSpec of invalid spec succeeded and should have failed")
		}
	})
	t.Run("set-artifact", func(t *testing.T) {
		artifactName := specName.Artifact("x")
		err := SetArtifact(ctx, registryClient, &rpc.Artifact{
			Name:     artifactName.String(),
			Contents: []byte("123"),
		})
		if err != nil {
			t.Fatalf("SetArtifact failed to create artifact")
		}
		artifact := &rpc.Artifact{Name: artifactName.String()}
		err = FetchArtifactContents(ctx, registryClient, artifact)
		if err != nil {
			t.Fatalf("FetchArtifactContents failed to read new artifact")
		}
		if !bytes.Equal(artifact.Contents, []byte("123")) {
			t.Fatalf("FetchArtifactContents read incorrect content for new artifact")
		}
		err = SetArtifact(ctx, registryClient, &rpc.Artifact{
			Name:     specName.Artifact("x").String(),
			Contents: []byte("456"),
		})
		if err != nil {
			t.Fatalf("SetArtifact failed to update artifact")
		}
		artifact = &rpc.Artifact{Name: artifactName.String()}
		err = FetchArtifactContents(ctx, registryClient, artifact)
		if err != nil {
			t.Fatalf("FetchArtifactContents failed to read updated artifact")
		}
		if !bytes.Equal(artifact.Contents, []byte("456")) {
			t.Fatalf("FetchArtifactContents read incorrect content for updated artifact")
		}
	})
}
