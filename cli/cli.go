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

package cli

import "github.com/alecthomas/kingpin/v2"

var (
	App   = kingpin.New("tlum-exporter", "The Telemetruum Leaf exporter")
	Serve = App.Command("serve", "Serve metrics")

	PathRootFs           = Serve.Flag("path-rootfs", "Path of the root fs").Default("/").String()
	BindAddress          = Serve.Flag("bind", "Bind address").Default(":2545").String()
	DockerEnabled        = Serve.Flag("docker", "Enable Docker Provider").Default("true").Bool()
	KubernetesEnabled    = Serve.Flag("kubernetes", "Enable Kubernetes Provider").Default("true").Bool()
	FileEnabled          = Serve.Flag("file", "Enable File Provider").Default("true").Bool()
	IpinfoEnabled        = Serve.Flag("ipinfo", "Enable IPInfo Provider").Default("true").Bool()
	SystemEnabled        = Serve.Flag("system", "Enable System Provider").Default("true").Bool()
	HostInfoInterval     = Serve.Flag("host-info-interval", "Interval for Host Info Metrics").Default("10m").String()
	OrchInfoInterval     = Serve.Flag("orch-info-interval", "Interval for Orchestrator Info Metrics").Default("2m").String()
	RuntimeInfoInterval  = Serve.Flag("runtime-info-interval", "Interval for Runtime Info Metrics").Default("2m").String()
	VnetInfoInterval     = Serve.Flag("vnet-info-interval", "Interval for VNet Info Metrics").Default("1m").String()
	HostLabelsInterval   = Serve.Flag("host-labels-interval", "Interval for Host Labels Metrics").Default("5m").String()
	WorkloadInfoInterval = Serve.Flag("workload-info-interval", "Interval for Workload Info Metrics").Default("1m").String()
	NodeMountedInterval  = Serve.Flag("node-mount-interval", "Interval for Node Mounted Metrics").Default("1m").String()
	InfoProps            = Serve.Flag("info-props", "comma separated list of prop=value pairs").Default("").String()
	StaticHostLabels     = Serve.Flag("static-host-label", "Static Host Label always added to the host_labels metric").Default().Strings()

	PrintHostId = App.Command("print-host-id", "Print the host id")
)
