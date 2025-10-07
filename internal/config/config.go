package config

type Config struct {
	MQTT MQTTConfig
}

type MQTTConfig struct {
	URL      string
	Username string
	Password string
}
