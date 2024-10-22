package util

import (
	"fmt"
)

func HandleRecoverError(r interface{}) error {
	switch x := r.(type) {
	case string:
		return fmt.Errorf("%s", x)
	case error:
		return x
	default:
		return fmt.Errorf("unknown error:%v", x)
	}
}
