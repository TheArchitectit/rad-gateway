package controlroom

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TokenType represents the type of lexical token in a tag query.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenTag             // category:value or category:wildcard
	TokenAnd             // AND
	TokenOr              // OR
	TokenNot             // NOT
	TokenLParen          // (
	TokenRParen          // )
	TokenInvalid         // Invalid token
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenTag:
		return "TAG"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenInvalid:
		return "INVALID"
	default:
		return "UNKNOWN"
	}
}

// Token represents a lexical token in the input.
type Token struct {
	Type  TokenType
	Value string
	Pos   int // Position in input string
}

// Lexer tokenizes tag query strings.
type Lexer struct {
	input string
	pos   int
	ch    byte
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch != 0 && (l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r') {
		l.readChar()
	}
}

func (l *Lexer) readString() string {
	start := l.pos - 1
	for l.ch != 0 && l.ch != ' ' && l.ch != '\t' && l.ch != '\n' && l.ch != '\r' &&
		l.ch != '(' && l.ch != ')' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.ch == 0 {
		return Token{Type: TokenEOF, Pos: l.pos}
	}

	pos := l.pos - 1

	switch l.ch {
	case '(':
		l.readChar()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}
	case ')':
		l.readChar()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}
	}

	// Read the next string token
	start := l.pos - 1
	for l.ch != 0 && (l.ch != ' ' && l.ch != '\t' && l.ch != '\n' && l.ch != '\r' &&
		l.ch != '(' && l.ch != ')') {
		l.readChar()
	}
	value := l.input[start : l.pos-1]

	// Check for keywords (case-insensitive)
	upperValue := strings.ToUpper(value)
	switch upperValue {
	case "AND":
		return Token{Type: TokenAnd, Value: value, Pos: start}
	case "OR":
		return Token{Type: TokenOr, Value: value, Pos: start}
	case "NOT":
		return Token{Type: TokenNot, Value: value, Pos: start}
	}

	// Must be a tag expression
	if strings.Contains(value, ":") {
		return Token{Type: TokenTag, Value: value, Pos: start}
	}

	return Token{Type: TokenInvalid, Value: value, Pos: start}
}

// Tokenize returns all tokens in the input.
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		token := l.NextToken()
		if token.Type == TokenInvalid {
			return nil, fmt.Errorf("invalid token at position %d: %s", token.Pos, token.Value)
		}
		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}

// ExpressionType represents the type of expression node.
type ExpressionType int

const (
	ExprTag ExpressionType = iota
	ExprAnd
	ExprOr
	ExprNot
)

func (e ExpressionType) String() string {
	switch e {
	case ExprTag:
		return "TAG"
	case ExprAnd:
		return "AND"
	case ExprOr:
		return "OR"
	case ExprNot:
		return "NOT"
	default:
		return "UNKNOWN"
	}
}

// TagExpression is the interface for all tag expression nodes.
type TagExpression interface {
	Type() ExpressionType
	String() string
}

// TagExpr represents a single tag match.
type TagExpr struct {
	Category string
	Pattern  string // Can include wildcards like "*"
}

// Type returns the expression type.
func (t *TagExpr) Type() ExpressionType { return ExprTag }

// String returns the string representation.
func (t *TagExpr) String() string {
	return fmt.Sprintf("%s:%s", t.Category, t.Pattern)
}

// Matches checks if a tag matches this expression.
func (t *TagExpr) Matches(tag Tag) bool {
	if tag.Category != t.Category {
		return false
	}
	return tag.MatchesWildcard(t.Pattern)
}

// AndExpr represents an AND combination of expressions.
type AndExpr struct {
	Left, Right TagExpression
}

// Type returns the expression type.
func (a *AndExpr) Type() ExpressionType { return ExprAnd }

// String returns the string representation.
func (a *AndExpr) String() string {
	return fmt.Sprintf("(%s AND %s)", a.Left.String(), a.Right.String())
}

// OrExpr represents an OR combination of expressions.
type OrExpr struct {
	Left, Right TagExpression
}

// Type returns the expression type.
func (o *OrExpr) Type() ExpressionType { return ExprOr }

// String returns the string representation.
func (o *OrExpr) String() string {
	return fmt.Sprintf("(%s OR %s)", o.Left.String(), o.Right.String())
}

