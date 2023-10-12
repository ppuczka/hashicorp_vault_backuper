package vault_service

import (
	"fmt"
	"github.com/gorilla/websocket"
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

const (
	eventType     = "kv-v2/*"
	websocketPath = "/v1/sys/events/subscribe/" + eventType
)

func pareJsonEvent(json string) string {
	return gjson.Get(json, "data.event_type").String()
}

func handleWebSocketEvents(conn *websocket.Conn, events chan string) {
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

func scheduledTimeBackup() {}

func CreateVaultBackups(config config.AppConfig, vault *Vault, googleDrive *google_drive.GoogleDrive) {
	wsURL := "wss://0.0.0.0:8300" + websocketPath + "?json=true"
	wsHeader := http.Header{"X-Vault-Token": []string{config.VaultToken}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, wsHeader)
	if err != nil {
		log.Fatalf("WebSocket connection error: %v", err)
	}

	defer conn.Close()

	events := make(chan string, 10)
	go handleWebSocketEvents(conn, events)
	go onEventVaultBackup(events, vault, googleDrive)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("Received interrupt. Closing WebSocket connection.")
}
