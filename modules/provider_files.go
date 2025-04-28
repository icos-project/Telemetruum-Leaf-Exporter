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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"telemetruum/leaf-exporter/cli"

	smbios "github.com/fenglyu/go-dmidecode"
)

type FileProvider struct {
	BaseProvider
}

func (p *FileProvider) Start(context.Context, *sync.WaitGroup) {

}

func (p *FileProvider) ProvideHostInfo(ctx context.Context, hic *HostInfoCollector) {

	loc := p.getMachineLocation()

	// consider location from file only if the location from ipinfo was invalid or disabled
	if hic.Latitutde == "" && hic.Longitude == "" {
		hic.Latitutde = loc[0]
		hic.Longitude = loc[1]
	}
	hic.Id = p.getMachineId()
}

func (p *FileProvider) ProvideLabels(ctx context.Context, hlc *HostLabelsCollector) {
	for k, v := range p.getMachineLabels() {
		hlc.Labels[k] = v
	}
}

func (p *FileProvider) GetHostId() string {
	return p.getMachineId()
}

func (p *FileProvider) getMachineLabels() map[string]string {
	labels := map[string]string{}
	content, err := os.ReadFile(filepath.Join(*cli.PathRootFs, "/etc/machine-labels"))
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*cli.PathRootFs, "/etc/machine-labels"))
		return labels
	}
	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		tokens := strings.Split(l, "=")
		if len(tokens) != 2 {
			p.Logger.Error().Msgf("Error parsing label %s, expected two tokens, but got %d", l, len(tokens))
			continue
		}
		labels[tokens[0]] = tokens[1]
		//if m1.MatchString(tokens[0]) {
		//  newK := m1.ReplaceAllString(tokens[0], "icos.$1.$2")
		//  labels[tokens[0]] = tokens[1]
		//}
	}

	return labels
}

func (p *FileProvider) getMachineId() string {
	if _, err := os.Stat("/sys/firmware/dmi/tables/smbios_entry_point"); err == nil {
		dmit, err := smbios.NewDMITable()
		if err != nil {
			p.Logger.Warn().Msgf("Failed to get Machine Id from DMI table: %s. If running in a container, it needs to be privileged", err)
		} else {
			return dmit.Query(smbios.KeywordSystemUUID)
		}
	} else {
		p.Logger.Debug().Msg("DMI table not found. Falling back to /etc/machine-id file to get the host unique identifier")
	}

	b, err := os.ReadFile(filepath.Join(*cli.PathRootFs, "/etc/machine-id")) // just pass the file name
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*cli.PathRootFs, "/etc/machine-id"))
		fmt.Print(err)
		return ""
	}
	return strings.Trim(string(b), "\n")
}

func (p *FileProvider) getMachineLocation() []string {
	content, err := os.ReadFile(filepath.Join(*cli.PathRootFs, "/etc/machine-location"))
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*cli.PathRootFs, "/etc/machine-location"))
		return []string{"", ""}
	}
	return strings.Split(strings.Trim(string(content), "\n"), ":")
}
