package xinvoice_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	xinvoice "github.com/invopop/gobl.de.xinvoice"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"

// loadUBLEnvelope loads, calculates, and validates a GOBL envelope from
// test/data/ubl, with fixed IDs so the output is stable.
func loadUBLEnvelope(t *testing.T, name string) *gobl.Envelope {
	t.Helper()
	path := filepath.Join("test", "data", "ubl", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(data, env))
	env.Head.UUID = staticUUID
	if inv, ok := env.Extract().(*bill.Invoice); ok {
		inv.UUID = staticUUID
	}
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())
	if *update {
		data, err := json.MarshalIndent(env, "", "\t")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0644))
	}
	return env
}

// TestContextXRechnungUBLGolden converts each envelope in
// test/data/ubl/xrechnung with the gobl.ubl context and compares the XML
// against the golden files.
func TestContextXRechnungUBLGolden(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join("test", "data", "ubl", "xrechnung", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, examples)
	for _, example := range examples {
		name := filepath.Base(example)
		t.Run(name, func(t *testing.T) {
			env := loadUBLEnvelope(t, filepath.Join("xrechnung", name))
			doc, err := ubl.ConvertInvoice(env, ubl.WithContext(xinvoice.ContextXRechnungUBL))
			require.NoError(t, err)
			data, err := ubl.Bytes(doc)
			require.NoError(t, err)

			golden := filepath.Join("test", "data", "ubl", "xrechnung", "out", strings.Replace(name, ".json", ".xml", 1))
			if *update {
				require.NoError(t, os.WriteFile(golden, data, 0644))
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(data))
		})
	}
}

func TestContextXRechnungUBL(t *testing.T) {
	t.Run("identity headers", func(t *testing.T) {
		env := loadUBLEnvelope(t, "xrechnung/invoice-xr-minimal.json")
		doc, err := ubl.Convert(env, ubl.WithContext(xinvoice.ContextXRechnungUBL))
		require.NoError(t, err)
		out, ok := doc.(*ubl.Invoice)
		require.True(t, ok)
		assert.Equal(t, xinvoice.CustomizationIDXRechnung, out.CustomizationID)
		assert.Equal(t, xinvoice.ProfileIDPeppolBilling, out.ProfileID.Value)
	})

	t.Run("found by its identifiers", func(t *testing.T) {
		ctx := ubl.FindContext(xinvoice.CustomizationIDXRechnung, xinvoice.ProfileIDPeppolBilling)
		require.NotNil(t, ctx)
		assert.Equal(t, xinvoice.KeyXRechnungUBL, ctx.Key)
	})

	t.Run("VESIDs", func(t *testing.T) {
		env := loadUBLEnvelope(t, "xrechnung/invoice-xr-minimal.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, "de.xrechnung:ubl-invoice:3.0.2", xinvoice.ContextXRechnungUBL.GetVESID(inv))
	})
}

func TestConvertRegister(t *testing.T) {
	t.Run("detect and import", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("test", "data", "parse", "invoice-xrechnung-ubl-v3.xml"))
		require.NoError(t, err)
		ctx, err := convert.Detect(data)
		require.NoError(t, err)
		assert.Equal(t, xinvoice.KeyXRechnungUBL, ctx.Key)

		env, err := convert.Import(data)
		require.NoError(t, err)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Contains(t, inv.GetAddons(), xinvoice.ContextXRechnungUBL.Addons[0])
	})

	t.Run("export", func(t *testing.T) {
		env := loadUBLEnvelope(t, "xrechnung/invoice-xr-minimal.json")
		out, err := convert.Export(env, xinvoice.KeyXRechnungUBL, ubl.ContextEN16931.Key)
		require.NoError(t, err)
		assert.Equal(t, xinvoice.KeyXRechnungUBL, out.Context.Key)

		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, xinvoice.KeyXRechnungUBL, ctx.Key, "detected again")
	})
}
