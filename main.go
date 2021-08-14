package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"time"
)

var (
	logFile        *os.File
	logger         *logrus.Logger
	configFilePath string
	configPath     string
)

type Config struct {
	Update int    `json:"update"`
	List   []List `json:"list"`
}
type List struct {
	Email      string    `json:"email"`
	DuckDNS    string    `json:"duckDNS"`
	IP         string    `json:"ip"`
	Status     int       `json:"status"` // if == 0 deletes entry
	LastUpdate time.Time `json:"last_update"`
}

func main() {
	// Set config Path
	getConfigDir()
	// Generate default config file
	generateConfigFiles()
	// Set Log file
	setLogger()
	// Close logger file
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			logger.Error(err)
		}
	}(logFile)
	// run app
	run()

}

func setLogger() {
	var err error
	// create the logger
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.SetOutput(os.Stdout)
	logFile, err = os.OpenFile(fmt.Sprint(configPath, "/log/duckSSH.logger"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetOutput(logFile)
}

func run() {
	logger.Info("Start duckSSH")
	for {
		logger.Info("Start Verification")
		changes := false
		appConfig := loadConfig()
		var newConfig []List
		for i, entry := range appConfig.List {

			// delete entry from firewall
			if entry.Status == 0 {
				//ufw show added | awk '/192.168.0.4/{ gsub("ufw","ufw delete",$0); system($0)}'
				cmd := exec.Command("ufw", "show", "added", "|", "awk", "'/"+entry.IP+"/{ gsub(\"ufw\",\"ufw delete\",$0); system($0)}'")
				err := cmd.Run()
				if err != nil {
					logger.Warn("There was a problem removing the firewall entry: %v", err)
				}
				logger.Warn("Entry was removed with success")
				changes = true
				continue
			}

			ips, _ := net.LookupIP(entry.DuckDNS)
			for _, duckdnsIP := range ips {
				if ipv4 := duckdnsIP.To4(); ipv4 != nil {
					if entry.IP != ipv4.String() {
						// Remove entry from firewall
						cmd := exec.Command("ufw", "show", "added", "|", "awk", "'/"+entry.IP+"/{ gsub(\"ufw\",\"ufw delete\",$0); system($0)}'")
						err := cmd.Run()
						if err != nil {
							logger.Warn("There was a problem removing the firewall entry: %v", err)
						}

						// Add new ip to firewall
						cmd = exec.Command("ufw", "allow", "from", ipv4.String(), "to", "any", "port", "22", "proto", "tcp")
						err = cmd.Run()
						if err != nil {
							logger.Warn("It was not possible to add an entry firewall: %v", err)
						}
						logger.Warn(entry.DuckDNS, " IP has changed to: ", ipv4.String())
						appConfig.List[i].IP = ipv4.String()
						appConfig.List[i].LastUpdate = time.Now()
						changes = true
					}
				}
			}
			newConfig = append(newConfig, entry)
		}

		// Save changes to config file
		if changes {
			appConfig.List = newConfig
			saveToFile(appConfig)
		}

		// Time to sleep for the next update
		time.Sleep(time.Duration(appConfig.Update) * time.Minute)
	}
}
func loadConfig() *Config {

	jsonFile, err := os.Open(configFilePath)
	if err != nil {
		logger.Error(err)
	}
	logger.Info("Config file loaded with success.")
	// defer the closing of our jsonFile so that we can parse it later on
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			logger.Error(err)
		}
	}(jsonFile)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var result *Config
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		logger.Error(err)
	}
	return result

}

func saveToFile(appConfig *Config) {
	file, err := json.MarshalIndent(appConfig, "", " ")
	if err != nil {
		logger.Error(err)
	}
	_ = ioutil.WriteFile(configFilePath, file, 0644)
}

func generateConfigFiles() {
	configFilePath = fmt.Sprint(configPath, "/config.json")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		appConfig := &Config{
			Update: 5,
			List: []List{
				{
					Email:      "duckSSH@domain.com",
					DuckDNS:    "duckSSH.duckdns.org",
					IP:         "127.0.0.1",
					Status:     1,
					LastUpdate: time.Now(),
				}, {
					Email:      "duckSSH2@domain.com",
					DuckDNS:    "duckSSH2.duckdns.org",
					IP:         "127.0.0.2",
					Status:     1,
					LastUpdate: time.Now(),
				},
			},
		}
		saveToFile(appConfig)
	}
}

func getConfigDir() string {
	configPath = fmt.Sprint(os.Getenv("HOME"), "/.duckSSH")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// default config dir
		if err := os.Mkdir(configPath, 0755); os.IsExist(err) {
		}
		// create logger dir in config dir
		if err := os.Mkdir(configPath+"/logger", 0755); os.IsExist(err) {
		}
	}
	return configPath
}
