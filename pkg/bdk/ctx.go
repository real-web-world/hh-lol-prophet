package bdk

import "context"

func IsCtxDone(c context.Context) bool {
	select {
	case <-c.Done():
		return true
	default:
		return false
	}
}
