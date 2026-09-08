package xinvoice

import (
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.de.xinvoice/addon/xrechnung"
	"github.com/invopop/gobl.de.xinvoice/addon/zugferd"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/cbc"
)

// German document identifiers. The XRechnung customization ID is the
// value receivers use to recognize the document as XRechnung (BT-24).
const (
	// CustomizationIDXRechnung identifies XRechnung 3.0 documents in
	// both syntaxes.
	CustomizationIDXRechnung = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
	// GuidelineIDEN16931 identifies plain EN 16931 CII documents, the
	// guideline ZUGFeRD's EN 16931 profile uses.
	GuidelineIDEN16931 = "urn:cen.eu:en16931:2017"
	// ProfileIDPeppolBilling is the Peppol billing process identifier.
	ProfileIDPeppolBilling = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

// The German conversion contexts live in this module so the base
// libraries stay free of country specifics. gobl.ubl and gobl.cii still
// carry their own German context values today; those are removed
// together with the deprecation of the German document types in the ubl
// and cii apps.
//
// The addon definitions live in this module's addon packages, which
// register themselves with GOBL when imported. GOBL core lists the keys
// as approved external addons in its addons/external.go.

// contextXRechnungUBL configures gobl.ubl for XRechnung 3.0 in UBL
// syntax.
var contextXRechnungUBL = ubl.Context{
	CustomizationID: CustomizationIDXRechnung,
	ProfileID:       ProfileIDPeppolBilling,
	Addons:          []cbc.Key{xrechnung.V3},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "de.xrechnung:ubl-invoice:3.0.2",
		CreditNote: "de.xrechnung:ubl-creditnote:3.0.2",
	},
}

// contextXRechnungCII configures gobl.cii for XRechnung 3.0 in CII
// syntax.
var contextXRechnungCII = cii.Context{
	GuidelineID: CustomizationIDXRechnung,
	BusinessID:  ProfileIDPeppolBilling,
	Version:     cii.VersionD16B,
	Addons:      []cbc.Key{xrechnung.V3},
	VESID:       "de.xrechnung:cii:3.0.2",
}

// contextZUGFeRD configures gobl.cii for ZUGFeRD's EN 16931 profile.
var contextZUGFeRD = cii.Context{
	GuidelineID: GuidelineIDEN16931,
	Version:     cii.VersionD16B,
	Addons:      []cbc.Key{zugferd.V2},
	VESID:       "de.zugferd:en16931:2.5.2",
}
