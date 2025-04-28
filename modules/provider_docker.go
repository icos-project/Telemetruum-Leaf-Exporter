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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"telemetruum/leaf-exporter/cli"

	"github.com/rs/zerolog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerProvider struct {
	BaseProvider
	DockerClient *client.Client
	Id           string
}

func InizializeDockerProvider(logger zerolog.Logger) (*DockerProvider, error) {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	info, err := cli.Info(context.TODO())
	if err != nil {
		logger.Debug().Msgf("Error initializing Docker with message: %s", err)
		return nil, err
	}

	return &DockerProvider{Id: info.Swarm.NodeID, DockerClient: cli, BaseProvider: BaseProvider{Logger: logger}}, nil
}

func (kd *DockerProvider) Start(ctx context.Context, wg *sync.WaitGroup) {
}

func dockerStatus2WIStatus(dockerStatus string) WorkloadStatus {
	switch dockerStatus {
	case "created":
		return Pending
	case "restarting":
		return Pending
	case "paused":
		return Pending
	case "running":
		return Running
	case "exited":
		return Exited
	case "dead":
		return Failed
	}
	return Unknown
}

func (kd *DockerProvider) ProvideWorkloadInfo(ctx context.Context, c *WorkloadInfoCollector) {
	containers, err := kd.DockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		panic(err)
	}
	res := []*WorkloadInfo{}

	kd.Logger.Debug().Msgf("Listing containers... found %d", len(containers))

	for _, ctr := range containers {
		wi := &WorkloadInfo{Name: ctr.Names[0], Id: ctr.ID, Annotations: map[string]string{}, Status: dockerStatus2WIStatus(ctr.State)}
		res = append(res, wi)

		for k, v := range ctr.Labels {
			if m1.MatchString(k) {
				newK := m1.ReplaceAllString(k, "icos.$1.$2")
				wi.Annotations[newK] = v
			}
		}
	}

	c.RunningWorkloads = res
	//c.ClusterId = kd.Id
}

func (kd *DockerProvider) ProvideNuvlaOrchestratorInfo(ctx context.Context, oic *OrchInfoCollector) {
	// /var/lib/nuvlaedge/%s/data/nuvlaedge_session.json
	CommonProvideNuvlaOrchestratorInfo(ctx, filepath.Join(*cli.PathRootFs, "/var/lib/docker/volumes/nuvlaedge_nuvlaedge-data/_data/nuvlaedge_session.json"), oic, kd.Logger)
}

func (kp *DockerProvider) ProvideLabels(ctx context.Context, hlc *HostLabelsCollector) {
	info, err := kp.DockerClient.Info(ctx)
	if err != nil {
		kp.Logger.Error().Msg("Error getting Docker Server info")
	} else {

		for _, s := range info.Labels {
			tokens := strings.Split(s, "=")
			if len(tokens) != 2 {
				kp.Logger.Error().Msgf("Error parsing label %s, expected two tokens, but got %d", s, len(tokens))
				continue
			}
			if m2.MatchString(tokens[0]) {
				newK := m2.ReplaceAllString(tokens[0], "$1")
				hlc.Labels[newK] = tokens[1]
			}
		}
	}
}

func (kd *DockerProvider) ProvideRuntimeOrchestartorInfo(ctx context.Context, ric *RuntimeInfoCollector) {
	ric.Type = "Docker"
	version, err := kd.DockerClient.ServerVersion(ctx)

	if err != nil {
		kd.Logger.Error().Msg("Error getting Docker Server version")
		return
	}

	info, err := kd.DockerClient.Info(ctx)
	if err != nil {
		kd.Logger.Error().Msg("Error getting Docker Server Node Id")
		return
	}

	ric.Version = version.Version
	ric.NodeName = info.Swarm.NodeID
	ric.ClusterId = info.Swarm.Cluster.ID
}

type NuvlaPeripheralFileStruct struct {
	Identifier string `json:"identifier"`
	Available  bool   `json:"available"`
	Interface  string `json:"interface"`
	DevicePath string `json:"device-path"`
	Name       string `json:"name"`
}

func (kd *DockerProvider) ProvideNuvlaAttachedPeripherals(ctx context.Context, oic *NodeMountedCollector) {

	// peripherals are taken from the nuvla edge cache because NFD is not running in Docker nodes. In Kubernetes
	// nodes, the same data is produced by NFD and scraped by the Telemetruum Leaf
	peripheral_files := filepath.Join(*cli.PathRootFs, "/var/lib/docker/volumes/nuvlaedge_nuvlaedge-data/_data/.peripherals/local_peripherals.json")

	nuvlaFile, err := os.ReadFile(peripheral_files) // just pass the file name
	if err != nil {
		kd.Logger.Warn().Msgf("Error reading Nuvla context file at %s: %s\n", peripheral_files, err)
		return
	}

	var nuvlaPeripherals map[string]NuvlaPeripheralFileStruct

	err = json.Unmarshal(nuvlaFile, &nuvlaPeripherals)

	if err != nil {
		kd.Logger.Warn().Msgf("Error unmarshalling Nuvla peripherals file: %s", err)
		return
	}

	res := []*Peripheral{}
	for _, v := range nuvlaPeripherals {

		if v.Interface != "USB" {
			continue
		}

		device := strings.ToLower(strings.ReplaceAll(v.Name, " ", "-")) + "_" + strings.ToLower(strings.ReplaceAll(v.Identifier, ":", "_"))

		p := &Peripheral{Device: device, Available: v.Available, ResourcePath: v.DevicePath}

		res = append(res, p)
	}

	oic.AttachedPeripherals = res
}
