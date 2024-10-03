package backend

import (
	"sync"
)

type Backend struct {
	Url                string
	IsDead             bool
	Mutex              sync.Mutex
	ReviveAttempts     int
	IsMarkedForRemoval bool
}
