package google

import (
	"fmt"
	"testing"
)

func TestGetName(t *testing.T) {
	t.Run("test1", func(t *testing.T) {
		got, err := GetName("jcracchiolo@tucowsinc.com")
		if err != nil {
			t.Errorf("GetName() error = %v", err)
			return
		}
		fmt.Println(got)
	})
}
