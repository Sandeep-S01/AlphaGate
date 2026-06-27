package pine

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// TokenType represents the lexical token type.
type TokenType int

const (
	TokError TokenType = iota
	TokEOF
	TokIdentifier
	TokNumber
	TokAssign          // = or :=
	TokLParen          // (
	TokRParen          // )
	TokLBracket        // [
	TokRBracket        // ]
	TokComma           // ,
	TokCompare         // >, <, >=, <=, ==, !=
	TokAnd             // and
	TokOr              // or
	TokNot             // not
	TokIf              // if
	TokTernaryQuestion // ?
	TokTernaryColon    // :
	TokNewline         // \n or \r\n
)

// Token represents a lexical token.
type Token struct {
	Type TokenType
	Val  string
	Line int
}

// Lexer tokenizes Pine Script code.
type Lexer struct {
	input        []rune
	pos          int
	line         int
	parenDepth   int
	bracketDepth int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input:        []rune(input),
		pos:          0,
		line:         1,
		parenDepth:   0,
		bracketDepth: 0,
	}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return Token{Type: TokEOF, Val: "", Line: l.line}
	}

	ch := l.input[l.pos]

	// Handle string literals (single or double quotes)
	if ch == '"' || ch == '\'' {
		quoteCh := ch
		start := l.pos
		l.pos++ // consume opening quote
		for l.pos < len(l.input) {
			if l.input[l.pos] == quoteCh {
				l.pos++ // consume closing quote
				return Token{Type: TokIdentifier, Val: string(l.input[start+1 : l.pos-1]), Line: l.line}
			}
			l.pos++
		}
		return Token{Type: TokError, Val: string(l.input[start:]), Line: l.line}
	}

	// Handle Newlines (if not inside parens/brackets)
	if ch == '\n' {
		l.pos++
		tok := Token{Type: TokNewline, Val: "\n", Line: l.line}
		l.line++
		return tok
	}
	if ch == '\r' {
		l.pos++
		if l.pos < len(l.input) && l.input[l.pos] == '\n' {
			l.pos++
		}
		tok := Token{Type: TokNewline, Val: "\n", Line: l.line}
		l.line++
		return tok
	}

	// Double character or single character operators
	if ch == '(' {
		l.pos++
		l.parenDepth++
		return Token{Type: TokLParen, Val: "(", Line: l.line}
	}
	if ch == ')' {
		l.pos++
		if l.parenDepth > 0 {
			l.parenDepth--
		}
		return Token{Type: TokRParen, Val: ")", Line: l.line}
	}
	if ch == '[' {
		l.pos++
		l.bracketDepth++
		return Token{Type: TokLBracket, Val: "[", Line: l.line}
	}
	if ch == ']' {
		l.pos++
		if l.bracketDepth > 0 {
			l.bracketDepth--
		}
		return Token{Type: TokRBracket, Val: "]", Line: l.line}
	}
	if ch == ',' {
		l.pos++
		return Token{Type: TokComma, Val: ",", Line: l.line}
	}
	if ch == '=' {
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokCompare, Val: "==", Line: l.line}
		}
		l.pos++
		return Token{Type: TokAssign, Val: "=", Line: l.line}
	}
	if ch == ':' {
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokAssign, Val: ":=", Line: l.line}
		}
		l.pos++
		return Token{Type: TokTernaryColon, Val: ":", Line: l.line}
	}
	if ch == '?' {
		l.pos++
		return Token{Type: TokTernaryQuestion, Val: "?", Line: l.line}
	}
	if ch == '!' {
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokCompare, Val: "!=", Line: l.line}
		}
		l.pos++
		return Token{Type: TokError, Val: "!", Line: l.line}
	}
	if ch == '>' {
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokCompare, Val: ">=", Line: l.line}
		}
		l.pos++
		return Token{Type: TokCompare, Val: ">", Line: l.line}
	}
	if ch == '<' {
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokCompare, Val: "<=", Line: l.line}
		}
		l.pos++
		return Token{Type: TokCompare, Val: "<", Line: l.line}
	}

	// Numbers (integer or float)
	if unicode.IsDigit(ch) || ch == '.' {
		start := l.pos
		hasDec := ch == '.'
		l.pos++
		for l.pos < len(l.input) {
			nextCh := l.input[l.pos]
			if unicode.IsDigit(nextCh) {
				l.pos++
			} else if nextCh == '.' && !hasDec {
				hasDec = true
				l.pos++
			} else {
				break
			}
		}
		return Token{Type: TokNumber, Val: string(l.input[start:l.pos]), Line: l.line}
	}

	// Identifiers or keywords
	if unicode.IsLetter(ch) || ch == '_' {
		start := l.pos
		l.pos++
		for l.pos < len(l.input) {
			nextCh := l.input[l.pos]
			if unicode.IsLetter(nextCh) || unicode.IsDigit(nextCh) || nextCh == '_' || nextCh == '.' {
				l.pos++
			} else {
				break
			}
		}
		val := string(l.input[start:l.pos])
		lowerVal := strings.ToLower(val)

		switch lowerVal {
		case "and":
			return Token{Type: TokAnd, Val: "and", Line: l.line}
		case "or":
			return Token{Type: TokOr, Val: "or", Line: l.line}
		case "not":
			return Token{Type: TokNot, Val: "not", Line: l.line}
		case "if":
			return Token{Type: TokIf, Val: "if", Line: l.line}
		default:
			return Token{Type: TokIdentifier, Val: val, Line: l.line}
		}
	}

	l.pos++
	return Token{Type: TokError, Val: string(ch), Line: l.line}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' {
			l.pos++
			continue
		}

		// Skip newlines if inside parens or brackets
		if (l.parenDepth > 0 || l.bracketDepth > 0) && (ch == '\n' || ch == '\r') {
			if ch == '\n' {
				l.line++
			} else if ch == '\r' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '\n' {
				l.pos++
				l.line++
			}
			l.pos++
			continue
		}

		// Comment starting with //
		if ch == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/' {
			l.pos += 2
			for l.pos < len(l.input) && l.input[l.pos] != '\n' && l.input[l.pos] != '\r' {
				l.pos++
			}
			continue
		}
		break
	}
}

