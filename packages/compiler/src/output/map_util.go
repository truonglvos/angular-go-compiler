package output

// MapEntry represents an entry in a map literal
type MapEntry struct {
	Key    string
	Quoted bool
	Value  OutputExpression
}

// MapLiteral represents a map literal
type MapLiteral []MapEntry

// MapEntry creates a new MapEntry
func NewMapEntry(key string, value OutputExpression) MapEntry {
	return MapEntry{
		Key:    key,
		Quoted: false,
		Value:  value,
	}
}

// MapLiteralFromObject creates a map literal from an object
// This is a placeholder function that will be properly implemented when output_ast is available
func MapLiteralFromObject(obj map[string]OutputExpression, quoted bool) OutputExpression {
	// TODO: Implement when output_ast is available
	// return o.literalMap(
	//   Object.keys(obj).map((key) => ({
	//     key,
	//     quoted,
	//     value: obj[key],
	//   })),
	// )
	return nil
}
