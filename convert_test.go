package xinvoice_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl"
	xinvoice "github.com/invopop/gobl.de.xinvoice"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update the golden files")

// loadEnvelope builds a calculated and validated envelope from a bare
// invoice fixture.
func loadEnvelope(t *testing.T, path string) *gobl.Envelope {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	inv := new(bill.Invoice)
	require.NoError(t, json.Unmarshal(data, inv))
	env := gobl.NewEnvelope()
	require.NoError(t, env.Insert(inv))
	return env
}

// TestConvertGolden converts each fixture in test/data/convert into
// every applicable format and compares the XML against the golden files
// in test/data/convert/out. Run with -update to regenerate them.
func TestConvertGolden(t *testing.T) {
	cases := []struct {
		file    string
		formats []cbc.Key
	}{
		{
			file: "invoice.json",
			formats: []cbc.Key{
				xinvoice.FormatXRechnungUBL,
				xinvoice.FormatXRechnungCII,
				xinvoice.FormatZUGFeRD,
			},
		},
		{
			// ZUGFeRD credit notes are left out until a fixture
			// satisfying the profile's corrective rules is added.
			file: "credit-note.json",
			formats: []cbc.Key{
				xinvoice.FormatXRechnungUBL,
				xinvoice.FormatXRechnungCII,
			},
		},
	}

	for _, tc := range cases {
		for _, format := range tc.formats {
			name := fmt.Sprintf("%s to %s", tc.file, format)
			t.Run(name, func(t *testing.T) {
				env := loadEnvelope(t, filepath.Join("test", "data", "convert", tc.file))
				doc, err := xinvoice.Convert(env, format)
				require.NoError(t, err)

				base := tc.file[:len(tc.file)-len(".json")]
				golden := filepath.Join("test", "data", "convert", "out",
					fmt.Sprintf("%s-%s.xml", base, format))
				if *update {
					require.NoError(t, os.WriteFile(golden, doc.Data, 0644))
					return
				}
				want, err := os.ReadFile(golden)
				require.NoError(t, err, "golden file missing, run with -update")
				assert.Equal(t, string(want), string(doc.Data))
			})
		}
	}
}

func TestConvertDocumentMetadata(t *testing.T) {
	// Each subtest loads its own envelope: Convert adds the format's
	// addon to the invoice, so sharing one would order-couple them.
	invoicePath := filepath.Join("test", "data", "convert", "invoice.json")

	t.Run("XRechnung UBL", func(t *testing.T) {
		env := loadEnvelope(t, invoicePath)
		doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL)
		require.NoError(t, err)
		assert.Equal(t, "Invoice", doc.Element)
		assert.Equal(t, xinvoice.CustomizationIDXRechnung, doc.CustomizationID)
		assert.Equal(t, "de.xrechnung:ubl-invoice:3.0.2", doc.VESID)
		assert.Equal(t, "xrechnung-ubl.xml", doc.Format.FileName)
	})

	t.Run("XRechnung CII", func(t *testing.T) {
		env := loadEnvelope(t, invoicePath)
		doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungCII)
		require.NoError(t, err)
		assert.Equal(t, "CrossIndustryInvoice", doc.Element)
		assert.Equal(t, "de.xrechnung:cii:3.0.2", doc.VESID)
	})

	t.Run("ZUGFeRD", func(t *testing.T) {
		env := loadEnvelope(t, invoicePath)
		doc, err := xinvoice.Convert(env, xinvoice.FormatZUGFeRD)
		require.NoError(t, err)
		assert.Equal(t, "de.zugferd:en16931:2.4", doc.VESID)
		assert.Equal(t, "factur-x.xml", doc.Format.FileName)
	})

	t.Run("credit note switches rule set", func(t *testing.T) {
		env := loadEnvelope(t, filepath.Join("test", "data", "convert", "credit-note.json"))
		doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL)
		require.NoError(t, err)
		assert.Equal(t, "CreditNote", doc.Element)
		assert.Equal(t, "de.xrechnung:ubl-creditnote:3.0.2", doc.VESID)
	})

	t.Run("unknown format", func(t *testing.T) {
		env := loadEnvelope(t, invoicePath)
		_, err := xinvoice.Convert(env, "peppol-bis")
		assert.ErrorIs(t, err, xinvoice.ErrUnsupportedFormat)
	})
}

func TestConvertWithAttachment(t *testing.T) {
	env := loadEnvelope(t, filepath.Join("test", "data", "convert", "invoice.json"))
	doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL,
		xinvoice.WithAttachment(xinvoice.BinaryAttachment{
			ID:       "att-1",
			Data:     []byte("%PDF-1.7 fake"),
			MimeCode: "application/pdf",
			Filename: "invoice.pdf",
		}))
	require.NoError(t, err)
	assert.Contains(t, string(doc.Data), "EmbeddedDocumentBinaryObject")
	assert.Contains(t, string(doc.Data), "invoice.pdf")
}

func TestConvertIncompleteInvoice(t *testing.T) {
	env := loadEnvelope(t, filepath.Join("test", "data", "convert", "invoice.json"))
	inv := env.Extract().(*bill.Invoice)
	inv.Payment = nil // BR-DE-1
	require.NoError(t, env.Calculate())

	_, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BR-DE-1")
}
