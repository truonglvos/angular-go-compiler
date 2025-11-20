package view

import (
	"strings"

	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/output"
	constant "ngc-go/packages/compiler/pool"
	"ngc-go/packages/compiler/render3"
	r3_identifiers "ngc-go/packages/compiler/render3/r3_identifiers"
)

// QueryFlags is a set of flags to be used with Queries.
//
// NOTE: Ensure changes here are in sync with `packages/core/src/render3/interfaces/query.ts`
type QueryFlags int

const (
	// QueryFlagsNone - No flags
	QueryFlagsNone QueryFlags = 0b0000

	// QueryFlagsDescendants - Whether or not the query should descend into children.
	QueryFlagsDescendants QueryFlags = 0b0001

	// QueryFlagsIsStatic - The query can be computed statically and hence can be assigned eagerly.
	//
	// NOTE: Backwards compatibility with ViewEngine.
	QueryFlagsIsStatic QueryFlags = 0b0010

	// QueryFlagsEmitDistinctChangesOnly - If the `QueryList` should fire change event only if actual change to query was computed (vs old
	// behavior where the change was fired whenever the query was recomputed, even if the recomputed
	// query resulted in the same list.)
	QueryFlagsEmitDistinctChangesOnly QueryFlags = 0b0100
)

// ToQueryFlags translates query flags into `TQueryFlags` type in
// packages/core/src/render3/interfaces/query.ts
func ToQueryFlags(query R3QueryMetadata) int {
	flags := QueryFlagsNone
	if query.Descendants {
		flags |= QueryFlagsDescendants
	}
	if query.Static {
		flags |= QueryFlagsIsStatic
	}
	if query.EmitDistinctChangesOnly {
		flags |= QueryFlagsEmitDistinctChangesOnly
	}
	return int(flags)
}

// GetQueryPredicate gets the query predicate expression
func GetQueryPredicate(
	query R3QueryMetadata,
	constantPool *constant.ConstantPool,
) output.OutputExpression {
	// Check if predicate is a string array
	if predicateArray, ok := query.Predicate.([]string); ok {
		predicate := []output.OutputExpression{}
		for _, selector := range predicateArray {
			// Each item in predicates array may contain strings with comma-separated refs
			// (for ex. 'ref, ref1, ..., refN'), thus we extract individual refs and store them
			// as separate array entities
			parts := strings.Split(selector, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				predicate = append(predicate, output.NewLiteralExpr(trimmed, output.InferredType, nil))
			}
		}
		forceShared := true
		return constantPool.GetConstLiteral(
			output.NewLiteralArrayExpr(predicate, nil, nil),
			&forceShared,
		)
	}

	// Otherwise, predicate is a MaybeForwardRefExpression
	if maybeRef, ok := query.Predicate.(render3.MaybeForwardRefExpression); ok {
		// The original predicate may have been wrapped in a `forwardRef()` call.
		switch maybeRef.ForwardRef {
		case render3.ForwardRefHandlingNone, render3.ForwardRefHandlingUnwrapped:
			return maybeRef.Expression
		case render3.ForwardRefHandlingWrapped:
			resolveForwardRef := output.NewExternalExpr(r3_identifiers.ResolveForwardRef, nil, nil, nil)
			return output.NewInvokeFunctionExpr(
				resolveForwardRef,
				[]output.OutputExpression{maybeRef.Expression},
				nil,
				nil,
				false,
			)
		}
	}

	// Fallback: try to cast as OutputExpression directly
	if expr, ok := query.Predicate.(output.OutputExpression); ok {
		return expr
	}

	return nil
}

// QueryTypeFns represents query type functions (signal-based and non-signal)
type QueryTypeFns struct {
	SignalBased output.ExternalReference
	NonSignal   output.ExternalReference
}

// CreateQueryCreateCall creates a query create call
func CreateQueryCreateCall(
	query R3QueryMetadata,
	constantPool *constant.ConstantPool,
	queryTypeFns QueryTypeFns,
	prependParams []output.OutputExpression,
) *output.InvokeFunctionExpr {
	parameters := []output.OutputExpression{}
	if prependParams != nil {
		parameters = append(parameters, prependParams...)
	}
	if query.IsSignal {
		// For signal queries, add the context property read first
		ctxVar := output.NewReadVarExpr(CONTEXT_NAME, output.DynamicType, nil)
		propRead := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		parameters = append(parameters, propRead)
	}
	parameters = append(parameters, GetQueryPredicate(query, constantPool))
	parameters = append(parameters, output.NewLiteralExpr(ToQueryFlags(query), output.InferredType, nil))
	if query.Read != nil {
		parameters = append(parameters, *query.Read)
	}

	queryCreateFn := queryTypeFns.NonSignal
	if query.IsSignal {
		queryCreateFn = queryTypeFns.SignalBased
	}
	fnExpr := output.NewExternalExpr(&queryCreateFn, nil, nil, nil)
	return output.NewInvokeFunctionExpr(fnExpr, parameters, nil, nil, false)
}

