package structured

import "fmt"

type Error struct {
	// MoreInfo is additional information about the error e.g. a link to context describing the context for the error.
	MoreInfo string
	// Impact is the likely impact of the error on system function e.g. "Proxies are unable to communicate with Istiod."
	Impact string
	// Action is the next step the user should take e.g. "Open an issue or bug report."
	Action string
	// LikelyCause is the likely cause for the error e.g. "Software bug."
	LikelyCause string
	// Err is the original error string.
	Err error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("\tmoreInfo=%s impact=%s action=%s likelyCause=%s err=%v",
		e.MoreInfo, e.Impact, e.Action, e.LikelyCause, e.Err)
}

func NewErr(serr *Error, err error) *Error {
	// Make a copy so that dictionary entry is not modified.
	ne := *serr
	ne.Err = err
	return &ne
}

func (e *Error) Unwrap() error { return e }
