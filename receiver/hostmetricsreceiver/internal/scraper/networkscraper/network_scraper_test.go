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

package networkscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/scrapererror"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/networkscraper/internal/metadata"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name                 string
		config               Config
		bootTimeFunc         func() (uint64, error)
		ioCountersFunc       func(bool) ([]net.IOCountersStat, error)
		connectionsFunc      func(string) ([]net.ConnectionStat, error)
		expectNetworkMetrics bool
		expectedStartTime    pdata.Timestamp
		newErrRegex          string
		initializationErr    string
		expectedErr          string
		expectedErrCount     int
	}

	testCases := []testCase{
		{
			name: "Standard",
			config: Config{
				Metrics: metadata.DefaultMetricsSettings(),
			},
			expectNetworkMetrics: true,
		},
		{
			name: "Validate Start Time",
			config: Config{
				Metrics: metadata.DefaultMetricsSettings(),
			},
			bootTimeFunc:         func() (uint64, error) { return 100, nil },
			expectNetworkMetrics: true,
			expectedStartTime:    100 * 1e9,
		},
		{
			name: "Include Filter that matches nothing",
			config: Config{
				Metrics: metadata.DefaultMetricsSettings(),
				Include: MatchConfig{filterset.Config{MatchType: "strict"}, []string{"@*^#&*$^#)"}},
			},
			expectNetworkMetrics: false,
		},
		{
			name: "Invalid Include Filter",
			config: Config{
				Metrics: metadata.DefaultMetricsSettings(),
				Include: MatchConfig{Interfaces: []string{"test"}},
			},
			newErrRegex: "^error creating network interface include filters:",
		},
		{
			name: "Invalid Exclude Filter",
			config: Config{
				Metrics: metadata.DefaultMetricsSettings(),
				Exclude: MatchConfig{Interfaces: []string{"test"}},
			},
			newErrRegex: "^error creating network interface exclude filters:",
		},
		{
			name:              "Boot Time Error",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
		{
			name:             "IOCounters Error",
			ioCountersFunc:   func(bool) ([]net.IOCountersStat, error) { return nil, errors.New("err2") },
			expectedErr:      "failed to read network IO stats: err2",
			expectedErrCount: networkMetricsLen,
		},
		{
			name:             "Connections Error",
			connectionsFunc:  func(string) ([]net.ConnectionStat, error) { return nil, errors.New("err3") },
			expectedErr:      "failed to read TCP connections: err3",
			expectedErrCount: connectionsMetricsLen,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newNetworkScraper(context.Background(), &test.config)
			if test.newErrRegex != "" {
				require.Error(t, err)
				require.Regexp(t, test.newErrRegex, err)
				return
			}
			require.NoError(t, err, "Failed to create network scraper: %v", err)

			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}
			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}
			if test.connectionsFunc != nil {
				scraper.connections = test.connectionsFunc
			}

			err = scraper.start(context.Background(), componenttest.NewNopHost())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize network scraper: %v", err)

			md, err := scraper.scrape(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, test.expectedErrCount, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			expectedMetricCount := 1
			if test.expectNetworkMetrics {
				expectedMetricCount += 4
			}
			assert.Equal(t, expectedMetricCount, md.MetricCount())

			metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
			idx := 0
			assertNetworkConnectionsMetricValid(t, metrics.At(idx))
			if test.expectNetworkMetrics {
				assertNetworkIOMetricValid(t, metrics.At(idx+1), "system.network.dropped", test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+2), "system.network.errors", test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+3), "system.network.io", test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+4), "system.network.packets", test.expectedStartTime)
				internal.AssertSameTimeStampForMetrics(t, metrics, 1, 5)
				idx += 4
			}

			internal.AssertSameTimeStampForMetrics(t, metrics, idx, idx+1)
		})
	}
}

func assertNetworkIOMetricValid(t *testing.T, metric pdata.Metric, expectedName string, startTime pdata.Timestamp) {
	assert.Equal(t, expectedName, metric.Name())
	if startTime != 0 {
		internal.AssertSumMetricStartTimeEquals(t, metric, startTime)
	}
	assert.GreaterOrEqual(t, metric.Sum().DataPoints().Len(), 2)
	internal.AssertSumMetricHasAttribute(t, metric, 0, "device")
	internal.AssertSumMetricHasAttributeValue(t, metric, 0, "direction", pdata.NewValueString(metadata.AttributeDirection.Transmit))
	internal.AssertSumMetricHasAttributeValue(t, metric, 1, "direction", pdata.NewValueString(metadata.AttributeDirection.Receive))
}

func assertNetworkConnectionsMetricValid(t *testing.T, metric pdata.Metric) {
	assert.Equal(t, metric.Name(), "system.network.connections")
	internal.AssertSumMetricHasAttributeValue(t, metric, 0, "protocol", pdata.NewValueString(metadata.AttributeProtocol.Tcp))
	internal.AssertSumMetricHasAttribute(t, metric, 0, "state")
	assert.Equal(t, 12, metric.Sum().DataPoints().Len())
}
