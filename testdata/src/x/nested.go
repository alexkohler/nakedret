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
	return // want "naked return in func `Bad` with 6 lines of code"
}

func BadNested() {
	_ = func() (i int) {
		return // want "naked return in func `BadNested.<func..:20>` with 2 lines of code"
	}
	return
}

func MoreBad() {
	var _ = func() (err error) {
		return // want "naked return in func `MoreBad.<func..:27>` with 2 lines of code"
	}

	func() (err error) {
		return // want "naked return in func `MoreBad.<func..:31>` with 2 lines of code"
	}()

	defer func() (err error) {
		return // want "naked return in func `MoreBad.<func..:35>` with 2 lines of code"
	}()

	go func() (err error) {
		return // want "naked return in func `MoreBad.<func..:39>` with 2 lines of code"
	}()
}

func LiteralFuncCallReturn() int {
	// function literal nested within a return statement
	return func() (x int) {
		return // want "naked return in func `LiteralFuncCallReturn.<func..:46>` with 2 lines of code"
	}()
}

func LiteralFuncCallReturn2() int {
	// function literal nested within a return statement
	return func() (x int) {
		return func() (x int) {
			return // want "naked return in func `LiteralFuncCallReturn2.<func..:53>.<func..:54>` with 2 lines of code"
		}()
	}()
}

func ManyReturns() (x, y, z int, w int, s string, err error) {
	switch {
	case true:
		return // want "naked return in func `ManyReturns` with 8 lines of code"
	case false:
		return // want "naked return in func `ManyReturns` with 8 lines of code"
	}
	return // want "naked return in func `ManyReturns` with 8 lines of code"
}

func DeeplyNested(b int) (f func() (x, y int)) {
	f = func() (x, y int) {
		defer func() {
			x, y = func() (x, y int) {
				switch {
				case true:
					x = func() (a int) {
						a = b
						return // want "naked return in func `DeeplyNested.<func..:71>.<func..:72>.<func..:73>.<func..:76>` with 3 lines of code"
					}()
					if x > y {
						return // want "naked return in func `DeeplyNested.<func..:71>.<func..:72>.<func..:73>` with 12 lines of code"
					}
				}
				return // want "naked return in func `DeeplyNested.<func..:71>.<func..:72>.<func..:73>` with 12 lines of code"
			}()
		}()
		return // want "naked return in func `DeeplyNested.<func..:71>` with 17 lines of code"
	}
	return // want "naked return in func `DeeplyNested` with 20 lines of code"
}

var ToplevelFuncLit = func(x int) (err error) {
	if x > 0 {
		return func() (err error) {
			return // want "naked return in func `<func..:92>.<func..:94>` with 2 lines of code"
		}()
	}
	return // want "naked return in func `<func..:92>` with 7 lines of code"
}

func SingleLine() (err error) { return } // want "naked return in func `SingleLine` with 1 lines of code"

var SingleLit = func() (err error) { return } // want "naked return in func `<func..:103>` with 1 lines of code"

func SingleLineNested() (err error) {
	return func() (err error) { return }() // want "naked return in func `SingleLineNested.<func..:106>` with 1 lines of code"
}
