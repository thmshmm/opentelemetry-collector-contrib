module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/stackdriverexporter

go 1.17

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter v0.48.0
	github.com/stretchr/testify v1.8.2
	go.opentelemetry.io/collector v0.48.0
	go.uber.org/zap v1.21.0
)

require (
	cloud.google.com/go/compute v1.5.0 // indirect
	cloud.google.com/go/monitoring v1.2.0 // indirect
	cloud.google.com/go/trace v1.0.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.11 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector v0.26.1-0.20220307211504-dc45061a44f9 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.3.0 // indirect
	github.com/aws/aws-sdk-go v1.43.32 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/googleapis/gax-go/v2 v2.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/knadh/koanf v1.4.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.48.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.48.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/collector/model v0.48.0 // indirect
	go.opentelemetry.io/otel v1.6.3 // indirect
	go.opentelemetry.io/otel/metric v0.29.0 // indirect
	go.opentelemetry.io/otel/sdk v1.6.1 // indirect
	go.opentelemetry.io/otel/trace v1.6.3 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/net v0.0.0-20220325170049-de3da57026de // indirect
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220328115105-d36c6a25d886 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/api v0.74.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220324131243-acbaeb5b85eb // indirect
	google.golang.org/grpc v1.45.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter => ../googlecloudexporter

replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus => ../../pkg/translator/opencensus

replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal => ../../internal/coreinternal
