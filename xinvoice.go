// Package xinvoice converts GOBL envelopes into the German electronic
// invoicing formats and back: XRechnung 3.0 in UBL and CII syntax, and
// ZUGFeRD's EN 16931 profile. The XML mapping is done by gobl.ubl and
// gobl.cii; this module owns the German contexts, formats, and the
// dispatch between the two syntaxes.
package xinvoice

import (
	"errors"
	"fmt"
	"slices"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

// Format keys for the supported German document formats.
const (
	// FormatXRechnungUBL is XRechnung 3.0 in UBL syntax.
	FormatXRechnungUBL cbc.Key = "xrechnung-ubl-v3"
	// FormatXRechnungCII is XRechnung 3.0 in CII syntax.
	FormatXRechnungCII cbc.Key = "xrechnung-cii-v3"
	// FormatZUGFeRD is the XML side of ZUGFeRD's EN 16931 profile. The
	// file name factur-x.xml is set by the ZUGFeRD / Factur-X
	// specification: readers locate the XML inside the PDF/A-3 by that
	// exact name.
	FormatZUGFeRD cbc.Key = "zugferd-v2"
)

// ErrUnsupportedFormat is returned when the format key is not one of
// the supported German formats.
var ErrUnsupportedFormat = errors.New("unsupported format")

// Format describes one supported German document format. Exactly one
// of the two context fields is set: it decides which conversion library
// handles the format.
type Format struct {
	Key      cbc.Key
	Name     string
	FileName string

	ublContext *ubl.Context
	ciiContext *cii.Context
}

// Addons returns the GOBL addon keys the format requires. The slice is
// a copy: mutating it does not change the format's requirements.
func (f *Format) Addons() []cbc.Key {
	if f.ublContext != nil {
		return slices.Clone(f.ublContext.Addons)
	}
	return slices.Clone(f.ciiContext.Addons)
}

// formats defines the supported German document formats.
var formats = []*Format{
	{
		Key:        FormatXRechnungUBL,
		Name:       "XRechnung UBL Invoice/CreditNote V3",
		FileName:   "xrechnung-ubl.xml",
		ublContext: &contextXRechnungUBL,
	},
	{
		Key:        FormatXRechnungCII,
		Name:       "XRechnung CII Invoice/CreditNote V3",
		FileName:   "xrechnung-cii.xml",
		ciiContext: &contextXRechnungCII,
	},
	{
		Key:        FormatZUGFeRD,
		Name:       "ZUGFeRD V2 (CII)",
		FileName:   "factur-x.xml",
		ciiContext: &contextZUGFeRD,
	},
}

// Formats returns the supported German document formats. The entries
// are copies: mutating them does not affect the package's registry.
func Formats() []*Format {
	out := make([]*Format, len(formats))
	for i, f := range formats {
		c := *f
		out[i] = &c
	}
	return out
}

// FormatFor returns a copy of the format with the given key, or nil.
func FormatFor(key cbc.Key) *Format {
	for _, f := range formats {
		if f.Key == key {
			c := *f
			return &c
		}
	}
	return nil
}

// Document carries the serialized XML and the descriptive values used
// to register or transmit it.
type Document struct {
	// Data is the XML document.
	Data []byte
	// Format is the format the document was generated in.
	Format *Format
	// Namespace and Element identify the XML root.
	Namespace string
	Element   string
	// CustomizationID and ProfileID identify the document flavor and
	// business process (CII: guideline and business process IDs).
	CustomizationID string
	ProfileID       string
	// Version is the syntax version (UBL "2.1", CII "D16B").
	Version string
	// VESID is the validation rule set the document must satisfy.
	VESID string
}

// Option configures a conversion.
type Option func(*options)

type options struct {
	attachments []BinaryAttachment
}

// WithAttachment embeds a file inside the generated XML as a binary
// attachment, such as the PDF rendition of the invoice.
func WithAttachment(a BinaryAttachment) Option {
	return func(o *options) {
		o.attachments = append(o.attachments, a)
	}
}

// Convert converts a GOBL envelope into the given German format. The
// envelope's invoice gains the format's addon when it does not declare
// it yet, so the German rules run before any mapping: an invoice that
// cannot satisfy the format fails here with the rule violations.
func Convert(env *gobl.Envelope, format cbc.Key, opts ...Option) (*Document, error) {
	f := FormatFor(format)
	if f == nil {
		return nil, fmt.Errorf("%w: '%s'", ErrUnsupportedFormat, format)
	}
	o := new(options)
	for _, opt := range opts {
		opt(o)
	}

	if env == nil {
		return nil, errors.New("nil envelope")
	}
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, errors.New("envelope does not contain an invoice")
	}
	if err := ensureAddons(env, inv, f.Addons()); err != nil {
		return nil, err
	}

	if f.ublContext != nil {
		return convertUBL(env, inv, f, o)
	}
	return convertCII(env, f, o)
}

