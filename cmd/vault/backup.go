package vault

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/gorilla/websocket"
	vault "github.com/hashicorp/vault/api"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
	"vault_backup/cmd/config"
	"vault_backup/cmd/google"
)

const vaultWebsocketPath = "v1/sys/events/subscribe"

type Event int64

const (
	WssEvent Event = iota
	ScheduledEvent
)

func (e Event) String() string {
	switch e {
	case WssEvent:
		return "vault event"
	case ScheduledEvent:
		return "scheduled event"
	}
	return "unknown"
}

type BackupType struct {
	eventType    Event
	gDriveFileId string
}

type BackupScheduler struct {
	vault             *Vault
	appConfig         *config.AppConfig
	googleDriveClient *google.DriveClient
	wsConnection      *websocket.Conn
	scheduler         *gocron.Scheduler
}

func GetBackupScheduler(
	vault *Vault,
	appConfig *config.AppConfig,
	googleDriveClient *google.DriveClient,
	token vault.Secret) (*BackupScheduler, error) {

	wsURL := fmt.Sprintf("%s/%s/%s?json=true",
		appConfig.VaultConfig.WebSocketEventBaseUrl,
		vaultWebsocketPath,
		appConfig.VaultConfig.ListenedEventsType)

	wsHeader := http.Header{"X-Vault-Token": []string{token.Auth.ClientToken}}
	wsDialer := websocket.DefaultDialer

	conn, _, err := wsDialer.Dial(wsURL, wsHeader)
	if err != nil {
		return nil, err
	}

	return &BackupScheduler{
			vault:             vault,
			appConfig:         appConfig,
			googleDriveClient: googleDriveClient,
			wsConnection:      conn,
			scheduler:         gocron.NewScheduler(time.UTC),
		},
		nil
}

func (bs BackupScheduler) vaultEventListener(events chan BackupType) {
	log.Println("Connected to vault events. Listening...")
	for {
		_, _, err := bs.wsConnection.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		eventType := BackupType{WssEvent, bs.appConfig.GoogleDriveConfig.OnEventDeployFolderId}
		events <- eventType
	}
}

func (bs BackupScheduler) scheduledTimeBackup(events chan BackupType) {
	_, err := bs.scheduler.Every(bs.appConfig.VaultConfig.ScheduledSnapshotInterval).Do(func() {
		log.Println("Performing scheduled backup...")
		events <- BackupType{ScheduledEvent, bs.appConfig.GoogleDriveConfig.ScheduledDeployFolderId}
	})

	if err != nil {
		log.Fatalf("error while scheduling cron job %v", err)
	}
}

func (bs BackupScheduler) scheduledTimeBackupCleanup() {
	_, err := bs.scheduler.Every(bs.appConfig.VaultConfig.ScheduledSnapshotInterval).Do(func() error {
		deletedFilesNumber, err := bs.googleDriveClient.RemoveOutdatedBackups()
		if err != nil {
			return fmt.Errorf("scheduledTimeBackupCleanup: error when removinig outdated backups %w", err)
		}
		log.Printf("scheduledTimeBackupCleanup: successfully deleted %d backup files\n", deletedFilesNumber)
		return nil
	})

	if err != nil {
		log.Printf("error while scheduling cron job %v", err)
	}
}

func (bs BackupScheduler) onEventBackup(events chan BackupType) {
	for {
		select {
		case e := <-events:
			nowTimestamp := time.Now().Unix()

			log.Printf("Event %s recived. Performing backup...", e.eventType)
			filePath := filepath.Join(bs.appConfig.VaultConfig.SnapshotFolder, fmt.Sprintf("%d.snap", nowTimestamp))
			backupFile, _ := bs.vault.RaftSnapshot(filePath)
			log.Printf("Backup %s created succesfully \n", backupFile.Name())

			fileId, err := bs.googleDriveClient.DeployBackupToGoogleDrive(filePath, e.gDriveFileId)
			if err != nil {
				log.Printf("onEventBackup: error while deploying backup to Google Drive %v \n", err)
			}
			log.Printf("New file id: %d\n", fileId)
		}
	}
}

func (bs BackupScheduler) CreateVaultBackups() {

	defer bs.wsConnection.Close()

	events := make(chan BackupType, 10)
	go bs.vaultEventListener(events)
	go bs.onEventBackup(events)
	go bs.scheduledTimeBackup(events)

	bs.scheduledTimeBackupCleanup()
	bs.scheduler.StartBlocking()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("Received interrupt. Closing WebSocket connection.")
}
