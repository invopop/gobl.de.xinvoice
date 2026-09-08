package xinvoice

// German document identifiers. These are the values this module writes
// into the identity headers of the converted documents: UBL's
// cbc:CustomizationID and cbc:ProfileID, and CII's guideline and
// business process context parameters. The XRechnung customization ID
// is the value receivers use to recognize the document as XRechnung
// (BT-24).
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

// The base libraries convert with their default EN 16931 behavior and
// know nothing about Germany: this module ensures the German addon on
// the invoice before converting (the addon shapes the document content)
// and writes the identity headers onto the converted document after.
// The addon definitions still live in GOBL core (gobl/addons/de);
// moving them into this module is a planned follow-up.
