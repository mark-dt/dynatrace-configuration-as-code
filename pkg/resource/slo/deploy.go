/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package slo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type DeploySource interface {
	List(ctx context.Context) (api.PagedListResponse, error)
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
}

type DeployAPI struct {
	sloSource DeploySource
}

func NewDeployAPI(sloSource DeploySource) *DeployAPI {
	return &DeployAPI{sloSource}
}

type sloResponse struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx = logr.NewContextWithSlogLogger(ctx, slog.Default())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	externalID := idutils.GenerateExternalID(c.Coordinate)
	requestPayload, err := addExternalIdAndValidate(externalID, renderedConfig)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "failed to validate slo payload").WithError(err)
	}

	//Strategy 1 when OriginObjectId is set we update the object
	if c.OriginObjectId != "" {
		_, err = d.sloSource.Update(ctx, c.OriginObjectId, requestPayload)
		if err == nil {
			return createResolveEntity(c.OriginObjectId, properties, c), nil
		}

		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy slo: %s", c.OriginObjectId)).WithError(err)
		}
	}

	//Strategy 2 is to try to find a match with external id and update it
	matchID, match, err := d.findMatchOnRemote(ctx, externalID)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding slo with externalID: %s", externalID)).WithError(err)
	}

	if match {
		_, err := d.sloSource.Update(ctx, matchID, requestPayload)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update slo with externalID: %s", externalID)).WithError(err)
		}
		return createResolveEntity(matchID, properties, c), nil
	}

	//Strategy 3 is to create a new slo
	createResponse, err := d.sloSource.Create(ctx, requestPayload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy slo with externalID: %s", externalID)).WithError(err)
	}

	response, err := responseFromHttpData(createResponse)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to unmarshal slo with externalID: %s", externalID)).WithError(err)
	}

	return createResolveEntity(response.ID, properties, c), nil
}

func addExternalIdAndValidate(externalId string, renderedConfig string) ([]byte, error) {
	var request map[string]any
	err := json.Unmarshal([]byte(renderedConfig), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to add externalID to slo request payload: %w", err)
	}
	request["externalId"] = externalId
	if _, exists := request["evaluationType"]; exists {
		return nil, errors.New("tried to deploy an slo-v1 configuration to slo-v2")
	}
	return json.Marshal(request)
}

func responseFromHttpData(rawResponse api.Response) (sloResponse, error) {
	var response sloResponse
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return sloResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}

func (d DeployAPI) findMatchOnRemote(ctx context.Context, externalId string) (id string, match bool, err error) {
	apiResponse, err := d.sloSource.List(ctx)
	if err != nil {
		return "", false, err
	}

	res := sloResponse{}
	for _, raw := range apiResponse.All() {
		if err := json.Unmarshal(raw, &res); err != nil {
			return "", false, err
		}
		if res.ExternalID == externalId {
			return res.ID, true, nil
		}
	}

	return "", false, nil
}
