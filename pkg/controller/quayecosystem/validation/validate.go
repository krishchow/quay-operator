package validation

import (
	"context"
	"fmt"
	"reflect"

	"time"

	redhatcopv1alpha1 "github.com/redhat-cop/quay-operator/pkg/apis/redhatcop/v1alpha1"

	"github.com/redhat-cop/quay-operator/pkg/controller/quayecosystem/constants"
	"github.com/redhat-cop/quay-operator/pkg/controller/quayecosystem/logging"
	"github.com/redhat-cop/quay-operator/pkg/controller/quayecosystem/resources"
	"github.com/redhat-cop/quay-operator/pkg/controller/quayecosystem/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Validate performs validation across all resources
func Validate(client client.Client, quayConfiguration *resources.QuayConfiguration) (bool, error) {

	// Validate Initial Superuser Credentials Secret
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.SuperuserCredentialsSecretName) && !quayConfiguration.QuayEcosystem.Spec.Quay.SkipSetup && !quayConfiguration.QuayEcosystem.Status.SetupComplete {

		validQuaySuperuserSecret, superuserSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.SuperuserCredentialsSecretName, constants.DefaultQuaySuperuserCredentials)

		if err != nil {
			return false, err
		}

		if !validQuaySuperuserSecret {
			return false, fmt.Errorf("Failed to validate provided Quay Superuser Secret")
		}

		quayConfiguration.InitialQuaySuperuserEmail = string(superuserSecret.Data[constants.InitialQuaySuperuserEmailKey])
		quayConfiguration.InitialQuaySuperuserUsername = string(superuserSecret.Data[constants.InitialQuaySuperuserUsernameKey])
		quayConfiguration.InitialQuaySuperuserPassword = string(superuserSecret.Data[constants.InitialQuaySuperuserPasswordKey])
		quayConfiguration.ValidProvidedInitialQuaySuperuserSecret = true
	}

	if len(quayConfiguration.InitialQuaySuperuserPassword) < 8 {
		return false, fmt.Errorf("Quay Superuser Password Must Be At Least 8 Characters in Length")
	}

	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.ConfigSecretName) {

		validQuayConfigSecret, quayConfigSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.ConfigSecretName, constants.DefaultQuayConfigCredentials)

		if err != nil {
			return false, err
		}

		if !validQuayConfigSecret {
			return false, fmt.Errorf("Failed to validate provided Quay Config Secret")
		}

		quayConfiguration.QuayConfigPassword = string(quayConfigSecret.Data[constants.QuayConfigPasswordKey])
		quayConfiguration.QuayConfigPasswordSecret = quayConfiguration.QuayEcosystem.Spec.Quay.ConfigSecretName
		quayConfiguration.ValidProvidedQuayConfigPasswordSecret = true

	}

	// Validate Quay ImagePullSecret
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.ImagePullSecretName) {

		validImagePullSecret, _, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.ImagePullSecretName, nil)

		if err != nil {
			return false, err
		}

		if !validImagePullSecret {
			return false, fmt.Errorf("Failed to validate provided Quay Image Pull Secret")
		}

	}

	// Validate Redis ImagePullSecret
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Redis.ImagePullSecretName) {

		validImagePullSecret, _, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Redis.ImagePullSecretName, nil)

		if err != nil {
			return false, err
		}

		if !validImagePullSecret {
			return false, fmt.Errorf("Failed to validate provided Redis Image Pull Secret")
		}
	}

	// Validate Redis CredentialsSecret
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Redis.CredentialsSecretName) {

		validRedisCredentialSecret, redisSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Redis.CredentialsSecretName, []string{constants.RedisPasswordKey})

		if err != nil {
			return false, err
		}

		if !validRedisCredentialSecret {
			return false, fmt.Errorf("Failed to validate provided Redis Credentials Secret")
		}

		quayConfiguration.RedisPassword = string(redisSecret.Data[constants.RedisPasswordKey])
		quayConfiguration.ValidProvidedRedisPasswordSecret = true
	}

	// Validate Quay Database ImagePullSecret
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database.ImagePullSecretName) {

		validImagePullSecret, _, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.Database.ImagePullSecretName, nil)

		if err != nil {
			return false, err
		}

		if !validImagePullSecret {
			return false, fmt.Errorf("Failed to validate provided Data Database Image Pull Secret")
		}
	}

	// Validate Quay Database Credential
	if !quayConfiguration.QuayEcosystem.Spec.Quay.SkipSetup {
		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database) && !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database.Server) && utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database.CredentialsSecretName) {
			return false, fmt.Errorf("Failed to locate a Quay Database Credential for Externally Provisioned Instance")
		}

		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database) && !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database.CredentialsSecretName) {

			validQuayDatabaseSecret, databaseSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.Database.CredentialsSecretName, constants.RequiredDatabaseCredentialKeys)

			if err != nil {
				return false, err
			}

			if !validQuayDatabaseSecret {
				return false, fmt.Errorf("Failed to validate provided Quay Database Secret")
			}

			quayConfiguration.QuayDatabase.Username = string(databaseSecret.Data[constants.DatabaseCredentialsUsernameKey])
			quayConfiguration.QuayDatabase.Password = string(databaseSecret.Data[constants.DatabaseCredentialsPasswordKey])
			quayConfiguration.QuayDatabase.Database = string(databaseSecret.Data[constants.DatabaseCredentialsDatabaseKey])

			if _, found := databaseSecret.Data[constants.DatabaseCredentialsServerKey]; found {
				quayConfiguration.QuayEcosystem.Spec.Quay.Database.Server = string(databaseSecret.Data[constants.DatabaseCredentialsServerKey])
				quayConfiguration.QuayDatabase.Server = quayConfiguration.QuayEcosystem.Spec.Quay.Database.Server
				quayConfiguration.QuayEcosystem.Spec.Quay.Database.Image = ""
				quayConfiguration.QuayEcosystem.Spec.Quay.Database.DeploymentStrategy = ""
				quayConfiguration.QuayEcosystem.Spec.Quay.Database.ReadinessProbe = nil
				quayConfiguration.QuayEcosystem.Spec.Quay.Database.LivenessProbe = nil
			}

			if _, found := databaseSecret.Data[constants.DatabaseCredentialsRootPasswordKey]; found {
				quayConfiguration.QuayDatabase.RootPassword = string(databaseSecret.Data[constants.DatabaseCredentialsRootPasswordKey])
			}

			quayConfiguration.ValidProvidedQuayDatabaseSecret = true
		}
	}

	// Validate Quay Database
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.Database.VolumeSize) {

		_, err := resource.ParseQuantity(quayConfiguration.QuayEcosystem.Spec.Quay.Database.VolumeSize)

		if err != nil {
			return false, err
		}
	}

	// Quay PVC Generation
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.RegistryStorage) {

		_, err := resource.ParseQuantity(quayConfiguration.QuayEcosystem.Spec.Quay.RegistryStorage.PersistentVolumeSize)

		if err != nil {
			return false, err
		}

	}

	// Validate Quay Config Files
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.ConfigFiles) {

		quayConfigFiles, err := validateConfigFiles(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.ConfigFiles)

		if err != nil {
			return false, err
		}

		quayConfiguration.QuayConfigFiles = append(quayConfiguration.QuayConfigFiles, quayConfigFiles...)

	}

	// Validate Hostname Provided if NodePort external access
	if (redhatcopv1alpha1.NodePortExternalAccessType == quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.Type || redhatcopv1alpha1.IngressExternalAccessType == quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.Type) && utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.Hostname) {
		return false, fmt.Errorf("Cannot use %s External Access Type Without Hostname Defined", quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.Type)
	}

	// Validate Route not specified when not running in OpenShift
	if redhatcopv1alpha1.RouteExternalAccessType == quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.Type && !quayConfiguration.IsOpenShift {
		return false, fmt.Errorf("Cannot use 'Route` as External Access Type when not running in OpenShift")
	}

	// Registry Backends
	for _, registryBackend := range quayConfiguration.QuayEcosystem.Spec.Quay.RegistryBackends {

		// Validate replication is not enabled when using a Local backend
		if quayConfiguration.QuayEcosystem.Spec.Quay.EnableStorageReplication {
			if registryBackend.Local != nil {
				return false, fmt.Errorf("Cannot have make use of local storage when replication enabled. Local storage: %s", registryBackend.Name)
			}
		}

		managedRegistryBackend := registryBackend.DeepCopy()

		// Validate S3 backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.S3) {

			if managedRegistryBackend.S3.StoragePath == "" || managedRegistryBackend.S3.BucketName == "" {
				return false, fmt.Errorf("Failed to validate required properties for registry backend. Name: %s", managedRegistryBackend.Name)
			}

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {
				validS3Secret, s3Secret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredS3CredentialKeys)

				if err != nil {
					return false, err
				}
				if !validS3Secret {
					return false, fmt.Errorf("Failed to validate required credentials secret name for the provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.S3.AccessKey = string(s3Secret.Data[constants.S3AccessKey])
				managedRegistryBackend.S3.SecretKey = string(s3Secret.Data[constants.S3SecretKey])
				managedRegistryBackend.CredentialsSecretName = ""

			}

		}

		// Validate Azure backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.Azure) {

			if managedRegistryBackend.Azure.StoragePath == "" || managedRegistryBackend.Azure.ContainerName == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {
				validAzureSecret, azureSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredAzureCredentialKeys)

				if err != nil {
					return false, err
				}
				if !validAzureSecret {
					return false, fmt.Errorf("Failed to validate required credentials secret name for the provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.Azure.AccountName = string(azureSecret.Data[constants.AzureAccountName])
				managedRegistryBackend.Azure.AccountKey = string(azureSecret.Data[constants.AzureAccountKey])

				if _, found := azureSecret.Data[constants.AzureSasToken]; found {
					managedRegistryBackend.Azure.SasToken = string(azureSecret.Data[constants.AzureSasToken])
				}

				managedRegistryBackend.CredentialsSecretName = ""

			}

		}

		// Validate Google Cloud backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.GoogleCloud) {

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {

				validGoogleCloudSecret, googleCloudSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredGoogleCloudCredentialKeys)

				if err != nil {
					return false, err
				}
				if !validGoogleCloudSecret {
					return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.GoogleCloud.AccessKey = string(googleCloudSecret.Data[constants.GoogleCloudAccessKey])
				managedRegistryBackend.GoogleCloud.SecretKey = string(googleCloudSecret.Data[constants.GoogleCloudAccessKey])

				managedRegistryBackend.CredentialsSecretName = ""

			}

			if managedRegistryBackend.GoogleCloud.StoragePath == "" || managedRegistryBackend.GoogleCloud.BucketName == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

		}

		// Validate RHOCS backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.RHOCS) {

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {

				validRHOCSSecret, RHOCSSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredRHOCSCredentialKeys)

				if err != nil {
					return false, err
				}
				if !validRHOCSSecret {
					return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.RHOCS.AccessKey = string(RHOCSSecret.Data[constants.RHOCSAccessKey])
				managedRegistryBackend.RHOCS.SecretKey = string(RHOCSSecret.Data[constants.RHOCSSecretKey])

				managedRegistryBackend.CredentialsSecretName = ""

			}

			if managedRegistryBackend.RHOCS.StoragePath == "" || managedRegistryBackend.RHOCS.BucketName == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

		}

		// Validate RADOS backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.RADOS) {

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {

				validRADOSSecret, RADOSSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredRADOSCredentialKeys)

				if err != nil {
					return false, err
				}
				if !validRADOSSecret {
					return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.RADOS.AccessKey = string(RADOSSecret.Data[constants.RADOSAccessKey])
				managedRegistryBackend.RADOS.SecretKey = string(RADOSSecret.Data[constants.RADOSSecretKey])

				managedRegistryBackend.CredentialsSecretName = ""

			}

			if managedRegistryBackend.RADOS.StoragePath == "" || managedRegistryBackend.RADOS.BucketName == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

		}

		// Validate Swift backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.Swift) {

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {

				validSwiftSecret, SwiftSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredSwiftCredentialKeys)

				if err != nil {
					return false, err
				}
				if !validSwiftSecret {
					return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.Swift.User = string(SwiftSecret.Data[constants.SwiftUser])
				managedRegistryBackend.Swift.Password = string(SwiftSecret.Data[constants.SwiftPassword])

				managedRegistryBackend.CredentialsSecretName = ""

			}

			if managedRegistryBackend.Swift.StoragePath == "" || managedRegistryBackend.Swift.Container == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

		}

		// Validate Cloudfront S3 backend
		if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CloudfrontS3) {

			if !utils.IsZeroOfUnderlyingType(managedRegistryBackend.CredentialsSecretName) {

				validCloudfrontS3Secret, cloudfrontS3Secret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, registryBackend.CredentialsSecretName, constants.RequiredCloudfrontS3CredentialKeys)

				if err != nil {
					return false, err
				}
				if !validCloudfrontS3Secret {
					return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
				}

				managedRegistryBackend.CloudfrontS3.AccessKey = string(cloudfrontS3Secret.Data[constants.CloudfrontS3AccessKey])
				managedRegistryBackend.CloudfrontS3.SecretKey = string(cloudfrontS3Secret.Data[constants.CloudfrontS3SecretKey])

				managedRegistryBackend.CredentialsSecretName = ""

			}

			if managedRegistryBackend.CloudfrontS3.StoragePath == "" || managedRegistryBackend.CloudfrontS3.BucketName == "" {
				return false, fmt.Errorf("Failed to validate provided registry backend. Name: %s", managedRegistryBackend.Name)
			}

		}

		quayConfiguration.RegistryBackends = append(quayConfiguration.RegistryBackends, *managedRegistryBackend)

	}

	// Validate Quay SSL Certificates
	if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.TLS.SecretName) {
		validQuaySslCertificateSecret, quaySslCertificateSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Quay.ExternalAccess.TLS.SecretName, constants.RequiredSslCertificateKeys)

		if err != nil {
			return false, err
		}

		if !validQuaySslCertificateSecret {
			return false, fmt.Errorf("Failed to validate provided Quay SSL Certificate")
		}

		quayConfiguration.QuaySslCertificate = quaySslCertificateSecret.Data[corev1.TLSCertKey]
		quayConfiguration.QuaySslPrivateKey = quaySslCertificateSecret.Data[corev1.TLSPrivateKeyKey]

	}

	if quayConfiguration.QuayEcosystem.Spec.Clair != nil && quayConfiguration.QuayEcosystem.Spec.Clair.Enabled {

		// Validate Clair ImagePullSecret
		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.ImagePullSecretName) {

			validImagePullSecret, _, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Clair.ImagePullSecretName, nil)

			if err != nil {
				return false, err
			}

			if !validImagePullSecret {
				return false, fmt.Errorf("Failed to validate provided Clair Image Pull Secret")
			}

		}

		// Validate Update Interval
		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.UpdateInterval) {

			duration, durationErr := time.ParseDuration(quayConfiguration.QuayEcosystem.Spec.Clair.UpdateInterval)

			if durationErr != nil {
				return false, durationErr
			}

			quayConfiguration.ClairUpdateInterval = duration
		}

		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.Database) && !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.Database.Server) && utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.Database.CredentialsSecretName) {
			return false, fmt.Errorf("Failed to locate a Clair Database Credential for Externally Provisioned Instance")
		}

		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.Database) && !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.Database.CredentialsSecretName) {

			validClairDatabaseSecret, databaseSecret, err := validateSecret(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Clair.Database.CredentialsSecretName, constants.RequiredDatabaseCredentialKeys)

			if err != nil {
				return false, err
			}

			if !validClairDatabaseSecret {
				return false, fmt.Errorf("Failed to validate provided Clair Database Secret")
			}

			quayConfiguration.ClairDatabase.Username = string(databaseSecret.Data[constants.DatabaseCredentialsUsernameKey])
			quayConfiguration.ClairDatabase.Password = string(databaseSecret.Data[constants.DatabaseCredentialsPasswordKey])
			quayConfiguration.ClairDatabase.Database = string(databaseSecret.Data[constants.DatabaseCredentialsDatabaseKey])

			// If the server is provided in the Secret, we override the defaults set in defaults.go
			if _, found := databaseSecret.Data[constants.DatabaseCredentialsServerKey]; found {
				quayConfiguration.ClairDatabase.Server = string(databaseSecret.Data[constants.DatabaseCredentialsServerKey])
				quayConfiguration.QuayEcosystem.Spec.Clair.Database.Server = string(databaseSecret.Data[constants.DatabaseCredentialsServerKey])
				quayConfiguration.QuayEcosystem.Spec.Clair.Database.ReadinessProbe = nil
				quayConfiguration.QuayEcosystem.Spec.Clair.Database.LivenessProbe = nil
				quayConfiguration.QuayEcosystem.Spec.Clair.Database.DeploymentStrategy = ""
				quayConfiguration.QuayEcosystem.Spec.Clair.Database.Image = ""
			}

			if _, found := databaseSecret.Data[constants.DatabaseCredentialsRootPasswordKey]; found {
				quayConfiguration.ClairDatabase.RootPassword = string(databaseSecret.Data[constants.DatabaseCredentialsRootPasswordKey])
			}

			quayConfiguration.ValidProvidedClairDatabaseSecret = true
		}

		// Validate Clair Config Files
		if !utils.IsZeroOfUnderlyingType(quayConfiguration.QuayEcosystem.Spec.Clair.ConfigFiles) {

			clairConfigFiles, err := validateConfigFiles(client, quayConfiguration.QuayEcosystem.Namespace, quayConfiguration.QuayEcosystem.Spec.Clair.ConfigFiles)

			if err != nil {
				return false, err
			}

			quayConfiguration.ClairConfigFiles = append(quayConfiguration.ClairConfigFiles, clairConfigFiles...)

		}

	}

	return true, nil
}

