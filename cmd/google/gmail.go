package google

import (
	"context"
	"fmt"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"vault_backup/cmd/config"
)

type GmailClient struct {
	service     *gmail.Service
	driveConfig *config.GoogleDriveConfig
}

func GetGmailClient(ctx context.Context, config config.AppConfig, credentialsJson string) (*GmailClient, error) {

	service, err := gmail.NewService(ctx, option.WithCredentialsJSON([]byte(credentialsJson)))
	if err != nil {
		return nil, fmt.Errorf("GetGmailClient: error while creating gmail service %w", err)
	}

	return &GmailClient{service: service, driveConfig: &config.GoogleDriveConfig}, nil
}
