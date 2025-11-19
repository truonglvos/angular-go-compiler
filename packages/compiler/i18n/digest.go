package i18n

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
)

// Digest returns the message id or computes it using the XLIFF1 digest
func Digest(message *Message) string {
	if message.ID != "" {
		return message.ID
	}
	return ComputeDigest(message)
}

// ComputeDigest computes the message id using the XLIFF1 digest
func ComputeDigest(message *Message) string {
	serialized := SerializeNodes(message.Nodes)
	content := strings.Join(serialized, "") + "[" + message.Meaning + "]"
	return SHA1(content)
}

// DecimalDigest returns the message id or computes it using the XLIFF2/XMB/$localize digest
func DecimalDigest(message *Message) string {
	if message.ID != "" {
		return message.ID
	}
	return ComputeDecimalDigest(message)
}

// ComputeDecimalDigest computes the message id using the XLIFF2/XMB/$localize digest
func ComputeDecimalDigest(message *Message) string {
	visitor := &SerializerIgnoreIcuExpVisitor{}
	parts := make([]string, len(message.Nodes))
	for i, node := range message.Nodes {
		result := node.Visit(visitor, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	return ComputeMsgID(strings.Join(parts, ""), message.Meaning)
}

// SerializerVisitor serializes the i18n ast to something xml-like in order to generate an UID
type SerializerVisitor struct{}

// VisitText serializes a Text node
func (v *SerializerVisitor) VisitText(text *Text, context interface{}) interface{} {
	return text.Value
}

// VisitContainer serializes a Container node
func (v *SerializerVisitor) VisitContainer(container *Container, context interface{}) interface{} {
	parts := make([]string, len(container.Children))
	for i, child := range container.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// VisitIcu serializes an Icu node
func (v *SerializerVisitor) VisitIcu(icu *Icu, context interface{}) interface{} {
	strCases := make([]string, 0, len(icu.Cases))
	for k, node := range icu.Cases {
		result := node.Visit(v, nil)
		if str, ok := result.(string); ok {
			strCases = append(strCases, k+" {"+str+"}")
		}
	}
	return "{" + icu.Expression + ", " + icu.Type + ", " + strings.Join(strCases, ", ") + "}"
}

// VisitTagPlaceholder serializes a TagPlaceholder node
func (v *SerializerVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context interface{}) interface{} {
	if ph.IsVoid {
		return `<ph tag name="` + ph.StartName + `"/>`
	}
	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	return `<ph tag name="` + ph.StartName + `">` + strings.Join(parts, ", ") + `</ph name="` + ph.CloseName + `">`
}

// VisitPlaceholder serializes a Placeholder node
func (v *SerializerVisitor) VisitPlaceholder(ph *Placeholder, context interface{}) interface{} {
	if ph.Value != "" {
		return `<ph name="` + ph.Name + `">` + ph.Value + `</ph>`
	}
	return `<ph name="` + ph.Name + `"/>`
}

// VisitIcuPlaceholder serializes an IcuPlaceholder node
func (v *SerializerVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context interface{}) interface{} {
	result := ph.Value.Visit(v, nil)
	if str, ok := result.(string); ok {
		return `<ph icu name="` + ph.Name + `">` + str + `</ph>`
	}
	return `<ph icu name="` + ph.Name + `"></ph>`
}

// VisitBlockPlaceholder serializes a BlockPlaceholder node
func (v *SerializerVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context interface{}) interface{} {
	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	return `<ph block name="` + ph.StartName + `">` + strings.Join(parts, ", ") + `</ph name="` + ph.CloseName + `">`
}

var serializerVisitor = &SerializerVisitor{}

// SerializeNodes serializes nodes to strings
func SerializeNodes(nodes []Node) []string {
	result := make([]string, len(nodes))
	for i, node := range nodes {
		visitResult := node.Visit(serializerVisitor, nil)
		if str, ok := visitResult.(string); ok {
			result[i] = str
		} else {
			result[i] = ""
		}
	}
	return result
}

// SerializerIgnoreIcuExpVisitor serializes the i18n ast but ignores ICU expressions
type SerializerIgnoreIcuExpVisitor struct {
	*SerializerVisitor
}

// VisitIcu serializes an Icu node without the expression
func (v *SerializerIgnoreIcuExpVisitor) VisitIcu(icu *Icu, context interface{}) interface{} {
	strCases := make([]string, 0, len(icu.Cases))
	for k, node := range icu.Cases {
		result := node.Visit(v, nil)
		if str, ok := result.(string); ok {
			strCases = append(strCases, k+" {"+str+"}")
		}
	}
	// Do not take the expression into account
	return "{" + icu.Type + ", " + strings.Join(strCases, ", ") + "}"
}

// SHA1 computes the SHA1 of the given string
// WARNING: this function has not been designed not tested with security in mind.
//          DO NOT USE IT IN A SECURITY SENSITIVE CONTEXT.
func SHA1(str string) string {
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// Fingerprint computes the fingerprint of the given string
// The output is 64 bit number encoded as a decimal string
// based on:
// https://github.com/google/closure-compiler/blob/master/src/com/google/javascript/jscomp/GoogleJsMessageIdGenerator.java
func Fingerprint(str string) uint64 {
	utf8 := []byte(str)
	length := len(utf8)
	
	hi := hash32(utf8, length, 0)
	lo := hash32(utf8, length, 102072)
	
	if hi == 0 && (lo == 0 || lo == 1) {
		hi = hi ^ 0x130f9bef
		lo = lo ^ uint32(0x6b5f56d8)
	}
	
	return (uint64(hi) << 32) | uint64(lo)
}

// ComputeMsgID computes the message ID
func ComputeMsgID(msg string, meaning string) string {
	msgFingerprint := Fingerprint(msg)
	
	if meaning != "" {
		// Rotate the 64-bit message fingerprint one bit to the left and then add the meaning fingerprint
		msgFingerprint = ((msgFingerprint << 1) | ((msgFingerprint >> 63) & 1))
		msgFingerprint += Fingerprint(meaning)
	}
	
	// Return as 63-bit number (to avoid negative numbers in JavaScript)
	return fmt.Sprintf("%d", msgFingerprint&0x7fffffffffffffff)
}

// hash32 computes a 32-bit hash
func hash32(data []byte, length int, c int) uint32 {
	var a, b uint32 = 0x9e3779b9, 0x9e3779b9
	var c32 uint32 = uint32(c)
	index := 0
	
	end := length - 12
	for index <= end {
		a += getUint32LE(data, index)
		b += getUint32LE(data, index+4)
		c32 += getUint32LE(data, index+8)
		res := mix(a, b, c32)
		a, b, c32 = res[0], res[1], res[2]
		index += 12
	}
	
	remainder := length - index
	
	// the first byte of c is reserved for the length
	c32 += uint32(length)
	
	if remainder >= 4 {
		a += getUint32LE(data, index)
		index += 4
		
		if remainder >= 8 {
			b += getUint32LE(data, index)
			index += 4
			
			// Partial 32-bit word for c
			if remainder >= 9 {
				c32 += uint32(data[index]) << 8
				index++
			}
			if remainder >= 10 {
				c32 += uint32(data[index]) << 16
				index++
			}
			if remainder == 11 {
				c32 += uint32(data[index]) << 24
				index++
			}
		} else {
			// Partial 32-bit word for b
			if remainder >= 5 {
				b += uint32(data[index])
				index++
			}
			if remainder >= 6 {
				b += uint32(data[index]) << 8
				index++
			}
			if remainder == 7 {
				b += uint32(data[index]) << 16
				index++
			}
		}
	} else {
		// Partial 32-bit word for a
		if remainder >= 1 {
			a += uint32(data[index])
			index++
		}
		if remainder >= 2 {
			a += uint32(data[index]) << 8
			index++
		}
		if remainder == 3 {
			a += uint32(data[index]) << 16
			index++
		}
	}
	
	return mix(a, b, c32)[2]
}

// mix mixes three 32-bit values
func mix(a, b, c uint32) [3]uint32 {
	a -= b
	a -= c
	a ^= c >> 13
	b -= c
	b -= a
	b ^= a << 8
	c -= a
	c -= b
	c ^= b >> 13
	a -= b
	a -= c
	a ^= c >> 12
	b -= c
	b -= a
	b ^= a << 16
	c -= a
	c -= b
	c ^= b >> 5
	a -= b
	a -= c
	a ^= c >> 3
	b -= c
	b -= a
	b ^= a << 10
	c -= a
	c -= b
	c ^= b >> 15
	return [3]uint32{a, b, c}
}

// getUint32LE gets a uint32 in little-endian format
func getUint32LE(data []byte, index int) uint32 {
	if index+4 > len(data) {
		return 0
	}
	return binary.LittleEndian.Uint32(data[index:])
}