// Parser parses a token stream into an IRConfig.
type Parser struct {
	tokens    []Token
	pos       int
	config    IRConfig
	errors    []string
	warnings  []string
	autoVar   int
	constants map[string]float64
}

func NewParser(input string) *Parser {
	lexer := NewLexer(input)
	var tokens []Token
	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokEOF {
			break
		}
	}
	return &Parser{
		tokens: tokens,
		pos:    0,
		config: IRConfig{
			Indicators: make(map[string]IndicatorDef),
			Conditions: make(map[string]Expression),
			Rules:      []ExecutionRule{},
		},
		constants: make(map[string]float64),
	}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF, Val: "", Line: 0}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) expect(typ TokenType, errMsg string) bool {
	if p.current().Type != typ {
		p.addError(fmt.Sprintf("line %d: %s (got %q)", p.current().Line, errMsg, p.current().Val))
		return false
	}
	p.advance()
	return true
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

func (p *Parser) addWarning(msg string) {
	p.warnings = append(p.warnings, msg)
}

func (p *Parser) nextAutoVar() string {
	name := fmt.Sprintf("_auto_var_%d", p.autoVar)
	p.autoVar++
	return name
}

// Parse runs the parsing logic.
func (p *Parser) Parse() ParseResult {
	for p.current().Type != TokEOF {
		p.skipNewlines()
		if p.current().Type == TokEOF {
			break
		}

		p.parseStatement()
	}

	p.validateStrategy()

	return ParseResult{
		Config:   p.config,
		Warnings: p.warnings,
		Errors:   p.errors,
	}
}

func (p *Parser) skipNewlines() {
	for p.current().Type == TokNewline {
		p.advance()
	}
}

func (p *Parser) isSkippableStatement(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "plot", "plotshape", "plotchar", "plotarrow", "plotbar", "plotcandle", "bgcolor", "alert", "alertcondition", "indicator":
		return true
	}
	if lower == "strategy" {
		return true
	}
	if strings.HasPrefix(lower, "table.") || strings.HasPrefix(lower, "color.") || strings.HasPrefix(lower, "str.") || strings.HasPrefix(lower, "math.") {
		return true
	}
	return false
}

