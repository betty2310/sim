package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var mutex sync.Mutex

func LogMessageToFile(mac, topic, message string) error {
	timestamp := time.Now()
	logsDir := "./logs"

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	dateStr := timestamp.Format("2006-01-02")
	filename := filepath.Join(logsDir, fmt.Sprintf("%s_%s.log", mac, dateStr))

	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logEntry := fmt.Sprintf("%s | %s: [%s] - %s",
		timestamp.Format(time.RFC3339), mac, topic, message)
	logger.Println(logEntry)

	return nil
}
