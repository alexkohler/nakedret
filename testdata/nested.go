package x

func Okay() (err error) {
	defer func() {
		// This is okay because it belongs to the function literal
		return
	}()
	return err
}

func Bad() (err error) {
	defer func() {
		// This is okay because it belongs to the function literal
		return
	}()
	return
}

func BadNested() {
	f := func() (i int) {
		return
	}
	return
}

func MoreBad() {
	var _ = func() (err error) {
		return
	}

	func() (err error) {
		return
	}()

	defer func() (err error) {
		return
	}()

	go func() (err error) {
		return
	}()
}
