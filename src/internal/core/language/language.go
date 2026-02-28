package language

import "time"

// Language represents a spoken/written language supported by the system.
type Language struct {
	ID         int       `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	NativeName string    `json:"native_name"`
	CreatedAt  time.Time `json:"-"`
}
