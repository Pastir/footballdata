package utility

import (
	"math/rand"
	"time"
)

//Rand - Возвращает рандомное число
func Rand(i int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(i)
}
