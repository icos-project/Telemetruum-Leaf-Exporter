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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"telemetruum/leaf-exporter/cli"
	"telemetruum/leaf-exporter/modules"
	"time"

	"github.com/alecthomas/kingpin/v2"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/exporters/prometheus"
	metric2 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)



func serveMetrics(logger zerolog.Logger) {

	logger.Info().Msgf("serving metrics at %s/metrics", *cli.BindAddress)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(*cli.BindAddress, nil) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
	if err != nil {
		fmt.Printf("error serving http: %v", err)
		return
	}
}

func setupOtel() metric2.Meter {

	prom_client.Unregister(collectors.NewGoCollector())
	prom_client.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	exporter, err := prometheus.New(
		prometheus.WithoutTargetInfo(),
		prometheus.WithoutScopeInfo())
	if err != nil {
		log.Fatal()
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("telemetruum-leaf-exporter")

	return meter
}

func main() {

	switch kingpin.MustParse(cli.App.Parse(os.Args[1:])) {
	case cli.Serve.FullCommand():
		serveCommand()

	case cli.PrintHostId.FullCommand():
		logger := zerolog.New(
			zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC822},
		).Level(zerolog.FatalLevel).With().Timestamp().Logger()
		fileProvider := &modules.FileProvider{
			BaseProvider: modules.BaseProvider{Logger: logger.With().Str("Provider", "File").Logger()}}

		fmt.Print(fileProvider.GetHostId())
	}
}

