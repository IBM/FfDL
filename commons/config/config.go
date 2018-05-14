/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"fmt"
	"runtime"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/units"
	"google.golang.org/grpc/grpclog"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	v1core "k8s.io/api/core/v1"
)

const (
	// PortKey is a key for looking up the Port setting in the viper config.
	PortKey = "port"
	// ObjectStoreKey is a key for looking up the ObjectStore setting in the viper config.
	ObjectStoreKey = "objectstore"
	// LogLevelKey is a key for looking up the LogLevel setting in the viper config.
	LogLevelKey = "loglevel"
	// EnvKey is a key for looking up the Environment setting in the viper config.
	EnvKey = "env"
	// DNSServerKey is a key for looking up the DNS server setting in the viper config.
	DNSServerKey = "dns_server"
	// ETCDEndpoints ...
	ETCDEndpoints = "etcd.address"
	//ETCDUsername ...
	ETCDUsername = "etcd.username"
	//ETCDPassword ...
	ETCDPassword = "etcd.password"
	//ETCDPrefix ...
	ETCDPrefix = "etcd.prefix"

	// TLSKey is the config key for looking up whether TLS is enabled.
	TLSKey = "tls"
	// ServerCertKey is the config key for the TLS server certificate.
	ServerCertKey = "server_cert"
	// ServerPrivateKey is the config key for the TLS server private key.
	ServerPrivateKey = "server_private_key"
	// ServerNameKey is the config key for the TLS cert server name.
	ServerNameKey = "server_name"
	// CACertKey is the config key for the TLS cert authority.
	CACertKey = "ca_cert"

	// DLaaS image tag key is the config key that indicates which DLAAS images to use
	DLaaSImageTagKey = "image_tag"

	// ServiceTagKey is the config key that indicates the namespace under which the services are deployed.
	ServicesTagKey = "image_tag"

	// LearnerTagKey is the config key that indicates which learner images to use.
	LearnerTagKey = "learner_tag"

	// DataBrokerTagKey is the config key that indicates which data broker images to use.
	DataBrokerTagKey = "databroker_tag"

	// IBMDockerRegistryKey is the config key for the docker registry to use for hybrid deployments.
	IBMDockerRegistryKey = "ibm_docker_registry"

	// LearnerRegistryKey is the config key for the docker registry to use.
	LearnerRegistryKey = "learner_registry"

	// LearnerImagePullSecretKey is the config key for docker registry pull secret
	LearnerImagePullSecretKey = "learner_image_pull_secret"

	//Use by LCM to know where it is deployed
	LCMDeploymentKey = "lcm_deployment"

	// envPrefix is the DLaaS prefix that viper uses for prefixing env variables (it is used upper case).
	envPrefix = "dlaas"

	// HybridEnv denotes a hybrid deployment on minikube and remote cluster. Used by LCM only
	HybridEnv = "hybrid"
	// LocalEnv denotes the value for the "local" environment configuration (non-Softlayer)
	LocalEnv = "local"
	// DevelopmentEnv denotes the value for the "development" SL environment configuration
	DevelopmentEnv = "development"
	// StagingEnv denotes the value for the "staging" SL environment configuration
	StagingEnv = "staging"
	// ProductionEnv denotes the value for the "production" SL environment configuration
	ProductionEnv = "production"

	LoggingType = "logging_type"

	LoggingTypeJson = "json"
	LoggingTypeText = "text"

	//PushMetricsEnabled whether push based metrics is enabled
	PushMetricsEnabled = "push_metrics_enabled"

	PodName = "pod.name"

	// PodNamespaceKey is the key to find the K8S namespace a pod is running in.
	PodNamespaceKey = "pod.namespace"

	// LearnerNamespaceKey is the key to find the K8S namespace learners are supposed to run in.
	LearnerKubeNamespaceKey = "learner.kube.namespace"

	// The path in the filesystem where the learner kubernetes cluster secrets are stored.
	learnerKubeSecretsRoot = "/var/run/secrets/learner-kube"

	learnerKubeCAFileKey    = "learner.kube.CAFile"
	learnerKubeKeyFileKey   = "learner.kube.KeyFile"
	learnerKubeCertFileKey  = "learner.kube.CertFile"
	learnerKubeTokenFileKey = "learner.kube.TokenFile"
	learnerKubeTokenKey     = "learner.kube.Token"
	learnerKubeURLKey       = "learner.kube.Url"

	// This is temporary until we support specifying storage requirements in the manifest.
	VolumeSize = "external_volume_size"

	// TODO this a duplication with the storage package - needs to align
	UsernameKey = "user_name"
	PasswordKey = "password"
	AuthURLKey  = "auth_url"
	DomainKey   = "domain_name"
	RegionKey   = "region"
	ProjectKey  = "project_id"
	StorageType = "type"

	DebugLearnerOptions = "learner.debug"
	// possible value for DebugLearnerOptions
	// learner pods should not be cleaned up
	NoCleanup = "nocleanup"

	DlaasResourceLimit          = "resource.limit"
	DlaasResourceLimitQuerySize = "resource.limit.query.size"

	ImagePullPolicy = "image_pull_policy"

	SharedVolumeStorageClassKey = "shared_volume_storage_class"
)

