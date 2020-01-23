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
