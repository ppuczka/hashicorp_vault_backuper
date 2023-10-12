package config

import (
	vault "github.com/hashicorp/vault/api"
	"log"
	"os"
)

const defaultVaultAddress = "https://127.0.0.0:8200"
const serviceAccountKeyPath = "key.json"
const googleDriveDeployFolderId = "1gJCvUBRdry1JZISJ6MCoCEcKFncK9fZD"

type AppConfig struct {
	VaultAddr                    string
	VaultToken                   string
	VaultAppRoleId               string
	VaultAppSecretId             string
	GoogleServiceAccountFilePath string
	GoogleDriveDeployFolderId    string
}

func GetVaultConfig() AppConfig {
	appConfig := AppConfig{}
	if len(os.Getenv(vault.EnvVaultAddress)) == 0 {
		log.Printf("Vault address not found defaulting to %s", defaultVaultAddress)
		appConfig.VaultAddr = defaultVaultAddress
	} else {
		log.Printf("Vault address: %s", os.Getenv(vault.EnvVaultAddress))
		appConfig.VaultAddr = os.Getenv(vault.EnvVaultAddress)
	}

	if len(os.Getenv(vault.EnvVaultToken)) != 28 {
		log.Print("Vault token is not valid")
	} else {
		appConfig.VaultToken = os.Getenv(vault.EnvVaultToken)
	}

	// Todo error handling
	appConfig.VaultAppRoleId = os.Getenv("APPROLE_ROLE_ID")
	appConfig.VaultAppSecretId = os.Getenv("APPROLE_SECRET_ID")

	appConfig.GoogleServiceAccountFilePath = serviceAccountKeyPath
	appConfig.GoogleDriveDeployFolderId = googleDriveDeployFolderId
	return appConfig
}
