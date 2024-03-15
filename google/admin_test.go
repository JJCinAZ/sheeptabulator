package google

import (
	"fmt"
	"os"
	"testing"
)

func TestGetName(t *testing.T) {
	t.Run("test1", func(t *testing.T) {
		got, err := GetName(os.Getenv("EMAIL"))
		if err != nil {
			t.Errorf("GetName() error = %v", err)
			return
		}
		fmt.Println(got)
	})
}
