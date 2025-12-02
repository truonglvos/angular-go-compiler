package ml_parser

import "ngc-go/packages/compiler/src/util"

// TokenType represents the type of a token
type TokenType int

const (
	TokenTypeTAG_OPEN_START TokenType = iota
	TokenTypeTAG_OPEN_END
	TokenTypeTAG_OPEN_END_VOID
	TokenTypeTAG_CLOSE
	TokenTypeINCOMPLETE_TAG_OPEN
	TokenTypeTEXT
	TokenTypeESCAPABLE_RAW_TEXT
	TokenTypeRAW_TEXT
	TokenTypeINTERPOLATION
	TokenTypeENCODED_ENTITY
	TokenTypeCOMMENT_START
	TokenTypeCOMMENT_END
	TokenTypeCDATA_START
	TokenTypeCDATA_END
	TokenTypeATTR_NAME
	TokenTypeATTR_QUOTE
	TokenTypeATTR_VALUE
	TokenTypeATTR_VALUE_TEXT
	TokenTypeATTR_VALUE_INTERPOLATION
	TokenTypeDOC_TYPE
	TokenTypeEXPANSION_FORM_START
	TokenTypeEXPANSION_CASE_VALUE
	TokenTypeEXPANSION_CASE_EXP_START
	TokenTypeEXPANSION_CASE_EXP_END
	TokenTypeEXPANSION_FORM_END
	TokenTypeBLOCK_OPEN_START
	TokenTypeBLOCK_OPEN_END
	TokenTypeBLOCK_CLOSE
	TokenTypeBLOCK_PARAMETER
	TokenTypeINCOMPLETE_BLOCK_OPEN
	TokenTypeLET_START
	TokenTypeLET_VALUE
	TokenTypeLET_END
	TokenTypeINCOMPLETE_LET
	TokenTypeCOMPONENT_OPEN_START
	TokenTypeCOMPONENT_OPEN_END
	TokenTypeCOMPONENT_OPEN_END_VOID
	TokenTypeCOMPONENT_CLOSE
	TokenTypeINCOMPLETE_COMPONENT_OPEN
	TokenTypeDIRECTIVE_NAME
	TokenTypeDIRECTIVE_OPEN
	TokenTypeDIRECTIVE_CLOSE
	TokenTypeEOF
)

// Token represents a token in the HTML/XML source
type Token interface {
	Type() TokenType
	Parts() []string
	SourceSpan() *util.ParseSourceSpan
}

// TokenBase is the base implementation of Token
type TokenBase struct {
	tokenType  TokenType
	parts      []string
	sourceSpan *util.ParseSourceSpan
}

// Type returns the token type
func (t *TokenBase) Type() TokenType {
	return t.tokenType
}

// Parts returns the token parts
func (t *TokenBase) Parts() []string {
	return t.parts
}

// SourceSpan returns the source span
func (t *TokenBase) SourceSpan() *util.ParseSourceSpan {
	return t.sourceSpan
}

// NewTokenBase creates a new TokenBase
func NewTokenBase(tokenType TokenType, parts []string, sourceSpan *util.ParseSourceSpan) *TokenBase {
	return &TokenBase{
		tokenType:  tokenType,
		parts:      parts,
		sourceSpan: sourceSpan,
	}
}

// TagOpenStartToken represents a tag open start token
type TagOpenStartToken struct {
	*TokenBase
}

// NewTagOpenStartToken creates a new TagOpenStartToken
func NewTagOpenStartToken(prefix, name string, sourceSpan *util.ParseSourceSpan) *TagOpenStartToken {
	return &TagOpenStartToken{
		TokenBase: NewTokenBase(TokenTypeTAG_OPEN_START, []string{prefix, name}, sourceSpan),
	}
}

// TagOpenEndToken represents a tag open end token
type TagOpenEndToken struct {
	*TokenBase
}

// NewTagOpenEndToken creates a new TagOpenEndToken
func NewTagOpenEndToken(sourceSpan *util.ParseSourceSpan) *TagOpenEndToken {
	return &TagOpenEndToken{
		TokenBase: NewTokenBase(TokenTypeTAG_OPEN_END, []string{}, sourceSpan),
	}
}

// TagOpenEndVoidToken represents a tag open end void token
type TagOpenEndVoidToken struct {
	*TokenBase
}

// NewTagOpenEndVoidToken creates a new TagOpenEndVoidToken
func NewTagOpenEndVoidToken(sourceSpan *util.ParseSourceSpan) *TagOpenEndVoidToken {
	return &TagOpenEndVoidToken{
		TokenBase: NewTokenBase(TokenTypeTAG_OPEN_END_VOID, []string{}, sourceSpan),
	}
}

