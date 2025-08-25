package common

import "time"

var Loc, _ = time.LoadLocation("Asia/Jakarta")

func Now() time.Time {
	return time.Now().In(Loc)
}
