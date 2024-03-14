package backend

import "time"

type Backend struct {
	Url         string
	IsDead      bool
	LastChecked time.Time
}
