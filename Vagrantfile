
#
# Copyright 2017-2018 IBM Corporation
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

repo_url = 'https://github.com/pires/kubernetes-vagrant-coreos-cluster'
base_dir = '/tmp/kubernetes.vagrant.setup'

if !File.exists?(base_dir)
  `git clone #{repo_url} #{base_dir}`
end

# config values
ENV['NODES'] = '1'
ENV['MASTER_MEM'] = '900'
ENV['MASTER_CPUS'] = '1'
ENV['NODE_MEM'] = '2800'
ENV['NODE_CPUS'] = '3'

# forward the Vagrant command to the Kubernetes Vagrant folder
Dir.chdir(base_dir)
system("cd #{base_dir} && vagrant '#{ARGV.join("' '")}'")

# need to terminate the process here, because of the way we're importing the parent Vagrant box
Kernel.exit(0)
