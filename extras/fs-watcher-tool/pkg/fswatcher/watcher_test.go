// Copyright 2026 Google LLC
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

package fswatcher

import "testing"

func TestIsTempFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".file.swp", true},
		{".file.tmp", true},
		{"file~", true},
		{".hidden", true},
		{"normal.go", false},
		{"dir/.backup.swp", true},
		{"dir/file.swo", true},
		{"Makefile", false},
	}

	for _, tt := range tests {
		got := isTempFile(tt.path)
		if got != tt.want {
			t.Errorf("isTempFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