func validateSecret(client client.Client, namespace string, name string, requiredParameters interface{}) (bool, *corev1.Secret, error) {

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, secret)
	if err != nil && errors.IsNotFound(err) {
		logging.Log.Error(fmt.Errorf("Secret not Found"), "Secret Validation", "Namespace", namespace, "Name", name)
		return false, nil, err
	} else if err != nil && !errors.IsNotFound(err) {
		logging.Log.Error(fmt.Errorf("Error retrieving secret"), "Secret Validation", "Namespace", namespace, "Name", name)
		return false, nil, err
	}

	if requiredParameters != nil {

		validSecret := false
		if reflect.TypeOf(requiredParameters).Kind() == reflect.Map {
			validSecret = validateProvidedSecretMap(secret, requiredParameters.(map[string]string))

		}
		if reflect.TypeOf(requiredParameters).Kind() == reflect.Slice {
			validSecret = validateProvidedSecretSlice(secret, requiredParameters.([]string))

		}

		if !validSecret {
			logging.Log.Error(fmt.Errorf("Failed to validate provided secret with required parameters"), "Secret Validation", "Namespace", namespace, "Name", name)
			return false, secret, fmt.Errorf("Failed to validate provided secret with required parameters. Namespace: %s, Name: %s", namespace, name)
		}
	}

	return true, secret, nil

}