func (p *Parser) skipRestOfStatement() {
	parenDepth := 0
	bracketDepth := 0
	for p.current().Type != TokEOF {
		tok := p.current()
		if tok.Type == TokLParen {
			parenDepth++
		} else if tok.Type == TokRParen {
			parenDepth--
		} else if tok.Type == TokLBracket {
			bracketDepth++
		} else if tok.Type == TokRBracket {
			bracketDepth--
		}

		if tok.Type == TokNewline && parenDepth <= 0 && bracketDepth <= 0 {
			p.advance() // consume the newline
			break
		}
		p.advance()
	}
}

func isTypeName(name string) bool {
	switch strings.ToLower(name) {
	case "int", "float", "string", "bool", "table", "color":
		return true
	}
	return false
}

func (p *Parser) parseStatement() {
	tok := p.current()

	if p.isStatementContinuation(tok) {
		p.skipRestOfStatement()
		return
	}

	// Skip 'var' keyword
	if tok.Type == TokIdentifier && strings.ToLower(tok.Val) == "var" {
		p.advance()
		tok = p.current()
	}

	// Skip type declarations: int, float, string, bool, table, color
	if tok.Type == TokIdentifier && isTypeName(tok.Val) {
		p.advance()
		tok = p.current()
	}

	// Check if statement should be skipped
	if tok.Type == TokIdentifier && p.isSkippableStatement(tok.Val) {
		p.skipRestOfStatement()
		return
	}

	if tok.Type == TokIdentifier {
		p.advance()
		if p.current().Type == TokAssign {
			p.advance() // consume '=' or ':='
			// Check if the right-hand side is a skippable function call
			nextTok := p.current()
			if nextTok.Type == TokIdentifier && p.isSkippableStatement(nextTok.Val) {
				p.skipRestOfStatement()
				return
			}
			p.parseAssignment(tok.Val)
		} else {
			// Backtrack and check if it's a directive (like strategy.entry or strategy.close/exit)
			p.pos-- // go back to TokIdentifier
			p.parseDirective("")
		}
	} else if tok.Type == TokIf {
		p.advance() // consume 'if'
		p.parseIfStatement()
	} else {
		p.addError(fmt.Sprintf("line %d: unexpected token %q", tok.Line, tok.Val))
		p.advance()
	}
}

func (p *Parser) isStatementContinuation(tok Token) bool {
	if tok.Type == TokAnd || tok.Type == TokOr || tok.Type == TokCompare || tok.Type == TokTernaryColon || tok.Type == TokError {
		return true
	}
	return tok.Type == TokIdentifier && strings.EqualFold(tok.Val, "else")
}

func (p *Parser) parseAssignment(varName string) {
	expr, err := p.parseExpression(0)
	if err != nil {
		p.addError(fmt.Sprintf("line %d: invalid assignment: %v", p.current().Line, err))
		p.skipRestOfStatement()
		return
	}

	// Record constant value if applicable
	if expr.Op == "ref" {
		if val, err := strconv.ParseFloat(expr.Val, 64); err == nil {
			p.constants[varName] = val
		}
	}

	// Check if this is an indicator function call directly
	if expr.Op == "indicator" {
		indType := expr.Val
		source := ""
		var params []float64

		if len(expr.Args) > 0 {
			if indType == "atr" {
				if val, err := p.resolveConstant(expr.Args[0]); err == nil {
					params = append(params, val)
				} else {
					p.addError(fmt.Sprintf("line %d: atr period must be a literal number or constant variable", p.current().Line))
				}
			} else {
				if expr.Args[0].Op == "ref" {
					source = expr.Args[0].Val
				} else {
					p.addError(fmt.Sprintf("line %d: indicator source must be a variable or input name", p.current().Line))
				}

				for i := 1; i < len(expr.Args); i++ {
					if val, err := p.resolveConstant(expr.Args[i]); err == nil {
						params = append(params, val)
					} else {
						p.addError(fmt.Sprintf("line %d: indicator parameters must be literal numbers or constant variables", p.current().Line))
					}
				}
			}
		}

		p.config.Indicators[varName] = IndicatorDef{
			Type:   indType,
			Source: source,
			Params: params,
		}
	} else {
		p.config.Conditions[varName] = expr
	}
}

