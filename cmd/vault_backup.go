package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"sync"
	"vault_backup/cmd/config"
	"vault_backup/cmd/google_drive"
	"vault_backup/cmd/vault_service"
)

func main() {
	viperCnf, _ := viperInit("config.yaml")
	appConfig := config.GetVaultConfig(viperCnf)
	ctx := context.Background()

	vault, authToken, err := vault_service.GetVaultAppRoleClient(ctx, appConfig)
	if err != nil {
		log.Fatalf("unable to initialize vault_service connection @ %s: %v", appConfig.VaultConfig.Address, err)
	}

	gDriveJsonSecret := vault.GetGoogleDriveJsonSecret(ctx)
	googleDrive, err := google_drive.GetGoogleDriveService(ctx, appConfig, gDriveJsonSecret)
	if err != nil {
		log.Fatalf("unable to initialize GoogleDriveService %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		vault.RenewTokenPeriodically(ctx, authToken, appConfig)
		wg.Done()
	}()

	defer func() {
		wg.Wait()
	}()

	backupScheduler, err := vault_service.GetBackupScheduler(vault, &appConfig, googleDrive, *authToken)
	if err != nil {
		log.Fatalf("unable to initialize BackupScheduler %v", err)
	}
	backupScheduler.CreateVaultBackups()
}

func viperInit(configFilePath string) (*viper.Viper, error) {
	viper.AddConfigPath(configFilePath)
	viper.AddConfigPath("$HOME/.vault_backup")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("viperInit: %w", err)
		}
	}
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("viperInit: %w", err)
	}

	return viper.GetViper(), nil
}
