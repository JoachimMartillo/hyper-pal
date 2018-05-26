package system

import (
	"github.com/satori/go.uuid"
	"strconv"
	"time"
	"math/rand"
)

func NewV4String() string {
	uuid4, err := uuid.NewV4()
	if err != nil {
		return strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Int())
	}
	return uuid4.String()
}