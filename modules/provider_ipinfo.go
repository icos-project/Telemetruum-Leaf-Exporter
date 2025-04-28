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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/rs/zerolog"
)

type IPInfoProvider struct {
	BaseProvider
}

func (p *IPInfoProvider) Start(context.Context, *sync.WaitGroup) {

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

func (p *IPInfoProvider) ProvideHostInfo(ctx context.Context, hic *HostInfoCollector) {
	p.setLocationIPApiCom(p.Logger, hic)
}

func (p *IPInfoProvider) setLocationIPApiCom(logger zerolog.Logger, hic *HostInfoCollector) {
	logger.Debug().Msg("Probing location calling ip-api.com")
	resp, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		logger.Error().Msgf("Error calling ipapi: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		logger.Warn().Msg("ip-api.com rate limit reached.")
		return
	}
	body, _ := io.ReadAll(resp.Body) // response body is []byte

	type IpAPIResponse struct {
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lon"`
	}
	var result IpAPIResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	hic.Latitutde = strconv.FormatFloat(result.Latitude, 'f', -1, 64)
	hic.Longitude = strconv.FormatFloat(result.Longitude, 'f', -1, 64)
	logger.Debug().Msgf("Location from ip-api.com set to %s:%s", hic.Latitutde, hic.Longitude)

}

/*
func (p *IPInfoProvider) setLocationIPApi(logger zerolog.Logger, hic *HostInfoCollector) {
	logger.Debug().Msg("Probing location calling ipapi.co")
	resp, err := http.Get("https://ipapi.co/json/")
	if err != nil {
		logger.Error().Msgf("Error calling ipapi: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		logger.Warn().Msg("ipapi rate limit reached.")
		return
	}
	body, _ := io.ReadAll(resp.Body) // response body is []byte

	type IpAPIResponse struct {
		// the full response from ipinfo contains additional fields:
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	}
	var result IpAPIResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	hic.Latitutde = result.Latitude
	hic.Longitude = result.Longitude
	logger.Debug().Msgf("Location from ipinfo set to %s:%s", hic.Latitutde, hic.Longitude)

}


// OTHER IMPLEMENTATION TO GET THE LOCATION THAT USE DIFFERENT ONLINE SERVICES

func (p *IPInfoProvider) setLocationIPInfo(logger zerolog.Logger, hic *HostInfoCollector) {
	logger.Debug().Msg("Probing location calling ipinfo.io")
	resp, err := http.Get("https://ipinfo.io/")
	if err != nil {
		logger.Error().Msgf("Error calling ipinfo: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		logger.Warn().Msg("ipinfo rate limit reached.")
		return
	}
	body, _ := io.ReadAll(resp.Body) // response body is []byte

	type IpInfoResponse struct {
		// the full response from ipinfo contains additional fields:
		//
		//	{
		//	"ip": "91.109.56.214",
		//	"city": "Prato",
		//	"region": "Tuscany",
		//	"country": "IT",
		//	"loc": "43.8805,11.0970",
		//	"org": "AS21176 Engineering D.HUB S.p.A.",
		//	"postal": "59100",
		//	"timezone": "Europe/Rome",
		//	"readme": "https://ipinfo.io/missingauth"
		//	}
		//
		Location string `json:"loc"`
	}
	var result IpInfoResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	tokens := strings.Split(result.Location, ",")

	if len(tokens) == 2 {
		hic.Latitutde = tokens[0]
		hic.Longitude = tokens[1]
		logger.Debug().Msgf("Location from ipinfo set to %s:%s", hic.Latitutde, hic.Longitude)
	} else {
		logger.Error().Msgf("Error parsing location from ipinfo: %s", result.Location)
	}
}

func (p *SystemProvider) setLocation(logger zerolog.Logger, hic *HostInfoCollector) {
	logger.Debug().Msg("Probing location calling ipinfo.io")
	resp, err := http.Get("https://ipinfo.io/")
	if err != nil {
		logger.Error().Msgf("Error calling ipinfo: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		logger.Warn().Msg("ipinfo rate limit reached.")
		return
	}
	body, err := io.ReadAll(resp.Body) // response body is []byte

	type IpInfoResponse struct {
		// the full response from ipinfo contains additional fields:
		Location string `json:"loc"`
	}
	var result IpInfoResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	tokens := strings.Split(result.Location, ",")

	if len(tokens) == 2 {
		hic.Latitutde = tokens[0]
		hic.Longitude = tokens[1]
		logger.Debug().Msgf("Location from ipinfo set to %s:%s", hic.Latitutde, hic.Longitude)
	} else {
		logger.Error().Msgf("Error parsing location from ipinfo: %s", result.Location)
	}
}

*/
