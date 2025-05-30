//go:build unit

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

package dynatrace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

var classicCheckPayload = []byte(`
		{
		  "id": "abc-xy",
		  "name": "my-token",
		  "enabled": true,
		  "personalAccessToken": false,
		  "owner": "my-owner-email",
		  "creationDate": "2024-01-11T16:56:05.499Z",
		  "scopes": [
			"settings.read",
			"settings.write"
		  ]
		}`)

func TestVerifyEnvironmentGeneration_TurnedOffByFF(t *testing.T) {
	t.Setenv("MONACO_FEAT_VERIFY_ENV_TYPE", "0")
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(404)
	}))
	defer server.Close()

	ok := VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
		"env": manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
		},
	})
	assert.True(t, ok)
}
func TestVerifyEnvironmentGeneration_OneOfManyFails(t *testing.T) {

	envCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if envCount > 0 {
			rw.WriteHeader(404)
			return
		}
		rw.WriteHeader(200)
		_, _ = rw.Write(classicCheckPayload)
		envCount++
	}))
	defer server.Close()

	ok := VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
		"env": manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
		},
		"env2": manifest.EnvironmentDefinition{
			Name: "env2",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
		},
	})
	assert.False(t, ok)

}

func TestVerifyEnvironmentGen(t *testing.T) {
	type args struct {
		envs manifest.EnvironmentDefinitionsByName
	}
	tests := []struct {
		name                 string
		args                 args
		classicEnvCheckFails bool
		handler              http.HandlerFunc
		wantErr              bool
	}{
		{
			name: "empty environment - passes",
			args: args{
				envs: manifest.EnvironmentDefinitionsByName{},
			},
			wantErr: false,
		},
		{
			name: "single environment without fields set - fails",
			args: args{
				envs: manifest.EnvironmentDefinitionsByName{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ok := VerifyEnvironmentGeneration(t.Context(), tt.args.envs); ok == tt.wantErr {
				t.Errorf("VerifyEnvironmentGeneration() error = %v, wantErr %v", ok, tt.wantErr)
			}
		})
	}

	t.Run("Call classic endpoint - ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(200)
			_, _ = rw.Write(classicCheckPayload)
		}))
		defer server.Close()

		ok := VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
			"env": manifest.EnvironmentDefinition{
				Name: "env",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "URL",
					Value: server.URL,
				},
				Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"}},
			},
		})
		assert.True(t, ok)
	})

	t.Run("Call Platform endpoint - ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "sso") {
				token := &oauth2.Token{
					AccessToken: "test-access-token",
					TokenType:   "Bearer",
					Expiry:      time.Now().Add(time.Hour),
				}

				rw.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(rw).Encode(token)
				return
			}

			rw.WriteHeader(200)
			_, _ = rw.Write(classicCheckPayload)
		}))
		defer server.Close()

		ok := VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
			"env": manifest.EnvironmentDefinition{
				Name: "env",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "URL",
					Value: server.URL,
				},
				Auth: manifest.Auth{
					OAuth: &manifest.OAuth{
						TokenEndpoint: &manifest.URLDefinition{
							Value: server.URL + "/sso",
						},
					},
				},
			},
		})
		assert.True(t, ok)
	})

	t.Run("classic endpoint not available ", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "sso") {
				token := &oauth2.Token{
					AccessToken: "test-access-token",
					TokenType:   "Bearer",
					Expiry:      time.Now().Add(time.Hour),
				}

				rw.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(rw).Encode(token)
				return
			}

			rw.WriteHeader(404)
			_, _ = rw.Write(classicCheckPayload)
		}))
		defer server.Close()

		ok := VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
			"env1": manifest.EnvironmentDefinition{
				Name: "env1",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "URL",
					Value: server.URL + "/WRONG_URL",
				},
			},
		})
		assert.False(t, ok)

		ok = VerifyEnvironmentGeneration(t.Context(), manifest.EnvironmentDefinitionsByName{
			"env2": manifest.EnvironmentDefinition{
				Name: "env2",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "URL",
					Value: server.URL + "/WRONG_URL",
				},
				Auth: manifest.Auth{
					OAuth: &manifest.OAuth{
						TokenEndpoint: &manifest.URLDefinition{
							Value: server.URL + "/sso",
						},
					},
				},
			},
		})
		assert.False(t, ok)
	})
}
