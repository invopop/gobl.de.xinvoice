package xinvoice

// BinaryAttachment represents a file embedded inside the XML document,
// unifying the attachment shape of the two conversion libraries.
type BinaryAttachment struct {
	// ID is the identifier for this attachment reference.
	ID string
	// Description provides a human-readable description.
	Description string
	// Data contains the raw binary data.
	Data []byte
	// MimeCode specifies the MIME type (e.g. "application/pdf").
	MimeCode string
	// Filename is the name of the file.
	Filename string
}
