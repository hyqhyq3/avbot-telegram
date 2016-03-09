package joke

import (
	"fmt"
	"testing"
)

func TestJoke(t *testing.T) {
	j := New()
	fmt.Println(j.getJoke())
}
