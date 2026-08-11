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
// error. XRechnung UBL documents are recognized by their customization
// ID and converted with this module's context; other UBL flavors fall
// back to gobl.ubl's own context detection.
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
	var opts []ubl.Option
	if in.CustomizationID == CustomizationIDXRechnung {
		opts = append(opts, ubl.WithContext(ContextXRechnungUBL))
	}
	if o.from != "" && o.to != "" {
		opts = append(opts, ubl.WithRouting(o.from, o.to))
	}
	env, err := in.Convert(opts...)
	if err != nil {
		return nil, fmt.Errorf("converting UBL document to GOBL: %w", err)
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
	env, err := cii.Parse(data, opts...)
	if err != nil {
		return nil, fmt.Errorf("converting CII document to GOBL: %w", err)
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
