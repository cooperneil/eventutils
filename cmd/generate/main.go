package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/alecthomas/jsonschema"
	"github.com/mikehelmick/eventutils/pkg/registry"
)

var reverseTypes = map[string]string{
	"integer": "int64",
	"string":  "string",
}

func caseSegment(s string) string {
	return fmt.Sprintf("%s%s", strings.ToUpper(s[0:1]), strings.ToLower(s[1:]))
}

// https://schemaregistry-byacsc76eq-uc.a.run.app/download/com.mikehelmick.eventutils.user.user.json
func getTypeName(ref string) string {
	ref = strings.TrimSuffix(ref, ".json")
	cutPoint := strings.LastIndex(ref, ".")
	typeName := ref[cutPoint+1:]
	typeName = caseSegment(typeName)
	return typeName
}

func getFieldName(field string) string {
	res := ""
	for _, segment := range strings.Split(field, "_") {
		res = fmt.Sprintf("%s%s", res, caseSegment(segment))
	}
	return res
}

func downloadSchema(schemaURL string, schema *jsonschema.Type) {
	log.Printf("Downloading from %v", schemaURL)
	resp, err := http.Get(schemaURL)
	if err != nil {
		log.Fatalf("Unable to download schama: %v", err)
		os.Exit(1)
	}

	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&schema)
}

func processFields(schema jsonschema.Type, required map[string]bool) []StructField {
	rtn := make([]StructField, 0)

	for prop, data := range schema.Properties {
		name := getFieldName(prop)
		fieldType := reverseTypes[data.Type]
		extra := ",omitempty"
		if required[prop] {
			extra = ""
		}
		tags := fmt.Sprintf("`json:\"prop%s\"`", extra)

		rtn = append(rtn, StructField{name, fieldType, tags})
	}

	return rtn
}

func renderTemplate(tFile, oFile string, data map[string]interface{}) {
	t, err := template.ParseFiles(tFile)
	if err != nil {
		log.Fatalf("Unable to load template %s : %v", tFile, err)
		panic(err)
	}
	f, err := os.Create(oFile)
	if err != nil {
		log.Fatalf("Unable to create file %s : %v", oFile, err)
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	t.Execute(w, data)
	w.Flush()
}

type StructField struct {
	Name, FieldType, Tags string
}

func outputFile(dir, fname string) string {
	return fmt.Sprintf("%s/%s", dir, fname)
}

func getServiceName(goType string) string {
	return strings.ToLower(goType)
}

func generateData(url, ceType, ceSource, gitRepo string) map[string]interface{} {
	schemaURL := fmt.Sprintf("%sdownload/%s", url, ceType)
	var schema jsonschema.Type
	downloadSchema(schemaURL, &schema)

	required := make(map[string]bool)
	for _, req := range schema.Required {
		required[req] = true
	}

	goType := getTypeName(schema.Ref)
	svcName := getServiceName(goType)
	fields := processFields(schema, required)

	data := make(map[string]interface{})
	data["GoType"] = goType
	data["StructFields"] = fields
	data["ServiceName"] = svcName
	data["GitRepo"] = gitRepo
	data["cetype"] = ceType
	data["cesource"] = ceSource
	data["schema"] = schemaURL

	return data
}

func runCommand(command, dir string) {
	log.Printf("Running %s", command)
	cmd := exec.Command(command)
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running dep ensure: %v", err)
	}

}

func generateReceiver(url, ceType, ceSource, dir, gitRepo string) {
	data := generateData(url, ceType, ceSource, gitRepo)

	// Generate files
	os.Mkdir(dir, os.ModePerm)
	log.Printf("Generating golang function...")
	renderTemplate("templates/Gopkg.toml.tmpl", outputFile(dir, "Gopkg.toml"), data)
	renderTemplate("templates/receiver_main.go.tmpl", outputFile(dir, "main.go"), data)
	renderTemplate("templates/service.yaml.tmpl", outputFile(dir, "service.yaml"), data)
	renderTemplate("templates/trigger.yaml.tmpl", outputFile(dir, "trigger.yaml"), data)

	runCommand("dep ensure", dir)
	runCommand("go fmt ./...", dir)
	log.Printf("You're good to go: cd %s; ko apply -f service.yaml", dir)
}

func generateSender(url, ceType, ceSource, dir, gitRepo string) {
	data := generateData(url, ceType, ceSource, gitRepo)

	// Generate files
	os.Mkdir(dir, os.ModePerm)
	log.Printf("Generating golang client...")
	renderTemplate("templates/client.go.tmpl", outputFile(dir, "main.go"), data)

	runCommand("dep ensure", dir)
	runCommand("go fmt ./...", dir)
	log.Printf("Ready: cd %s", dir)
}

func main() {
	gitRepo := flag.String("gitrepo", "", "github repo to write to.")
	ceType := flag.String("type", "", "ce type to generate application for.")
	urlFlag := flag.String("registry", registry.Default, "schema registry")
	mode := flag.String("mode", "receiver", "One of: receiver, both, sender")
	ceSource := flag.String("source", "", "CloudEvents source to use when generating clients")
	//returnType := flag.String("rtype", "", "CE Return Type for mode both")
	flag.Parse()

	if *gitRepo == "" {
		log.Fatalf("You must provide a gitrepo to write to")
		os.Exit(1)
	}
	dir := fmt.Sprintf("%s/src/github.com/%s", os.Getenv("GOPATH"), *gitRepo)

	switch {
	case *mode == "receiver":
		log.Printf("Generating receiver code to %s", dir)
		generateReceiver(*urlFlag, *ceType, *ceSource, dir, *gitRepo)
	case *mode == "sender":
		log.Printf("Generating sender code to %s", dir)
		generateSender(*urlFlag, *ceType, *ceSource, dir, *gitRepo)
	default:
		log.Fatalf("Unsupported mode: `%s`, must be one of receiver, both, or sender.", *mode)
	}

}
