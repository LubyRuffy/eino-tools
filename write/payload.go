package write

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/shared"
)

// nearbyParamNames maps common wrong keys to the canonical write params.
// This is a field-name hint, not a sample-specific branch.
var nearbyParamNames = map[string]string{
	"contents": "content",
	"path":     "file_path",
}

type writePayload struct {
	filePath string
	content  string
}

func resolveWritePayload(params map[string]interface{}) (writePayload, error) {
	filePath, filePresent, fileErr := shared.LookupStringParam(params, "file_path")
	content, contentPresent, contentErr := shared.LookupStringParam(params, "content")
	if err := joinParamTypeErrors(fileErr, contentErr); err != nil {
		return writePayload{}, err
	}

	keys := receivedKeys(params)
	if !filePresent || filePath == "" {
		return writePayload{}, missingParamError(params, keys, "file_path")
	}
	if !contentPresent {
		return writePayload{}, missingParamError(params, keys, "content")
	}
	return writePayload{filePath: filePath, content: content}, nil
}

func missingParamError(params map[string]interface{}, keys, canonical string) error {
	msg := fmt.Sprintf("%s is required; received keys [%s]", canonical, keys)
	for got, want := range nearbyParamNames {
		if want != canonical {
			continue
		}
		if _, ok := params[got]; !ok {
			continue
		}
		if _, hasCanonical := params[canonical]; hasCanonical {
			continue
		}
		msg += fmt.Sprintf("; use %s not %s", want, got)
	}
	return errors.New(msg)
}

func joinParamTypeErrors(errs ...error) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, "; "))
}

func receivedKeys(params map[string]interface{}) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
