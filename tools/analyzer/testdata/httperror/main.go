package main

import (
	"errors"
	"net/http"
)

func testIfElseOk1() {
	if true {
		http.Error(nil, "string", http.StatusIMUsed)
	} else {
		http.Error(nil, "string", http.StatusAccepted)
	}
}

func testIfElseOk2() {
	if true {
		x := 0
		_ = x
	} else {
		http.Error(nil, "string", http.StatusBadRequest)
	}
}

func testIfElseOk3() {
	if true {
		http.Error(nil, "string", http.StatusBadRequest)
	} else {
		x := 0
		_ = x
	}
}

func testIfElseBad1() {
	x := 0
	if true { // want `if statement not followed by return`
		http.Error(nil, "string", http.StatusIMUsed)
	} else {
		http.Error(nil, "string", http.StatusAccepted)
	}
	x = 1
	_ = x
}

func testIfElseBad2() {
	x := 0
	if true {
		x = 1
	} else {
		http.Error(nil, "string", http.StatusAccepted) // want `http.Error not followed by return`
	}
	x = 2
	_ = x
}

func testIfElseBad3() {
	x := 0
	if true {
		http.Error(nil, "string", http.StatusAccepted) // want `http.Error not followed by return`
	} else {
		x = 1
	}
	x = 2
	_ = x
}

func testIfElseIfElseOk1() {
	if true {
		http.Error(nil, "string1", http.StatusBadRequest)
	} else if false {
		http.Error(nil, "string2", http.StatusBadRequest)
	} else {
		http.Error(nil, "string3", http.StatusBadRequest)
	}
}

func testIfElseIfElseOk2() {
	x := 0
	if true {
		x = 1
	} else if false {
		x = 2
	} else {
		_ = x
		http.Error(nil, "string", http.StatusBadRequest)
	}
}

func testIfElseIfElseOk3() {
	x := 0
	if true {
		x = 1
	} else if false {
		http.Error(nil, "string", http.StatusBadRequest)
	} else {
		x = 2
		_ = x
	}
}

func testIfElseIfElseBad1() {
	x := 0
	if true { // want `if statement not followed by return`
		http.Error(nil, "string1", http.StatusBadRequest)
	} else if false {
		http.Error(nil, "string2", http.StatusBadRequest)
	} else {
		http.Error(nil, "string3", http.StatusBadRequest)
	}
	x = 1
	_ = x
}

func testIfElseIfElseBad2() {
	x := 0
	if true {
		http.Error(nil, "string", http.StatusBadRequest) // want `http.Error not followed by return`
	} else if false {
		x = 1
	} else {
		x = 2
	}
	x = 3
	_ = x
}

func testIfElseIfElseBad3() {
	x := 0
	if true {
		x = 1
	} else if false {
		http.Error(nil, "string", http.StatusBadRequest) // want `http.Error not followed by return`
	} else {
		x = 2
	}
	x = 3
	_ = x
}

func testIfElseIfElseBad4() {
	x := 0
	if true {
		x = 1
	} else if false {
		x = 2
	} else {
		http.Error(nil, "string", http.StatusBadRequest) // want `http.Error not followed by return`
	}
	x = 3
	_ = x
}

func testNestedIfElseOk() {
	x := 0
	if true {
		if true {
			http.Error(nil, "string", http.StatusBadRequest)
		} else {
			x = 1
		}
	} else {
		if true {
			x = 2
		} else {
			_ = x
			http.Error(nil, "string", http.StatusBadRequest)
		}
	}
}

func testNestedIfElseBad() {
	x := 0
	if true { // want `if statement not followed by return`
		if true {
			http.Error(nil, "string", http.StatusBadRequest)
		} else {
			x = 1
		}
	} else {
		if true {
			x = 2
		} else {
			http.Error(nil, "string", http.StatusBadRequest)
		}
	}
	x = 3
	_ = x
}

func testChainedIfElseOk() {
	x := 0
	if x < -10 {
		x = 1
	} else if x < -5 {
		x = 2
	} else if x < 0 {
		x = 3
	} else {
		_ = x
		http.Error(nil, "string", http.StatusBadRequest)
	}
}

