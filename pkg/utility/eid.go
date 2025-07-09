package utility

import (
	"github.com/google/uuid"
	"sync"
)

type ExecutionID = uuid.UUID

var (
	executionID     ExecutionID
	executionIDOnce sync.Once
	executionIDMu   sync.RWMutex
)

func GetExecutionID() ExecutionID {
	executionIDOnce.Do(func() {
		executionID = uuid.Must(uuid.NewV7())
	})

	executionIDMu.RLock()
	defer executionIDMu.RUnlock()
	return executionID
}

func ResetExecutionID() ExecutionID {
	executionIDMu.Lock()
	defer executionIDMu.Unlock()

	executionID = uuid.Must(uuid.NewV7())
	return executionID
}
