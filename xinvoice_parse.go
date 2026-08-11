package xinvoice

import (
	"errors"
	"fmt"

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
// a GOBL envelope. XRechnung UBL documents are recognized by their
// customization ID and converted with this module's context; other UBL
// flavors fall back to gobl.ubl's own context detection.
func Parse(data []byte, opts ...ParseOption) (*Parsed, error) {
	o := new(parseOptions)
	for _, opt := range opts {
		opt(o)
	}

	// UBL first: covers XRechnung UBL invoices and credit notes.
	if doc, err := ubl.Parse(data); err == nil {
		in, ok := doc.(*ubl.Invoice)
		if !ok {
			return nil, errors.New("unsupported UBL document type")
		}
		var ublOpts []ubl.Option
		if in.CustomizationID == CustomizationIDXRechnung {
			ublOpts = append(ublOpts, ubl.WithContext(ContextXRechnungUBL))
		}
		if o.from != "" && o.to != "" {
			ublOpts = append(ublOpts, ubl.WithRouting(o.from, o.to))
		}
		env, err := in.Convert(ublOpts...)
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

	// CII second: covers XRechnung CII and the XML inside ZUGFeRD PDFs.
	wire, err := cii.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownDocument, err.Error())
	}
	in, ok := wire.(*cii.Invoice)
	if !ok {
		return nil, errors.New("unsupported CII document type")
	}
	var ciiOpts []cii.ParseOption
	if o.from != "" && o.to != "" {
		ciiOpts = append(ciiOpts, cii.WithRouting(o.from, o.to))
	}
	env, err := cii.Parse(data, ciiOpts...)
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
