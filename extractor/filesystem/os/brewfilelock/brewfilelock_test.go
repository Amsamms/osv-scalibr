// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brewfilelock_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/extractor/filesystem/os/brewfilelock"
	"github.com/google/osv-scalibr/extractor/filesystem/simplefileapi"
	"github.com/google/osv-scalibr/inventory"
	"github.com/google/osv-scalibr/purl"
	"github.com/google/osv-scalibr/testing/extracttest"

	cpb "github.com/google/osv-scalibr/binary/proto/config_go_proto"
)

func TestExtractor_FileRequired(t *testing.T) {
	tests := []struct {
		name      string
		inputPath string
		want      bool
	}{
		{
			name:      "empty name",
			inputPath: "",
			want:      false,
		},
		{
			name:      "Brewfile.lock.json from root",
			inputPath: "Brewfile.lock.json",
			want:      true,
		},
		{
			name:      "Brewfile.lock.json from subpath",
			inputPath: "path/to/my/Brewfile.lock.json",
			want:      true,
		},
		{
			name:      "Brewfile.lock.json as a dir",
			inputPath: "path/to/my/Brewfile.lock.json/file",
			want:      false,
		},
		{
			name:      "Brewfile.lock.json with additional extension",
			inputPath: "path/to/my/Brewfile.lock.json.bak",
			want:      false,
		},
		{
			name:      "wrong filename",
			inputPath: "Brewfile.json",
			want:      false,
		},
		{
			name:      "just Brewfile",
			inputPath: "Brewfile",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := brewfilelock.New(&cpb.PluginConfig{})
			if err != nil {
				t.Fatalf("brewfilelock.New: %v", err)
			}
			got := e.FileRequired(simplefileapi.New(tt.inputPath, nil))
			if got != tt.want {
				t.Errorf("FileRequired(%s, FileInfo) got = %v, want %v", tt.inputPath, got, tt.want)
			}
		})
	}
}

func TestExtractor_Extract(t *testing.T) {
	tests := []extracttest.TestTableEntry{
		{
			Name: "invalid json",
			InputConfig: extracttest.ScanInputMockConfig{
				Path: "testdata/invalid.json",
			},
			WantPackages: nil,
			WantErr:      extracttest.ContainsErrStr{Str: "could not extract"},
		},
		{
			Name: "empty entries",
			InputConfig: extracttest.ScanInputMockConfig{
				Path: "testdata/empty_entries.json",
			},
			WantPackages: nil,
		},
		{
			Name: "brew only",
			InputConfig: extracttest.ScanInputMockConfig{
				Path: "testdata/brew_only.json",
			},
			WantPackages: []*extractor.Package{
				{
					Name:     "curl",
					Version:  "8.7.1",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/brew_only.json"),
				},
			},
		},
		{
			Name: "valid with brew cask tap and mas",
			InputConfig: extracttest.ScanInputMockConfig{
				Path: "testdata/valid.json",
			},
			WantPackages: []*extractor.Package{
				{
					Name:     "git",
					Version:  "2.44.0",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/valid.json"),
				},
				{
					Name:     "node",
					Version:  "21.7.1",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/valid.json"),
				},
				{
					Name:     "openssl@3",
					Version:  "3.3.0",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/valid.json"),
				},
				{
					Name:     "firefox",
					Version:  "124.0.1",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/valid.json"),
				},
				{
					Name:     "visual-studio-code",
					Version:  "1.87.2",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/valid.json"),
				},
			},
		},
		{
			Name: "entries with missing or empty versions are skipped",
			InputConfig: extracttest.ScanInputMockConfig{
				Path: "testdata/missing_version.json",
			},
			WantPackages: []*extractor.Package{
				{
					Name:     "has-version",
					Version:  "1.0.0",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/missing_version.json"),
				},
				{
					Name:     "has-version",
					Version:  "2.0.0",
					PURLType: purl.TypeBrew,
					Location: extractor.LocationFromPath("testdata/missing_version.json"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			extr, err := brewfilelock.New(&cpb.PluginConfig{})
			if err != nil {
				t.Fatalf("brewfilelock.New: %v", err)
			}

			scanInput := extracttest.GenerateScanInputMock(t, tt.InputConfig)
			defer extracttest.CloseTestScanInput(t, scanInput)

			got, err := extr.Extract(t.Context(), &scanInput)

			if diff := cmp.Diff(tt.WantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s.Extract(%q) error diff (-want +got):\n%s", extr.Name(), tt.InputConfig.Path, diff)
				return
			}

			wantInv := inventory.Inventory{Packages: tt.WantPackages}
			if diff := cmp.Diff(wantInv, got, cmpopts.SortSlices(extracttest.PackageCmpLess)); diff != "" {
				t.Errorf("%s.Extract(%q) diff (-want +got):\n%s", extr.Name(), tt.InputConfig.Path, diff)
			}
		})
	}
}
