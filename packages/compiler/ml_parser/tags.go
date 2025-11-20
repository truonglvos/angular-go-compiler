package ml_parser

import (
	"strings"
)

// TagContentType represents the content type of a tag
type TagContentType int

const (
	TagContentTypeRAW_TEXT TagContentType = iota
	TagContentTypeESCAPABLE_RAW_TEXT
	TagContentTypePARSABLE_DATA
)

// TagDefinition defines the behavior of an HTML tag
type TagDefinition interface {
	ClosedByParent() bool
	ImplicitNamespacePrefix() *string
	IsVoid() bool
	IgnoreFirstLf() bool
	CanSelfClose() bool
	PreventNamespaceInheritance() bool
	IsClosedByChild(name string) bool
	GetContentType(prefix *string) TagContentType
}

// SplitNsName splits a namespace:name string into namespace and name
func SplitNsName(elementName string, fatal bool) (string, string) {
	if len(elementName) == 0 || elementName[0] != ':' {
		return "", elementName
	}

	colonIndex := strings.Index(elementName[1:], ":")
	if colonIndex == -1 {
		if fatal {
			panic("Unsupported format \"" + elementName + "\" expecting \":namespace:name\"")
		}
		return "", elementName
	}

	colonIndex++ // Adjust for the slice
	namespace := elementName[1:colonIndex]
	name := elementName[colonIndex+1:]
	return namespace, name
}

// IsNgContainer checks if a tag name is ng-container
func IsNgContainer(tagName string) bool {
	_, name := SplitNsName(tagName, false)
	return name == "ng-container"
}

// IsNgContent checks if a tag name is ng-content
func IsNgContent(tagName string) bool {
	_, name := SplitNsName(tagName, false)
	return name == "ng-content"
}

// IsNgTemplate checks if a tag name is ng-template
func IsNgTemplate(tagName string) bool {
	_, name := SplitNsName(tagName, false)
	return name == "ng-template"
}

// GetNsPrefix gets the namespace prefix from a full name
func GetNsPrefix(fullName *string) *string {
	if fullName == nil {
		return nil
	}
	prefix, _ := SplitNsName(*fullName, false)
	if prefix == "" {
		return nil
	}
	return &prefix
}

// MergeNsAndName merges namespace prefix and local name
func MergeNsAndName(prefix, localName string) string {
	if prefix != "" {
		return ":" + prefix + ":" + localName
	}
	return localName
}
