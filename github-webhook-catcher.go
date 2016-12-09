package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
)

func main() {
	sourceHost := flag.String(
		"source-host",
		"github.com",
		"Source host where the webhook notification will come from",
	)

	command := flag.String(
		"command",
		"",
		"Command to execute once a webhook notification is received. Required",
	)

	queue := flag.String(
		"queue-addr",
		"",
		"Siberite/Memcache address to queue push event in. Required, but mutually exclusive to command option.",
	)

	tlsKey := flag.String(
		"tls-key",
		"",
		"Path to TLS key used to support SSL encryption",
	)

	tlsCert := flag.String(
		"tls-cert",
		"",
		"Path to TLS certificate used to support SSL encryption",
	)

	port := flag.String(
		"port",
		"8088",
		"Port to listen for webhook notifications on",
	)

	accessToken := flag.String(
		"access-token",
		"",
		"If provided, any webhook notification must pass the same token in the query string of the request",
	)

	flag.Parse()

	if *command == "" && *queue == "" {
		fmt.Println("-command or -queue is required")
		flag.Usage()
		os.Exit(1)
	}

	var mc *memcache.Client
	if *queue != "" {
		mc = memcache.New(*queue)
	}

	http.HandleFunc("/", handleWebHook(*sourceHost, *command, *accessToken, mc))
	if *tlsKey != "" && *tlsCert != "" {
		log.Fatal(http.ListenAndServeTLS(":"+*port, *tlsCert, *tlsKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(":"+*port, nil))
	}
}

func handleWebHook(source, cmd, accessToken string, mc *memcache.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		remote := r.RemoteAddr[0:strings.LastIndex(r.RemoteAddr, ":")]

		if remote != source {
			log.Printf("[WARN] Got request from '%s' but expected '%s'. Ignoring\n", remote, source)
			w.WriteHeader(401)
			return
		}

		if accessToken != "" && accessToken != r.URL.RawQuery {
			log.Println("[WARN] Request did not provide expected access token. Ignoring")
			w.WriteHeader(401)
			return
		}

		jsonBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("[ERROR] Problem reading body of http request: %s\n", err)
			w.WriteHeader(400)
			return
		}
		w.WriteHeader(200)

		if mc != nil { // Push to a queue and do not send to a command
			// Determine repo name (e.g. jq -r '.repository.name')
			var jsonObj map[string]interface{}
			err = json.Unmarshal(jsonBytes, &jsonObj)
			if err != nil {
				log.Println("[ERROR] Could not unmarshal json received from github: ", err)
				return
			}

			repo := jsonObj["repository"].(map[string]interface{})["name"].(string)
			err = mc.Set(&memcache.Item{Key: repo, Value: jsonBytes})
			if err != nil {
				log.Println("[ERROR] Could not set key in the queue. Check the queue service.")
				return
			}

			log.Printf("[INFO] Stored push event in queue '%s'\n", repo)
			return
		}

		command := exec.Command(cmd)
		command.Stderr = os.Stderr
		command.Stdout = os.Stdout

		cw, err := command.StdinPipe()
		if err != nil {
			log.Printf("[ERROR] Problem creating stdin pipe to the command '%s': %s\n", cmd, err)
			return
		}
		defer cw.Close()

		err = command.Start()
		if err != nil {
			log.Printf("[ERROR] Problem running the command '%s': %s\n", cmd, err)
			return
		}

		_, err = fmt.Fprintf(cw, string(jsonBytes))
		if err != nil {
			log.Printf("[ERROR] Problem writing to the stdin pipe of the command '%s': %s\n", cmd, err)
			return
		}

		log.Println("[INFO] Got WebHook notification from", r.RemoteAddr, ". Sent to", cmd)
	}
}
