package xinvoice

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

// Syntax identifies which of the two German XML syntaxes a parsed
// document uses.
type Syntax string

// The two syntaxes German invoices arrive in.
const (
	SyntaxUBL Syntax = "ubl"
	SyntaxCII Syntax = "cii"
)

// ErrUnknownDocument is returned when the data is neither a UBL nor a
// CII invoice.
var ErrUnknownDocument = errors.New("not a recognized UBL or CII invoice")

// Parsed carries the outcome of parsing a German XML document.
type Parsed struct {
	// Envelope contains the GOBL invoice.
	Envelope *gobl.Envelope
	// Attachments are the binary attachments embedded in the XML.
	Attachments []BinaryAttachment
	// Syntax the document was written in.
	Syntax Syntax
}

// ParseOption configures parsing.
type ParseOption func(*parseOptions)

type parseOptions struct {
	from, to cbc.URI
}

// WithRouting records the transport direction (sender and receiver
// participant URIs) on the parsed envelope's header. Both must be set
// for the option to take effect.
func WithRouting(from, to cbc.URI) ParseOption {
	return func(o *parseOptions) {
		o.from = from
		o.to = to
	}
}

// Parse reads a German XML invoice in either syntax and converts it to
// a GOBL envelope. The syntax is decided by the document's root
// namespace, so a failure inside one syntax reports that syntax's own
// error. XRechnung documents are recognized by their customization ID
// and get the German addon stamped from this module's identity values;
// other flavors keep the base library's own detection.
func Parse(data []byte, opts ...ParseOption) (*Parsed, error) {
	o := new(parseOptions)
	for _, opt := range opts {
		opt(o)
	}

	ns, err := rootNamespace(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnknownDocument, err)
	}

	switch ns {
	case ubl.NamespaceUBLInvoice, ubl.NamespaceUBLCreditNote:
		return parseUBL(data, o)
	case cii.NamespaceRSM:
		return parseCII(data, o)
	default:
		return nil, fmt.Errorf("%w: unexpected root namespace %q", ErrUnknownDocument, ns)
	}
}

// parseUBL covers XRechnung UBL invoices and credit notes.
func parseUBL(data []byte, o *parseOptions) (*Parsed, error) {
	doc, err := ubl.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing UBL document: %w", err)
	}
	in, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil, fmt.Errorf("unsupported UBL document type %T", doc)
	}
	// Pin the plain EN 16931 context, like gobl.dk.oioubl does: this
	// overrides gobl.ubl's detection of its own German context, so the
	// parse behaves the same before and after that context is removed.
	opts := []ubl.Option{ubl.WithContext(ubl.ContextEN16931)}
	if o.from != "" && o.to != "" {
		opts = append(opts, ubl.WithRouting(o.from, o.to))
	}
	env, err := in.Convert(opts...)
	if err != nil {
		return nil, fmt.Errorf("converting UBL document to GOBL: %w", err)
	}
	// gobl.ubl stamps the addon while it still carries a German context.
	// Stamp it from this module's own identity values too, so parsing
	// keeps producing the same envelopes when the base library loses it.
	if in.CustomizationID == CustomizationIDXRechnung {
		if err := stampAddons(env, []cbc.Key{xrechnung.V3}); err != nil {
			return nil, fmt.Errorf("applying German addon to parsed document: %w", err)
		}
	}
	out := &Parsed{Envelope: env, Syntax: SyntaxUBL}
	for _, ba := range in.ExtractBinaryAttachments() {
		out.Attachments = append(out.Attachments, BinaryAttachment{
			ID:          ba.ID,
			Description: ba.Description,
			Data:        ba.Data,
			MimeCode:    ba.MimeCode,
			Filename:    ba.Filename,
		})
	}
	return out, nil
}

// parseCII covers XRechnung CII and the XML inside ZUGFeRD PDFs. The
// data is deliberately processed twice: gobl.cii's Parse goes straight
// to the envelope and offers no wire document to extract attachments
// from, so Unmarshal provides that separately.
func parseCII(data []byte, o *parseOptions) (*Parsed, error) {
	wire, err := cii.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("parsing CII document: %w", err)
	}
	in, ok := wire.(*cii.Invoice)
	if !ok {
		return nil, fmt.Errorf("unsupported CII document type %T", wire)
	}
	var opts []cii.ParseOption
	if o.from != "" && o.to != "" {
		opts = append(opts, cii.WithRouting(o.from, o.to))
	}
	env, err := ciiParse(data, opts)
	if err != nil {
		return nil, fmt.Errorf("converting CII document to GOBL: %w", err)
	}
	// Same as parseUBL: stamp the German addon from this module's own
	// identity values. Only the XRechnung guideline is unambiguously
	// German; the generic EN 16931 guideline (which ZUGFeRD's EN 16931
	// profile shares with plain EN 16931 documents) is left to gobl.cii's
	// own detection.
	if in.ExchangedContext != nil && in.ExchangedContext.GuidelineContext != nil &&
		in.ExchangedContext.GuidelineContext.ID == CustomizationIDXRechnung {
		if err := stampAddons(env, []cbc.Key{xrechnung.V3}); err != nil {
			return nil, fmt.Errorf("applying German addon to parsed document: %w", err)
		}
	}
	out := &Parsed{Envelope: env, Syntax: SyntaxCII}
	for _, ba := range in.ExtractBinaryAttachments() {
		out.Attachments = append(out.Attachments, BinaryAttachment{
			ID:          ba.ID,
			Description: ba.Description,
			Data:        ba.Data,
			MimeCode:    ba.MimeCode,
			Filename:    ba.Filename,
		})
	}
	return out, nil
}

// stampAddons adds the missing addons to the envelope's invoice and
// recalculates. Unlike conversion, parsing does not validate: a received
// document that violates the addon's rules must still produce an
// envelope the caller can inspect.
func stampAddons(env *gobl.Envelope, required []cbc.Key) error {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil
	}
	var missing []cbc.Key
	existing := inv.GetAddons()
	for _, a := range required {
		if !a.In(existing...) {
			missing = append(missing, a)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	inv.SetAddons(append(existing, missing...)...)
	return env.Calculate()
}

// ciiParse wraps gobl.cii's Parse, turning panics into errors: gobl.cii
// dereferences required elements (such as the settlement) without
// checking a structurally incomplete document actually carries them,
// and Parse handles external data.
func ciiParse(data []byte, opts []cii.ParseOption) (env *gobl.Envelope, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("incomplete document: %v", r)
		}
	}()
	return cii.Parse(data, opts...)
}

// rootNamespace returns the namespace of the document's root element.
func rootNamespace(data []byte) (string, error) {
	dc := xml.NewDecoder(bytes.NewReader(data))
	for {
		tk, err := dc.Token()
		if err == io.EOF {
			return "", errors.New("no root element found")
		}
		if err != nil {
			return "", fmt.Errorf("reading XML: %w", err)
		}
		if t, ok := tk.(xml.StartElement); ok {
			return t.Name.Space, nil
		}
	}
}
