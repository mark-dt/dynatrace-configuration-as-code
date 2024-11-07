/*
 * @license
 * Copyright 2024 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package report

import (
	"bufio"
	"encoding/json"
	"time"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

const (
	TypeDeploy = "DEPLOY"
)

const (
	// StateDeploySuccess indicates a config was successfully deployed.
	StateDeploySuccess = "SUCCESS"

	// StateDeployError indicates a config could not be deployed due to an error.
	StateDeployError = "ERROR"

	// StateDeployExcluded indicates no attempt was made to deploy a config because it was marked by the user to skip.
	StateDeployExcluded = "EXCLUDED"

	// StateDeploySkipped indicates no attempt was made to deploy a config because one or mored dependencies were skipped or excluded.
	StateDeploySkipped = "SKIPPED"
)

// Record is a single entry in a report.
type Record struct {
	// Type is the type of record, currently TypeDeploy.
	Type string `json:"type"`

	// Time is the time associated with the Record.
	Time JSONTime `json:"time"`

	// Config provides the config ID, project and type of the config associated with the Record.
	Config coordinate.Coordinate `json:"config"`

	// State is the result of the deployment of the config, currently StateDeploySuccess, StateDeployError, StateDeployExcluded, StateDeploySkipped.
	State string `json:"state"`

	// Details optionally provides Detail log entries associated with the record.
	Details []Detail `json:"details,omitempty"`

	// Error optionally provides the string representation of any error associated with the Record.
	Error *string `json:"error,omitempty"`
}

// JSONTime represents a time.Time value that is serialized as a string in RFC3339 format.
type JSONTime time.Time

// MarshalJSON marshals a JSONTime value into a RFC3339 string.
func (t JSONTime) MarshalJSON() ([]byte, error) {
	s := time.Time(t).Format(time.RFC3339)
	return json.Marshal(s)
}

// UnmarshalJSON unmarshals a JSONTime value from a RFC3339 string.
func (t *JSONTime) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	tVal, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}

	*t = JSONTime(tVal)
	return nil
}

// ReadReportFile reads a report file and returns a slice of records or an error. It is intended for use in testing.
func ReadReportFile(fs afero.Fs, filename string) ([]Record, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	var records []Record
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r Record
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if s.Err() != nil {
		return nil, err
	}
	return records, nil
}