// ensureAddons checks that the invoice declares all required addons,
// adds missing ones (recalculating so the addon's normalizations run),
// and validates the envelope. Validation always runs, also when the
// addons were already declared: an envelope built outside the platform
// can declare an addon and still violate its rules.
func ensureAddons(env *gobl.Envelope, inv *bill.Invoice, required []cbc.Key) error {
	var missing []cbc.Key
	existing := inv.GetAddons()
	for _, a := range required {
		if !a.In(existing...) {
			missing = append(missing, a)
		}
	}
	if len(missing) > 0 {
		inv.SetAddons(append(existing, missing...)...)
		if err := env.Calculate(); err != nil {
			return fmt.Errorf("calculating envelope with addons: %w", err)
		}
	}
	if err := env.Validate(); err != nil {
		return fmt.Errorf("validating envelope with addons: %w", err)
	}
	return nil
}

// convertUBL produces the XRechnung UBL document.
func convertUBL(env *gobl.Envelope, inv *bill.Invoice, f *Format, o *options) (*Document, error) {
	out, err := ubl.ConvertInvoice(env, ubl.WithContext(*f.ublContext))
	if err != nil {
		return nil, fmt.Errorf("converting to UBL: %w", err)
	}

	for _, a := range o.attachments {
		out.AddBinaryAttachment(ubl.BinaryAttachment{
			ID:          a.ID,
			Description: a.Description,
			Data:        a.Data,
			MimeCode:    a.MimeCode,
			Filename:    a.Filename,
		})
	}

	vesID := f.ublContext.VESIDs.Invoice
	if inv.GetType().Has(bill.InvoiceTypeCreditNote) {
		vesID = f.ublContext.VESIDs.CreditNote
	}

	data, err := ubl.Bytes(out)
	if err != nil {
		return nil, fmt.Errorf("serializing UBL document: %w", err)
	}

	return &Document{
		Data:            data,
		Format:          f,
		Namespace:       out.UBLNamespace,
		Element:         out.XMLName.Local,
		CustomizationID: f.ublContext.CustomizationID,
		ProfileID:       f.ublContext.ProfileID,
		Version:         ubl.Version,
		VESID:           vesID,
	}, nil
}

// convertCII produces the XRechnung CII or ZUGFeRD document.
func convertCII(env *gobl.Envelope, f *Format, o *options) (*Document, error) {
	out, err := cii.ConvertInvoice(env, cii.WithContext(*f.ciiContext))
	if err != nil {
		return nil, fmt.Errorf("converting to CII: %w", err)
	}

	for _, a := range o.attachments {
		out.AddBinaryAttachment(cii.BinaryAttachment{
			ID:          a.ID,
			Description: a.Description,
			Data:        a.Data,
			MimeCode:    a.MimeCode,
			Filename:    a.Filename,
		})
	}

	data, err := out.Bytes()
	if err != nil {
		return nil, fmt.Errorf("serializing CII document: %w", err)
	}

	return &Document{
		Data:            data,
		Format:          f,
		Namespace:       cii.NamespaceRSM,
		Element:         "CrossIndustryInvoice",
		CustomizationID: f.ciiContext.GuidelineID,
		ProfileID:       f.ciiContext.BusinessID,
		Version:         f.ciiContext.Version,
		VESID:           f.ciiContext.VESID,
	}, nil
}
