package services

import (
	"net/http"
	"time"
)

type AppStatusProvider interface {
	DbAppStatusProvider
	FileAppStatusProvider
}

type DbAppStatusProvider interface {
	SaveStatusToDb()
	LoadStatusFromDb()
}

type FileAppStatusProvider interface {
	SaveStatusToFile()
	LoadStatusFromFile()
}

type LastSuccessfulBackup struct {
	time       time.Time
	status     string
	backupInfo BackupInfo
}

type LastFailedBackup struct {
	time       time.Time
	status     string
	backupInfo BackupInfo
}

type BackupInfo struct {
	numberOfBackupFiles   int
	remoteBackupType      string
	totalRemoteBackupSize float32
	totalLocalBackupSize  float32
	lastBackupSize        float32
}

type FileAppStatus struct {
	statusFilePath       string
	startTime            time.Time
	lastFailedBackup     LastFailedBackup
	lastSuccessfulBackup LastSuccessfulBackup
}

func GetFileAppStatus(statusFile string, appStartTime time.Time) (*FileAppStatus, error) {
	return &FileAppStatus{}, nil
}

func (fs *FileAppStatus) SaveStatusToFile() {

}

func (fs *FileAppStatus) LoadStatusFromFile() {

}

type DbAppStatus struct {
	dbConnString         string
	startTime            time.Time
	lastFailedBackup     LastFailedBackup
	lastSuccessfulBackup LastSuccessfulBackup
}

func (db *DbAppStatus) SaveStatusToDb() {

}

func (db *DbAppStatus) LoadStatusFromDb() {

}

type StatusServer struct {
	server http.Server
}

func (s *StatusServer) StartServer(port string) {

}
