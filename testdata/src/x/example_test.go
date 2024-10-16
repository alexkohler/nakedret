package x

import "testing"

func TestCheckNakedReturns(t *testing.T) {
	SomeTestHelperFunction()
}

func SomeTestHelperFunction() (res string) {
	res = "res"
	return
}
