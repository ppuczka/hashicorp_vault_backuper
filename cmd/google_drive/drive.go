package google_drive

import (
	"context"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"log"
	"os"
	"vault_backup/cmd/config"
)

type GoogleDriveClient struct {
	service     *drive.Service
	driveConfig *config.GoogleDriveConfig
}

func GetGoogleDriveService(ctx context.Context, config config.AppConfig) (*GoogleDriveClient, error) {
	service, err := drive.NewService(ctx, option.WithCredentialsFile(config.GoogleDriveConfig.ServiceAccountFilePath))
	if err != nil {
		log.Fatalf("Warning: Unable to create drive Client %v", err)
		return nil, err
	}

	gd := GoogleDriveClient{
		service:     service,
		driveConfig: &config.GoogleDriveConfig,
	}

	return &gd, nil
}

func (g *GoogleDriveClient) ListFiles() {
	res, err := g.service.Files.List().Do()
	if err != nil {
		log.Fatalf("Warning: unable to list files %v", err)
	}
	log.Printf("Files %s", (res.Files))
}

func (g *GoogleDriveClient) DeployBackupToGoogleDrive(backupFilePath string) {
	file, err := os.Open(backupFilePath)
	if err != nil {
		log.Fatalf("Warning: unable to load a file %s, %v", backupFilePath, err)
	}

	info, err := file.Stat()
	if err != nil {
		log.Fatalf("Warning: unable to get fileInfo %s, %v", backupFilePath, err)
	}

	defer file.Close()

	fileMetadata := &drive.File{
		Name:    info.Name(),
		Parents: []string{g.driveConfig.DeployFolderId},
	}

	res, err := g.service.Files.
		Create(fileMetadata).
		Media(file).
		SupportsAllDrives(true).
		ProgressUpdater(func(now, size int64) { log.Printf("%d, %d\r", now, size) }).
		Do()

	if err != nil {
		log.Fatalf("Warning: unable to upload file %v", err)
	}

	log.Printf("New file id: %s\n", res.Id)
}
