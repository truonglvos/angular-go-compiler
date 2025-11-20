package ml_parser

// XmlTagDefinition implements TagDefinition for XML tags
type XmlTagDefinition struct {
	closedByParent              bool
	implicitNamespacePrefix     *string
	isVoid                      bool
	ignoreFirstLf               bool
	canSelfClose                bool
	preventNamespaceInheritance bool
}

// NewXmlTagDefinition creates a new XmlTagDefinition
func NewXmlTagDefinition() *XmlTagDefinition {
	return &XmlTagDefinition{
		closedByParent:              false,
		implicitNamespacePrefix:     nil,
		isVoid:                      false,
		ignoreFirstLf:               false,
		canSelfClose:                true,
		preventNamespaceInheritance: false,
	}
}

// ClosedByParent returns whether this tag is closed by parent
func (x *XmlTagDefinition) ClosedByParent() bool {
	return x.closedByParent
}

// ImplicitNamespacePrefix returns the implicit namespace prefix
func (x *XmlTagDefinition) ImplicitNamespacePrefix() *string {
	return x.implicitNamespacePrefix
}

// IsVoid returns whether this tag is void
func (x *XmlTagDefinition) IsVoid() bool {
	return x.isVoid
}

// IgnoreFirstLf returns whether to ignore first line feed
func (x *XmlTagDefinition) IgnoreFirstLf() bool {
	return x.ignoreFirstLf
}

// CanSelfClose returns whether this tag can self-close
func (x *XmlTagDefinition) CanSelfClose() bool {
	return x.canSelfClose
}

// PreventNamespaceInheritance returns whether to prevent namespace inheritance
func (x *XmlTagDefinition) PreventNamespaceInheritance() bool {
	return x.preventNamespaceInheritance
}

// IsClosedByChild returns whether this tag is closed by a child
func (x *XmlTagDefinition) IsClosedByChild(name string) bool {
	return false
}

// GetContentType returns the content type for this tag
func (x *XmlTagDefinition) GetContentType(prefix *string) TagContentType {
	return TagContentTypePARSABLE_DATA
}

var xmlTagDefinition = NewXmlTagDefinition()

// GetXmlTagDefinition returns the XML tag definition for a tag name
func GetXmlTagDefinition(tagName string) TagDefinition {
	return xmlTagDefinition
}
