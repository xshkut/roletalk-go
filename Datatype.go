package roletalk

//Datatype represents type of data defined by roletalk communication protocol.
//Can be checked with corresponding constants.
//Implements Stringer (https://golang.org/pkg/fmt/#Stringer) interface
type Datatype byte

const (
	//DatatypeBinary represents []byte
	DatatypeBinary Datatype = 0

	//DatatypeNull represents nil
	DatatypeNull Datatype = 1

	//DatatypeBool represents bool
	DatatypeBool Datatype = 2

	//DatatypeString represents string
	DatatypeString Datatype = 3

	//DatatypeNumber represents float64
	DatatypeNumber Datatype = 4

	//DatatypeJSON represents []byte of JSON stringified object
	DatatypeJSON Datatype = 5
)

func (d Datatype) String() string {
	switch d {
	case 0:
		return "[]byte"
	case 1:
		return "nil"
	case 2:
		return "bool"
	case 3:
		return "string"
	case 4:
		return "float64"
	case 5:
		return "[]byte (raw JSON)"
	default:
		return "unknown"
	}
}
