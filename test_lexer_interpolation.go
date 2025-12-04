package main

import (
	"fmt"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
)

func main() {
	source := "{{foo // comment}}"
	file := util.NewParseSourceFile(source, "test.html")

	trueVal := true
	options := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: &trueVal,
	}

	tokenizer := ml_parser.NewTokenizer(file, func(tagName string) ml_parser.TagDefinition {
		return nil
	}, options)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic: %v\n", r)
		}
	}()

	tokenizer.Tokenize()
	fmt.Println("Success! Tokenized without error")
}
