package main

import (
	"time"
)

type LastSuccessfulBackup struct {
	time   time.Time
	status string
}

type LastFailedBackup struct {
	time   time.Time
	status string
}

type AppStatus struct {
	uptime               time.Duration
	lastFailedBackup     LastFailedBackup
	lastSuccessfulBackup LastSuccessfulBackup
}

func (status *AppStatus) SaveStatusToFile() {

}

func (status *AppStatus) LoadStatusFromFile() {

}