func (p *Parser) resolveConstant(expr Expression) (float64, error) {
	if expr.Op == "ref" {
		if val, exists := p.constants[expr.Val]; exists {
			return val, nil
		}
		if val, err := strconv.ParseFloat(expr.Val, 64); err == nil {
			return val, nil
		}
	}
	return 0, fmt.Errorf("not a float constant")
}

func (p *Parser) parseIfStatement() {
	expr, err := p.parseExpression(0)
	if err != nil {
		p.addError(fmt.Sprintf("line %d: invalid if condition: %v", p.current().Line, err))
		return
	}

	p.skipNewlines()

	var condVar string
	if expr.Op == "ref" {
		condVar = expr.Val
	} else {
		condVar = p.nextAutoVar()
		p.config.Conditions[condVar] = expr
	}

	if p.current().Type != TokIdentifier || !strings.HasPrefix(strings.ToLower(p.current().Val), "strategy.") {
		p.skipRestOfStatement()
		return
	}

	p.parseDirective(condVar)
}

func (p *Parser) parseDirective(condVar string) {
	tok := p.current()
	if tok.Type != TokIdentifier {
		p.addError(fmt.Sprintf("line %d: expected directive starting with strategy.entry/close/exit, got %q", tok.Line, tok.Val))
		return
	}

	p.advance()
	if !p.expect(TokLParen, "expected '(' after directive name") {
		return
	}

	args := []string{}
	for p.current().Type != TokRParen && p.current().Type != TokEOF {
		argTok := p.current()
		if argTok.Type == TokIdentifier || argTok.Type == TokNumber {
			val := argTok.Val
			p.advance()

			// Handle named parameters: name=val
			if p.current().Type == TokAssign {
				p.advance() // consume '='
				valVal := p.current().Val
				p.advance()
				args = append(args, val+"="+valVal)
			} else {
				val = strings.Trim(val, `"'`)
				args = append(args, val)
			}
		} else {
			p.addError(fmt.Sprintf("line %d: unexpected directive argument %q", argTok.Line, argTok.Val))
			p.advance()
		}

		if p.current().Type == TokComma {
			p.advance()
		} else if p.current().Type != TokRParen {
			p.addError(fmt.Sprintf("line %d: expected ',' or ')' in directive arguments", p.current().Line))
			break
		}
	}

	if !p.expect(TokRParen, "expected ')' after directive arguments") {
		return
	}

	action := ""
	direction := ""
	id := ""

	directiveName := strings.ToLower(tok.Val)
	if directiveName == "strategy.entry" {
		action = "entry"
		if len(args) >= 1 {
			id = args[0]
		}
		if len(args) >= 2 {
			dirVal := strings.ToLower(args[1])
			if strings.Contains(dirVal, "long") {
				direction = "long"
			} else if strings.Contains(dirVal, "short") {
				direction = "short"
			} else {
				p.addError(fmt.Sprintf("line %d: strategy.entry direction must be strategy.long or strategy.short", tok.Line))
			}
		} else {
			direction = "long"
		}
	} else if directiveName == "strategy.close" || directiveName == "strategy.exit" {
		action = "close"
		if len(args) >= 1 {
			// For strategy.exit("Long Exit", "Long", ...), args[1] is the entry ID "Long"
			if len(args) >= 2 && !strings.Contains(args[1], "=") {
				id = args[1]
			} else {
				id = args[0]
			}
			if strings.Contains(strings.ToLower(id), "short") {
				direction = "short"
			} else if strings.Contains(strings.ToLower(id), "long") {
				direction = "long"
			}
		}
	} else {
		// Ignore metadata strategies or other strategies gracefully
		return
	}

	if condVar == "" {
		condVar = "_always_true"
		p.config.Conditions[condVar] = Expression{Op: "ref", Val: "true"}
	}

	p.config.Rules = append(p.config.Rules, ExecutionRule{
		Condition: condVar,
		Action:    action,
		ID:        id,
		Direction: direction,
	})
}

