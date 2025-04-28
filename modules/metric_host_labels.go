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

type HostLabelsCollector struct {
	Labels       map[string]string
	staticLabels map[string]string
	gauge        metric.Int64ObservableGauge
}

func (c *HostLabelsCollector) Init(logger zerolog.Logger) {

	c.staticLabels = map[string]string{}

	for _, s := range *cli.StaticHostLabels {
		tokens := strings.Split(s, "=")
		if len(tokens) != 2 {
			logger.Error().Msgf("Error parsing static host label \"%s\". Ignoring it. Static Host Labels should be declared as --static-host-label \"<name>=<value>\"", s)
			continue
		}

		c.staticLabels[tokens[0]] = tokens[1]
	}

	logger.Debug().Msgf("Adding static host labels %+v", c.staticLabels)
}

func (c *HostLabelsCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_host_labels", metric.WithDescription("labels defined by the host and runtime"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *HostLabelsCollector) CreateObservations(ctx context.Context, o metric.Observer, logger zerolog.Logger) {

	if len(c.Labels) > 0 || len(c.staticLabels) > 0 {
		metricsAttributes := []attribute.KeyValue{}

		for k, v := range c.Labels {
			// append the "label_" prefix to distinguish labels from other attributes
			metricsAttributes = append(metricsAttributes, attribute.Key("label_"+k).String(v))
		}

		for k, v := range c.staticLabels {
			metricsAttributes = append(metricsAttributes, attribute.Key("label_"+k).String(v))
		}

		opt := metric.WithAttributeSet(attribute.NewSet(metricsAttributes...))

		o.ObserveInt64(c.gauge, 1, opt)
	}
}
