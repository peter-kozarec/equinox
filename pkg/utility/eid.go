package utility

import "github.com/google/uuid"

type ExecutionID = uuid.UUID

var (
	eid = uuid.New()
)

func GetExecutionID() ExecutionID {
	return eid
}
