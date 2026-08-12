# GOBL ➡️ German XRechnung and ZUGFeRD

German electronic invoicing formats for [GOBL](https://github.com/invopop/gobl):
XRechnung 3.0 in UBL and CII syntax, and the EN 16931 profile of ZUGFeRD.

Copyright [Invopop S.L.](https://invopop.com) 2026. Released publicly under the
[Apache License Version 2.0](LICENSE). For commercial licenses, please contact
the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to
this library, we require transferring copyrights to Invopop S.L.

[![Lint](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/lint.yaml/badge.svg)](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/lint.yaml)
[![Test Go](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/test.yaml/badge.svg)](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/invopop/gobl.de.xinvoice)](https://goreportcard.com/report/github.com/invopop/gobl.de.xinvoice)
[![codecov](https://codecov.io/gh/invopop/gobl.de.xinvoice/graph/badge.svg)](https://codecov.io/gh/invopop/gobl.de.xinvoice)
[![GoDoc](https://godoc.org/github.com/invopop/gobl.de.xinvoice?status.svg)](https://godoc.org/github.com/invopop/gobl.de.xinvoice)
![Latest Tag](https://img.shields.io/github/v/tag/invopop/gobl.de.xinvoice)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/invopop/gobl.de.xinvoice)

The XML mapping is done by [gobl.ubl](https://github.com/invopop/gobl.ubl) and
[gobl.cii](https://github.com/invopop/gobl.cii). This module owns the German
contexts and formats, the dispatch between the two syntaxes, and the
detection of the syntax when parsing, so the base libraries stay free of
anything German.

## Formats

| Key | Format | File name |
|---|---|---|
| `xrechnung-ubl-v3` | XRechnung 3.0, UBL syntax | `xrechnung-ubl.xml` |
| `xrechnung-cii-v3` | XRechnung 3.0, CII syntax | `xrechnung-cii.xml` |
| `zugferd-v2` | ZUGFeRD EN 16931 profile, CII syntax | `factur-x.xml` |

ZUGFeRD and France's Factur-X are the same specification published under two
national names. The ZUGFeRD file name is set by that specification: readers
locate the XML inside the PDF/A-3 by the exact name `factur-x.xml`. Embedding
the XML into the PDF is not part of this module; the gov-de app does that with
the factur-x CLI.

## Usage

### Convert GOBL to a German format

```go
package main

import (
	"encoding/json"
	"os"

	"github.com/invopop/gobl"
	xinvoice "github.com/invopop/gobl.de.xinvoice"
	"github.com/invopop/gobl/bill"
)

func main() {
	// The fixtures store the bare invoice document; wrap it in an
	// envelope. An existing envelope JSON unmarshals directly instead.
	data, err := os.ReadFile("./test/data/convert/invoice.json")
	if err != nil {
		panic(err)
	}

	inv := new(bill.Invoice)
	if err := json.Unmarshal(data, inv); err != nil {
		panic(err)
	}
	env := gobl.NewEnvelope()
	if err := env.Insert(inv); err != nil {
		panic(err)
	}

	doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL)
	if err != nil {
		// The error carries the BR-DE rule violations when the invoice
		// does not satisfy the format.
		panic(err)
	}

	// doc.Data is the XML; doc.VESID names the validation rule set;
	// doc.Format.FileName is the conventional file name.
	if err := os.WriteFile(doc.Format.FileName, doc.Data, 0644); err != nil {
		panic(err)
	}
}
```

The conversion adds the format's GOBL addon (`de-xrechnung-v3` or
`de-zugferd-v2`) to invoices that do not declare it, recalculating and
revalidating the envelope so the German rules run before any mapping.

To embed a file inside the generated XML, such as the PDF rendition of the
invoice, pass `xinvoice.WithAttachment`.

### Parse a German XML document

```go
package main

import (
	"encoding/json"
	"os"

	xinvoice "github.com/invopop/gobl.de.xinvoice"
)

func main() {
	data, err := os.ReadFile("./test/data/parse/invoice-zugferd-v2.xml")
	if err != nil {
		panic(err)
	}

	parsed, err := xinvoice.Parse(data)
	if err != nil {
		panic(err)
	}

	// parsed.Envelope is the GOBL envelope; parsed.Syntax says UBL or
	// CII; parsed.Attachments carries files embedded in the XML.
	out, err := json.MarshalIndent(parsed.Envelope, "", "  ")
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(out)
}
```

The syntax is detected from the document's root namespace. Both XRechnung
syntaxes and the XML inside ZUGFeRD PDFs are supported, including UBL credit
notes.

## Testing

The library uses testify. Run the tests with:

```bash
go test ./...
```

The golden files under `test/data/convert/out` and `test/data/parse/out` are
regenerated with:

```bash
go test ./... -update
```

## Considerations

- Mapping limitations are those of the base libraries; see
  [gobl.ubl](https://github.com/invopop/gobl.ubl) and
  [gobl.cii](https://github.com/invopop/gobl.cii) directly.
- The addon definitions live in this module's `addon/` packages and
  register themselves when imported. GOBL core lists the keys as approved
  external addons in its `addons/external.go`.
- `gobl.ubl` and `gobl.cii` still carry their own German context values.
  Those are removed together with the deprecation of the German document
  types in the ubl and cii apps.
- Schematron validation is not part of this module. The gov-de app validates
  every generated document against the KoSIT and FeRD rule sets (named by
  `Document.VESID`) through phorm before persisting it.

## References

### XRechnung

- [XStandard Einkauf](https://xeinkauf.de/): the XRechnung organization.
- [XRechnung 3.0.2 specification](https://xeinkauf.de/app/uploads/2024/07/302-XRechnung-2024-06-20.pdf)
  and its [English summary](https://xeinkauf.de/app/uploads/2024/10/XRechnung-EnglishSummary-v302.pdf).
- [eInvoicing in Germany](https://ec.europa.eu/digital-building-blocks/sites/display/DIGITAL/eInvoicing+in+Germany):
  the European Commission's country page.
- Submission portals for invoices to German federal authorities:
  [ZRE](https://xrechnung.bund.de/prod/authenticate.do) and
  [OZG-RE](https://xrechnung-bdr.de/edi/auth/login).

### ZUGFeRD

- [Forum elektronische Rechnung Deutschland](https://www.ferd-net.de/): the
  ZUGFeRD organization.
- [ZUGFeRD specification downloads](https://www.ferd-net.de/en/standards/zugferd).
