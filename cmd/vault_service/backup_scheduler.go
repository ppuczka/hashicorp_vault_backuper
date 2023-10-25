package vault_service

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/gorilla/websocket"
	vault "github.com/hashicorp/vault/api"
	"github.com/tidwall/gjson"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
	"vault_backup/cmd/config"
	"vault_backup/cmd/google_drive"
)

const vaultWebsocketPath = "v1/sys/events/subscribe"

type BackupScheduler struct {
	vault             *Vault
	vaultConfig       *config.VaultConfig
	googleDriveClient *google_drive.GoogleDriveClient
	wsConnection      *websocket.Conn
	scheduler         *gocron.Scheduler
}

func GetBackupScheduler(
	vault *Vault,
	appConfig *config.AppConfig,
	googleDriveClient *google_drive.GoogleDriveClient,
	token vault.Secret) (*BackupScheduler, error) {

	wsURL := fmt.Sprintf("%s/%s/%s?json=true",
		appConfig.VaultConfig.WebSocketEventBaseUrl,
		vaultWebsocketPath,
		appConfig.VaultConfig.ListenedEventsType)

	wsHeader := http.Header{"X-Vault-Token": []string{token.Auth.ClientToken}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, wsHeader)
	if err != nil {
		return nil, err
	}

	return &BackupScheduler{
			vault:             vault,
			vaultConfig:       &appConfig.VaultConfig,
			googleDriveClient: googleDriveClient,
			wsConnection:      conn,
			scheduler:         gocron.NewScheduler(time.UTC),
		},
		nil
}

func (bs BackupScheduler) vaultEventListener(events chan string) {
	log.Println("Connected to vault_service events. Listening...")
	for {
		_, message, err := bs.wsConnection.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		eventType := bs.pareJsonEvent(string(message))
		events <- eventType
	}
}

func (bs BackupScheduler) scheduledTimeBackup(events chan string) {
	_, err := bs.scheduler.Every(bs.vaultConfig.ScheduledSnapshotInterval).Do(func() {
		log.Println("Performing scheduled backup...")
		events <- "scheduled backup"
	})

	if err != nil {
		log.Fatalf("error while scheduling cron job %v", err)
	}
}

func (bs BackupScheduler) scheduledTimeBackupCleanup() {
	_, err := bs.scheduler.Every(bs.vaultConfig.ScheduledSnapshotInterval).Do(func() {
		bs.googleDriveClient.RemoveOutdatedBackups()
	})

	if err != nil {
		log.Fatalf("error while scheduling cron job %v", err)
	}
}

func (bs BackupScheduler) onEventBackup(events chan string) {
	for {
		select {
		case e := <-events:
			nowTimestamp := time.Now().Unix()

			log.Printf("Event %s recived. Performing backup...", e)
			filePath := filepath.Join(bs.vaultConfig.SnapshotFolder, fmt.Sprintf("%d.snap", nowTimestamp))
			backupFile, _ := bs.vault.RaftSnapshot(filePath)
			log.Printf("Backup %s created succesfully \n", backupFile.Name())

			bs.googleDriveClient.DeployBackupToGoogleDrive(filePath)
		}
	}
}

func (bs BackupScheduler) pareJsonEvent(json string) string {
	return gjson.Get(json, "data.event_type").String()
}

func (bs BackupScheduler) CreateVaultBackups() {

	defer bs.wsConnection.Close()

	events := make(chan string, 10)
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
