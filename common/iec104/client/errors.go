package client

import (
	"errors"
	"fmt"
)

var (
	NotConnected = errors.New("the service request can not be executed because the client is not yet connected")
)

// CommandRejectedError represents an IEC 104 command rejection from the slave station.
// This error carries structured ACK metadata for proper error classification and logging.
type CommandRejectedError struct {
	TypeID     int
	Coa        uint
	Ioa        uint
	Cot        string
	CotCause   int
	IsNegative bool
	Status     CommandAckStatus
}

func (e *CommandRejectedError) Error() string {
	return fmt.Sprintf("command rejected: cot=%s isNegative=%v typeId=%d coa=%d ioa=%d",
		e.Cot, e.IsNegative, e.TypeID, e.Coa, e.Ioa)
}

// Is implements errors.Is matching for CommandRejectedError type.
func (e *CommandRejectedError) Is(target error) bool {
	_, ok := target.(*CommandRejectedError)
	return ok
}
