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
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"telemetruum/leaf-exporter/cli"

	"github.com/rs/zerolog"
)

type SystemProvider struct {
	BaseProvider
}

var (
	ipHint = cli.Serve.Flag("ip-hint", "An ip:port to use to help identify the device's ip (the specified endpoint is never called)").Default("8.8.8.8:80").String()
)

func (p *SystemProvider) Start(context.Context, *sync.WaitGroup) {

}

/*
func (p *SystemProvider) ProvideWorkloadInfoLabels(ctx context.Context, wic *WorkloadInfoCollector) {

	b, err := os.ReadFile(filepath.Join(*pathRootFs, "/etc/machine-id")) // just pass the file name
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*pathRootFs, "/etc/machine-id"))
		fmt.Print(err)
	}

	wic.HostId = strings.Trim(string(b), "\n")
}
*/

func (p *SystemProvider) ProvideHostInfo(ctx context.Context, hic *HostInfoCollector) {
	p.Logger.Debug().Msg("Collecting host metrics")

	hostname, err := os.Hostname()
	if err != nil {
		p.Logger.Fatal().Msg(err.Error())
	}

	hic.Os = runtime.GOOS
	hic.Arch = runtime.GOARCH
	hic.Ip = p.getOutboundIP(p.Logger).String()
	hic.Hostname = hostname
}

// Get preferred outbound ip of this machine
func (p *SystemProvider) getOutboundIP(logger zerolog.Logger) net.IP {

	// the ipHint can be an IP or an URL (with or without schema and port)
	// 1.1.1.1, 1.1.1.1:100, google.com/test, google.com:443, https://google.com
	// all will be accepted.
	// However we just need the hostname and the port to test the network interface
	// so we try to extract it from ipHint string
	urlToParse := *ipHint
	if !strings.Contains(*ipHint, "://") {
		urlToParse = fmt.Sprintf("http://%s", *ipHint)
	}
	u, err := url.Parse(urlToParse)
	if err != nil {
		panic(err)
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:80", host)
	}

	logger.Debug().Msgf("Using %s to test outgoing network interface (the url will never be really called)", host)
	conn, err := net.Dial("udp", host)

	if err != nil {
		logger.Error().Msg(err.Error())
		logger.Debug().Msg("Trying with 1.1.1.1:53")
		conn, err = net.Dial("udp", "1.1.1.1:53")
		if err != nil {
			logger.Fatal().Msg(err.Error())
		}
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
