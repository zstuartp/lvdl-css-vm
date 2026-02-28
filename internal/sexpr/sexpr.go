// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

// Package sexpr provides a minimal s-expression parser and
// formatter. It handles parenthesized lists, bare atoms, and
// semicolon-delimited line comments.
package sexpr

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Node interface{ isNode() }

type Atom struct{ Value string }

func (Atom) isNode() {}

type List struct{ Items []Node }

func (List) isNode() {}

// ParseAll parses one or more top-level s-expressions.
// Comments starting with ';' run to end of line.
func ParseAll(r io.Reader) ([]Node, error) {
	t := newTokenizer(r)
	var out []Node
	for {
		_, ok, err := t.peek()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		n, err := parseNode(t)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func parseNode(t *tokenizer) (Node, error) {
	tok, ok, err := t.next()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, io.EOF
	}

	switch tok.kind {
	case tokLParen:
		var items []Node
		for {
			p, ok, err := t.peek()
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf(
					"unexpected EOF (missing ')')",
				)
			}
			if p.kind == tokRParen {
				_, _, _ = t.next() // consume ')'
				break
			}
			n, err := parseNode(t)
			if err != nil {
				return nil, err
			}
			items = append(items, n)
		}
		return List{Items: items}, nil
	case tokRParen:
		return nil, fmt.Errorf("unexpected ')'")
	case tokAtom:
		return Atom{Value: tok.text}, nil
	default:
		return nil, fmt.Errorf("unknown token: %#v", tok)
	}
}

// Format serializes a Node back into s-expression text.
func Format(n Node) string {
	switch v := n.(type) {
	case Atom:
		return v.Value
	case List:
		var b strings.Builder
		b.WriteByte('(')
		for i, it := range v.Items {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(Format(it))
		}
		b.WriteByte(')')
		return b.String()
	default:
		return "<?>"
	}
}

/* --- tokenizer --- */

type tokenKind int

const (
	tokLParen tokenKind = iota
	tokRParen
	tokAtom
)

type token struct {
	kind tokenKind
	text string
}

type tokenizer struct {
	r        *bufio.Reader
	peeked   *token
	peekedOk bool
}

func newTokenizer(r io.Reader) *tokenizer {
	return &tokenizer{r: bufio.NewReader(r)}
}

func (t *tokenizer) peek() (token, bool, error) {
	if t.peekedOk {
		return *t.peeked, true, nil
	}
	tok, ok, err := t.readToken()
	if err != nil {
		return token{}, false, err
	}
	if !ok {
		return token{}, false, nil
	}
	t.peeked = &tok
	t.peekedOk = true
	return tok, true, nil
}

func (t *tokenizer) next() (token, bool, error) {
	if t.peekedOk {
		tok := *t.peeked
		t.peekedOk = false
		t.peeked = nil
		return tok, true, nil
	}
	return t.readToken()
}

func (t *tokenizer) readToken() (token, bool, error) {
	for {
		r, _, err := t.r.ReadRune()
		if err == io.EOF {
			return token{}, false, nil
		}
		if err != nil {
			return token{}, false, err
		}

		if unicode.IsSpace(r) {
			continue
		}

		// Skip line comments.
		if r == ';' {
			for {
				rr, _, e := t.r.ReadRune()
				if e == io.EOF {
					return token{}, false, nil
				}
				if e != nil {
					return token{}, false, e
				}
				if rr == '\n' {
					break
				}
			}
			continue
		}

		switch r {
		case '(':
			return token{tokLParen, "("}, true, nil
		case ')':
			return token{tokRParen, ")"}, true, nil
		default:
			var b strings.Builder
			b.WriteRune(r)
			for {
				rr, _, e := t.r.ReadRune()
				if e == io.EOF {
					break
				}
				if e != nil {
					return token{}, false, e
				}
				if rr == ';' {
					_ = t.r.UnreadRune()
					break
				}
				if rr == '(' || rr == ')' || unicode.IsSpace(rr) {
					_ = t.r.UnreadRune()
					break
				}
				b.WriteRune(rr)
			}
			s := b.String()
			if s == "" {
				continue
			}
			return token{tokAtom, s}, true, nil
		}
	}
}
