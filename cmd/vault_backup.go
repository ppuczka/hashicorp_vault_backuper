package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"sync"
	"vault_backup/cmd/config"
	"vault_backup/cmd/google_drive"
	"vault_backup/cmd/vault_service"
)

func main() {

	flag.String("config", "", "path to the yaml config file")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	configFilePath, err := pflag.CommandLine.GetString("config")
	if err != nil {
		log.Fatalf("error while parsing run parameters")
	}

	viperCnf, err := viperInit(configFilePath)
	if err != nil {
		log.Fatalf("error while loading config file %s, %v", configFilePath, err)
	}
	appConfig := config.GetVaultConfig(viperCnf)
	ctx := context.Background()

	logFile, err := os.OpenFile(appConfig.VaultConfig.LogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("main: error while creating log file %s, %v \n", appConfig.VaultConfig.LogFilePath, err)
	}
	log.SetOutput(logFile)

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
	cwd, _ := os.Getwd()
	cnf := filepath.Join(cwd, configFilePath)
	viper.SetConfigFile(cnf)
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
