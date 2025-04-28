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
	"strings"
	"telemetruum/leaf-exporter/cli"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type InfoCollector struct {
	gauge metric.Int64ObservableGauge
}

func (c *InfoCollector) Init(logger zerolog.Logger) {

}

func (c *InfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_info", metric.WithDescription("info about the Telemetruum Installation"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *InfoCollector) CreateObservations(ctx context.Context, o metric.Observer, logger zerolog.Logger) {

	if *cli.InfoProps != "" {
		pairs := strings.Split(*cli.InfoProps, ",")

		metricsAttributes := []attribute.KeyValue{}

		for _, p := range pairs {
			tokens := strings.Split(p, "=")

			if len(tokens) != 2 {
				logger.Error().Msgf("Error parsing info prop '%s'. It should be in the form <name>=<value>", p)
				continue
			}

			metricsAttributes = append(metricsAttributes, attribute.Key("prop_"+tokens[0]).String(tokens[1]))
		}

		opt := metric.WithAttributeSet(attribute.NewSet(metricsAttributes...))

		o.ObserveInt64(c.gauge, 1, opt)
	}

}
