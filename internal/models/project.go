package models

import "time"

type Project struct {
	ID        int
	Name      string
	Objective string
	CreatedAt time.Time
}
