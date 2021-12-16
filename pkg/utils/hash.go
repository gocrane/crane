package utils

import (
	"hash"
	"hash/adler32"

	"github.com/davecgh/go-spew/spew"
)

func GetHash(object interface{}) uint32 {
	var hashObj = adler32.New()
	deepHashObject(hashObj, object)
	return hashObj.Sum32()
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func deepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}
