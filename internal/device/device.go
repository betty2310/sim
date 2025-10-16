package device

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/betty2310/sim/internal/config"
	"github.com/betty2310/sim/internal/utils"
	"github.com/betty2310/sim/pkg/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Device struct {
	macAddress      string
	doorId          int
	client          mqtt.Client
	pushBioTopic    string
	syncBioTopic    string
	receiveBioTopic string
	biometrics      map[int]struct{} // go don't have set, use map[int]struct{} to simulate set
	config          *config.Config
	stopChan        chan struct{}
	mutex           sync.RWMutex
	wg              sync.WaitGroup
	sync            bool
}

func NewSimulateDevice(mac string, doorId int, cfg *config.Config) (*Device, error) {
	device := &Device{
		macAddress:      mac,
		doorId:          doorId,
		config:          cfg,
		pushBioTopic:    fmt.Sprintf("iot/server/push_biometric/%s", mac),
		syncBioTopic:    "iot/devices/device_sync_bio",
		receiveBioTopic: "iot/devices/device_received_bio",
		biometrics:      make(map[int]struct{}),
		stopChan:        make(chan struct{}),
		sync:            true,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTT.URL)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.SetClientID(fmt.Sprintf("device_sim_%s", mac))
	opts.SetAutoReconnect(true)

	opts.SetOnConnectHandler(device.handleConnect)
	opts.SetConnectionLostHandler(device.handleConnectionLost)
	opts.SetReconnectingHandler(device.handleReconnect)

	device.client = mqtt.NewClient(opts)
	return device, nil
}

func (d *Device) Start() error {
	utils.LogMessageToFile(d.macAddress, "", "Connecting to MQTT broker...")
	if token := d.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	d.wg.Add(1)
	go d.syncBio()
	return nil
}

func (d *Device) Stop() error {
	utils.LogMessageToFile(d.macAddress, "", "Stopping devices...")

	close(d.stopChan)
	d.wg.Wait()

	if d.client.IsConnected() {
		d.client.Disconnect(250)
	}

	utils.LogMessageToFile(d.macAddress, "", "Devices stopped")
	return nil
}

func (d *Device) handleConnect(client mqtt.Client) {
	utils.LogMessageToFile(d.macAddress, "", "Successfully connected to MQTT broker")

	token := client.Subscribe(d.pushBioTopic, 1, d.handleMessage)
	token.Wait()

	if token.Error() != nil {
		utils.LogMessageToFile(d.macAddress, d.pushBioTopic,
			fmt.Sprintf("Failed to subscribe to topic: %s. Error: %v",
				d.pushBioTopic, token.Error()))
	} else {
		utils.LogMessageToFile(d.macAddress, d.pushBioTopic,
			fmt.Sprintf("Subscribed to topic: %s", d.pushBioTopic))
	}

}

func (d *Device) handleMessage(client mqtt.Client, msg mqtt.Message) {
	go func() {
		topic := msg.Topic()
		payload := msg.Payload()

		if strings.Contains(topic, "push_bio") {
			var data types.ServerPushBiometricMessage
			if err := json.Unmarshal(payload, &data); err != nil {
				utils.LogMessageToFile(d.macAddress, msg.Topic(), fmt.Sprintf("Failed to parse biometric message: %v", err))
				return
			}
			utils.LogMessageToFile(d.macAddress, msg.Topic(),
				fmt.Sprintf("BioID=%d, IdNumber=%s, FullName=%s, SyncEnds=%t", data.BioID, data.IDNumber, data.FullName, data.SyncEnded))
			d.sync = data.SyncEnded

			time.Sleep(500 * time.Millisecond)
			d.mutex.Lock()
			d.biometrics[data.BioID] = struct{}{}
			bioCount := len(d.biometrics)
			d.mutex.Unlock()

			d.publishReceivedBiometric(data.BioID)
			fmt.Printf("Device %s has %d biometrics\n", d.macAddress, bioCount)

		} else {
			utils.LogMessageToFile(d.macAddress, topic, string(payload))
		}
	}()
}

func (d *Device) syncBio() {
	defer d.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	if d.sync {
		d.pulishDeviceSyncBio()
	}

	for {
		select {
		case <-ticker.C:
			if d.sync {
				d.pulishDeviceSyncBio()
			}
		case <-d.stopChan:
			return
		}
	}
}

func (d *Device) pulishDeviceSyncBio() {
	d.mutex.RLock()
	ids := make([]int, 0, len(d.biometrics))

	for id := range d.biometrics {
		ids = append(ids, id)
	}
	d.mutex.RUnlock()

	msg := types.DeviceSyncBioMessage{
		MacAddress: d.macAddress,
		LstBioId:   ids,
		DoorID:     d.doorId,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		utils.LogMessageToFile(d.macAddress, d.syncBioTopic,
			fmt.Sprintf("Failed to marshal sync bio message: %v", err))
		return
	}

	token := d.client.Publish(d.syncBioTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		utils.LogMessageToFile(d.macAddress, d.syncBioTopic,
			fmt.Sprintf("Failed to publish sync bio. Error: %v", token.Error()))
	} else {
		utils.LogMessageToFile(d.macAddress, d.syncBioTopic,
			fmt.Sprintf("%s", string(payload)))
	}

}

func (d *Device) publishReceivedBiometric(id int) {
	msg := types.DeviceReceivedBioMessage{
		MacAddress: d.macAddress,
		BioID:      id,
		DeviceTime: time.Now().Format(time.RFC3339),
		CmdType:    "NEW_BIO",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		utils.LogMessageToFile(d.macAddress, d.receiveBioTopic,
			fmt.Sprintf("Failed to marshal received bio message: %v", err))
		return
	}

	token := d.client.Publish(d.receiveBioTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		utils.LogMessageToFile(d.macAddress, d.receiveBioTopic,
			fmt.Sprintf("Failed to publish received bio. Error: %v", token.Error()))
	} else {
		utils.LogMessageToFile(d.macAddress, d.receiveBioTopic,
			fmt.Sprintf("Published ACK for bio %d: %s", id, string(payload)))
	}
}

func (d *Device) handleConnectionLost(client mqtt.Client, err error) {
	utils.LogMessageToFile(d.macAddress, "", fmt.Sprintf("Connection lost: %v", err))
}

func (d *Device) handleReconnect(client mqtt.Client, opts *mqtt.ClientOptions) {
	utils.LogMessageToFile(d.macAddress, "", "Reconnecting to MQTT broker...")
}
