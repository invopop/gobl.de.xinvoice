package xinvoice_test

import (
	"testing"

	// Register the German addons so example documents declaring
	// de-xrechnung-v3 or de-zugferd-v2 normalize and validate.
	_ "github.com/invopop/gobl.de.xinvoice/addon/xrechnung"
	_ "github.com/invopop/gobl.de.xinvoice/addon/zugferd"

	"github.com/invopop/gobl/pkg/examples"
)

// TestExamples converts every document under examples/ to a calculated,
// validated JSON envelope and compares it against its golden output, using
// the shared GOBL example helpers. Run with -update to regenerate them.
func TestExamples(t *testing.T) {
	examples.Run(t, "examples", *update)
}
