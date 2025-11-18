package core

// Character code constants
const (
	CharEOF       = 0
	CharBSPACE    = 8
	CharTAB       = 9
	CharLF        = 10
	CharVTAB      = 11
	CharFF        = 12
	CharCR        = 13
	CharSPACE     = 32
	CharBANG      = 33
	CharDQ        = 34
	CharHASH      = 35
	CharDollar    = 36
	CharPERCENT   = 37
	CharAMPERSAND = 38
	CharSQ        = 39
	CharLPAREN    = 40
	CharRPAREN    = 41
	CharSTAR      = 42
	CharPLUS      = 43
	CharCOMMA     = 44
	CharMINUS     = 45
	CharPERIOD    = 46
	CharSLASH     = 47
	CharCOLON     = 58
	CharSEMICOLON = 59
	CharLT        = 60
	CharEQ        = 61
	CharGT        = 62
	CharQUESTION  = 63

	Char0 = 48
	Char7 = 55
	Char9 = 57

	CharA = 65
	CharE = 69
	CharF = 70
	CharX = 88
	CharZ = 90

	CharLBRACKET   = 91
	CharBACKSLASH  = 92
	CharRBRACKET   = 93
	CharCARET      = 94
	CharUnderscore = 95

	CharLowerA = 97
	CharLowerB = 98
	CharLowerE = 101
	CharLowerF = 102
	CharLowerN = 110
	CharLowerR = 114
	CharLowerT = 116
	CharLowerU = 117
	CharLowerV = 118
	CharLowerX = 120
	CharLowerZ = 122

	CharLBRACE = 123
	CharBAR    = 124
	CharRBRACE = 125
	CharNBSP   = 160

	CharPIPE  = 124
	CharTILDA = 126
	CharAT    = 64
	CharBT    = 96
)

// IsWhitespace checks if a character code represents whitespace
func IsWhitespace(code int) bool {
	return (code >= CharTAB && code <= CharSPACE) || code == CharNBSP
}

// IsDigit checks if a character code represents a digit
func IsDigit(code int) bool {
	return Char0 <= code && code <= Char9
}

// IsAsciiLetter checks if a character code represents an ASCII letter
func IsAsciiLetter(code int) bool {
	return (code >= CharLowerA && code <= CharLowerZ) || (code >= CharA && code <= CharZ)
}

// IsAsciiHexDigit checks if a character code represents a hexadecimal digit
func IsAsciiHexDigit(code int) bool {
	return (code >= CharLowerA && code <= CharLowerF) || (code >= CharA && code <= CharF) || IsDigit(code)
}

// IsNewLine checks if a character code represents a newline
func IsNewLine(code int) bool {
	return code == CharLF || code == CharCR
}

// IsOctalDigit checks if a character code represents an octal digit
func IsOctalDigit(code int) bool {
	return Char0 <= code && code <= Char7
}

// IsQuote checks if a character code represents a quote character
func IsQuote(code int) bool {
	return code == CharSQ || code == CharDQ || code == CharBT
}

