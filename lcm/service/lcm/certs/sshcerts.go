package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"golang.org/x/crypto/ssh"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//GenerateSSHCertAsK8sSecret ...
func GenerateSSHCertAsK8sSecret(secretName, trainingID, framework, version string) (*v1core.Secret, error) {
	if needsMountedSSHCerts(framework, version) {

		log.Infof("provisioning mounted secret with framework %s and version %s", framework, version)
		tmp, err := ioutil.TempDir("", trainingID)
		if err != nil {
			log.WithError(err).Errorf("failed to create temp dir for certs for training %s", trainingID)
			return nil, err
		}
		publicKeyFilePath := fmt.Sprintf("%s/public.pub", tmp)
		privateKeyFilePath := fmt.Sprintf("%s/private.pem", tmp)
		defer os.RemoveAll(tmp) //delete the folder once done
		if err := generatePublicPrivateKeyPair(publicKeyFilePath, privateKeyFilePath); err != nil {
			log.WithError(err).Errorf("failed to generate public private key for %s , %s", publicKeyFilePath, privateKeyFilePath)
			return nil, err
		}

		publicKeyBytes, err := ioutil.ReadFile(publicKeyFilePath)
		if err != nil {
			log.WithError(err).Errorf("failed to read in public cert for training %s", publicKeyFilePath)
			return nil, err
		}
		privateKeyBytes, err := ioutil.ReadFile(privateKeyFilePath)
		if err != nil {
			log.WithError(err).Errorf("failed to read in private cert for training %s", privateKeyFilePath)
			return nil, err
		}
		secret := provisionSSHCertsAsK8sSecret(secretName, publicKeyBytes, privateKeyBytes, map[string]string{"training_id": trainingID})
		return secret, nil

	}

	return nil, nil //nil, nil if the no provisioning is needed
}

// ------------- private functions -------------- //

//generatePublicPrivateKeyPair ...
func generatePublicPrivateKeyPair(publicKeyFilePath, privateKeyFilePath string) error {

	privateKeyPEMOnDisk, err := os.Create(privateKeyFilePath)
	if err != nil {
		log.WithError(err).Errorf("failed to create file for private cert at path %s", privateKeyFilePath)
		return err
	}
	defer privateKeyPEMOnDisk.Close()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.WithError(err).Errorf("unexpected error, failed when generating key")
		return err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(privateKeyPEMOnDisk, privateKeyPEM); err != nil {
		log.WithError(err).Errorf("failed to encode private key")
		return err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.WithError(err).Errorf("failed to create the public key for private key at path %s", privateKeyFilePath)
		return err
	}

	if err := ioutil.WriteFile(publicKeyFilePath, ssh.MarshalAuthorizedKey(pub), 0655); err != nil {
		log.WithError(err).Errorf("failed to create the public key at path %s", publicKeyFilePath)
		return err
	}

	return nil
}

//provisionSSHCertsAsK8sSecret ...
func provisionSSHCertsAsK8sSecret(secretName string, publicKey, privateKey []byte, labels map[string]string) *v1core.Secret {

	secret := &v1core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: config.GetLearnerNamespace(),
			Labels:    labels,
		},
		Type: "generic",
		Data: map[string][]byte{
			"ssh-privatekey": privateKey,
			"ssh-publickey":  publicKey,
		},
	}

	return secret
}

func needsMountedSSHCerts(framework, version string) bool {
	result := false
	if strings.EqualFold(framework, "tensorflow") && (strings.HasSuffix(version, "horovod") || strings.HasSuffix(version, "ddl")) {
		result = true
	}
	return result
}
