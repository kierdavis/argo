package argo

import (
	"testing"
)


func TestResourceEqual(t *testing.T) {

	r1, r2 := NewResource("http://example.com/"), NewResource("http://example.com/")


	if !r1.Equal(r2) {
		t.Errorf("Expected %s but got %s", r1, r2)
	}

}
