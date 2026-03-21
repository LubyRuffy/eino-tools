package shared

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func ParseToolArgs(argumentsInJSON string) (map[string]interface{}, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	return params, nil
}

func GetStringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func GetBoolParam(params map[string]interface{}, key string) bool {
	if params == nil {
		return false
	}
	value, ok := params[key]
	if !ok || value == nil {
		return false
	}
	boolean, _ := value.(bool)
	return boolean
}

func GetIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}
	value, ok := params[key]
	if !ok || value == nil {
		return defaultValue
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
