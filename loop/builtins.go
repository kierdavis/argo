package loop

import (
	"fmt"
)

type Builtin struct {
	ValArgs []string
	RefArgs []string
	Func    func([]interface{}, []Resource) (interface{}, error)
}

var builtins = map[string]Builtin{
	// loop:Add
	(LOOPBase + "Add"): {[]string{LOOPBase + "a", LOOPBase + "b"}, []string{}, func(args []interface{}, refs []Resource) (value interface{}, err error) {
		aRaw := args[0]
		bRaw := args[1]

		switch a := aRaw.(type) {
		case int64:
			switch b := bRaw.(type) {
			case int64:
				return a + b, nil
			case float64:
				return float64(a) + b, nil
			default:
				return nil, fmt.Errorf("loop:Add expects loop:b to be an integer or float")
			}

		case float64:
			switch b := bRaw.(type) {
			case int64:
				return a + float64(b), nil
			case float64:
				return a + b, nil
			default:
				return nil, fmt.Errorf("loop:Add expects loop:b to be an integer or float")
			}
		}

		return nil, fmt.Errorf("loop:Add expects loop:a to be an integer or float")
	}},
}