func serveCommand() {
	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC822},
	).Level(zerolog.TraceLevel).With().Timestamp().Logger()

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	meter := setupOtel()

	// Setup Metric Collectors

	icr := &modules.AsyncCollectorRunner[*modules.InfoCollector]{
		Collector: &modules.InfoCollector{},
		Logger:    logger.With().Str("Collector", "HostInfo").Logger()}

	hicr_interval, _ := time.ParseDuration(*cli.HostInfoInterval)
	hicr := &modules.AsyncCollectorRunner[*modules.HostInfoCollector]{
		Collector: &modules.HostInfoCollector{},
		Interval:  hicr_interval,
		Logger:    logger.With().Str("Collector", "HostInfo").Logger()}

	oicr_interval, _ := time.ParseDuration(*cli.OrchInfoInterval)
	oicr := &modules.AsyncCollectorRunner[*modules.OrchInfoCollector]{
		Collector: &modules.OrchInfoCollector{},
		Interval:  oicr_interval,
		Logger:    logger.With().Str("Collector", "OrchInfo").Logger()}

	wicr_interval, _ := time.ParseDuration(*cli.WorkloadInfoInterval)
	wicr := &modules.AsyncCollectorRunner[*modules.WorkloadInfoCollector]{
		Collector: &modules.WorkloadInfoCollector{},
		Interval:  wicr_interval,
		Logger:    logger.With().Str("Collector", "WorkloadInfo").Logger()}

	pecr_interval, _ := time.ParseDuration(*cli.NodeMountedInterval)
	pecr := &modules.AsyncCollectorRunner[*modules.NodeMountedCollector]{
		Collector: &modules.NodeMountedCollector{},
		Interval:  pecr_interval,
		Logger:    logger.With().Str("Collector", "NodeMounted").Logger()}

	ricr_interval, _ := time.ParseDuration(*cli.RuntimeInfoInterval)
	ricr := &modules.AsyncCollectorRunner[*modules.RuntimeInfoCollector]{
		Collector: &modules.RuntimeInfoCollector{},
		Interval:  ricr_interval,
		Logger:    logger.With().Str("Collector", "RuntimeInfo").Logger()}

	hlcr_interval, _ := time.ParseDuration(*cli.HostLabelsInterval)
	hlcr := &modules.AsyncCollectorRunner[*modules.HostLabelsCollector]{
		Collector: &modules.HostLabelsCollector{Labels: map[string]string{}},
		Interval:  hlcr_interval,
		Logger:    logger.With().Str("Collector", "Labels").Logger()}

	vncr_interval, _ := time.ParseDuration(*cli.VnetInfoInterval)
	vncr := &modules.AsyncCollectorRunner[*modules.VNetInfoCollector]{
		Collector: &modules.VNetInfoCollector{},
		Interval:  vncr_interval,
		Logger:    logger.With().Str("Collector", "VNet").Logger()}

	// Setup Providers
	var kubernetesProvider *modules.KubernetesProvider
	var systemProvider *modules.SystemProvider
	var fileProvider *modules.FileProvider
	var dockerProvider *modules.DockerProvider
	var ipinfoProvider *modules.IPInfoProvider
	var providerErr error

	if *cli.KubernetesEnabled {
		kubernetesProvider, providerErr = modules.InizializeKubernetesProvider(logger.With().Str("Provider", "Kubernetes").Logger())
		if providerErr != nil {
			logger.Warn().Msgf("Error initializing Kubernetes (\"%s\"). The Kubernetes provider will not be used", providerErr)
		} else {

			kubernetesProvider.Start(ctx, wg)
			logger.Info().Msg("Kubernetes Provider successfully started")

			oicr.AppendAsyncDataProvider(kubernetesProvider.ProvideOCMOrchInfo)
			wicr.AppendAsyncDataProvider(kubernetesProvider.ProvideWorkloadInfo)
			oicr.AppendAsyncDataProvider(kubernetesProvider.ProvideNuvlaOrchestratorInfo)
			ricr.AppendAsyncDataProvider(kubernetesProvider.ProvideRuntimeOrchestartorInfo)
			hlcr.AppendAsyncDataProvider(kubernetesProvider.ProvideLabels)
			vncr.AppendAsyncDataProvider(kubernetesProvider.ProvideVNetInfo)

			// after the leader has been elected, run the following collectors that did not run
			// beofre because the leader was not elected yet. Without this, the collector might
			// run a long time after the leader was elected (<= collector interval)
			kubernetesProvider.OnLeaderElected(func(iAmTheLeader bool) {
				if iAmTheLeader {
					logger.Info().Msg("We are the leader, running the collector that are affected without waiting for their interval")
					kubernetesProvider.ProvideOCMOrchInfo(context.TODO(), oicr.Collector)
					kubernetesProvider.ProvideVNetInfo(context.TODO(), vncr.Collector)
				}
			})

		}

	} else {
		logger.Debug().Msg("Kubernetes Provider disabled")
	}

	if *cli.SystemEnabled {
		systemProvider = &modules.SystemProvider{
			BaseProvider: modules.BaseProvider{Logger: logger.With().Str("Provider", "System").Logger()}}

		systemProvider.Start(ctx, wg)
		logger.Info().Msg("System Provider successfully started")

		hicr.AppendAsyncDataProvider(systemProvider.ProvideHostInfo)

	} else {
		logger.Debug().Msg("System Provider disabled")
	}

	if *cli.IpinfoEnabled {
		ipinfoProvider = &modules.IPInfoProvider{
			BaseProvider: modules.BaseProvider{Logger: logger.With().Str("Provider", "IPInfo").Logger()}}

		fileProvider.Start(ctx, wg)
		logger.Info().Msg("IPInfo Provider successfully started")

		hicr.AppendAsyncDataProvider(ipinfoProvider.ProvideHostInfo)

	} else {
		logger.Debug().Msg("IPInfo Provider disabled")
	}

	if *cli.FileEnabled {
		fileProvider = &modules.FileProvider{
			BaseProvider: modules.BaseProvider{Logger: logger.With().Str("Provider", "File").Logger()}}

		fileProvider.Start(ctx, wg)
		logger.Info().Msg("File Provider successfully started")

		hicr.AppendAsyncDataProvider(fileProvider.ProvideHostInfo)
		hlcr.AppendAsyncDataProvider(fileProvider.ProvideLabels)

	} else {
		logger.Debug().Msg("File Provider disabled")
	}

	if *cli.DockerEnabled {
		dockerProvider, providerErr = modules.InizializeDockerProvider(logger.With().Str("Provider", "Docker").Logger())
		if providerErr != nil {
			logger.Warn().Msgf("Error initializing Docker (\"%s\"). The Docker provider will not be used", providerErr)
		} else {
			dockerProvider.Start(ctx, wg)
			logger.Info().Msg("Docker Provider successfully started")

			wicr.AppendAsyncDataProvider(dockerProvider.ProvideWorkloadInfo)
			oicr.AppendAsyncDataProvider(dockerProvider.ProvideNuvlaOrchestratorInfo)
			pecr.AppendAsyncDataProvider(dockerProvider.ProvideNuvlaAttachedPeripherals)
			ricr.AppendAsyncDataProvider(dockerProvider.ProvideRuntimeOrchestartorInfo)
			hlcr.AppendAsyncDataProvider(dockerProvider.ProvideLabels)

		}
	} else {
		logger.Debug().Msg("Docker Provider disabled")
	}

	// Start Metrics Collectors

	icr.Init(meter)
	icr.Start(context.TODO())
	hicr.Init(meter)
	hicr.Start(context.TODO())
	oicr.Init(meter)
	oicr.Start(context.TODO())
	wicr.Init(meter)
	wicr.Start(context.TODO())
	pecr.Init(meter)
	pecr.Start(context.TODO())
	ricr.Init(meter)
	ricr.Start(context.TODO())
	hlcr.Init(meter)
	hlcr.Start(context.TODO())
	vncr.Init(meter)
	vncr.Start(context.TODO())
	go serveMetrics(logger)

	<-ch
	cancel()

	wg.Wait()
}
