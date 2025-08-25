package gitconfig

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseEnvParameters parses GIT_CONFIG_PARAMETERS format into key-value pairs.
func ParseEnvParameters(params string) (map[string]string, error) {
	tokenizer := newEnvConfigParameterTokenizer(params)
	result := make(map[string]string)

	for keyToken := tokenizer.NextToken(); keyToken.Type != envConfigParameterTokenEOF; keyToken = tokenizer.NextToken() {
		if keyToken.Type != envConfigParameterTokenQuotedString {
			return nil, fmt.Errorf("expected quoted key: got %s", keyToken)
		}

		equalsToken := tokenizer.NextToken()
		if equalsToken.Type != envConfigParameterTokenEquals {
			return nil, fmt.Errorf("expected equals after key: got %s", equalsToken)
		}

		valueToken := tokenizer.NextToken()
		if valueToken.Type != envConfigParameterTokenQuotedString && valueToken.Type != envConfigParameterTokenEOF {
			return nil, fmt.Errorf("expected either no value or a quoted value after equal: got %s", valueToken)
		}

		result[keyToken.Value] = valueToken.Value
	}

	return result, nil
}

type envConfigParameterTokenType int

const (
	envConfigParameterTokenQuotedString envConfigParameterTokenType = iota + 1
	envConfigParameterTokenEquals
	envConfigParameterTokenEOF
	envConfigParameterTokenError
)

type envConfigParameterToken struct {
	Type  envConfigParameterTokenType
	Value string
	Pos   int
}

func (t envConfigParameterToken) String() string {
	var str string

	str += "at position " + strconv.Itoa(t.Pos) + ": "

	switch t.Type {
	case envConfigParameterTokenQuotedString:
		str += "<quoted string>"
	case envConfigParameterTokenEquals:
		str += "<equal sign>"
	case envConfigParameterTokenEOF:
		str += "<EOF>"
	case envConfigParameterTokenError:
		str += "<parsing error>"
	default:
		str += "<unknown type>"
	}

	if t.Type != envConfigParameterTokenEOF {
		str += " " + t.Value
	}

	return str
}

type envConfigParameterTokenizer struct {
	input string
	pos   int
	start int
}

func newEnvConfigParameterTokenizer(input string) *envConfigParameterTokenizer {
	return &envConfigParameterTokenizer{input: input}
}

func (t *envConfigParameterTokenizer) NextToken() envConfigParameterToken {
	for t.pos < len(t.input) && t.current() == ' ' {
		t.advance()
	}

	if t.reachedEOF() {
		return envConfigParameterToken{Type: envConfigParameterTokenEOF, Pos: t.pos}
	}

	t.start = t.pos

	switch t.current() {
	case '=':
		return t.handleEqual()
	case '\'':
		return t.handleQuotedString(t.start)
	default:
		return envConfigParameterToken{Type: envConfigParameterTokenError, Value: fmt.Sprintf("unexpected character: %c(ascii %d)", t.current(), t.current()), Pos: t.start}
	}
}

func (t *envConfigParameterTokenizer) current() byte {
	if t.pos >= len(t.input) {
		return 0
	}

	c := t.input[t.pos]

	return c
}

func (t *envConfigParameterTokenizer) peak() byte {
	if t.pos+1 >= len(t.input) {
		return 0
	}

	return t.input[t.pos+1]
}

func (t *envConfigParameterTokenizer) advance() { t.pos++ }

func (t *envConfigParameterTokenizer) reachedEOF() bool { return t.pos >= len(t.input) }

func (t *envConfigParameterTokenizer) handleEqual() envConfigParameterToken {
	if t.current() != '=' {
		return envConfigParameterToken{Type: envConfigParameterTokenError, Value: "expected equal sign", Pos: t.pos}
	}

	t.advance()

	return envConfigParameterToken{Type: envConfigParameterTokenEquals, Value: "=", Pos: t.start}
}

func (t *envConfigParameterTokenizer) handleQuotedString(start int) envConfigParameterToken {
	if t.current() != '\'' {
		return envConfigParameterToken{Type: envConfigParameterTokenError, Value: "expected single quote", Pos: start}
	}

	t.advance()

	var value strings.Builder
	for t.current() != '\'' && !t.reachedEOF() {
		value.WriteByte(t.current())
		t.advance()
	}

	if t.current() != '\'' {
		return envConfigParameterToken{Type: envConfigParameterTokenError, Value: "unterminated quoted string", Pos: start}
	}

	t.advance()

	if t.current() == '\\' && t.peak() == '\'' {
		t.advance()
		t.advance()

		token := t.handleQuotedString(t.pos)
		if token.Type != envConfigParameterTokenQuotedString {
			return envConfigParameterToken{Type: envConfigParameterTokenError, Value: fmt.Sprintf("%s: invalid quote inside quoted string", token), Pos: start}
		}

		value.WriteString("'" + token.Value)
	}

	return envConfigParameterToken{Type: envConfigParameterTokenQuotedString, Value: value.String(), Pos: start}
}