// queryAdvancePlaceholder represents a placeholder for query advance statements
type queryAdvancePlaceholder struct{}

var queryAdvancePlaceholderInstance = &queryAdvancePlaceholder{}

// CollapseAdvanceStatements collapses query advance placeholders in a list of statements.
//
// This allows for less generated code because multiple sibling query advance
// statements can be collapsed into a single call with the count as argument.
//
// e.g.
//
// ```ts
//   bla();
//   queryAdvance();
//   queryAdvance();
//   bla();
// ```
//
//   --> will turn into
//
// ```ts
//   bla();
//   queryAdvance(2);
//   bla();
// ```
func CollapseAdvanceStatements(
	statements []interface{}, // []output.OutputStatement | *queryAdvancePlaceholder
) []output.OutputStatement {
	result := []output.OutputStatement{}
	advanceCollapseCount := 0

	flushAdvanceCount := func() {
		if advanceCollapseCount > 0 {
			var args []output.OutputExpression
			if advanceCollapseCount == 1 {
				args = []output.OutputExpression{}
			} else {
				args = []output.OutputExpression{output.NewLiteralExpr(advanceCollapseCount, output.InferredType, nil)}
			}
			queryAdvanceExpr := output.NewExternalExpr(r3_identifiers.QueryAdvance, nil, nil, nil)
			callExpr := output.NewInvokeFunctionExpr(queryAdvanceExpr, args, nil, nil, false)
			stmt := output.NewExpressionStatement(callExpr, nil, nil)
			// Insert at beginning (unshift equivalent in Go)
			result = append([]output.OutputStatement{stmt}, result...)
			advanceCollapseCount = 0
		}
	}

	// Iterate through statements in reverse and collapse advance placeholders.
	for i := len(statements) - 1; i >= 0; i-- {
		st := statements[i]
		if _, ok := st.(*queryAdvancePlaceholder); ok {
			advanceCollapseCount++
		} else {
			flushAdvanceCount()
			if stmt, ok := st.(output.OutputStatement); ok {
				// Insert at beginning (unshift equivalent in Go)
				result = append([]output.OutputStatement{stmt}, result...)
			}
		}
	}
	flushAdvanceCount()
	return result
}

// renderFlagCheckIfStmt creates an if statement that checks render flags
// if (rf & flags) { .. }
func renderFlagCheckIfStmt(flags core.RenderFlags, statements []output.OutputStatement) *output.IfStmt {
	rfVar := output.NewReadVarExpr(RENDER_FLAGS, output.DynamicType, nil)
	flagsLiteral := output.NewLiteralExpr(int(flags), output.InferredType, nil)
	condition := output.NewBinaryOperatorExpr(
		output.BinaryOperatorBitwiseAnd,
		rfVar,
		flagsLiteral,
		nil,
		nil,
	)
	return output.NewIfStmt(condition, statements, nil, nil, nil)
}

// CreateViewQueriesFunction defines and updates any view queries
func CreateViewQueriesFunction(
	viewQueries []R3QueryMetadata,
	constantPool *constant.ConstantPool,
	name string,
) output.OutputExpression {
	createStatements := []output.OutputStatement{}
	updateStatements := []interface{}{} // []output.OutputStatement | *queryAdvancePlaceholder

	pushStatement := func(st output.OutputStatement) {
		updateStatements = append(updateStatements, st)
	}
	tempAllocator := TemporaryAllocator(pushStatement, TEMPORARY_NAME)

	for _, query := range viewQueries {
		// creation call, e.g. r3.viewQuery(somePredicate, true) or
		//                r3.viewQuerySignal(ctx.prop, somePredicate, true);
		queryDefinitionCall := CreateQueryCreateCall(query, constantPool, QueryTypeFns{
			SignalBased: *r3_identifiers.ViewQuerySignal,
			NonSignal:   *r3_identifiers.ViewQuery,
		}, nil)
		createStatements = append(createStatements, output.NewExpressionStatement(queryDefinitionCall, nil, nil))

		// Signal queries update lazily and we just advance the index.
		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholderInstance)
			continue
		}

		// update, e.g. (r3.queryRefresh(tmp = r3.loadQuery()) && (ctx.someDir = tmp));
		temporary := tempAllocator()
		loadQueryExpr := output.NewExternalExpr(r3_identifiers.LoadQuery, nil, nil, nil)
		getQueryList := output.NewInvokeFunctionExpr(loadQueryExpr, []output.OutputExpression{}, nil, nil, false)
		queryRefreshExpr := output.NewExternalExpr(r3_identifiers.QueryRefresh, nil, nil, nil)
		setExpr := temporary.Set(getQueryList) // temporary.set(getQueryList)
		refresh := output.NewInvokeFunctionExpr(queryRefreshExpr, []output.OutputExpression{setExpr}, nil, nil, false)
		ctxVar := output.NewReadVarExpr(CONTEXT_NAME, output.DynamicType, nil)
		ctxProp := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		var updateDirective output.OutputExpression
		if query.First {
			firstProp := output.NewReadPropExpr(temporary, "first", output.DynamicType, nil)
			updateDirective = ctxProp.Set(firstProp) // ctx.prop = temporary.first
		} else {
			updateDirective = ctxProp.Set(temporary) // ctx.prop = temporary
		}
		combined := output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			refresh,
			updateDirective,
			nil,
			nil,
		)
		updateStatements = append(updateStatements, output.NewExpressionStatement(combined, nil, nil))
	}

	var viewQueryFnName *string
	if name != "" {
		fnName := name + "_Query"
		viewQueryFnName = &fnName
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam(RENDER_FLAGS, output.NumberType),
			output.NewFnParam(CONTEXT_NAME, nil),
		},
		[]output.OutputStatement{
			renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
			renderFlagCheckIfStmt(core.RenderFlagsUpdate, CollapseAdvanceStatements(updateStatements)),
		},
		output.InferredType,
		nil,
		viewQueryFnName,
	)
}

