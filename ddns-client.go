package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// config.yaml
type YamlConfig struct {
	WaitTime      float32 `yaml:"waitTime"`
	MaxAttempts   int     `yaml:"maxAttempts"`
	CheckIPChange bool    `yaml:"checkIPChange"`
	LogPath       string  `yaml:"logPath"`
	Domain        string  `yaml:"domain"`
	DDNSUser      string  `yaml:"ddnsUser"`
	Token         string  `yaml:"token"`
}

func readYaml(configPath string, cfgPtr *YamlConfig) error {
	configFile, err := os.Open(configPath) // *os.File, error
	if err != nil {
		log.Panicln("Config file open failed:", configPath)
		return err
	}
	defer configFile.Close()
	configData, err := io.ReadAll(configFile) // []uint8, error
	if err != nil {
		log.Panicln("Config file io.ReadAll() failed:", err)
		return err
	}
	err = yaml.Unmarshal(configData, cfgPtr)
	if err != nil {
		log.Panicln("Config file yaml.Unmarshal() failed:", err)
		return err
	}
	return nil
}

// TODO: Rewrite with context.Context
// http get wrapper
func httpReq(payload string, maxAttempts int, waitTime float32) (*http.Response, error) {
	for i := 0; i < maxAttempts; i++ {
		resp, err := http.Get(payload)
		if err != nil {
			log.Println("HTTP GET Retrying:", err)
			time.Sleep(time.Duration(waitTime) * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			log.Println("HTTP GET 200 Success.")
			return resp, nil
		}
		log.Println("HTTP GET Retrying:", resp.StatusCode)

	}
	log.Println("HTTP GET Failed.") // Don't log payload, might include token
	// resp is not retuned for non-200 responses
	return nil, fmt.Errorf("HTTP GET failed.")
}

// This function uses https://ip4.me/ to get the global IP of this machine.
func whatsMyIP(cfgPtr *YamlConfig) (string, error) {
	resp, err := httpReq(
		"https://ip4.me/api/", // API URI
		cfgPtr.MaxAttempts,
		cfgPtr.WaitTime,
	)
	if err != nil {
		log.Println("httpReq() failed.")
		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("io.ReadAll() failed.")
		return "", err
	}
csvLoop:
	// ex) string(body): "IPv4,61.194.159.231,v1.1,,,
	for _, cell := range strings.Split(string(body), ",") {
		cellAry := strings.Split(cell, ".")
		if len(cellAry) != 4 { // IPv4 address has 4 segments
			continue
		}
		for _, val := range cellAry {
			digit, err := strconv.Atoi(val)
			if err != nil {
				continue csvLoop
			}
			if digit < 1 || 256 < digit {
				continue csvLoop
			}
			// TODO: Match against IP Address blacklist and raise error.
			//  - private IP
			//  - Well known static IP
		}
		return cell, nil
	}
	return "", fmt.Errorf("Valid IP address not in response: %q", string(body))
}

func updateDDNS(cfgPtr *YamlConfig) (*http.Response, error) {
	updateIpPayload :=
		"https://f5.si/update.php?domain=" + cfgPtr.DDNSUser +
			"&password=" + cfgPtr.Token + "&format=json"
	resp, err := httpReq(
		updateIpPayload,
		cfgPtr.MaxAttempts,
		cfgPtr.WaitTime,
	)
	if err != nil {
		log.Println("DDNS update failed at http request.")
		return resp, err
	}

	var respJSON map[string]interface{}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("DDNS update failed at io.ReadAll().")
		return resp, err
	}
	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		log.Println("DDNS update failed at decoding JSON.")
		return resp, err
	}

	// Success return
	if respJSON["result"] == "OK" {
		log.Println("DDNS updated to:", respJSON["remote_ip"])
		return resp, nil
	}

	// Errors
	log.Printf("DDNS Now ERROR %q: %q\n", respJSON["errorcode"], respJSON["errormsg"])
	return resp, fmt.Errorf("DDNS Now ERROR %q: %q", respJSON["errorcode"], respJSON["errormsg"])
}

func main() {
	// Read config
	// configPath := "config.yaml"
	configPath := os.Getenv("HOME") + "/.config/ddns-client/config.yaml"
	cfg := YamlConfig{}
	err := readYaml(configPath, &cfg)
	if err != nil {
		log.Panicln("config yaml could not be read.")
		return
	}

	// Open log
	logFile, err := os.OpenFile(cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panicln("logfile open failed:", cfg.LogPath)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	for {
		// Not sure if this is more effective.
		if cfg.CheckIPChange {
			// Check resolved IP
			ipSlc, err := net.LookupIP(cfg.Domain)
			if err != nil {
				log.Println("IP lookup Failed for:", cfg.Domain)
				time.Sleep(time.Duration(60) * time.Second)
				continue
			}
			if len(ipSlc) != 1 {
				log.Panicln("Lookup returned multiple IP address for:", cfg.Domain)
				return
			}
			if ipSlc[0].To4() == nil {
				log.Panicln("IPv6 might be returned for:", cfg.Domain)
				return
			}
			registeredIPv4 := ipSlc[0].String()
			fmt.Println("Resolved IP Address: ", registeredIPv4)

			// Check current IP
			currentIPv4, err := whatsMyIP(&cfg)
			if err != nil {
				log.Println("Couldn't get current IP.")
				time.Sleep(time.Duration(60) * time.Second)
				continue
			}
			log.Println("Current IP Address: ", currentIPv4)

			if registeredIPv4 == currentIPv4 {
				log.Println("IP registered correctly, doing nothing.")
				time.Sleep(time.Duration(60) * time.Second)
				continue
			}
		}

		_, err := updateDDNS(&cfg)
		if err != nil {
			log.Println("DDNS update failed.")
		} else {
			log.Println("DDNS update success")
		}

		// ToL < 60secs is ignored by server
		time.Sleep(time.Duration(60) * time.Second)
	}
}
