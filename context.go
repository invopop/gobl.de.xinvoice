package xinvoice

import (
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/schema"
)

// KeyXRechnungUBL identifies XRechnung 3.0 in UBL syntax in the GOBL
// convert register.
const KeyXRechnungUBL cbc.Key = "ubl+de-xrechnung-v3"

// ContextXRechnungUBL is the gobl.ubl context for XRechnung 3.0 in UBL
// syntax. It is registered with gobl.ubl and the GOBL convert register
// when this package is imported.
var ContextXRechnungUBL = ubl.Context{
	Key:             KeyXRechnungUBL,
	Name:            i18n.NewString("UBL XRechnung 3"),
	Countries:       []l10n.Code{l10n.DE},
	Schemas:         []schema.ID{schema.Lookup(bill.Invoice{})},
	CustomizationID: CustomizationIDXRechnung,
	ProfileID:       ProfileIDPeppolBilling,
	Addons:          []cbc.Key{xrechnung.V3},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "de.xrechnung:ubl-invoice:3.0.2",
		CreditNote: "de.xrechnung:ubl-creditnote:3.0.2",
	},
}

func init() {
	ubl.RegisterContexts(ContextXRechnungUBL)
}
