// Copyright 2024 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loggers

import (
	"time"

	"github.com/bep/logg"
)

// LogFields provides a common implementation for structured logging fields
type LogFields struct {
	fields logg.Fields
}

// Fields implements the logg.Fielder interface
func (f *LogFields) Fields() logg.Fields {
	return f.fields
}

// NewLogFields creates a new LogFields instance with common fields
func NewLogFields() *LogFields {
	return &LogFields{
		fields: make(logg.Fields, 0),
	}
}

// AddField adds a new field to the LogFields
func (f *LogFields) AddField(name string, value interface{}) *LogFields {
	f.fields = append(f.fields, logg.Field{Name: name, Value: value})
	return f
}

// AddFields adds multiple fields to the LogFields
func (f *LogFields) AddFields(fields ...logg.Field) *LogFields {
	f.fields = append(f.fields, fields...)
	return f
}

// WithCommonFields creates a new LogFields instance with common fields like timestamp and level
func NewLogFieldsWithCommon(operation string, sessionID string) *LogFields {
	return &LogFields{
		fields: logg.Fields{
			{Name: "timestamp", Value: time.Now().UTC()},
			{Name: "level", Value: "info"}, // default level, can be overridden
			{Name: "sessionID", Value: sessionID},
			{Name: "operation", Value: operation},
		},
	}
}

// WithLevel sets the log level field
func (f *LogFields) WithLevel(level string) *LogFields {
	for i, field := range f.fields {
		if field.Name == "level" {
			f.fields[i].Value = level
			return f
		}
	}
	return f.AddField("level", level)
}
