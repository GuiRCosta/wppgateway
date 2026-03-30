package antiban_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/guilhermecosta/wpp-gateway/internal/antiban"
)

func TestProcessSpintaxBasic(t *testing.T) {
	result := antiban.ProcessSpintax("{Olá|Oi|E aí}")
	assert.True(t, result == "Olá" || result == "Oi" || result == "E aí",
		"got: %s", result)
}

func TestProcessSpintaxMultipleGroups(t *testing.T) {
	text := "{Olá|Oi}, {João|Maria}!"

	for range 20 {
		result := antiban.ProcessSpintax(text)
		assert.Contains(t, result, "!")
		assert.True(t,
			strings.HasPrefix(result, "Olá") || strings.HasPrefix(result, "Oi"),
			"got: %s", result)
	}
}

func TestProcessSpintaxNoSpintax(t *testing.T) {
	text := "Hello world, no spintax here"
	assert.Equal(t, text, antiban.ProcessSpintax(text))
}

func TestProcessSpintaxPreservesVariables(t *testing.T) {
	text := "{Olá|Oi}, {{nome}}!"
	result := antiban.ProcessSpintax(text)
	assert.Contains(t, result, "{{nome}}")
}

func TestProcessSpintaxNested(t *testing.T) {
	text := "{a|{b|c}}"
	results := map[string]bool{}
	for range 50 {
		results[antiban.ProcessSpintax(text)] = true
	}
	// Should produce "a", "b", or "c"
	for k := range results {
		assert.True(t, k == "a" || k == "b" || k == "c", "unexpected: %s", k)
	}
}

func TestProcessSpintaxGeneratesVariation(t *testing.T) {
	text := "{a|b|c|d|e|f|g|h|i|j}"
	results := map[string]bool{}
	for range 50 {
		results[antiban.ProcessSpintax(text)] = true
	}
	assert.Greater(t, len(results), 1, "should produce more than one variation")
}

func TestProcessSpintaxEmpty(t *testing.T) {
	assert.Equal(t, "", antiban.ProcessSpintax(""))
}

func TestProcessSpintaxUnmatchedBrace(t *testing.T) {
	text := "hello {world"
	result := antiban.ProcessSpintax(text)
	assert.Equal(t, "hello {world", result)
}
