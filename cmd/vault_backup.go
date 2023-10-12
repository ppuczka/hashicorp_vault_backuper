package main

import (
	"context"
	"log"
	"sync"
	"vault_backup/cmd/config"
	"vault_backup/cmd/vault_service"

	"vault_backup/cmd/google_drive"
)

func main() {
	vaultConfig := config.GetVaultConfig()
	ctx := context.Background()

	vault, authToken, err := vault_service.GetVaultAppRoleClient(ctx, vaultConfig)
	if err != nil {
		log.Fatalf("unable to initialize vault_service connection @ %s: %w", vaultConfig.VaultAddr, err)
	}

	googleDrive, err := google_drive.GetGoogleDriveService(vaultConfig)
	if err != nil {
		log.Fatalf("unable to initialize GoogleDriveService %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		vault.RenewTokenPeriodically(ctx, authToken, vaultConfig)
		wg.Done()
	}()

	defer func() {
		wg.Wait()
	}()

	vault_service.CreateVaultBackups(*authToken, vault, googleDrive)
}
