// Package addon registers the German addons:
//
//   - de-xrechnung-v3 (XRechnung documents)
//   - de-zugferd-v2   (ZUGFeRD documents)
//
// Import it for its side effects to make both available without the
// converters, or import the individual subpackages directly.
package addon

import (
	// Register the addons.
	_ "github.com/invopop/gobl.de.xinvoice/addon/xrechnung"
	_ "github.com/invopop/gobl.de.xinvoice/addon/zugferd"
)
