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
	"vault_backup/cmd/google_drive"
)

const (
	eventType     = "kv-v2/*"
	websocketPath = "/v1/sys/events/subscribe/" + eventType
)

type BackupScheduler struct {
	vault     *Vault
	appConfig AppConfig
}

func eventListenerBackup(conn *websocket.Conn, events chan string) {
	log.Println("Connected to vault_service events. Listening...")
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		eventType := pareJsonEvent(string(message))
		events <- eventType
	}
}

func scheduledTimeBackup(events chan string) {
	s := gocron.NewScheduler(time.UTC)

	_, err := s.Every(10).Hour().Do(func() {
		log.Println("Performing scheduled backup...")

		events <- "scheduled backup"
	})
	if err != nil {
		log.Fatalf("error while scheduling cron job %v", err)
	}

	s.StartBlocking()
}

func onEventVaultBackup(events chan string, vault *Vault, googleDrive *google_drive.GoogleDrive) {
	for {
		select {
		case e := <-events:
			nowTimestamp := time.Now().Unix()

			log.Printf("Event %s recived. Performing backup...", e)
			filePath := filepath.Join("/home/azureuser/vault", fmt.Sprintf("%d.snap", nowTimestamp))
			backupFile, _ := vault.RaftSnapshot(filePath)
			log.Printf("Backup %s created succesfully \n", backupFile.Name())

			googleDrive.DeployBackupToGoogleDrive(filePath)
		}
	}
}

func pareJsonEvent(json string) string {
	return gjson.Get(json, "data.event_type").String()
}

func CreateVaultBackups(token vault.Secret, vault *Vault, googleDrive *google_drive.GoogleDrive) {
	wsURL := "wss://0.0.0.0:8300" + websocketPath + "?json=true"
	wsHeader := http.Header{"X-Vault-Token": []string{token.Auth.ClientToken}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, wsHeader)
	if err != nil {
		log.Fatalf("WebSocket connection error: %v", err)
	}

	defer conn.Close()

	events := make(chan string, 10)
	go eventListenerBackup(conn, events)
	go onEventVaultBackup(events, vault, googleDrive)
	go scheduledTimeBackup(events)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("Received interrupt. Closing WebSocket connection.")
}
