package xinvoice_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xinvoice "github.com/invopop/gobl.de.xinvoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGolden parses every XML document in test/data/parse and
// compares the resulting GOBL invoice against the golden files in
// test/data/parse/out. Run with -update to regenerate them.
func TestParseGolden(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("test", "data", "parse", "*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no parse fixtures found")

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			require.NoError(t, err)

			parsed, err := xinvoice.Parse(data)
			require.NoError(t, err)

			// Compare the document only: the envelope header carries a
			// fresh UUID and digest on every run.
			got, err := json.MarshalIndent(parsed.Envelope.Extract(), "", "\t")
			require.NoError(t, err)
			got = normalizeUUID(t, got)

			golden := strings.TrimSuffix(file, ".xml") + ".json"
			golden = filepath.Join("test", "data", "parse", "out", filepath.Base(golden))
			if *update {
				require.NoError(t, os.WriteFile(golden, append(got, '\n'), 0644))
				return
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err, "golden file missing, run with -update")
			assert.JSONEq(t, string(want), string(got))
		})
	}
}

// normalizeUUID stamps a zero UUID on the document: the wire formats
// carry none, so parsing assigns a fresh one on every run.
func normalizeUUID(t *testing.T, doc []byte) []byte {
	t.Helper()
	raw := map[string]any{}
	require.NoError(t, json.Unmarshal(doc, &raw))
	if _, ok := raw["uuid"]; ok {
		raw["uuid"] = "00000000-0000-0000-0000-000000000000"
	}
	out, err := json.MarshalIndent(raw, "", "\t")
	require.NoError(t, err)
	return out
}

func TestParseSyntaxDetection(t *testing.T) {
	t.Run("UBL", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("test", "data", "parse", "invoice-xrechnung-ubl-v3.xml"))
		require.NoError(t, err)
		parsed, err := xinvoice.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, xinvoice.SyntaxUBL, parsed.Syntax)
	})

	t.Run("CII", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("test", "data", "parse", "invoice-zugferd-v2.xml"))
		require.NoError(t, err)
		parsed, err := xinvoice.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, xinvoice.SyntaxCII, parsed.Syntax)
	})

	t.Run("garbage", func(t *testing.T) {
		_, err := xinvoice.Parse([]byte("<html><body>not an invoice</body></html>"))
		assert.ErrorIs(t, err, xinvoice.ErrUnknownDocument)
	})
}
