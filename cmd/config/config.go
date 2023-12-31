package config

import (
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	"log"
	"os"
)

const appName = "VaultBackup"

type AppConfig struct {
	AppName           string
	VaultConfig       VaultConfig
	GoogleDriveConfig GoogleDriveConfig
}

type VaultConfig struct {
	Address                   string
	AppRoleId                 string
	AppSecretId               string
	WebSocketEventBaseUrl     string
	ListenedEventsType        string
	ScheduledSnapshotInterval string
	SnapshotFolder            string
	LogFilePath               string
	EmailHost                 string
	EmailHostPort             string
	Mailbox                   string
	NotifyEmails              []string
}

type GoogleDriveConfig struct {
	OnEventDeployFolderId   string
	ScheduledDeployFolderId string
	ServiceAccountFilePath  string
	BackupFileRetentionDays int
}

func GetVaultConfig(viper *viper.Viper) AppConfig {
	appConfig := AppConfig{
		AppName: appName,
	}

	if len(os.Getenv(vault.EnvVaultAddress)) == 0 {
		appConfig.VaultConfig.Address = viper.GetString("vault.address")
	} else {
		log.Printf("Vault address: %s", os.Getenv(vault.EnvVaultAddress))
		appConfig.VaultConfig.Address = os.Getenv(vault.EnvVaultAddress)
	}

	appConfig.VaultConfig.AppRoleId = viper.GetString("vault.app_role_id")
	appConfig.VaultConfig.WebSocketEventBaseUrl = viper.GetString("vault.web_socket_event_base_url")
	appConfig.VaultConfig.ListenedEventsType = viper.GetString("vault.listened_event_type")
	appConfig.VaultConfig.ScheduledSnapshotInterval = viper.GetString("vault.scheduled_snapshot_interval")
	appConfig.VaultConfig.SnapshotFolder = viper.GetString("vault.snapshot_folder")
	appConfig.VaultConfig.LogFilePath = viper.GetString("vault.log_file_path")
	appConfig.VaultConfig.NotifyEmails = viper.GetStringSlice("vault.notify_email_addresses")
	appConfig.VaultConfig.EmailHost = viper.GetString("vault.email_host")
	appConfig.VaultConfig.EmailHostPort = viper.GetString("vault.email_host_port")
	appConfig.VaultConfig.Mailbox = viper.GetString("vault.mailbox")

	appConfig.VaultConfig.AppSecretId = os.Getenv("APPROLE_SECRET_ID")

	appConfig.GoogleDriveConfig.OnEventDeployFolderId = viper.GetString("google.on_event_deploy_folder_id")
	appConfig.GoogleDriveConfig.ScheduledDeployFolderId = viper.GetString("google.scheduled_deploy_folder_id")
	appConfig.GoogleDriveConfig.BackupFileRetentionDays = viper.GetInt("google.backup_file_retention_days")

	return appConfig
}
