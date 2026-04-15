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

// Package brewfilelock extracts package information from Homebrew Bundler
// lockfiles (Brewfile.lock.json).
package brewfilelock

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/extractor/filesystem"
	"github.com/google/osv-scalibr/inventory"
	"github.com/google/osv-scalibr/plugin"
	"github.com/google/osv-scalibr/purl"

	cpb "github.com/google/osv-scalibr/binary/proto/config_go_proto"
)

const (
	// Name is the unique name of this extractor.
	Name = "os/brewfilelock"
)

// brewfileLock represents the top-level structure of a Brewfile.lock.json file.
type brewfileLock struct {
	Entries brewfileEntries `json:"entries"`
}

// brewfileEntries contains the categorized entries in a Brewfile.lock.json.
type brewfileEntries struct {
	Brew map[string]brewEntry `json:"brew"`
	Cask map[string]brewEntry `json:"cask"`
}

// brewEntry represents a single formula or cask entry.
type brewEntry struct {
	Version string `json:"version"`
}

// Extractor extracts packages from Brewfile.lock.json files.
type Extractor struct{}

// New returns a new instance of the extractor.
func New(_ *cpb.PluginConfig) (filesystem.Extractor, error) { return &Extractor{}, nil }

// Name of the extractor.
func (e Extractor) Name() string { return Name }

// Version of the extractor.
func (e Extractor) Version() int { return 0 }

// Requirements of the extractor.
func (e Extractor) Requirements() *plugin.Capabilities {
	return &plugin.Capabilities{}
}

// FileRequired returns true if the specified file matches Brewfile.lock.json.
func (e Extractor) FileRequired(api filesystem.FileAPI) bool {
	return filepath.Base(api.Path()) == "Brewfile.lock.json"
}

// Extract extracts packages from a Brewfile.lock.json file.
func (e Extractor) Extract(ctx context.Context, input *filesystem.ScanInput) (inventory.Inventory, error) {
	var lockfile brewfileLock
	if err := json.NewDecoder(input.Reader).Decode(&lockfile); err != nil {
		return inventory.Inventory{}, fmt.Errorf("could not extract from %s: %w", input.Path, err)
	}

	var packages []*extractor.Package

	// Extract formulae (brew entries).
	for _, name := range sortedKeys(lockfile.Entries.Brew) {
		entry := lockfile.Entries.Brew[name]
		if entry.Version == "" {
			continue
		}
		packages = append(packages, &extractor.Package{
			Name:     name,
			Version:  entry.Version,
			PURLType: purl.TypeBrew,
			Location: extractor.LocationFromPath(input.Path),
		})
	}

	// Extract casks.
	for _, name := range sortedKeys(lockfile.Entries.Cask) {
		entry := lockfile.Entries.Cask[name]
		if entry.Version == "" {
			continue
		}
		packages = append(packages, &extractor.Package{
			Name:     name,
			Version:  entry.Version,
			PURLType: purl.TypeBrew,
			Location: extractor.LocationFromPath(input.Path),
		})
	}

	return inventory.Inventory{Packages: packages}, nil
}

// sortedKeys returns the keys of a map in sorted order for deterministic output.
func sortedKeys(m map[string]brewEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ filesystem.Extractor = Extractor{}
