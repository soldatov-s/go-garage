package log

// Field represents field information that should be passed to
// context.Logger.GetLogger function.
type Field struct {
	// Name is a field name.
	Name  string
	Value interface{}
}
