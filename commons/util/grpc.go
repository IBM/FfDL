package util

import (
	log "github.com/sirupsen/logrus"
	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateClientDialOpts creates the TLC/non-TLS and other common dial options for
// establishing a grpc server connection to other microservices.
func CreateClientDialOpts() ([]grpc.DialOption, error) {
	var opts []grpc.DialOption
	if config.IsTLSEnabled() {
		creds, err := credentials.NewClientTLSFromFile(config.GetCAKey(), config.GetServerName())
		if err != nil {
			log.Errorf("Could not read TLS credentials: %v", err)
			return nil, err
		}
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithBlock()}
	} else {
		opts = []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}
	}
	return opts, nil
}
