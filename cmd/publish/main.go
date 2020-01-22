package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/mikehelmick/eventutils/pkg/registry"
)

type Publish struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Schema string `json:"schema"`
}

func checkFlag(val *string, name string) {
	if *val == "" {
		log.Fatalf("%v flag is requred, exiting", name)
		os.Exit(1)
	}
}

func main() {
	ceType := flag.String("type", "", "CloudEvents type name")
	ceSource := flag.String("source", "", "CloudEvents source")
	schemaFile := flag.String("schema", "", "Filename that contains schema")
	registry := flag.String("registry", registry.Default, "schema registry")
	flag.Parse()
	checkFlag(ceType, "type")
	checkFlag(ceSource, "source")
	checkFlag(schemaFile, "schema")

	content, err := ioutil.ReadFile(*schemaFile)
	if err != nil {
		log.Fatalf("Error reading schema file: %v", err)
		os.Exit(1)
	}

	data := Publish{*ceType, *ceSource, string(content)}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error encoding JSON payload for RPC: %v", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/publish", *registry)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode == http.StatusOK {
		log.Printf("Schema accepted into registry")
	} else {
		log.Printf("Error publishing schema: %v", resp.Status)
	}
	defer resp.Body.Close()
}