// CreateContentQueriesFunction defines and updates any content queries
func CreateContentQueriesFunction(
	queries []R3QueryMetadata,
	constantPool *constant.ConstantPool,
	name string,
) output.OutputExpression {
	createStatements := []output.OutputStatement{}
	updateStatements := []interface{}{} // []output.OutputStatement | *queryAdvancePlaceholder

	pushStatement := func(st output.OutputStatement) {
		updateStatements = append(updateStatements, st)
	}
	tempAllocator := TemporaryAllocator(pushStatement, TEMPORARY_NAME)

	for _, query := range queries {
		// creation, e.g. r3.contentQuery(dirIndex, somePredicate, true, null) or
		//                r3.contentQuerySignal(dirIndex, propName, somePredicate, <flags>, <read>).
		dirIndexVar := output.NewReadVarExpr("dirIndex", output.DynamicType, nil)
		queryDefinitionCall := CreateQueryCreateCall(query, constantPool, QueryTypeFns{
			NonSignal:   *r3_identifiers.ContentQuery,
			SignalBased: *r3_identifiers.ContentQuerySignal,
		}, []output.OutputExpression{dirIndexVar})
		createStatements = append(createStatements, output.NewExpressionStatement(queryDefinitionCall, nil, nil))

		// Signal queries update lazily and we just advance the index.
		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholderInstance)
			continue
		}

		// update, e.g. (r3.queryRefresh(tmp = r3.loadQuery()) && (ctx.someDir = tmp));
		temporary := tempAllocator()
		loadQueryExpr := output.NewExternalExpr(r3_identifiers.LoadQuery, nil, nil, nil)
		getQueryList := output.NewInvokeFunctionExpr(loadQueryExpr, []output.OutputExpression{}, nil, nil, false)
		queryRefreshExpr := output.NewExternalExpr(r3_identifiers.QueryRefresh, nil, nil, nil)
		setExpr := temporary.Set(getQueryList) // temporary.set(getQueryList)
		refresh := output.NewInvokeFunctionExpr(queryRefreshExpr, []output.OutputExpression{setExpr}, nil, nil, false)
		ctxVar := output.NewReadVarExpr(CONTEXT_NAME, output.DynamicType, nil)
		ctxProp := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		var updateDirective output.OutputExpression
		if query.First {
			firstProp := output.NewReadPropExpr(temporary, "first", output.DynamicType, nil)
			updateDirective = ctxProp.Set(firstProp) // ctx.prop = temporary.first
		} else {
			updateDirective = ctxProp.Set(temporary) // ctx.prop = temporary
		}
		combined := output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			refresh,
			updateDirective,
			nil,
			nil,
		)
		updateStatements = append(updateStatements, output.NewExpressionStatement(combined, nil, nil))
	}

	var contentQueriesFnName *string
	if name != "" {
		fnName := name + "_ContentQueries"
		contentQueriesFnName = &fnName
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam(RENDER_FLAGS, output.NumberType),
			output.NewFnParam(CONTEXT_NAME, nil),
			output.NewFnParam("dirIndex", nil),
		},
		[]output.OutputStatement{
			renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
			renderFlagCheckIfStmt(core.RenderFlagsUpdate, CollapseAdvanceStatements(updateStatements)),
		},
		output.InferredType,
		nil,
		contentQueriesFnName,
	)
}