var viperInitOnce sync.Once

func init() {
	InitViper()
}

// InitViper is initializing the configuration system
func InitViper() {

	viperInitOnce.Do(func() {

		viper.SetEnvPrefix(envPrefix) // will be capitalized automatically
		viper.SetConfigType("yaml")

		viper.SetDefault(ImagePullPolicy, v1core.PullIfNotPresent)

		// Most likely be "standard" in Minikube and "ibmc-s3fs-standard" in DIND, (other value is "default" or "")
		viper.SetDefault(SharedVolumeStorageClassKey, "")

		// enable ENV vars and defaults
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
		viper.SetDefault(EnvKey, "prod")
		viper.SetDefault(LoggingType, "default")
		setLogLevel()
		viper.SetDefault(PortKey, 8080)

		viper.SetDefault(ServicesTagKey, "prod") // or "latest" as the default?

		viper.SetDefault(LearnerTagKey, "prod")
		viper.SetDefault(DataBrokerTagKey, "prod")
		viper.SetDefault(LearnerRegistryKey, "docker.io/ffdl")
		viper.SetDefault(LearnerImagePullSecretKey, "regcred")

		// TLS defaults for microservices
		viper.SetDefault(TLSKey, true)
		viper.SetDefault(ServerNameKey, "dlaas.ibm.com")

		// find certificates locally or in known place (in docker image)
		baseDir := getBaseDir()
		if _, err := os.Stat(baseDir); err == nil { // path exists so we assume we run locally
			viper.SetDefault(ServerCertKey, path.Join(baseDir, "certs", "server.crt"))
			viper.SetDefault(ServerPrivateKey, path.Join(baseDir, "certs", "server.key"))
			viper.SetDefault(CACertKey, path.Join(baseDir, "certs", "ca.crt"))
		} else {
			viper.SetDefault(ServerCertKey, "/etc/ssl/dlaas/server.crt")
			viper.SetDefault(ServerPrivateKey, "/etc/ssl/dlaas/server.key")
			viper.SetDefault(CACertKey, "/etc/ssl/dlaas/ca.crt")
		}

		// Fine Kubernetes cluster credentials in secrets mount by default.
		viper.SetDefault(learnerKubeCAFileKey, path.Join(learnerKubeSecretsRoot, "ca.crt"))
		viper.SetDefault(learnerKubeKeyFileKey, path.Join(learnerKubeSecretsRoot, "client.key"))
		viper.SetDefault(learnerKubeCertFileKey, path.Join(learnerKubeSecretsRoot, "client.crt"))
		viper.SetDefault(learnerKubeTokenFileKey, path.Join(learnerKubeSecretsRoot, "token"))

		viper.SetDefault(VolumeSize, "10GiB")

		// config file is optional. we usually configure via ENV_VARS
		configFile := fmt.Sprintf("config-%s", viper.Get(EnvKey))
		viper.SetConfigName(configFile) // name of config file (without extension)
		viper.AddConfigPath("/etc/dlaas/")
		viper.AddConfigPath("$HOME/.dlaas")
		viper.AddConfigPath(".")
		if viper.GetString(EnvKey) == LocalEnv {
			viper.AddConfigPath(getBaseDir())
		}

		err := viper.ReadInConfig()
		if err != nil {
			switch err := err.(type) {
			case viper.ConfigFileNotFoundError:
				log.Debugf("No config file '%s.yml' found. Using environment variables only.", configFile)
			case viper.ConfigParseError:
				log.Panicf("Cannot read config file: %s.yml: %s\n", configFile, err)
			}
		} else {
			log.Debugf("Loading config from file %s\n", viper.ConfigFileUsed())
		}
	})

	grpclog.SetLogger(log.StandardLogger())
}

