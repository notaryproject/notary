package kivik

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
)

func TestAttachmentBytes(t *testing.T) {
	tests := []struct {
		name     string
		att      *Attachment
		expected string
		err      string
	}{
		{
			name:     "read success",
			att:      NewAttachment("test.txt", "text/plain", ioutil.NopCloser(strings.NewReader("test content"))),
			expected: "test content",
		},
		{
			name: "buffered read",
			att: func() *Attachment {
				att := NewAttachment("test.txt", "text/plain", ioutil.NopCloser(strings.NewReader("test content")))
				_, _ = att.Bytes()
				return att
			}(),
			expected: "test content",
		},
		{
			name: "read error",
			att:  NewAttachment("test.txt", "text/plain", errReader("read error")),
			err:  "read error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.att.Bytes()
			testy.Error(t, test.err, err)
			if d := diff.Text(test.expected, string(result)); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestAttachmentRead(t *testing.T) {
	tests := []struct {
		name     string
		input    Attachment
		expected string
		status   int
		err      string
	}{
		{
			name:   "nil reader",
			input:  Attachment{},
			status: StatusUnknownError,
			err:    "kivik: attachment content not read",
		},
		{
			name:     "reader set",
			input:    Attachment{ReadCloser: ioutil.NopCloser(strings.NewReader("foo"))},
			expected: "foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.input.Close() // nolint: errcheck
			result, err := ioutil.ReadAll(test.input)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Text(test.expected, string(result)); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestAttachmentMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		att      *Attachment
		expected string
		err      string
	}{
		{
			name: "foo.txt",
			att: &Attachment{
				ReadCloser:  ioutil.NopCloser(strings.NewReader("test attachment\n")),
				Filename:    "foo.txt",
				ContentType: "text/plain",
			},
			expected: `{
				"content_type": "text/plain",
				"data": "dGVzdCBhdHRhY2htZW50Cg=="
			}`,
		},
		{
			name: "read error",
			att: &Attachment{
				ReadCloser:  ioutil.NopCloser(&errorReader{}),
				Filename:    "foo.txt",
				ContentType: "text/plain",
			},
			err: "json: error calling MarshalJSON for type *kivik.Attachment: errorReader",
		},
	}
	for _, test := range tests {
		result, err := json.Marshal(test.att)
		testy.Error(t, test.err, err)
		if d := diff.JSON([]byte(test.expected), result); d != nil {
			t.Error(d)
		}
	}
}

func TestAttachmentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string

		body     string
		expected *Attachment
		err      string
	}{
		{
			name: "stub",
			input: `{
					"content_type": "text/plain",
					"stub": true
				}`,
			expected: &Attachment{
				ContentType: "text/plain",
				Stub:        true,
			},
		},
		{
			name: "simple",
			input: `{
					"content_type": "text/plain",
					"data": "dGVzdCBhdHRhY2htZW50Cg=="
				}`,
			body: "test attachment\n",
			expected: &Attachment{
				ContentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := new(Attachment)
			err := json.Unmarshal([]byte(test.input), result)
			testy.Error(t, test.err, err)
			var body []byte
			if result.ReadCloser != nil {
				body, err = ioutil.ReadAll(result)
				if err != nil {
					t.Fatal(err)
				}
				result.ReadCloser = nil
			}
			if d := diff.Text(test.body, string(body)); d != nil {
				t.Errorf("Unexpected body:\n%s", d)
			}
			if d := diff.Interface(test.expected, result); d != nil {
				t.Errorf("Unexpected result:\n%s", d)
			}
		})
	}
}

func TestAttachmentsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string

		expected Attachments
		err      string
	}{
		{
			name:     "no attachments",
			input:    "{}",
			expected: Attachments{},
		},
		{
			name: "one attachment",
			input: `{
				"foo.txt": {
					"content_type": "text/plain",
					"data": "dGVzdCBhdHRhY2htZW50Cg=="
				}
			}`,
			expected: Attachments{
				"foo.txt": &Attachment{
					Filename:    "foo.txt",
					ContentType: "text/plain",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var att Attachments
			err := json.Unmarshal([]byte(test.input), &att)
			testy.Error(t, test.err, err)
			for _, v := range att {
				v.ReadCloser = nil
			}
			if d := diff.Interface(test.expected, att); d != nil {
				t.Error(d)
			}
		})
	}
}
