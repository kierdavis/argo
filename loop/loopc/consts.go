package main

import (
	"github.com/kierdavis/argo"
)

var LOOP = argo.NewNamespace("http://kierdavis.com/data/vocab/loop/")

var (
	XSDbase = ""

	XSDboolean            = XSDbase + "boolean"
	XSDbase64Binary       = XSDbase + "base64Binary"
	XSDhexBinary          = XSDbase + "hexBinary"
	XSDfloat              = XSDbase + "float"
	XSDdecimal            = XSDbase + "decimal"
	XSDinteger            = XSDbase + "integer"
	XSDnonPositiveInteger = XSDbase + "nonPositiveInteger"
	XSDnegativeInteger    = XSDbase + "negativeInteger"
	XSDlong               = XSDbase + "long"
	XSDint                = XSDbase + "int"
	XSDshort              = XSDbase + "short"
	XSDbyte               = XSDbase + "byte"
	XSDnonNegativeInteger = XSDbase + "nonNegativeInteger"
	XSDunsignedLong       = XSDbase + "unsignedLong"
	XSDunsignedInt        = XSDbase + "unsignedInt"
	XSDunsignedShort      = XSDbase + "unsignedShort"
	XSDunsignedByte       = XSDbase + "unsignedByte"
	XSDpositiveInteger    = XSDbase + "positiveInteger"
	XSDdouble             = XSDbase + "double"
	XSDanyURI             = XSDbase + "anyURI"
	XSDQName              = XSDbase + "QName"
)