// GetInt returns the int value for the config key.
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetInt64 returns the int64 value for the config key.
func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

// GetFloat64 returns the float64 value for the config key.
func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

// GetString returns the string value for the config key.
func GetString(key string) string {
	return viper.GetString(key)
}

// SetDefault sets the default value for the config key.
func SetDefault(key string, value interface{}) {
	viper.SetDefault(key, value)
}

// Return the contents of a file. Return an empty string if the file doesn't exist or can't be read.
func GetFileContents(filename string) string {
	contents := ""
	if filename != "" {
		data, err := ioutil.ReadFile(filename)
		if err == nil {
			contents = string(data)
		}
	}
	return contents
}

// IsTLSEnabled is true if the microservices should all use TLS for communication, otherwise
// it is false.
func IsTLSEnabled() bool {
	return viper.GetBool(TLSKey)
}

// GetServerCert gets the server's TLS certificate file name.
func GetServerCert() string {
	return viper.GetString(ServerCertKey)
}

// GetServerPrivateKey gets the private key file name of the server.
func GetServerPrivateKey() string {
	return viper.GetString(ServerPrivateKey)
}

// GetCAKey gets the CA certificate file name for clients to use
func GetCAKey() string {
	return viper.GetString(CACertKey)
}

// GetServerName gets the server name that is encoded in the TLS certificate.
func GetServerName() string {
	return viper.GetString(ServerNameKey)
}

// DisableDNSServer disables the DNS server in the global config. This is
// usefule for testing.
func DisableDNSServer() {
	viper.Set(DNSServerKey, "disabled")
}

// GetValue returns the value associated with the given config key.
func GetValue(key string) string {
	return viper.GetString(key)
}

// FatalOnAbsentKey fails if the key is absent.
func FatalOnAbsentKey(key string) {
	if !viper.IsSet(key) {
		log.Fatalf("Please set config key %s or env var %s correctly.", key, configKey2EnvVar(key))
	}
}

// FatalOnAbsentKeyInMap fails if the key is absent in the conf map.
func FatalOnAbsentKeyInMap(key string, conf map[string]string) {
	if _, ok := conf[key]; !ok {
		log.Fatalf("Please set config key %s or env var %s correctly.", key, configKey2EnvVar(key))
	}
}

func GetPodName() string {
	if viper.IsSet(PodName) {
		return viper.GetString(PodName)
	}
	return "NOT_FOUND"
}

// GetPodNamespace gets the namespace of a POD or returns the default one.
func GetPodNamespace() string {
	if viper.IsSet(PodNamespaceKey) {
		return viper.GetString(PodNamespaceKey)
	}
	return "default"
}
func GetPodNamespaceForPrometheus() string {
	namespace := "default"
	if viper.IsSet(PodNamespaceKey) {
		namespace = viper.GetString(PodNamespaceKey)
	}
	namespace = strings.Replace(namespace, "-", "", -1)
	return namespace
}

// GetLearnerNamespace gets the namespace where the learners should run.
func GetLearnerNamespace() string {
	if viper.IsSet(LearnerKubeNamespaceKey) {
		return viper.GetString(LearnerKubeNamespaceKey)
	}
	return "default"
}

//CheckPushGatewayEnabled ... for sending out metrics
func CheckPushGatewayEnabled() bool {
	if viper.IsSet(PushMetricsEnabled) {
		return viper.GetBool(PushMetricsEnabled)
	}
	return false
}

