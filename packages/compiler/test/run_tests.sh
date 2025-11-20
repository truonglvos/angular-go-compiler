#!/bin/bash

cd /Users/truong/Documents/go/packages/compiler/test

echo "Running util tests..."
cd util && go test -v && cd ..

echo ""
echo "Running expression_parser tests..."
cd /Users/truong/Documents/go/packages/compiler/test/expression_parser && go test -v