// NotExpr represents a NOT (negation) of an expression.
type NotExpr struct {
	Expr TagExpression
}

// Type returns the expression type.
func (n *NotExpr) Type() ExpressionType { return ExprNot }

// String returns the string representation.
func (n *NotExpr) String() string {
	return fmt.Sprintf("(NOT %s)", n.Expr.String())
}

// TagQueryParser parses tag query strings into expression trees.
type TagQueryParser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

// NewTagQueryParser creates a new parser.
func NewTagQueryParser() *TagQueryParser {
	return &TagQueryParser{}
}

// Parse parses a tag query string and returns an expression tree.
func (p *TagQueryParser) Parse(input string) (TagExpression, error) {
	p.lexer = NewLexer(input)
	tokens, err := p.lexer.Tokenize()
	if err != nil {
		return nil, err
	}
	p.tokens = tokens
	p.pos = 0

	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", p.current().Pos, p.current().Value)
	}

	return expr, nil
}

func (p *TagQueryParser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *TagQueryParser) advance() Token {
	token := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return token
}

func (p *TagQueryParser) expect(tt TokenType) (Token, error) {
	token := p.current()
	if token.Type != tt {
		return Token{}, fmt.Errorf("expected %s at position %d, got %s", tt, token.Pos, token.Type)
	}
	p.advance()
	return token, nil
}

// parseOr parses OR expressions (lowest precedence).
func (p *TagQueryParser) parseOr() (TagExpression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOr {
		p.advance() // consume OR
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrExpr{Left: left, Right: right}
	}

	return left, nil
}

// parseAnd parses AND expressions (medium precedence).
func (p *TagQueryParser) parseAnd() (TagExpression, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenAnd {
		p.advance() // consume AND
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}

	// Implicit AND: tag1 tag2 means tag1 AND tag2
	for p.current().Type == TokenTag || p.current().Type == TokenLParen || p.current().Type == TokenNot {
		// Don't consume if it's an OR (handled in parseOr)
		if p.peekIsOr() {
			break
		}
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}

	return left, nil
}

// parseNot parses NOT expressions (high precedence).
func (p *TagQueryParser) parseNot() (TagExpression, error) {
	if p.current().Type == TokenNot {
		p.advance() // consume NOT
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Expr: expr}, nil
	}
	return p.parsePrimary()
}

// parsePrimary parses primary expressions (tags and parenthesized expressions).
func (p *TagQueryParser) parsePrimary() (TagExpression, error) {
	token := p.current()

	switch token.Type {
	case TokenTag:
		p.advance()
		return parseTagExpr(token.Value)
	case TokenLParen:
		p.advance() // consume (
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	default:
		return nil, fmt.Errorf("unexpected token at position %d: %s (expected tag or '(')", token.Pos, token.Type)
	}
}

func (p *TagQueryParser) peekIsOr() bool {
	// Look ahead without modifying position
	peekPos := p.pos
	for peekPos < len(p.tokens) {
		if p.tokens[peekPos].Type == TokenOr {
			return true
		}
		// Skip tokens that could be part of current expression
		if p.tokens[peekPos].Type == TokenTag || p.tokens[peekPos].Type == TokenAnd ||
			p.tokens[peekPos].Type == TokenNot || p.tokens[peekPos].Type == TokenLParen {
			peekPos++
			continue
		}
		break
	}
	return false
}

func parseTagExpr(s string) (*TagExpr, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tag expression: %q", s)
	}
	return &TagExpr{
		Category: strings.TrimSpace(parts[0]),
		Pattern:  strings.TrimSpace(parts[1]),
	}, nil
}

// Evaluate evaluates a tag expression against a set of resource tags.
func Evaluate(expr TagExpression, resourceTags []Tag) bool {
	switch e := expr.(type) {
	case *TagExpr:
		for _, tag := range resourceTags {
			if e.Matches(tag) {
				return true
			}
		}
		return false
	case *AndExpr:
		return Evaluate(e.Left, resourceTags) && Evaluate(e.Right, resourceTags)
	case *OrExpr:
		return Evaluate(e.Left, resourceTags) || Evaluate(e.Right, resourceTags)
	case *NotExpr:
		return !Evaluate(e.Expr, resourceTags)
	case nil:
		return true // Empty expression matches all
	default:
		return false
	}
}

