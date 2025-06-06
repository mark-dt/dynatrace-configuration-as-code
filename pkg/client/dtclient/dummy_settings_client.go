/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package dtclient

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"
)

type DummySettingsClient struct{}

func (c *DummySettingsClient) Cache(context.Context, string) error {
	return nil
}

func (c *DummySettingsClient) ClearCache() {
	// no-op
}

func (c *DummySettingsClient) Upsert(_ context.Context, obj SettingsObject, _ UpsertSettingsOptions) (DynatraceEntity, error) {

	id := obj.Coordinate.ConfigId

	// to ensure decoding of Management Zone Numeric IDs works for dry-runs the dummy client needs to produce a fake but validly formated objectID
	if obj.SchemaId == "builtin:management-zones" {
		uuid := uuid.New().String()
		id = base64.RawURLEncoding.EncodeToString([]byte(uuid))
	}

	return DynatraceEntity{
		Id:   id,
		Name: obj.Coordinate.ConfigId,
	}, nil
}

func (c *DummySettingsClient) ListSchemas(_ context.Context) (SchemaList, error) {
	return make(SchemaList, 0), nil
}

func (c *DummySettingsClient) GetSchema(_ context.Context, _ string) (schema Schema, err error) {
	return Schema{}, nil
}

func (c *DummySettingsClient) Get(_ context.Context, _ string) (*DownloadSettingsObject, error) {
	return &DownloadSettingsObject{}, nil
}
func (c *DummySettingsClient) List(_ context.Context, _ string, _ ListSettingsOptions) ([]DownloadSettingsObject, error) {
	return make([]DownloadSettingsObject, 0), nil
}

func (c *DummySettingsClient) Delete(_ context.Context, _ string) error {
	return nil
}

func (c *DummySettingsClient) GetPermission(_ context.Context, _ string) (PermissionObject, error) {
	return PermissionObject{}, nil
}

func (c *DummySettingsClient) UpsertPermission(_ context.Context, _ string, _ PermissionObject) error {
	return nil
}

func (c *DummySettingsClient) DeletePermission(_ context.Context, _ string) error {
	return nil
}
