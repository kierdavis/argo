package argo

import (
	"fmt"
)

type Triple struct {
	Subject   Term
	Predicate Term
	Object    Term
	Context   Term
}

func NewTriple(subject Term, predicate Term, object Term) (triple *Triple) {
	return &Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Context:   nil,
	}
}

func NewQuad(subject Term, predicate Term, object Term, context Term) (triple *Triple) {
	return &Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Context:   context,
	}
}

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

func (triple Triple) Equal(other *Triple) bool {
	return triple.Subject.Equal(other.Subject) &&
		triple.Predicate.Equal(other.Predicate) &&
		triple.Object.Equal(other.Object) &&
		((triple.Context == nil && other.Context == nil) || (triple.Context != nil && other.Context != nil && triple.Context.Equal(other.Context)))

}
