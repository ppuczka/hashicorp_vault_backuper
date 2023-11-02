package vault

import (
	"context"
	"encoding/json"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"log"
	"os"
	"vault_backup/cmd/config"
)

type Vault struct {
	client *vault.Client
}

type renewResult uint8

const (
	renewError renewResult = 1 << iota
	exitRequested
	expiringAuthToken
)

func GetVaultAppRoleClient(ctx context.Context, config config.AppConfig) (*Vault, *vault.Secret, error) {
	client, err := vault.NewClient(&vault.Config{Address: config.VaultConfig.Address})
	if err != nil {
		log.Fatalf("unable to initialize Vault client: %v", err)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("unable to initialize AppRole auth method: %w", err)
	}

	v := &Vault{
		client: client,
	}

	token, err := v.login(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("vault login error: %w", err)
	}

	log.Println("connecting to vault: success!")

	return v, token, nil
}

func (v *Vault) GetGoogleDriveJsonSecret(ctx context.Context) string {
	kvSecret, err := v.client.KVv2("google_drive").Get(ctx, "service_account")
	if err != nil {
		log.Fatalf("error while getting secret from vault %v", err)
	}

	secret, err := json.Marshal(kvSecret.Data)
	if err != nil {
		log.Fatalf("error while creating secret string")
	}

	return string(secret)
}

func (v *Vault) login(ctx context.Context, config config.AppConfig) (*vault.Secret, error) {
	appRoleAuth, err := auth.NewAppRoleAuth(
		config.VaultConfig.AppRoleId,
		&auth.SecretID{FromString: config.VaultConfig.AppSecretId})
	if err != nil {
		return nil, fmt.Errorf("unable to initialize AppRole auth method: %w", err)
	}

	authInfo, err := v.client.Auth().Login(ctx, appRoleAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to login to AppRole auth method: %w", err)
	}

	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}

	return authInfo, nil
}

func (v *Vault) RaftSnapshot(snapshotPath string) (*os.File, error) {
	snapshotFile, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		log.Printf("Snapshot file at %s could not be created", snapshotPath)
		fmt.Println(err)
		return nil, err
	}
	defer snapshotFileClose(snapshotFile)

	err = v.client.Sys().RaftSnapshot(snapshotFile)
	if err != nil {
		snapshotFile.Close()
		fmt.Println("Vault Raft snapshot invocation failed")
		fmt.Println(err)
		return nil, err
	}

	return snapshotFile, nil
}

func (v *Vault) RenewTokenPeriodically(ctx context.Context, authToken *vault.Secret, config config.AppConfig) {

	log.Println("Renew / Recreate secrets loop: begin")
	defer log.Println("Renew / Recreate secrets loop: end")

	currentAuthToken := authToken
	for {
		renewed, err := v.renewLeases(ctx, currentAuthToken)
		if err != nil {
			log.Fatalf("Renew error: %v", err) // simplified error handling
		}

		if renewed&exitRequested != 0 {
			return
		}

		if renewed&expiringAuthToken != 0 {
			log.Printf("Auth token: can no longer be renewed; will log in again")

			authToken, err := v.login(ctx, config)
			if err != nil {
				log.Fatalf("Login authentication error: %v", err)
			}
			currentAuthToken = authToken
		}
	}
}

func (v *Vault) renewLeases(ctx context.Context, authToken *vault.Secret) (renewResult, error) {
	log.Println("Renew cycle: begin")
	defer log.Println("Renew cycle: end")

	// auth token
	authTokenWatcher, err := v.client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret: authToken,
	})
	if err != nil {
		return renewError, fmt.Errorf("unable to initialize auth token lifetime watcher: %w", err)
	}

	go authTokenWatcher.Start()
	defer authTokenWatcher.Stop()

	// monitor events from both watchers
	for {
		select {
		case <-ctx.Done():
			return exitRequested, nil

		// DoneCh will return if renewal fails, or if the remaining lease
		// duration is under a built-in threshold and either renewing is not
		// extending it or renewing is disabled.  In both cases, the caller
		// should attempt a re-read of the secret. Clients should check the
		// return value of the channel to see if renewal was successful.
		case err := <-authTokenWatcher.DoneCh():
			// Leases created by a token get revoked when the token is revoked.
			return expiringAuthToken, err

		// RenewCh is a channel that receives a message when a successful
		// renewal takes place and includes metadata about the renewal.
		case info := <-authTokenWatcher.RenewCh():
			log.Printf("Auth token: successfully renewed; remaining duration: %ds\n", info.Secret.Auth.LeaseDuration)
		}
	}
}

func snapshotFileClose(snapshotFile *os.File) {
	err := snapshotFile.Close()
	if err != nil {
		fmt.Println("Vault raft snapshot file failed to close")
		log.Fatalln(err)
	}
}
