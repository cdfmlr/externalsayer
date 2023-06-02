package main

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/cdfmlr/ellipsis"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// Config is the configuration for the application.
type Config struct {
	// SrvAddr is the address:port to listen on.
	SrvAddr string
	// EnabledSayer is the name of the sayer to use: "azure".
	EnabledSayer string
	// AzureSayer is the configuration for the AzureSayer.
	AzureSayer AzureSayerConfig
}

func (c *Config) Write(dst io.Writer) error {
	return yaml.NewEncoder(dst).Encode(&c)
}

// DesensitizedCopy desensitize the config.
// Returns a pointer to the desensitized config copy.
//
// If it's failed to make it, it panics.
//
// Avoid keys being printed to the log.
func (c *Config) DesensitizedCopy() *Config {
	var cCopy Config

	// deep copy
	buf := bytes.NewBuffer(nil)
	if err := yaml.NewEncoder(buf).Encode(&c); err != nil {
		panic(err)
	}
	if err := yaml.NewDecoder(buf).Decode(&cCopy); err != nil {
		panic(err)
	}

	// api key
	cCopy.AzureSayer.SpeechKey = ellipsis.Centering(cCopy.AzureSayer.SpeechKey, 9)

	return &cCopy
}

// AzureSayerConfig is the configuration for the AzureSayer.
type AzureSayerConfig struct {
	// SpeechKey is the key for the Azure Speech API.
	SpeechKey string
	// SpeechRegion is the region for the Azure Speech API.
	SpeechRegion string
	// Roles is a map from role to voiceTemplate.
	Roles map[string]string
	// FormatMicrosoft is the format for the Azure Speech API.
	FormatMicrosoft string
	// FormatMimeSubtype is the mime subtype for the audio format.
	FormatMimeSubtype string
}

var config Config
var configChanged chan struct{}

func initConfig(paths ...string) {
	defaultConfig()
	setupConfig(paths...)

	if err := readConfig(); err != nil {
		log.Fatal(err)
	}

	if err := reloadConfig(); err != nil {
		log.Fatal("loading config failed: ", err)
	}

	log.Println("Config loaded:")
	config.DesensitizedCopy().Write(log.Writer())

	configChanged = watchConfig()
}

func defaultConfig() {
	viper.SetDefault("srv_addr", "50010")
	viper.SetDefault("enabled_sayer", "azure")
	viper.SetDefault("azure_sayer.speech_key", "")
	viper.SetDefault("azure_sayer.speech_region", "")
	viper.SetDefault("azure_sayer.roles", map[string]string{})
	viper.SetDefault("azure_sayer.format_microsoft", "audio-16khz-32kbitrate-mono-mp3")
	viper.SetDefault("azure_sayer.format_mime_subtype", "mp3")
}

func setupConfig(paths ...string) {
	// XXX: Env vars does not work: https://github.com/spf13/viper/issues/188
	// I tried all the workarounds, but none worked.
	// viper.SetEnvPrefix("mes")
	// viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/extremesayer")
	viper.AddConfigPath("$HOME/.extremesayer")

	for _, path := range paths {
		if path == "" {
			continue
		}
		viper.SetConfigFile(path)
	}
}

func readConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			return fmt.Errorf("no config file found, using defaults")
		default:
			return fmt.Errorf("reading config file failed. err: %v", err)
		}
	}

	return nil
}

// watchConfig watches the config file for changes and reloads it.
// It returns a channel that will be sent to when the config is reloaded.
func watchConfig() chan struct{} {
	ch := make(chan struct{}, 10)
	viper.OnConfigChange(func(e fsnotify.Event) {
		slog.Warn("Config file changed.", "file", e.Name)
		if err := reloadConfig(); err != nil {
			slog.Error("reloading config", "err", err)
		}

		slog.Info("reloaded config successfully.")
		ch <- struct{}{}
	})
	viper.WatchConfig()
	return ch
}

func checkConfig() error {
	if config.SrvAddr == "" {
		return fmt.Errorf("SrvAddr must be set")
	}

	if config.EnabledSayer == "azure" {
		if config.AzureSayer.SpeechKey == "" {
			return fmt.Errorf("azure_sayer.speech_key must be set")
		}

		if config.AzureSayer.SpeechRegion == "" {
			return fmt.Errorf("azure_sayer.speech_region must be set")
		}

		if len(config.AzureSayer.Roles) == 0 {
			return fmt.Errorf("azure_sayer.roles must be set")
		}
	}

	if config.EnabledSayer != "azure" {
		return fmt.Errorf("enabled_sayer must be 'azure'")
	}

	return nil
}

func reloadConfig() error {
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("reloaded config: Unmarshal failed. err=%v", err)
	}

	// j, _ := json.MarshalIndent(config, "", "  ")
	// fmt.Printf("reloaded config: %s\n", string(j))
	// fmt.Printf("roles: %v\n", config.AzureSayer.Roles)

	if err := checkConfig(); err != nil {
		return fmt.Errorf("reloaded config: checkConfig failed. err=%v", err)
	}
	return nil
}
