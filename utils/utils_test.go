package utils

import (
	"fmt"
	"testing"
)

func TestSession(t *testing.T) {
	fmt.Println(Sha256("123456", "123456"))
}
