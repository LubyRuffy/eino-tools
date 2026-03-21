package shared

import "fmt"

type ErrorPassthrough func(error) bool

func ToolInvokableDefer(result *string, errp *error, shouldPassthrough ErrorPassthrough) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("error: panic: %v", r)
		if result != nil && *result != "" {
			*result += "\n" + msg
		} else if result != nil {
			*result = msg
		}
		if errp != nil {
			*errp = nil
		}
		return
	}

	if errp == nil || *errp == nil {
		return
	}
	if shouldPassthrough != nil && shouldPassthrough(*errp) {
		return
	}

	msg := fmt.Sprintf("error: %v", *errp)
	if result != nil && *result != "" {
		*result += "\n" + msg
	} else if result != nil {
		*result = msg
	}
	*errp = nil
}
