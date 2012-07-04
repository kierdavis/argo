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

package argo

import (
	"testing"
)

func TestTripleEqual(t *testing.T) {
	s1, s2 := NewResource("http://example.com/s"), NewResource("http://example.com/s")
	p1, p2 := NewResource("http://example.com/p"), NewResource("http://example.com/p")
	o1, o2 := NewResource("http://example.com/o"), NewResource("http://example.com/o")

	t1, t2 := NewTriple(s1, p1, o1), NewTriple(s2, p2, o2)

	if !t1.Equal(t2) {
		t.Errorf("Expected %s but got %s", t1, t2)
	}
}
