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

import (
	"encoding/json"
	"io"
	"sync"
)

// Logger writes NDJSON-formatted events to an output writer.
type Logger struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewLogger creates a Logger that writes to the given writer.
func NewLogger(w io.Writer) *Logger {
	return &Logger{enc: json.NewEncoder(w)}
}

// Write serialises a single event as one NDJSON line.
func (l *Logger) Write(ev Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enc.Encode(ev)
}