func testChainedIfElseBad() {
	x := 0
	if x < -10 {
		x = 1
	} else if x < -5 {
		x = 2
	} else if x < 0 {
		x = 3
	} else {
		http.Error(nil, "string", http.StatusBadRequest) // want `http.Error not followed by return`
	}
	x = 4
	_ = x
}

func testFuncLiteral() {
	x := 0
	_ = func() {
		http.Error(nil, "foo", http.StatusAccepted)
	}

	_ = func() {
		http.Error(nil, "bar", http.StatusBadRequest) // want `http.Error not followed by return`
		x = 2
	}
	_ = x
}

func testSwitch() {
	x := 1
	switch x {
	case 1:
		x = 2
	case 2:
		x = 3
	}
	_ = x
	switch x { // want `switch statement not followed by return`
	case 1:
		http.Error(nil, "foo", http.StatusAccepted)
	case 2:
		http.Error(nil, "bar", http.StatusAccepted)
	}
	_ = x
	switch x {
	case 1:
		http.Error(nil, "foo", http.StatusAccepted)
	case 4:
		http.Error(nil, "bar", http.StatusAccepted) // want `http.Error not followed by return`
		x = 1
	}
}

func foo() bool {
	return true
}

func bar() bool {
	return false
}

func testSwitch2() {
	x := 1
	if err := errors.New("Test"); err != nil {
		switch { // want `switch statement not followed by return`
		case foo():
			http.Error(nil, "foo", http.StatusInternalServerError)
		case bar():
			http.Error(nil, "bar", http.StatusNoContent)
		default:
			http.Error(nil, "baz", http.StatusOK)
		}
	}
	_ = x
}

// TODO: rationalize the error reporting (sometimes on http.Error, sometimes after)
func THE() {
	http.Error(nil, "string", http.StatusInternalServerError) // want `http.Error not followed by return`
	x := 0

	if true {
		http.Error(nil, "string", http.StatusBadGateway) // want `http.Error not followed by return`
		x = 2
	}

	if true {
		http.Error(nil, "string", http.StatusBadGateway) // want `http.Error not followed by return`
	}
	x = 3

	// More advanced tests below.
	testIfElseOk1()
	testIfElseOk2()
	testIfElseOk3()
	testIfElseBad1()
	testIfElseBad2()
	testIfElseBad3()
	testIfElseIfElseOk1()
	testIfElseIfElseOk2()
	testIfElseIfElseOk3()
	testIfElseIfElseBad1()
	testIfElseIfElseBad2()
	testIfElseIfElseBad3()
	testIfElseIfElseBad4()
	testNestedIfElseOk()
	testNestedIfElseBad()
	testChainedIfElseOk()
	testChainedIfElseBad()
	testFuncLiteral()
	testSwitch()
	testSwitch2()

	_ = x
}

func THE2() {
	x := 0

	if true { // want `if statement not followed by return`
		http.Error(nil, "string", http.StatusIMUsed)
	} else {
		http.Error(nil, "string", http.StatusAccepted)
	}
	x = 1

	if true {
		http.Error(nil, "string", http.StatusBadGateway) // want `http.Error not followed by return`
		x = 2
	}

	if true {
		http.Error(nil, "string", http.StatusBadGateway) // want `http.Error not followed by return`
	}
	x = 3

	if true {
		http.Error(nil, "string", http.StatusAlreadyReported) // want `http.Error not followed by return`
	} else if false {
		x = 4
	} else {
		x = 5
	}

	if true {
		x = 6
	} else if false {
		http.Error(nil, "string", http.StatusCreated) // want `http.Error not followed by return`
		x = 7
	} else {
		x = 8
	}

	func() {
		if true {
			http.Error(nil, "string", http.StatusIMUsed)
		} else if false {
			http.Error(nil, "string", http.StatusOK)
		} else {
			http.Error(nil, "string", http.StatusForbidden)
		}
	}()

	if true {
		x = 9
	} else if false {
		x = 10
	} else {
		http.Error(nil, "string", http.StatusOK) // want `http.Error not followed by return`
	}

	_ = x
}