// TagCloseToken represents a tag close token
type TagCloseToken struct {
	*TokenBase
}

// NewTagCloseToken creates a new TagCloseToken
func NewTagCloseToken(prefix, name string, sourceSpan *util.ParseSourceSpan) *TagCloseToken {
	return &TagCloseToken{
		TokenBase: NewTokenBase(TokenTypeTAG_CLOSE, []string{prefix, name}, sourceSpan),
	}
}

// TextToken represents a text token
type TextToken struct {
	*TokenBase
}

// NewTextToken creates a new TextToken
func NewTextToken(text string, tokenType TokenType, sourceSpan *util.ParseSourceSpan) *TextToken {
	return &TextToken{
		TokenBase: NewTokenBase(tokenType, []string{text}, sourceSpan),
	}
}

// InterpolationToken represents an interpolation token
type InterpolationToken struct {
	*TokenBase
}

// NewInterpolationToken creates a new InterpolationToken
func NewInterpolationToken(startMarker, expression string, endMarker *string, sourceSpan *util.ParseSourceSpan) *InterpolationToken {
	parts := []string{startMarker, expression}
	if endMarker != nil {
		parts = append(parts, *endMarker)
	}
	return &InterpolationToken{
		TokenBase: NewTokenBase(TokenTypeINTERPOLATION, parts, sourceSpan),
	}
}

// EncodedEntityToken represents an encoded entity token
type EncodedEntityToken struct {
	*TokenBase
}

// NewEncodedEntityToken creates a new EncodedEntityToken
func NewEncodedEntityToken(decoded, encoded string, sourceSpan *util.ParseSourceSpan) *EncodedEntityToken {
	return &EncodedEntityToken{
		TokenBase: NewTokenBase(TokenTypeENCODED_ENTITY, []string{decoded, encoded}, sourceSpan),
	}
}

// AttributeNameToken represents an attribute name token
type AttributeNameToken struct {
	*TokenBase
}

// NewAttributeNameToken creates a new AttributeNameToken
func NewAttributeNameToken(prefix, name string, sourceSpan *util.ParseSourceSpan) *AttributeNameToken {
	return &AttributeNameToken{
		TokenBase: NewTokenBase(TokenTypeATTR_NAME, []string{prefix, name}, sourceSpan),
	}
}

// AttributeQuoteToken represents an attribute quote token
type AttributeQuoteToken struct {
	*TokenBase
}

// NewAttributeQuoteToken creates a new AttributeQuoteToken
func NewAttributeQuoteToken(quote string, sourceSpan *util.ParseSourceSpan) *AttributeQuoteToken {
	return &AttributeQuoteToken{
		TokenBase: NewTokenBase(TokenTypeATTR_QUOTE, []string{quote}, sourceSpan),
	}
}

// AttributeValueTextToken represents an attribute value text token
type AttributeValueTextToken struct {
	*TokenBase
}

// NewAttributeValueTextToken creates a new AttributeValueTextToken
func NewAttributeValueTextToken(value string, sourceSpan *util.ParseSourceSpan) *AttributeValueTextToken {
	return &AttributeValueTextToken{
		TokenBase: NewTokenBase(TokenTypeATTR_VALUE_TEXT, []string{value}, sourceSpan),
	}
}

// AttributeValueInterpolationToken represents an attribute value interpolation token
type AttributeValueInterpolationToken struct {
	*TokenBase
}

// NewAttributeValueInterpolationToken creates a new AttributeValueInterpolationToken
func NewAttributeValueInterpolationToken(startMarker, expression string, endMarker *string, sourceSpan *util.ParseSourceSpan) *AttributeValueInterpolationToken {
	parts := []string{startMarker, expression}
	if endMarker != nil {
		parts = append(parts, *endMarker)
	}
	return &AttributeValueInterpolationToken{
		TokenBase: NewTokenBase(TokenTypeATTR_VALUE_INTERPOLATION, parts, sourceSpan),
	}
}

// EndOfFileToken represents an end of file token
type EndOfFileToken struct {
	*TokenBase
}

// NewEndOfFileToken creates a new EndOfFileToken
func NewEndOfFileToken(sourceSpan *util.ParseSourceSpan) *EndOfFileToken {
	return &EndOfFileToken{
		TokenBase: NewTokenBase(TokenTypeEOF, []string{}, sourceSpan),
	}
}

// InterpolatedTextToken represents a token that can be part of interpolated text
type InterpolatedTextToken interface {
	Token
}

// InterpolatedAttributeToken represents a token that can be part of interpolated attribute
type InterpolatedAttributeToken interface {
	Token
}
