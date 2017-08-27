package avbot

import "time"

func GetNow() int {
	return int(time.Now().Unix())
}
