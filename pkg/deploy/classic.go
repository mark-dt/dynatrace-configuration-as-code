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

package deploy

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

func deployClassicConfig(configClient dtclient.ConfigClient, apis api.APIs, entityMap *entityMap, properties parameter.Properties, renderedConfig string, conf *config.Config) (*parameter.ResolvedEntity, error) {
	t, ok := conf.Type.(config.ClassicApiType)
	if !ok {
		return &parameter.ResolvedEntity{}, fmt.Errorf("config was not of expected type %q, but %q", config.ClassicApiTypeId, conf.Type.ID())
	}

	apiToDeploy, found := apis[t.Api]
	if !found {
		return &parameter.ResolvedEntity{}, fmt.Errorf("unknown api `%s`. this is most likely a bug", t.Api)
	}

	configName, err := extractConfigName(conf, properties)
	if err != nil {
		return &parameter.ResolvedEntity{}, err
	}
	if entityMap.contains(apiToDeploy.ID, configName) && !apiToDeploy.NonUniqueName {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(conf, fmt.Sprintf("duplicated config name `%s`", configName))
	}

	if apiToDeploy.DeprecatedBy != "" {
		log.Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", apiToDeploy.ID, apiToDeploy.DeprecatedBy)
	}

	var entity dtclient.DynatraceEntity
	if apiToDeploy.NonUniqueName {
		entity, err = upsertNonUniqueNameConfig(configClient, apiToDeploy, conf, configName, renderedConfig)
	} else {
		entity, err = configClient.UpsertConfigByName(apiToDeploy, configName, []byte(renderedConfig))
	}

	if err != nil {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(conf, err.Error())
	}

	properties[config.IdParameter] = entity.Id
	properties[config.NameParameter] = entity.Name

	return &parameter.ResolvedEntity{
		EntityName: entity.Name,
		Coordinate: conf.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil
}

func upsertNonUniqueNameConfig(client dtclient.ConfigClient, apiToDeploy api.API, conf *config.Config, configName string, renderedConfig string) (dtclient.DynatraceEntity, error) {
	configID := conf.Coordinate.ConfigId
	projectId := conf.Coordinate.Project

	entityUuid := configID

	isUUIDOrMeID := idutils.IsUUID(entityUuid) || idutils.IsMeId(entityUuid)
	if !isUUIDOrMeID {
		entityUuid = idutils.GenerateUUIDFromConfigId(projectId, configID)
	}

	return client.UpsertConfigByNonUniqueNameAndId(apiToDeploy, entityUuid, configName, []byte(renderedConfig))
}