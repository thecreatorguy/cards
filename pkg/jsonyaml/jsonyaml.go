package jsonyaml

import (
	// Base packages.
	"bytes"
	"encoding/json"
	"fmt"

	// Third party packages.
	"gopkg.in/yaml.v2"
)

// Unmarshal YAML to map[string]interface{} instead of map[interface{}]interface{}.
func Unmarshal(in []byte, out interface{}) error {
    var res interface{}

    if err := yaml.Unmarshal(in, &res); err != nil {
        return err
    }
    var b bytes.Buffer
    json.NewEncoder(&b).Encode(cleanupMapValue(res))
    json.NewDecoder(&b).Decode(out)

    return nil
}

// Marshal YAML wrapper function.
func Marshal(in interface{}) ([]byte, error) {
    return yaml.Marshal(in)
}

func cleanupInterfaceArray(in []interface{}) []interface{} {
    res := make([]interface{}, len(in))
    for i, v := range in {
        res[i] = cleanupMapValue(v)
    }
    return res
}

func cleanupInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
    res := make(map[string]interface{})
    for k, v := range in {
        res[fmt.Sprintf("%v", k)] = cleanupMapValue(v)
    }
    return res
}

func cleanupMapValue(v interface{}) interface{} {
    switch v := v.(type) {
    case []interface{}:
        return cleanupInterfaceArray(v)
    case map[interface{}]interface{}:
        return cleanupInterfaceMap(v)
    case string:
        return v
    default:
        return fmt.Sprintf("%v", v)
    }
}