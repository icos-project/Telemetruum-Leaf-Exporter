/*
ICOS Telemetruum Leaf Exporter
Copyright © 2022 - 2025 Engineering Ingegneria Informatica S.p.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This work has received funding from the European Union's HORIZON research
and innovation programme under grant agreement No. 101070177.
*/

package modules

import (
	"context"
	"log"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type RuntimeInfoCollector struct {
	Type      string
	Version   string
	NodeName  string
	ClusterId string
	gauge     metric.Int64ObservableGauge
}

func (c *RuntimeInfoCollector) Init(logger zerolog.Logger) {

}

func (c *RuntimeInfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_runtime_info", metric.WithDescription("info about the runtime"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *RuntimeInfoCollector) CreateObservations(ctx context.Context, o metric.Observer, logger zerolog.Logger) {
	if c.Type != "" {

		var metricsAttributes []attribute.KeyValue

		metricsAttributes = append(metricsAttributes, attribute.Key("type").String(c.Type))
		metricsAttributes = append(metricsAttributes, attribute.Key("version").String(c.Version))
		metricsAttributes = append(metricsAttributes, attribute.Key("node_name").String(c.NodeName))
		metricsAttributes = append(metricsAttributes, attribute.Key("cluster_id").String(c.ClusterId))

		opt := metric.WithAttributeSet(attribute.NewSet(metricsAttributes...))

		o.ObserveInt64(c.gauge, 1, opt)
	}
}
