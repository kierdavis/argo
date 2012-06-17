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