// FatalOnAbsentKeysets fatals if either set of keys is not set correctly.
// (i.e., keys of `keyset1` xor `keyset2` are not set)
func FatalOnAbsentKeysets(keyset1 []string, keyset2 []string) {
	keySetMsg1 := ""
	keysSet1Defined := true
	for _, v := range keyset1 {
		if !viper.IsSet(v) {
			keysSet1Defined = false
		}
		keySetMsg1 = fmt.Sprintf("%s\t %s\n", keySetMsg1, configKey2EnvVar(v))
	}
	keySetMsg2 := ""
	keysSet2Defined := true
	for _, v := range keyset2 {
		if !viper.IsSet(v) {
			keysSet2Defined = false
		}
		keySetMsg2 = fmt.Sprintf("%s\t %s\n", keySetMsg2, configKey2EnvVar(v))
	}

	msg := fmt.Sprintf("Please set the following config keys or env vars:\n %s\nOR\n%s", keySetMsg1, keySetMsg2)
	if !keysSet1Defined && !keysSet2Defined {
		log.Fatal(msg)
	}
}

// GetDataStoreConfig returns the configuration for the internal data store
// used to store model configuration, trained models, etc.
// TODO rename this config to data_store later
func GetDataStoreConfig() map[string]string {
	// FR: GetStringMapString cannot be used with ENV variables overrides
	// viper.GetStringMapString("objectstore") // TODO use constant
	m := make(map[string]string, 7)
	val := viper.GetString("objectstore." + UsernameKey)
	if val != "" {
		m[UsernameKey] = val
	}
	val = viper.GetString("objectstore." + PasswordKey)
	if val != "" {
		m[PasswordKey] = val
	}
	val = viper.GetString("objectstore." + AuthURLKey)
	if val != "" {
		m[AuthURLKey] = val
	}
	val = viper.GetString("objectstore." + DomainKey)
	if val != "" {
		m[DomainKey] = val
	}
	val = viper.GetString("objectstore." + UsernameKey)
	if val != "" {
		m[UsernameKey] = val
	}
	val = viper.GetString("objectstore." + ProjectKey)
	if val != "" {
		m[ProjectKey] = val
	}
	val = viper.GetString("objectstore." + StorageType)
	if val != "" {
		m[StorageType] = val
	}
	return m
}

// GetDataStoreType returns the configuration type for the internal data store.
func GetDataStoreType() string {
	return viper.GetString("objectstore.type")
}

// GetVolumeSize returns the size (in bytes) of the external volume to use. If 0, don't use an external volume.
func GetVolumeSize() int64 {
	size := viper.GetString(VolumeSize)
	// First, try to parse number of bytes (e.g., "111222333")
	bytes, err := strconv.ParseInt(size, 10, 0)
	if err != nil {
		// Next, try to parse with unit suffix (e.g,. "10GiB").
		bytes, err = units.ParseStrictBytes(size)
		if err != nil {
			bytes = 0
		}
	}
	return bytes
}

//GetResourceLimit ...
func GetResourceLimit() int {
	return viper.GetInt(DlaasResourceLimit)
}

//GetResourceLimitQuerySize ...
func GetResourceLimitQuerySize() int {
	return viper.GetInt(DlaasResourceLimitQuerySize)
}

func configKey2EnvVar(key string) string {
	r := strings.NewReplacer(".", "_", "-", "_")
	return strings.ToUpper(fmt.Sprintf("%s_%s", envPrefix, r.Replace(key)))
}

// setLogLevel sets the logging level based on the environment
func setLogLevel() {
	viper.SetDefault(LogLevelKey, "warn")

	env := viper.GetString(EnvKey)
	if env == "dev" || env == "test" {
		viper.Set(LogLevelKey, "debug")
	}

	if logLevel, err := log.ParseLevel(viper.GetString(LogLevelKey)); err == nil {
		log.SetLevel(logLevel)
	}

	log.Debugf("Log level set to '%s'", viper.Get(LogLevelKey))
}

func getBaseDir() string {
	baseDir := path.Join(os.Getenv("GOPATH"), "src", "github.com/IBM/FfDL/commons")
	if _, err := os.Stat(baseDir); err == nil {
		return baseDir
	}

	// NOTE: We cannot rely on GOPATH here anymore because the base dir needs to refer to dlaas-commons which is located
	// in the vendor/ folder. We use a heuristic here and jump up to three levels up to find the vendor/.. folder.
	cwd, _ := os.Getwd()
	baseDir = ""
	for i := 0; i < 3; i++ {
		baseDir = path.Join(cwd, strings.Repeat("../", i)+"vendor", "github.com", "IBM", "FfDL", "commons")
		if _, err := os.Stat(baseDir); err == nil {
			return baseDir
		}
	}
	return baseDir
}

