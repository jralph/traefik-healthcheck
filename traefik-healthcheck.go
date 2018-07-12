package main

import (
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/pborman/getopt/v2"
	"log"
	"net/http"
	"os"
	"time"
)

// Configuration settings.
type Configuration struct {
	ListenAddr   string
	PollInterval int
	TraefikHosts []string
	ConsulHost   string
}

// Traefik providers endpoint struct for json response.
type TraefikProviders struct {
	ConsulCatalog struct {
		Backends  map[string]interface{} `json:"backends"`
		Frontends map[string]interface{} `json:"frontends"`
	} `json:"consul_catalog"`
}

// Global variable to determine if the load-balancer is healthy or not.
var healthy bool

func main() {
	configFile := getopt.String('c', "./traefik-healthcheck.json", "The path to the traefik-healthcheck config file.", "string")

	opts := getopt.CommandLine
	opts.Parse(os.Args)

	log.Print("Starting Traefik Healthcheck...")
	log.Printf("Using config file \"%s\"", *configFile)

	config := newConfig(*configFile)

	go pollHealth(config)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !healthy {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	log.Printf("HTTP server listening on: %s", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, nil))

	log.Println("Fnished.")
}

// Create a new configuration setup.
func newConfig(path string) Configuration {
	config := Configuration{
		ListenAddr:   "0.0.0.0:10700",
		PollInterval: 10,
		TraefikHosts: []string{"localhost:8080"},
		ConsulHost:   "localhost:8500",
	}

	if _, err := os.Stat(path); err == nil {
		file, _ := os.Open(path)
		defer file.Close()

		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)

		if err != nil {
			log.Fatal("Unable to read config file. Check json is correct.", err)
		}
	}

	return config
}

// Check that consul is healthy.
func consulIsHealthy(consulAddress string) bool {
	config := api.Config{
		Address: consulAddress,
	}

	client, err := api.NewClient(&config)

	if err != nil {
		log.Print("Error connecting to consul client.", err)
		return false
	}

	status := client.Status()
	leader, err := status.Leader()

	if err != nil {
		log.Print("Error querying consul leader.", err)
		return false
	}

	if leader != "" {
		return true
	}

	return false
}

// Check traefik is healthy.
func traefikIsHealthy(traefikAddresses []string) bool {
	var traefikClient = &http.Client{
		Timeout: time.Second * 10,
	}

	for _, host := range traefikAddresses {
		response, err := traefikClient.Get("http://" + host + "/api/providers")

		if err != nil {
			log.Print("Error contacting traefik providers endpoint.", err)
			return false
		}

		if response.StatusCode != 200 {
			log.Printf("Error fetching traefik providers. Got status code %d", response.StatusCode)
			response.Body.Close()
			return false
		}

		providers := TraefikProviders{}
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&providers)

		if err != nil {
			log.Print(err)
			response.Body.Close()
			return false
		}

		if len(providers.ConsulCatalog.Backends) == 0 {
			log.Print("No backends found in Traefik.")
			response.Body.Close()
			return false
		}

		if len(providers.ConsulCatalog.Frontends) == 0 {
			log.Print("No frontends found in Traefik.")
			response.Body.Close()
			return false
		}

		response.Body.Close()
	}

	return true
}

// Check the overall load balancer is healthy.
func isLBHealthy(config Configuration) bool {
	return consulIsHealthy(config.ConsulHost) && traefikIsHealthy(config.TraefikHosts)
}

// Poll for health changes and save to the global healthy variable.
func pollHealth(config Configuration) {
	healthy = isLBHealthy(config)
	time.Sleep(time.Second * time.Duration(config.PollInterval))
	pollHealth(config)
}
