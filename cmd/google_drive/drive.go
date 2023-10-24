package google_drive

import (
	"context"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"log"
	"os"
	"sync"
	"time"
	"vault_backup/cmd/config"
)

type GoogleDriveClient struct {
	service     *drive.Service
	driveConfig *config.GoogleDriveConfig
}

func GetGoogleDriveService(ctx context.Context, config config.AppConfig, credentialsJson string) (*GoogleDriveClient, error) {

	service, err := drive.NewService(ctx, option.WithCredentialsJSON([]byte(credentialsJson)))
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

func (g *GoogleDriveClient) GetListOfOutdatedFiles() (*[]drive.File, error) {
	outdatedBackupFiles := make([]drive.File, 0)
	googleDateTimeLayout := "2006-01-02T15:04:05.999Z"

	res, err := g.service.Files.List().Fields("files(kind, id, name, createdTime, parents, mimeType)").Do()
	if err != nil {
		log.Fatalf("Warning: unable to list files %v", err)
		return &outdatedBackupFiles, err
	}

	for _, f := range res.Files {
		// skip folders
		if f.MimeType == "application/vnd.google-apps.folder" {
			continue
		}

		fileCreatedTime, _ := time.Parse(googleDateTimeLayout, f.CreatedTime)
		if err != nil {
			log.Fatalf("error parsing date: %v", err)
			return &outdatedBackupFiles, err
		}

		daysOutdated := int(time.Now().Sub(fileCreatedTime).Hours() / 24)
		if daysOutdated > g.driveConfig.BackupFileRetentionDays {
			//log.Printf("To delete %s => %.0f | %s", f.Name, daysOutdated, f.CreatedTime)
			outdatedBackupFiles = append(outdatedBackupFiles, *f)
		}
	}
	return &outdatedBackupFiles, nil
}

func (g *GoogleDriveClient) RemoveOutdatedBackups() {
	var numOfDeletedFiles = 0

	outdatedBackupFiles, err := g.GetListOfOutdatedFiles()
	if err != nil {
		log.Fatalf("error while getting outdated backup files %v \n", err)
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
	} else {
		log.Printf("successfully deleted %d backup files", numOfDeletedFiles)
	}
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