func validateProvidedSecretMap(secret *corev1.Secret, requiredParameters map[string]string) bool {

	for key := range requiredParameters {
		if _, found := secret.Data[key]; !found {
			return false
		}
	}

	return true

}

func validateProvidedSecretSlice(secret *corev1.Secret, requiredParameters []string) bool {

	for _, value := range requiredParameters {
		if _, found := secret.Data[value]; !found {
			return false
		}
	}

	return true

}

func validateConfigFiles(client client.Client, namespace string, inputConfigFiles []redhatcopv1alpha1.ConfigFiles) ([]redhatcopv1alpha1.ConfigFiles, error) {

	outputConfigFiles := []redhatcopv1alpha1.ConfigFiles{}

	for _, configFiles := range inputConfigFiles {

		managedConfigFiles := configFiles.DeepCopy()

		if managedConfigFiles.SecretName == "" {
			return nil, fmt.Errorf("Failed to validate provided config files. `secretName` must not be empty")
		}

		validConfigFilesSecret, configFilesSecret, err := validateSecret(client, namespace, managedConfigFiles.SecretName, managedConfigFiles.GetKeys())

		if err != nil {
			return nil, err
		}
		if !validConfigFilesSecret {
			return nil, fmt.Errorf("Failed to validate required provided config file parameters. Invalid Secret Name: %s", managedConfigFiles.SecretName)
		}

		// If the user did not provide a list of keys, grab all of the files
		if utils.IsZeroOfUnderlyingType(managedConfigFiles.Files) || len(managedConfigFiles.Files) == 0 {
			for secretDataFileKey, secretDataFileValue := range configFilesSecret.Data {

				fileType := redhatcopv1alpha1.ConfigConfigFileType

				if !utils.IsZeroOfUnderlyingType(managedConfigFiles.Type) {
					fileType = configFiles.Type
				}

				managedConfigFiles.Files = append(managedConfigFiles.Files, redhatcopv1alpha1.ConfigFile{
					Type:          fileType,
					Key:           secretDataFileKey,
					Filename:      secretDataFileKey,
					SecretContent: secretDataFileValue,
				})
			}
		} else {
			for _, managedConfigFile := range managedConfigFiles.Files {

				fileType := redhatcopv1alpha1.ConfigConfigFileType

				if !utils.IsZeroOfUnderlyingType(managedConfigFile.Type) {
					fileType = managedConfigFile.Type
				} else if !utils.IsZeroOfUnderlyingType(configFiles.Type) {
					fileType = configFiles.Type
				}

				managedConfigFile.Type = fileType
				managedConfigFile.SecretContent = configFilesSecret.Data[managedConfigFile.Key]
			}
		}

		outputConfigFiles = append(outputConfigFiles, *managedConfigFiles)

	}

	return outputConfigFiles, nil
}
