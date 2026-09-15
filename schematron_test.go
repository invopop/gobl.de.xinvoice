package xinvoice_test

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	xinvoice "github.com/invopop/gobl.de.xinvoice"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/phorm"
	"github.com/stretchr/testify/require"
)

// phormURL points at the validation service; PHORM_URL overrides it.
var phormURL = "http://127.0.0.1:8080"

func init() {
	if u := os.Getenv("PHORM_URL"); u != "" {
		phormURL = u
	}
}

var validate = flag.Bool("validate", false, "validate converted documents against the German schematron via phorm")

// TestSchematron converts every fixture into every applicable format and
// validates each result against the real KoSIT and FeRD rule sets, which
// are the only definition of correctness the formats have: unit tests
// and golden files only pin what the converter already does.
//
// Off by default, since it needs a phorm service. Run it with -validate
// and phorm listening on phormURL; CI does that with a service container.
func TestSchematron(t *testing.T) {
	if !*validate {
		t.Skip("schematron validation is off; run with -validate and phorm on " + phormURL)
	}

	client := phorm.New(phormURL, "")
	waitForRules(t, client)

	cases := []struct {
		file    string
		formats []cbc.Key
	}{
		{"invoice.json", []cbc.Key{
			xinvoice.FormatXRechnungUBL,
			xinvoice.FormatXRechnungCII,
			xinvoice.FormatZUGFeRD,
		}},
		{"credit-note.json", []cbc.Key{
			xinvoice.FormatXRechnungUBL,
			xinvoice.FormatXRechnungCII,
		}},
	}

	for _, tc := range cases {
		for _, format := range tc.formats {
			t.Run(tc.file+" as "+format.String(), func(t *testing.T) {
				env := loadEnvelope(t, filepath.Join("test", "data", "convert", tc.file))
				doc, err := xinvoice.Convert(env, format)
				require.NoError(t, err)

				resp, err := client.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{
					Vesid:      doc.VESID,
					XmlContent: doc.Data,
				})
				require.NoError(t, err)

				if !resp.Success {
					results, err := json.MarshalIndent(resp.Results, "", "  ")
					require.NoError(t, err)
					t.Fatalf("not valid against %s:\n%s", doc.VESID, results)
				}
			})
		}
	}
}

// waitForRules blocks until phorm reports the German rule sets loaded. A
// service still starting up refuses the connection or serves an empty
// list, which reads exactly like a converter bug.
func waitForRules(t *testing.T, client *phorm.Client) {
	t.Helper()

	const want = "de.xrechnung:ubl-invoice:3.0.2"
	deadline := time.Now().Add(120 * time.Second)
	for {
		resp, err := client.ListVesIds(context.Background(), &phorm.ListVesIdsRequest{})
		if err == nil {
			for _, v := range resp.Vesids {
				if v.Vesid == want {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("phorm at %s never became ready: %v", phormURL, err)
		}
		time.Sleep(2 * time.Second)
	}
}
