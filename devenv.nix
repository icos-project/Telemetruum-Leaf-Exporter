#
# ICOS Telemetruum Leaf Exporter
# Copyright © 2022 - 2025 Engineering Ingegneria Informatica S.p.A.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
# http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# 
# This work has received funding from the European Union's HORIZON research
# and innovation programme under grant agreement No. 101070177.
#

{ pkgs, ... }:
{
  languages.go.enable = true;

  scripts.build.exec = ''
    go build -o ./output/telemetruum-leaf-exporter
  '';

  scripts.run-dev-187.exec = ''
    POD_NAME=devpod NAMESPACE=icos-system NODE_NAME=node1 go run . --kube-config=/data/ICOS/infra/workspace-icos/ncsrd/staging/cluster_187/.credentials/cluster.kubeconfig --path-rootfs `pwd`/testrootfs --no-docker
  '';

  scripts.run-dev.exec = ''
    go run . --path-rootfs `pwd`/testrootfs
  '';
}
