package helper

import (
	"path/filepath"
	"reflect"
)

// ObjName return object name for passing it to errors messages
func ObjName(iface interface{}) string {
	return filepath.Base(reflect.TypeOf(iface).PkgPath()) +
		"." + reflect.TypeOf(iface).Name()
}
