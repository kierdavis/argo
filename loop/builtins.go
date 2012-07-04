/*
	Copyright (c) 2012 Kier Davis

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
	associated documentation files (the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial
	portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
	NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
	NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
	OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

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