// Expression parsing using recursive descent / precedence-climbing.
func (p *Parser) parseExpression(minPrecedence int) (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return Expression{}, err
	}

	for {
		tok := p.current()

		// Ternary Operator: left ? trueExpr : falseExpr
		if tok.Type == TokTernaryQuestion {
			p.advance() // consume '?'
			trueExpr, err := p.parseExpression(0)
			if err != nil {
				return Expression{}, err
			}
			if p.current().Type != TokTernaryColon {
				return Expression{}, fmt.Errorf("expected ':' in ternary expression")
			}
			p.advance() // consume ':'
			falseExpr, err := p.parseExpression(0)
			if err != nil {
				return Expression{}, err
			}
			left = Expression{
				Op:   "ternary",
				Args: []Expression{left, trueExpr, falseExpr},
			}
			continue
		}

		opPrecedence := p.getPrecedence(tok)
		if opPrecedence < minPrecedence {
			break
		}

		op := tok.Val
		p.advance() // consume operator

		right, err := p.parseExpression(opPrecedence + 1)
		if err != nil {
			return Expression{}, err
		}

		left = Expression{
			Op:   strings.ToLower(op),
			Args: []Expression{left, right},
		}
	}

	return left, nil
}

func (p *Parser) parsePrimary() (Expression, error) {
	tok := p.current()
	if tok.Type == TokLParen {
		p.advance() // consume '('
		expr, err := p.parseExpression(0)
		if err != nil {
			return Expression{}, err
		}
		if !p.expect(TokRParen, "expected ')'") {
			return Expression{}, fmt.Errorf("missing closing parenthesis")
		}
		return expr, nil
	}

	if tok.Type == TokNot {
		p.advance() // consume 'not'
		expr, err := p.parseExpression(p.getPrecedence(tok))
		if err != nil {
			return Expression{}, err
		}
		return Expression{
			Op:   "not",
			Args: []Expression{expr},
		}, nil
	}

	if tok.Type == TokNumber {
		p.advance()
		return Expression{Op: "ref", Val: tok.Val}, nil
	}

	if tok.Type == TokIdentifier {
		p.advance()

		// Could be a function call: identifier(args...)
		if p.current().Type == TokLParen {
			p.advance() // consume '('

			fnName := strings.ToLower(tok.Val)
			// Special case: input.* default value extraction
			if strings.HasPrefix(fnName, "input.") || fnName == "input" {
				firstArg, err := p.parseExpression(0)
				if err != nil {
					return Expression{}, err
				}
				// Skip all other arguments until the matching ')'
				parenDepth := 1
				for parenDepth > 0 && p.current().Type != TokEOF {
					t := p.current()
					if t.Type == TokLParen {
						parenDepth++
					} else if t.Type == TokRParen {
						parenDepth--
					}
					p.advance()
				}
				return firstArg, nil
			}

			args := []Expression{}
			for p.current().Type != TokRParen && p.current().Type != TokEOF {
				argExpr, err := p.parseExpression(0)
				if err != nil {
					return Expression{}, err
				}
				args = append(args, argExpr)

				if p.current().Type == TokComma {
					p.advance()
				} else if p.current().Type != TokRParen {
					return Expression{}, fmt.Errorf("expected ',' or ')' in argument list")
				}
			}

			if !p.expect(TokRParen, "expected ')' after argument list") {
				return Expression{}, fmt.Errorf("missing closing parenthesis in function call")
			}

			// Is it an indicator or built-in function?
			if strings.HasPrefix(fnName, "ta.") {
				shortName := strings.TrimPrefix(fnName, "ta.")
				if isIndicatorType(shortName) {
					flattenedArgs := make([]Expression, len(args))
					for i, arg := range args {
						if arg.Op == "indicator" {
							nestedVar := p.nextAutoVar()
							p.saveAutoIndicator(nestedVar, arg)
							flattenedArgs[i] = Expression{Op: "ref", Val: nestedVar}
						} else {
							flattenedArgs[i] = arg
						}
					}

					return Expression{
						Op:   "indicator",
						Val:  shortName,
						Args: flattenedArgs,
					}, nil
				}

				if shortName == "crossover" || shortName == "crossunder" {
					flattenedArgs := make([]Expression, len(args))
					for i, arg := range args {
						if arg.Op == "indicator" {
							nestedVar := p.nextAutoVar()
							p.saveAutoIndicator(nestedVar, arg)
							flattenedArgs[i] = Expression{Op: "ref", Val: nestedVar}
						} else {
							flattenedArgs[i] = arg
						}
					}
					return Expression{
						Op:   shortName,
						Args: flattenedArgs,
					}, nil
				}
			}

			// For any other function calls in conditions (like math.round, color.new), we can return a dummy value if skipped,
			// otherwise we return a placeholder reference.
			return Expression{Op: "ref", Val: "0"}, nil
		}

		// Simple variable/input reference
		return Expression{Op: "ref", Val: tok.Val}, nil
	}

	return Expression{}, fmt.Errorf("unexpected token %q", tok.Val)
}

