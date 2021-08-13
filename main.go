package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"time"
)

var configFilePath string

type Config struct {
	Update int    `json:"update"`
	List   []List `json:"list"`
}
type List struct {
	Email   string `json:"email"`
	DuckDNS string `json:"duckDNS"`
	IP      string `json:"ip"`
	Status  int    `json:"status"` // if == 0 deletes entry
}

func main() {
	configPath := getConfigDir()
	generateConfigFiles(configPath)
	run()
}

func run() {
	log.Info("Start duckSSH")
	for {
		changes := false
		appConfig := loadConfig()
		newConfig := []List{}
		for i, entry := range appConfig.List {

			if entry.Status == 0 {
				// delete from firewall
				// @todo remove from ufw and configfile

				changes = true
				continue
			}

			ips, _ := net.LookupIP(entry.DuckDNS)
			for _, duckdnsIP := range ips {
				if ipv4 := duckdnsIP.To4(); ipv4 != nil {
					if entry.IP != ipv4.String() {
						appConfig.List[i].IP = ipv4.String()
						// @todo remove old ip from ufw
						// @todo add new ip in ufw
						log.Warn(entry.DuckDNS, " ip has changes to: ", ipv4.String())
						changes = true
					}
				}
			}
			newConfig = append(newConfig, entry)
		}
		if changes {
			appConfig.List = newConfig
			saveToFile(appConfig)
		}
		time.Sleep(time.Minute * time.Duration(appConfig.Update))
	}
}
func loadConfig() *Config {

	jsonFile, err := os.Open(configFilePath)
	if err != nil {
		log.Error(err)
	}
	log.Info("Config file loaded with success.")
	// defer the closing of our jsonFile so that we can parse it later on
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			log.Error(err)
		}
	}(jsonFile)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var result *Config
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		log.Error(err)
	}
	return result

}

func saveToFile(appConfig *Config) {
	file, err := json.MarshalIndent(appConfig, "", " ")
	if err != nil {
		log.Error(err)
	}
	_ = ioutil.WriteFile(configFilePath, file, 0644)
}

func generateConfigFiles(configPath string) {
	configFilePath = fmt.Sprint(configPath, "/config.json")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		appConfig := &Config{
			Update: 5,
			List: []List{
				{
					Email:   "duckSSH@domain.com",
					DuckDNS: "duckSSH.duckdns.org",
					IP:      "127.0.0.1",
				}, {
					Email:   "duckSSH2@domain.com",
					DuckDNS: "duckSSH2.duckdns.org",
					IP:      "127.0.0.2",
				},
			},
		}
		saveToFile(appConfig)
	}
}

func getConfigDir() string {
	configPath := fmt.Sprint(os.Getenv("HOME"), "/.duckSSH")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.Mkdir(configPath, 0755); os.IsExist(err) {
		}
	}
	return configPath
}
