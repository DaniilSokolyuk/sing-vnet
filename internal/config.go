package internal

import (
	"encoding/json"
	"log"
	"os"
)

type Conf struct {
	LoggerLevel   string `json:"logger_level"`
	VnetInterface string `json:"vnet_interface"`
	Sing          struct {
		FileConfig   string `json:"file_config"`
		ForceVersion string `json:"force_version"`
		RenameExec   string `json:"rename_exec"`
		ExecPath     string `json:"exec_path"`
		InboundTag   string `json:"inbound_tag"`
	} `json:"sing"`
}

func LoadConfig() Conf {
	configPath := "vnet.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var conf Conf
	if err := json.Unmarshal(data, &conf); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	return conf
}

type MainConfig struct {
	Inbounds []struct {
		Type                   string `json:"type"`
		Tag                    string `json:"tag"`
		Inet4Address           string `json:"inet4_address"`
		StrictRoute            bool   `json:"strict_route"`
		Stack                  string `json:"stack"`
		Sniff                  bool   `json:"sniff"`
		DomainStrategy         string `json:"domain_strategy"`
		EndpointIndependentNat bool   `json:"endpoint_independent_nat"`
		InterfaceName          string `json:"interface_name"`
		MTU                    int    `json:"mtu"`
	} `json:"inbounds"`
	Route struct {
		DefaultInterface    bool `json:"default_interface"`
		AutoDetectInterface bool `json:"auto_detect_interface"`
	} `json:"route"`
}
