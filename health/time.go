package health

import "time"

var timestamp = func() time.Time {
	return time.Now().UTC()
}
