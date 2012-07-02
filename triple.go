package argo

import (
	"fmt"
)

// A Triple contains a subject, a predicate and an object term. It also contains a context term, but
// support for contexts is not fully implemented yet.
type Triple struct {
	Subject   Term
	Predicate Term
	Object    Term
	Context   Term
}

// Function NewTriple returns a new triple with the given subject, predicate and object.
func NewTriple(subject Term, predicate Term, object Term) (triple *Triple) {
	return &Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Context:   nil,
	}
}

// Function NewQuad returns a new triple with the given subject, predicate, object and context.
func NewQuad(subject Term, predicate Term, object Term, context Term) (triple *Triple) {
	return &Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Context:   context,
	}
}

// Method String returns the NTriples representation of this triple.
func (triple Triple) String() (str string) {
	subj_str := "nil"
	if triple.Subject != nil {
		subj_str = triple.Subject.String()
	}

	pred_str := "nil"
	if triple.Predicate != nil {
		pred_str = triple.Predicate.String()
	}

	obj_str := "nil"
	if triple.Object != nil {
		obj_str = triple.Object.String()
	}

	if triple.Context == nil {
		return fmt.Sprintf("%s %s %s .", subj_str, pred_str, obj_str)
	}

	ctx_str := triple.Context.String()
	return fmt.Sprintf("%s %s %s %s .", subj_str, pred_str, obj_str, ctx_str)
}

// Method Equal returns this triple is equivalent to the argument.
func (triple Triple) Equal(other *Triple) bool {
	return triple.Subject.Equal(other.Subject) &&
		triple.Predicate.Equal(other.Predicate) &&
		triple.Object.Equal(other.Object) &&
		((triple.Context == nil && other.Context == nil) || (triple.Context != nil && other.Context != nil && triple.Context.Equal(other.Context)))

}
