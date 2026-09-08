package xinvoice

// German document identifiers, written into the identity headers of the
// converted documents.
const (
	// CustomizationIDXRechnung identifies XRechnung 3.0 documents in
	// both syntaxes (BT-24).
	CustomizationIDXRechnung = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
	// GuidelineIDEN16931 identifies plain EN 16931 CII documents, the
	// guideline ZUGFeRD's EN 16931 profile uses.
	GuidelineIDEN16931 = "urn:cen.eu:en16931:2017"
	// ProfileIDPeppolBilling is the Peppol billing process identifier.
	ProfileIDPeppolBilling = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)
