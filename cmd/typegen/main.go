package main

import (
	"fmt"
	"os"

	"github.com/labstack/gommon/log"
	"github.com/mikehelmick/eventutils/pkg/generate"
	"github.com/mikehelmick/eventutils/pkg/user"
)

func writeFile(fName, content string) {
	file, err := os.Create(fName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	file.WriteString(content)
}

func main() {
	// Replace the type of this variable with one of the type you
	// wish to generate a JSON Schema for.
	// Your type will need to be included.
	var genType user.User
	// Replace this with the type string of the
	ceSource := "com.mikehelmick.example.login"
	ceType := "com.mikehelmick.eventutils.user.user"

	// This is a public registry of types.
	registry := "https://registry-XX-uc.a.run.app"

	// No need to modify below here.
	jsonString, err := generate.Schema(ceType, registry, genType)
	if err != nil {
		log.Errorf("Error generating schema: %v", err)
		return
	}
	fmt.Println(jsonString)
	fmt.Println("------------------------")
	yamlString, err := generate.EventType(ceType, ceSource, registry)
	if err != nil {
		log.Errorf("Error generating EventType: %v", err)
		return
	}
	fmt.Println(yamlString)

	// Write the files.
	writeFile(fmt.Sprintf("%s.json", ceType), jsonString)
	writeFile("event_type.yaml", yamlString)
}