func (p *Parser) saveAutoIndicator(varName string, indicatorExpr Expression) {
	indType := indicatorExpr.Val
	source := ""
	var params []float64

	if len(indicatorExpr.Args) > 0 {
		if indType == "atr" {
			if val, err := p.resolveConstant(indicatorExpr.Args[0]); err == nil {
				params = append(params, val)
			}
		} else {
			if indicatorExpr.Args[0].Op == "ref" {
				source = indicatorExpr.Args[0].Val
			}
			for i := 1; i < len(indicatorExpr.Args); i++ {
				if val, err := p.resolveConstant(indicatorExpr.Args[i]); err == nil {
					params = append(params, val)
				}
			}
		}
	}

	p.config.Indicators[varName] = IndicatorDef{
		Type:   indType,
		Source: source,
		Params: params,
	}
}

func (p *Parser) getPrecedence(tok Token) int {
	if tok.Type == TokOr {
		return 1
	}
	if tok.Type == TokAnd {
		return 2
	}
	if tok.Type == TokCompare {
		return 3
	}
	return -1
}

func (p *Parser) validateStrategy() {
	hasEntry := false
	hasClose := false

	for _, rule := range p.config.Rules {
		if rule.Action == "entry" {
			hasEntry = true
		}
		if rule.Action == "close" {
			hasClose = true
		}
	}

	if !hasEntry {
		p.addWarning("Strategy does not have any entry rules (e.g. strategy.entry). It will never open positions.")
	}
	if !hasClose {
		p.addWarning("Strategy does not have any exit/close rules (e.g. strategy.close). Positions will only exit on default risk settings or stops.")
	}

	// Check for direct hardcoded numbers in condition comparisons
	for condName, expr := range p.config.Conditions {
		p.checkExpressionForHardcodedConstants(condName, expr)
	}
}

func (p *Parser) checkExpressionForHardcodedConstants(condName string, expr Expression) {
	if expr.Op == ">" || expr.Op == "<" || expr.Op == ">=" || expr.Op == "<=" || expr.Op == "==" || expr.Op == "!=" {
		for _, arg := range expr.Args {
			if arg.Op == "ref" {
				if _, err := strconv.ParseFloat(arg.Val, 64); err == nil {
					p.addWarning(fmt.Sprintf("Condition %q contains a hardcoded constant %q. Consider defining it as an input variable for optimization.", condName, arg.Val))
				}
			}
		}
	}

	for _, arg := range expr.Args {
		p.checkExpressionForHardcodedConstants(condName, arg)
	}
}

func isIndicatorType(name string) bool {
	switch strings.ToLower(name) {
	case "sma", "ema", "rsi", "atr", "macd", "bb":
		return true
	}
	return false
}
