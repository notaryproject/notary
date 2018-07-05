package kivik

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/go-kivik/kivik/driver"
)

// Attachments is a collection of one or more file attachments.
type Attachments map[string]*Attachment

// MD5sum is a 128-bit MD5 checksum.
type MD5sum [16]byte

// Attachment represents a file attachment on a CouchDB document.
type Attachment struct {
	// Filiename is the name of the attachment.
	Filename string `json:"-"`

	// ContentType is the MIME type of the attachment contents.
	ContentType string `json:"content_type"`

	// Stub will be true if the data structure only represents file metadata,
	// and contains no actual content. Stub will be true when returned by the
	// GetAttachmentMeta function, or when included in a document without the
	// 'include_docs' option
	Stub bool `json:"stub"`

	// Content represents the attachment's content.
	//
	// Kivik will always return a non-nil Content, even for 0-byte attachments
	// or when Stub is true. It is the caller's responsibility to close
	// Content.
	Content io.ReadCloser `json:"-"`

	// Size records the uncompressed size of the attachment. The value -1
	// indicates that the length is unknown. Unless Stub is true, values >= 0
	// indicate that the given number of bytes may be read from Content.
	Size int64 `json:"length"`

	// Used compression codec, if any. Will be the empty string if the
	// attachment is uncompressed.
	ContentEncoding string `json:"encoding"`

	// EncodedLength records the compressed attachment size in bytes. Only
	// meaningful when ContentEncoding is defined.
	EncodedLength int64 `json:"encoded_length"`

	// RevPos is the revision number when attachment was added.
	RevPos int64 `json:"revpos"`

	// Digest is the content hash digest.
	Digest string `json:"digest"`
}

// bufCloser wraps a *bytes.Buffer to create an io.ReadCloser
type bufCloser struct {
	*bytes.Buffer
}

var _ io.ReadCloser = &bufCloser{}

func (b *bufCloser) Close() error { return nil }

// validate returns an error if the attachment is invalid.
func (a *Attachment) validate() error {
	if a.Filename == "" {
		return missingArg("filename")
	}
	return nil
}

type jsonAttachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

func readEncoder(in io.ReadCloser) io.ReadCloser {
	r, w := io.Pipe()
	enc := base64.NewEncoder(base64.StdEncoding, w)
	go func() {
		_, err := io.Copy(enc, in)
		_ = enc.Close()
		_ = w.CloseWithError(err)
	}()
	return r
}

// MarshalJSON satisfies the json.Marshaler interface.
func (a *Attachment) MarshalJSON() ([]byte, error) {
	r := readEncoder(a.Content)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	att := &jsonAttachment{
		ContentType: a.ContentType,
		Data:        string(data),
	}
	return json.Marshal(att)
}

// UnmarshalJSON implements the json.Unmarshaler interface for an Attachment.
func (a *Attachment) UnmarshalJSON(data []byte) error {
	type clone Attachment
	type jsonAtt struct {
		clone
		Data []byte `json:"data"`
	}
	var att jsonAtt
	if err := json.Unmarshal(data, &att); err != nil {
		return err
	}
	*a = Attachment(att.clone)
	if att.Data != nil {
		a.Content = ioutil.NopCloser(bytes.NewReader(att.Data))
	} else {
		a.Content = nilContent
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for a collection of
// Attachments.
func (a *Attachments) UnmarshalJSON(data []byte) error {
	atts := make(map[string]*Attachment)
	if err := json.Unmarshal(data, &atts); err != nil {
		return err
	}
	for filename, att := range atts {
		att.Filename = filename
	}
	*a = atts
	return nil
}

// AttachmentsIterator is an experimental way to read streamed attachments from
// a multi-part Get request.
type AttachmentsIterator struct {
	atti driver.Attachments
}

// Next returns the next attachment in the stream. io.EOF will be
// returned when there are no more attachments.
func (i *AttachmentsIterator) Next() (*Attachment, error) {
	att := new(driver.Attachment)
	if err := i.atti.Next(att); err != nil {
		return nil, err
	}
	return &Attachment{
		Filename:    att.Filename,
		ContentType: att.ContentType,
		Content:     att.Content,
		Size:        att.Size,
	}, nil
}
