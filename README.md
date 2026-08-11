# GOBL ➡️ German XRechnung and ZUGFeRD

German electronic invoicing formats for [GOBL](https://github.com/invopop/gobl):
XRechnung 3.0 in UBL and CII syntax, and the EN 16931 profile of ZUGFeRD.

Released under the Apache 2.0 [LICENSE](https://github.com/invopop/gobl.de.xinvoice/blob/main/LICENSE), Copyright 2026 [Invopop S.L.](https://invopop.com).

[![Lint](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/lint.yaml/badge.svg)](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/lint.yaml)
[![Test Go](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/test.yaml/badge.svg)](https://github.com/invopop/gobl.de.xinvoice/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/invopop/gobl.de.xinvoice)](https://goreportcard.com/report/github.com/invopop/gobl.de.xinvoice)
[![GoDoc](https://godoc.org/github.com/invopop/gobl.de.xinvoice?status.svg)](https://godoc.org/github.com/invopop/gobl.de.xinvoice)
![Latest Tag](https://img.shields.io/github/v/tag/invopop/gobl.de.xinvoice)

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

The ZUGFeRD file name is set by the ZUGFeRD / Factur-X specification: readers
locate the XML inside the PDF/A-3 by the exact name `factur-x.xml`. Embedding
the XML into the PDF is not part of this module; the gov-de app does that with
the factur-x CLI.

## Usage

Convert a GOBL envelope into a German format:

```go
doc, err := xinvoice.Convert(env, xinvoice.FormatXRechnungUBL)
if err != nil {
    // The error carries the BR-DE rule violations when the invoice
    // does not satisfy the format.
}
// doc.Data is the XML; doc.VESID names the validation rule set.
```

Parse a German XML document in either syntax:

```go
parsed, err := xinvoice.Parse(data)
if err != nil {
    // ...
}
// parsed.Envelope is the GOBL envelope; parsed.Syntax says UBL or CII;
// parsed.Attachments carries files embedded in the XML.
```

The conversion adds the format's GOBL addon (`de-xrechnung-v3` or
`de-zugferd-v2`) to invoices that do not declare it, recalculating and
revalidating the envelope so the German rules run before any mapping.

## Notes

- The addon definitions still live in GOBL core (`gobl/addons/de`). Moving
  them into this module with the external addon registration in GOBL's
  `addons/external.go` is a planned follow-up.
- `gobl.ubl` and `gobl.cii` still carry their own German context values.
  Those are removed together with the deprecation of the German document
  types in the ubl and cii apps.
- Schematron validation is not part of this module. The gov-de app validates
  every generated document against the KoSIT and FeRD rule sets (named by
  `Document.VESID`) through phorm before persisting it.

## Development

The golden files under `test/data/convert/out` and `test/data/parse/out` are
regenerated with:

```bash
go test ./... -update
```
