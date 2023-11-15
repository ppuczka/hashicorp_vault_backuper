package google

import (
	"context"
	"encoding/json"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"log"
	"os"
	"sync"
	"time"
	"vault_backup/cmd/config"
)

type DriveClient struct {
	service     *drive.Service
	driveConfig *config.GoogleDriveConfig
}

func GetGoogleDriveClient(ctx context.Context, config config.AppConfig, kvSecret vault.KVSecret) (*DriveClient, error) {
	secret, err := json.Marshal(kvSecret.Data)
	if err != nil {
		return nil, fmt.Errorf("GetKVSecret: error while creating secret string %w", err)
	}

	service, err := drive.NewService(ctx, option.WithCredentialsJSON(secret))
	if err != nil {
		return nil, fmt.Errorf("GetGoogleDriveClient: Unable to create drive Client %w", err)
	}

	gd := DriveClient{
		service:     service,
		driveConfig: &config.GoogleDriveConfig,
	}

	return &gd, nil
}

func (g *DriveClient) GetListOfOutdatedFiles() (*[]drive.File, error) {
	outdatedBackupFiles := make([]drive.File, 0)
	googleDateTimeLayout := "2006-01-02T15:04:05.999Z"

	res, err := g.service.Files.List().Fields("files(kind, id, name, createdTime, parents, mimeType)").Do()
	if err != nil {
		return nil, fmt.Errorf("GetListOfOutdatedFiles: unable to list files %w", err)
	}

	for _, f := range res.Files {
		// skip folders
		if f.MimeType == "application/vnd.google-apps.folder" {
			continue
		}

		fileCreatedTime, _ := time.Parse(googleDateTimeLayout, f.CreatedTime)
		if err != nil {
			return nil, fmt.Errorf("GetListOfOutdatedFiles: error parsing date: %w", err)
		}

		daysOutdated := int(time.Now().Sub(fileCreatedTime).Hours() / 24)
		if daysOutdated > g.driveConfig.BackupFileRetentionDays {
			//log.Printf("To delete %s => %.0f | %s", f.Name, daysOutdated, f.CreatedTime)
			outdatedBackupFiles = append(outdatedBackupFiles, *f)
		}
	}
	return &outdatedBackupFiles, nil
}

func (g *DriveClient) RemoveOutdatedBackups() (int, error) {
	var numOfDeletedFiles = 0

	outdatedBackupFiles, err := g.GetListOfOutdatedFiles()
	if err != nil {
		return 0, fmt.Errorf("RemoveOutdatedBackups: error while getting outdated backup files %w", err)
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(*outdatedBackupFiles))

	for _, f := range *outdatedBackupFiles {
		wg.Add(1)
		go func(f drive.File) {
			defer wg.Done()
			err := g.service.Files.Delete(f.Id).Do()
			if err != nil {
				errorChan <- err
			} else {
				numOfDeletedFiles = numOfDeletedFiles + 1
			}
		}(f)
		wg.Wait()
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	if len(errorChan) > 0 {
		for err := range errorChan {
			log.Printf("error while deleting file: %v\n", err)
		}
		return 0, fmt.Errorf("RemoveOutdatedBackups: error while deleting file %w", <-errorChan)
	}
	return numOfDeletedFiles, nil
}

func (g *DriveClient) DeployBackupToGoogleDrive(backupFilePath, googleDriveFolderId string) (*string, error) {
	file, err := os.Open(backupFilePath)
	if err != nil {
		log.Fatalf("DeployBackupToGoogleDrive: unable to load a file %s, %v", backupFilePath, err)
	}

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("DeployBackupToGoogleDrive: unable to get fileInfo %s, %w", backupFilePath, err)
	}

	defer file.Close()

	log.Printf("Uploading file %s to folder: %s", info.Name(), googleDriveFolderId)
	fileMetadata := &drive.File{
		Name:    info.Name(),
		Parents: []string{googleDriveFolderId},
	}

	res, err := g.service.Files.
		Create(fileMetadata).
		Media(file).
		SupportsAllDrives(true).
		ProgressUpdater(func(now, size int64) { log.Printf("%d, %d\r", now, size) }).
		Do()

	if err != nil {
		return nil, fmt.Errorf("DeployBackupToGoogleDrive: unable to upload file %w", err)
	}

	return &res.Id, nil
}
