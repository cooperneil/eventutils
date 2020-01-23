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

	// Convert - to Camel case
	i := strings.Index(typeName, "-")
	for i >= 0 {
		typeName = fmt.Sprintf("%s%s%s", typeName[0:i], strings.ToUpper(typeName[i+1:i+2]), typeName[i+2:])
		i = strings.Index(typeName, "-")
	}

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
		tags := fmt.Sprintf("`json:\"%s%s\"`", prop, extra)

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

// StructField is used to genreate go types.
type StructField struct {
	Name, FieldType, Tags string
}

func getServiceName(goType string) string {
	return strings.ToLower(goType)
}

// Config is the internal configuration of this generator.
type Config struct {
	Dir      string
	URL      string
	CEType   string
	CESource string
	GitRepo  string
	Target   string

	ReplyType   string
	ReplySource string
}

func (c Config) outputFile(fname string) string {
	return fmt.Sprintf("%s/%s", c.Dir, fname)
}

func (c Config) generateData() map[string]interface{} {
	schemaURL := fmt.Sprintf("%sdownload/%s", c.URL, c.CEType)
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
	data["GitRepo"] = c.GitRepo
	data["cetype"] = c.CEType
	data["cesource"] = c.CESource
	data["schema"] = schemaURL
	data["target"] = c.Target
	data["TypeComment"] = fmt.Sprintf("%s is a generated type for CloudEvents binding", goType)

	return data
}

func (c Config) addReplyData(data map[string]interface{}) {
	schemaURL := fmt.Sprintf("%sdownload/%s", c.URL, c.ReplyType)
	var schema jsonschema.Type
	downloadSchema(schemaURL, &schema)

	required := make(map[string]bool)
	for _, req := range schema.Required {
		required[req] = true
	}

	goType := getTypeName(schema.Ref)
	fields := processFields(schema, required)

	data["ReplyGoType"] = goType
	data["ReplyStructFields"] = fields
	data["ReplyCEType"] = c.ReplyType
	data["ReplyCESource"] = c.ReplySource
	data["ReplySchema"] = schemaURL
}

func runCommand(command, dir string) {
	log.Printf("Running %s", command)
	cmd := exec.Command(fmt.Sprintf("cd %s && %s", dir, command))
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running %s: %v", command, err)
	}
}

func generateReceiver(c Config) {
	data := c.generateData()

	// Generate files
	os.Mkdir(c.Dir, os.ModePerm)
	log.Printf("Generating golang function...")
	renderTemplate("templates/Gopkg.toml.tmpl", c.outputFile("Gopkg.toml"), data)
	renderTemplate("templates/receiver_main.go.tmpl", c.outputFile("main.go"), data)
	renderTemplate("templates/service.yaml.tmpl", c.outputFile("service.yaml"), data)
	renderTemplate("templates/trigger.yaml.tmpl", c.outputFile("trigger.yaml"), data)

	//runCommand("/usr/local/bin/dep ensure", c.Dir)
	//runCommand("/usr/local/go/bin/go fmt ./...", c.Dir)
	log.Printf("You're good to go: cd %s; ko apply -f service.yaml", c.Dir)
}

func generateBoth(c Config) {
	data := c.generateData()
	c.addReplyData(data)

	// Generate files
	os.Mkdir(c.Dir, os.ModePerm)
	log.Printf("Generating golang function...")
	renderTemplate("templates/Gopkg.toml.tmpl", c.outputFile("Gopkg.toml"), data)
	renderTemplate("templates/reply.go.tmpl", c.outputFile("main.go"), data)
	renderTemplate("templates/service.yaml.tmpl", c.outputFile("service.yaml"), data)
	renderTemplate("templates/trigger.yaml.tmpl", c.outputFile("trigger.yaml"), data)

	log.Printf("You're good to go: cd %s; ko apply -f service.yaml", c.Dir)
}

func generateSender(c Config) {
	data := c.generateData()

	// Generate files
	os.Mkdir(c.Dir, os.ModePerm)
	log.Printf("Generating golang client...")
	renderTemplate("templates/client.go.tmpl", c.outputFile("main.go"), data)
	renderTemplate("templates/Gopkg.toml.tmpl", c.outputFile("Gopkg.toml"), data)

	//runCommand("/usr/local/bin/dep ensure", c.Dir)
	//runCommand("/usr/local/go/bin/go fmt ./...", c.Dir)
	log.Printf("Ready: cd %s", c.Dir)
}

func main() {
	gitRepo := flag.String("gitrepo", "", "github repo to write to.")
	ceType := flag.String("type", "", "ce type to generate application for.")
	urlFlag := flag.String("registry", registry.Default, "schema registry")
	mode := flag.String("mode", "receiver", "One of: receiver, both, sender")
	ceSource := flag.String("source", "", "CloudEvents source to use when generating clients")
	target := flag.String("target", "http://default-broker.default.svc.cluster.local", "target for clients")
	returnType := flag.String("rtype", "", "CE Return Type for mode both")
	replySource := flag.String("rsource", "", "CE Reply Source")
	flag.Parse()

	if *gitRepo == "" {
		log.Fatalf("You must provide a gitrepo to write to")
		os.Exit(1)
	}
	dir := fmt.Sprintf("%s/src/github.com/%s", os.Getenv("GOPATH"), *gitRepo)

	cfg := Config{
		Dir:         dir,
		URL:         *urlFlag,
		CEType:      *ceType,
		CESource:    *ceSource,
		GitRepo:     *gitRepo,
		Target:      *target,
		ReplyType:   *returnType,
		ReplySource: *replySource,
	}
	if cfg.ReplySource == "" {
		cfg.ReplySource = cfg.CESource
	}

	switch {
	case *mode == "receiver":
		log.Printf("Generating receiver code to %s", dir)
		generateReceiver(cfg)
	case *mode == "sender":
		log.Printf("Generating sender code to %s", dir)
		generateSender(cfg)
	case *mode == "both":
		log.Printf("Generating receiver with rerply to %s", dir)
		generateBoth(cfg)
	default:
		log.Fatalf("Unsupported mode: `%s`, must be one of receiver, both, or sender.", *mode)
	}
}