// Validate validates a tag query string.
func Validate(query string) error {
	parser := NewTagQueryParser()
	_, err := parser.Parse(query)
	return err
}

// Normalize normalizes a tag query string (standardizes formatting).
func Normalize(query string) (string, error) {
	parser := NewTagQueryParser()
	expr, err := parser.Parse(query)
	if err != nil {
		return "", err
	}
	if expr == nil {
		return "", nil
	}
	return normalizeExpr(expr), nil
}

func normalizeExpr(expr TagExpression) string {
	switch e := expr.(type) {
	case *TagExpr:
		return e.String()
	case *AndExpr:
		return fmt.Sprintf("%s AND %s", normalizeExpr(e.Left), normalizeExpr(e.Right))
	case *OrExpr:
		return fmt.Sprintf("%s OR %s", normalizeExpr(e.Left), normalizeExpr(e.Right))
	case *NotExpr:
		return fmt.Sprintf("NOT %s", normalizeExpr(e.Expr))
	default:
		return ""
	}
}

// ExtractTags extracts all tag expressions from a query.
func ExtractTags(query string) ([]TagExpr, error) {
	parser := NewTagQueryParser()
	expr, err := parser.Parse(query)
	if err != nil {
		return nil, err
	}
	return extractTagsFromExpr(expr), nil
}

func extractTagsFromExpr(expr TagExpression) []TagExpr {
	switch e := expr.(type) {
	case *TagExpr:
		return []TagExpr{*e}
	case *AndExpr:
		return append(extractTagsFromExpr(e.Left), extractTagsFromExpr(e.Right)...)
	case *OrExpr:
		return append(extractTagsFromExpr(e.Left), extractTagsFromExpr(e.Right)...)
	case *NotExpr:
		return extractTagsFromExpr(e.Expr)
	default:
		return nil
	}
}

// ExtractUsedCategories returns all category names referenced in a query.
func ExtractUsedCategories(query string) []string {
	tags, err := ExtractTags(query)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var categories []string
	for _, tag := range tags {
		if !seen[tag.Category] {
			seen[tag.Category] = true
			categories = append(categories, tag.Category)
		}
	}
	return categories
}

// IsEmpty checks if a query is empty or only whitespace.
func IsEmpty(query string) bool {
	return strings.TrimSpace(query) == ""
}

// IsValidQuery checks if a query string is valid.
func IsValidQuery(query string) bool {
	return Validate(query) == nil
}

// simpleQueryRegex matches simple category:value patterns.
var simpleQueryRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*:[^\s]+$`)

// IsSimpleQuery checks if a query is a simple single-tag query.
func IsSimpleQuery(query string) bool {
	query = strings.TrimSpace(query)
	return simpleQueryRegex.MatchString(query)
}

// Sanitize removes potentially dangerous characters from a query string.
func Sanitize(query string) string {
	// Remove null bytes and control characters
	var result strings.Builder
	for _, r := range query {
		if r == 0 {
			continue // Skip null bytes
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue // Skip control chars except newlines and tabs
		}
		result.WriteRune(r)
	}
	return result.String()
}

// MatchResult represents the result of matching a resource against a filter.
type MatchResult struct {
	Matched   bool
	Expr      TagExpression
	Resource  TaggedResource
	// MatchedTags contains the tags that satisfied the filter
	MatchedTags []Tag
}

// Match matches a resource against a tag filter query.
func Match(query string, resource TaggedResource) (*MatchResult, error) {
	parser := NewTagQueryParser()
	expr, err := parser.Parse(query)
	if err != nil {
		return nil, err
	}

	matched := Evaluate(expr, resource.Tags)

	// Find which tags matched
	var matchedTags []Tag
	if matched && expr != nil {
		if tagExpr, ok := expr.(*TagExpr); ok {
			for _, tag := range resource.Tags {
				if tagExpr.Matches(tag) {
					matchedTags = append(matchedTags, tag)
				}
			}
		}
	}

	return &MatchResult{
		Matched:     matched,
		Expr:        expr,
		Resource:    resource,
		MatchedTags: matchedTags,
	}, nil
}
