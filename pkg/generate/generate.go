package generate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/jsonschema"
	"github.com/ghodss/yaml"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
)

// RefURI is the Ref tag that should be included for this particular schema.
func RefURI(ceType, registry string) string {
	return fmt.Sprintf("%sdownload/%s.json", registry, ceType)
}

// Schema writes a JSON schema that can be used for the specified type.
func Schema(ceType, registry string, t interface{}) (string, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		ExpandedStruct:            true,
	}
	schema := reflector.Reflect(t)

	schema.Version = "http://json-schema.org/schema#"
	schema.Ref = RefURI(ceType, registry)

	st := reflect.TypeOf(t)
	for i := 0; i < st.NumField(); i++ {

	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), err
}

// EventType returns the Knative binding necessary to register this custom type
func EventType(ceType, ceSource, registry string) (string, error) {
	eventType := &eventing.EventType{}
	eventType.TypeMeta.APIVersion = "eventing.knative.dev/v1alpha1"
	eventType.TypeMeta.Kind = "EventType"
	eventType.ObjectMeta.SetName(ceType)
	eventType.ObjectMeta.SetNamespace("default")

	eventType.Spec.Type = ceType
	eventType.Spec.Schema = RefURI(ceType, registry)
	eventType.Spec.Source = ceSource
	eventType.Spec.Broker = "default"

	data, err := yaml.Marshal(&eventType)
	if err != nil {
		return "", err
	}
	sData := string(data)
	sData = sData[0:strings.Index(sData, "status: {}")]
	return sData, err
}
