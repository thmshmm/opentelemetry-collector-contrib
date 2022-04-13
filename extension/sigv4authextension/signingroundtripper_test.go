// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sigv4authextension

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type errorRoundTripper struct{}

func (ert *errorRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("error")
}

func TestRoundTrip(t *testing.T) {
	awsCredsProvider := mockCredentials()

	defaultRoundTripper := (http.RoundTripper)(http.DefaultTransport.(*http.Transport).Clone())
	errorRoundTripper := &errorRoundTripper{}

	tests := []struct {
		name        string
		rt          http.RoundTripper
		shouldError bool
		cfg         *Config
	}{
		{
			"valid_round_tripper",
			defaultRoundTripper,
			false,
			&Config{Region: "region", Service: "service"},
		},
		{
			"error_round_tripper",
			errorRoundTripper,
			true,
			&Config{Region: "region", Service: "service", AssumeRole: AssumeRole{ARN: "rolearn"}},
		},
	}

	awsSDKInfo := "awsSDKInfo"
	body := "body"

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, awsSDKInfo, r.Header.Get("User-Agent"))

				reqBody := r.Body
				content, err := ioutil.ReadAll(reqBody)

				assert.NoError(t, err)
				assert.Equal(t, body, string(content))

				w.WriteHeader(200)
			}))
			defer server.Close()
			serverURL, _ := url.Parse(server.URL)

			testcase.cfg.credsProvider = awsCredsProvider
			sa := newSigv4Extension(testcase.cfg, awsSDKInfo, zap.NewNop())
			rt, err := sa.RoundTripper(testcase.rt)
			assert.NoError(t, err)

			newBody := strings.NewReader(body)
			req, err := http.NewRequest("POST", serverURL.String(), newBody)
			assert.NoError(t, err)

			res, err := rt.RoundTrip(req)
			if testcase.shouldError {
				assert.Nil(t, res)
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, res.StatusCode, 200)
		})
	}
}

func TestInferServiceAndRegion(t *testing.T) {
	req1, err := http.NewRequest("GET", "https://example.com", nil)
	assert.NoError(t, err)

	req2, err := http.NewRequest("GET", "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-XXX/api/v1/remote_write", nil)
	assert.NoError(t, err)

	req3, err := http.NewRequest("GET", "https://search-my-domain.us-east-1.es.amazonaws.com/_search?q=house", nil)
	assert.NoError(t, err)

	req4, err := http.NewRequest("GET", "https://example.com", nil)
	assert.NoError(t, err)

	req5, err := http.NewRequest("GET", "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-XXX/api/v1/remote_write", nil)
	assert.NoError(t, err)

	tests := []struct {
		name            string
		request         *http.Request
		cfg             *Config
		expectedService string
		expectedRegion  string
	}{
		{
			"no_service_or_region_match_with_no_config",
			req1,
			createDefaultConfig().(*Config),
			"",
			"",
		},
		{
			"amp_service_and_region_match_with_no_config",
			req2,
			createDefaultConfig().(*Config),
			"aps",
			"us-east-1",
		},
		{
			"es_service_and_region_match_with_no_config",
			req3,
			createDefaultConfig().(*Config),
			"es",
			"us-east-1",
		},
		{
			"no_match_with_config",
			req4,
			&Config{Region: "region", Service: "service", AssumeRole: AssumeRole{ARN: "rolearn"}},
			"service",
			"region",
		},
		{
			"match_with_config",
			req5,
			&Config{Region: "region", Service: "service", AssumeRole: AssumeRole{ARN: "rolearn"}},
			"service",
			"region",
		},
	}

	// run tests
	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			sa := newSigv4Extension(testcase.cfg, "awsSDKInfo", zap.NewNop())
			assert.NotNil(t, sa)

			rt, err := sa.RoundTripper((http.RoundTripper)(http.DefaultTransport.(*http.Transport).Clone()))
			assert.Nil(t, err)
			si := rt.(*signingRoundTripper)

			service, region := si.inferServiceAndRegion(testcase.request)
			assert.EqualValues(t, testcase.expectedService, service)
			assert.EqualValues(t, testcase.expectedRegion, region)
		})
	}
}