// GetDebugLearnerOptions returns comma separated list of options for debugging learners
func GetDebugLearnerOptions() string {
	return viper.GetString(DebugLearnerOptions)
}

//GetEtcdEndpoints ...
func GetEtcdEndpoints() []string {
	return strings.Split(viper.GetString(ETCDEndpoints), ",")
}

//GetEtcdUsername ...
func GetEtcdUsername() string {
	return viper.GetString(ETCDUsername)
}

//GetEtcdPassword ...
func GetEtcdPassword() string {
	return viper.GetString(ETCDPassword)
}

//GetEtcdPrefix ...
func GetEtcdPrefix() string {
	return viper.GetString(ETCDPrefix)
}

//GetEtcdCertLocation ...
func GetEtcdCertLocation() string {
	return getFileAtLocation("/etc/certs/etcd/etcd.cert") //the file should have been mounted at this path as a part of secrets
}

//GetMongoCertLocation ...
func GetMongoCertLocation() string {
	return getFileAtLocation("/etc/certs/mongo/mongo.cert") //the file should have been mounted at this path as a part of secrets
}

//Get LearnerKubeURLKey...
func GetLearnerKubeURL() string {
	return viper.GetString(learnerKubeURLKey)
}

//Get LearnerKubeCAFileKey...
func GetLearnerKubeCAFile() string {
	return viper.GetString(learnerKubeCAFileKey)
}

//Get LearnerKubetokenKey...
func GetLearnerKubeToken() string {
	return viper.GetString(learnerKubeTokenKey)
}

//Get LearnerKubeTokenFileKey
func GetLearnerKubeTokenFile() string {
	return viper.GetString(learnerKubeTokenFileKey)
}

//Get LearnerKubeKeyFileKey
func GetLearnerKubeKeyFile() string {
	return viper.GetString(learnerKubeKeyFileKey)
}

//Get LearnerKubeCertFileKey
func GetLearnerKubeCertFile() string {
	return viper.GetString(learnerKubeCertFileKey)
}

func GetCurrentLearnerConfigLocationFromCombination(nameversion string) string {
	learnerConfigDir := "/etc/learner-config"                          // default directory
	dir, dirPresent := os.LookupEnv("DLAAS_LEARNER_CONFIG_MAPPED_DIR") // may override default mapping for testing
	if dirPresent {
		learnerConfigDir = dir
	}

	return getFileAtLocation(fmt.Sprintf("%s/%s_CURRENT", learnerConfigDir, nameversion))
}

func GetCurrentLearnerConfigLocation(name string, version string) string {
	loc := GetCurrentLearnerConfigLocationFromCombination(fmt.Sprintf("%s_gpu_%s", name, version))
	if loc == "" {
		return GetCurrentLearnerConfigLocationFromCombination(fmt.Sprintf("%s_cpu_%s", name, version))
	} else {
		return loc
	}
}

func LogStackTrace() {
	pc := make([]uintptr, 30)
	stackDepth := runtime.Callers(0, pc)
	for i := 0; i < stackDepth; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])
		// Truncate package name
		funcName := f.Name()
		if slash := strings.LastIndex(funcName, "."); slash >= 0 {
			funcName = funcName[slash+1:]
		}
		locString := fmt.Sprintf("%s:%d %s -", file, line, funcName)
		log.Debugf("   ---> %s", locString)
	}
}

//return the location back if the file was present otherwise an empty string
func getFileAtLocation(location string) string {
	exists := func(filename string) bool {
		if _, err := os.Stat(filename); err != nil {
			if os.IsNotExist(err) {
				return false
			}
		}
		return true
	}
	if exists(location) {
		log.Debugf("file was found at location %s", location)
		return location
	}
	//log.Debugf("file certificate was missing at location %s", location)
	//LogStackTrace()

	return "" //empty location means that cert is not required
}

//GetPushgatewayURL ...
func GetPushgatewayURL() string {
	return fmt.Sprintf("http://pushgateway:%s", "9091")
}
