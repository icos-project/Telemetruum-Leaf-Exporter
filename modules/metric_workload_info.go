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

type WorkloadStatus string

const (
	Pending WorkloadStatus = "pending"
	Running WorkloadStatus = "running"
	Exited  WorkloadStatus = "exited"
	Failed  WorkloadStatus = "failed"
	Unknown WorkloadStatus = "unknown"
)

func (s WorkloadStatus) String() string {
	switch s {
	case Pending:
		return "pending"
	case Running:
		return "running"
	case Exited:
		return "exited"
	case Failed:
		return "failed"
	}
	return "unknown"
}

type WorkloadInfo struct {
	Name        string
	Id          string
	Type        string
	Status      WorkloadStatus
	Annotations map[string]string
}

type WorkloadInfoCollector struct {
	RunningWorkloads []*WorkloadInfo
	gauge            metric.Int64ObservableGauge
}

func (c *WorkloadInfoCollector) Init(logger zerolog.Logger) {

}

func (c *WorkloadInfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_workload_info", metric.WithDescription("info about the workloads running in the node"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *WorkloadInfoCollector) CreateObservations(ctx context.Context, o metric.Observer, logger zerolog.Logger) {

	for _, w := range c.RunningWorkloads {

		var annotationAttributes []attribute.KeyValue

		annotationAttributes = append(annotationAttributes, attribute.Key("name").String(w.Name))
		annotationAttributes = append(annotationAttributes, attribute.Key("status").String(w.Status.String()))
		annotationAttributes = append(annotationAttributes, attribute.Key("id").String(w.Id))

		// this is not required because the "icos_host_id" label is added by the otel collector
		//annotationAttributes = append(annotationAttributes, attribute.Key("host_id").String(c.HostId))
		// commented because it is not sure that it is needed
		//annotationAttributes = append(annotationAttributes, attribute.Key("cluster_id").String(c.ClusterId))

		for k, v := range w.Annotations {
			annotationAttributes = append(annotationAttributes, attribute.Key(k).String(v))
		}

		opt := metric.WithAttributeSet(attribute.NewSet(annotationAttributes...))

		o.ObserveInt64(c.gauge, 1, opt)
	}
}
