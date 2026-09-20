package edit

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/shared"
)

type editPayload struct {
	search    string
	replace   string
	patch     string
	useSearch bool
}

func resolveEditPayload(params map[string]interface{}) (editPayload, error) {
	search, searchPresent, searchErr := shared.LookupStringParam(params, "search_block")
	replace, replacePresent, replaceErr := shared.LookupStringParam(params, "replace_block")
	patch, patchPresent, patchErr := shared.LookupStringParam(params, "patch")
	if err := joinParamTypeErrors(searchErr, replaceErr, patchErr); err != nil {
		return editPayload{}, err
	}

	keys := receivedKeys(params)
	switch {
	case searchPresent && replacePresent:
		if search == "" {
			return editPayload{}, fmt.Errorf("search_block must be non-empty")
		}
		return editPayload{search: search, replace: replace, useSearch: true}, nil
	case searchPresent:
		return editPayload{}, fmt.Errorf("replace_block is required when search_block is set; use an empty string to delete; received keys [%s]", keys)
	case replacePresent:
		return editPayload{}, fmt.Errorf("search_block is required when replace_block is set; received keys [%s]", keys)
	case patchPresent && patch != "":
		return editPayload{patch: patch}, nil
	default:
		return editPayload{}, fmt.Errorf("missing edit payload: received keys [%s]; provide search_block and replace_block together (replace_block may be empty to delete), or patch", keys)
	}
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
