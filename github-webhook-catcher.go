package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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

	if *command == "" {
		fmt.Println("-command is required")
		flag.Usage()
		os.Exit(1)
	}

	http.HandleFunc("/", handleWebHook(*sourceHost, *command, *accessToken))
	if *tlsKey != "" && *tlsCert != "" {
		log.Fatal(http.ListenAndServeTLS(":"+*port, *tlsCert, *tlsKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(":"+*port, nil))
	}
}

func handleWebHook(source, cmd, accessToken string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host != source {
			log.Printf("[WARN] Got request from '%s' but expected '%s'. Ignoring\n", r.Host, source)
			w.WriteHeader(401)
			return
		}

		if accessToken != "" && accessToken != r.URL.RawQuery {
			log.Println("[WARN] Request did not provide expected access token. Ignoring")
			w.WriteHeader(401)
			return
		}

		command := exec.Command(cmd)
		command.Stderr = os.Stderr
		command.Stdout = os.Stdout

		json, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			log.Printf("[ERROR] Problem reading body of http request: %s\n", err)
			w.WriteHeader(400)
			return
		}
		w.WriteHeader(200)

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

		_, err = fmt.Fprintf(cw, string(json))
		if err != nil {
			log.Printf("[ERROR] Problem writing to the stdin pipe of the command '%s': %s\n", cmd, err)
			return
		}

		log.Printf("[INFO] Got WebHook notification from %s: %s", source, r.RemoteAddr)
	}
}
