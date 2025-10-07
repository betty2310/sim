package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/betty2310/sim/internal/config"
	"github.com/betty2310/sim/internal/device"
)

type mac struct {
	mac    string
	doorId int
}

var macs = []mac{
	{mac: "74:25:84:c0:01:aa", doorId: 138},
	{mac: "74:25:84:c0:02:4b", doorId: 138},
	{mac: "74:25:84:c0:02:15", doorId: 151},
}

func main() {
	broker := flag.String("broker", "tcp://172.18.10.66:1883", "The broker URI. ex: tcp://172.18.10.66:1883")
	password := flag.String("password", "admin", "The password (optional)")
	user := flag.String("user", "simulator", "The User (optional)")

	flag.Parse()

	cfg := &config.Config{
		MQTT: config.MQTTConfig{
			URL:      *broker,
			Username: *user,
			Password: *password,
		},
	}

	if cfg.MQTT.URL == "" {
		log.Fatal("Pls provide MQTT Broker url")
	}

	var wg sync.WaitGroup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	devices := make([]*device.Device, 0, len(macs))

	for i, mac := range macs {
		wg.Add(1)

		d, err := device.NewSimulateDevice(mac.mac, mac.doorId, cfg)
		if err != nil {
			log.Printf("Failed to create device with mac %s: %v", mac.mac, err)
			wg.Done()
			continue
		}
		devices = append(devices, d)

		go func(dev *device.Device, index int, mac string) {
			defer wg.Done()
			if err := dev.Start(); err != nil {
				log.Printf("Failed to start device %s: %v", mac, err)
				return
			}

			fmt.Printf("Initialized device with MAC: %s\n", mac)

		}(d, i, mac.mac)

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("--- All devices initialized and running ---")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n--- Shutting down gracefully ---")

	// Stop all devices
	for _, d := range devices {
		d.Stop()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	fmt.Println("--- Shutdown complete ---")
}
