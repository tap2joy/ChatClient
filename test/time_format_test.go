package test

import (
	"testing"
	"time"
)

func TestTimeFormat(t *testing.T) {
	println(time.Now().Format("2006-01-02 15:04:05"))

	now := time.Now().Unix()
	now += 3600
	println(time.Unix(now, 0).Format("2006-01-02 15:04:05"))
}
