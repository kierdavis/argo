package rdflib

import (
	"strings"
)

type Namespace struct {
	base string
}

func CreateNamespace(base string) (ns *Namespace) {
	return &Namespace{base: base}
}

func (ns *Namespace) Base() (base string) {
	return ns.base
}

func (ns *Namespace) SetBase(base string) {
	ns.base = base
}

func (ns *Namespace) Get(name string) (term *Term) {
	return CreateResource(strings.Join([]string{ns.base, name}, ""))
}