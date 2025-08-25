package gitconfig

import (
	"strings"
	"testing"

	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_ParseConfigParameters(t *testing.T) {
	for _, tc := range []struct {
		input                 string
		expectedResult        map[string]string
		expectedErrorContains string
	}{
		{
			input:          "",
			expectedResult: map[string]string{},
		},
		{
			input:          "    ",
			expectedResult: map[string]string{},
		},
		{
			input:          `'foo'=`,
			expectedResult: map[string]string{"foo": ""},
		},
		{
			input:          "'foo'='bar'",
			expectedResult: map[string]string{"foo": "bar"},
		},
		{
			input:          "'foo'='bar' 'bar'='foo'",
			expectedResult: map[string]string{"foo": "bar", "bar": "foo"},
		},
		{
			input:          "  'foo'='bar'   'bar'  =  'foo'   ",
			expectedResult: map[string]string{"foo": "bar", "bar": "foo"},
		},
		{
			input:          `'say'='hello '\'' world'`,
			expectedResult: map[string]string{"say": "hello ' world"},
		},
		{
			input:          `'key'='value'\''suffix'`,
			expectedResult: map[string]string{"key": "value'suffix"},
		},
		{
			input:          `'path'='/home/user'\''s file'`,
			expectedResult: map[string]string{"path": "/home/user's file"},
		},
		{
			input:          `'quote'=''\'''`,
			expectedResult: map[string]string{"quote": "'"},
		},
		{
			input:          `'multiple'='a'\''b'\''c'`,
			expectedResult: map[string]string{"multiple": "a'b'c"},
		},
		{
			input:          `'empty.middle'='start'\''end'`,
			expectedResult: map[string]string{"empty.middle": "start'end"},
		},
		{
			input:          `'complex'='He said: '\''Hello, world!'\'''`,
			expectedResult: map[string]string{"complex": "He said: 'Hello, world!'"},
		},
		{
			input:          `'section.subsection.key'='complex value with spaces'`,
			expectedResult: map[string]string{"section.subsection.key": "complex value with spaces"},
		},
		{
			input:          `'unicode'='café'\''s special chars: ñ, é, ü'`,
			expectedResult: map[string]string{"unicode": "café's special chars: ñ, é, ü"},
		},
		{
			input:          `'numbers'='123'\''456'`,
			expectedResult: map[string]string{"numbers": "123'456"},
		},
		{
			input:                 "f",
			expectedErrorContains: "expected quoted key: got at position 0: <parsing error> unexpected character: f",
		},
		{
			input:                 "='foo'='bar'",
			expectedErrorContains: "expected quoted key: got at position 0: <equal sign> =",
		},
		{
			input:                 "'foo",
			expectedErrorContains: "expected quoted key: got at position 0: <parsing error> unterminated quoted string",
		},
		{
			input:                 "'foo'",
			expectedErrorContains: "expected equals after key: got at position 5: <EOF>",
		},
		{
			input:                 "'foo'==",
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 6: <equal sign> =",
		},
		{
			input:                 "'foo'='",
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 6: <parsing error> unterminated quoted string",
		},
		{
			input:                 `'key'='value'\`,
			expectedErrorContains: "expected quoted key: got at position 13: <parsing error> unexpected character: \\",
		},
		{
			input:                 `'key'='value'\'missing'`,
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 6: <parsing error> at position 15: <parsing error> expected single quote: invalid quote inside quoted string",
		},
		{
			input:                 `'incomplete'='text'\'`,
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 13: <parsing error> at position 21: <parsing error> expected single quote: invalid quote inside quoted string",
		},
		{
			input:                 `'nested'='start'\'middle'unquoted`,
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 9: <parsing error> at position 18: <parsing error> expected single quote: invalid quote inside quoted string",
		},
		{
			input:                 `'malformed'=''\'`,
			expectedErrorContains: "expected either no value or a quoted value after equal: got at position 12: <parsing error> at position 16: <parsing error> expected single quote: invalid quote inside quoted string",
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParseEnvParameters(tc.input)
			if tc.expectedErrorContains != "" {
				test.Assert(t, err != nil && strings.Contains(err.Error(), tc.expectedErrorContains), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(check.Compare(t, result, tc.expectedResult))
			}
		})
	}
}

func Test_envConfigParameterToken_utilities(t *testing.T) {
	tokenizer := newEnvConfigParameterTokenizer("foo")

	test.Assert(t, !tokenizer.reachedEOF())

	test.Assert(t, tokenizer.current() == 'f')
	test.Assert(t, tokenizer.peak() == 'o')
	tokenizer.advance()
	test.Assert(t, tokenizer.current() == 'o')
	test.Assert(t, tokenizer.peak() == 'o')
	tokenizer.advance()
	test.Assert(t, tokenizer.current() == 'o')
	test.Assert(t, tokenizer.peak() == 0)

	test.Assert(t, !tokenizer.reachedEOF())
	tokenizer.advance()
	test.Assert(t, tokenizer.current() == 0)
	test.Assert(t, tokenizer.reachedEOF())
}
