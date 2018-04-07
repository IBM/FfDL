<!--
{% comment %}
Copyright 2017-2018 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
{% endcomment %}
-->

# Health Checker for gRPC

This is a simple CLI utility for checking if a gRPC service is healthy. It assumes
that the gRPC service provides the [Health Checking Service](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

This utility will connect to the above service and the following error codes:
* exit code 0: `status == HealthCheckResponse_SERVING`
* exit code 1: an error occurred connecting to the service or retrieving a health check response
* exit code 2: `status == HealthCheckResponse_UNKNOWN`
* exit code 3: `status == HealthCheckResponse_NOT_SERVING`

Generally speaking, any error code != 0 means the service is not healthy. The error code mapping
just helps to classify it a bit further, if needed.

# Compiling

Install dependencies:
```
make install-deps
```

Binary for your local machine:
```
make build-local
```

Static binary for x86_64 arch:
```
make build-x86-64:
```


# Running

```
grpc-health-checker -h
Usage of ./bin/grpc-health-checker:
  -help|h
	Print usage
  -host string
	Health endpoint host. (default "localhost")
  -p|port int
	Health endpoint port. (default 10000)
  -s|service string
	Service name to check.
  -t|timeout duration
	Timeout in Milliseconds (ms). (default 5µs)
```
