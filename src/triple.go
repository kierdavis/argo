package rdflib

import (
	"fmt"
)

type Triple struct {
	Subject *Term
	Predicate *Term
	Object *Term
	Context *Term
}

func CreateTriple(subject *Term, predicate *Term, object *Term) (triple *Triple) {
	return &Triple{
		Subject: subject,
		Predicate: predicate,
		Object: object,
		Context: nil,
	}
}

func CreateQuad(subject *Term, predicate *Term, object *Term, context *Term) (triple *Triple) {
	return &Triple{
		Subject: subject,
		Predicate: predicate,
		Object: object,
		Context: context,
	}
}

func (triple *Triple) String() (str string) {
	subj_str := "nil"
	if triple.Subject != nil {subj_str = triple.Subject.String()}

	pred_str := "nil"
	if triple.Predicate != nil {pred_str = triple.Predicate.String()}

	obj_str := "nil"
	if triple.Object != nil {obj_str = triple.Object.String()}

	if triple.Context == nil {
		return fmt.Sprintf("%s %s %s .", subj_str, pred_str, obj_str)
	
	} else {
		ctx_str := triple.Context.String()
		return fmt.Sprintf("%s %s %s %s .", subj_str, pred_str, obj_str, ctx_str)
	}
}

func termsEqual(a *Term, b *Term) (isEqual bool) {
	if a != nil {
		return a.EqualTo(b)
	} else if b != nil {
		return b.EqualTo(a)
	} else {
		return true // They're both nil
	}
}

func (triple *Triple) EqualTo(other *Triple) (isEqual bool) {
	return termsEqual(triple.Subject, other.Subject) && termsEqual(triple.Predicate, other.Predicate) && termsEqual(triple.Object, other.Object) && termsEqual(triple.Context, other.Context)
}