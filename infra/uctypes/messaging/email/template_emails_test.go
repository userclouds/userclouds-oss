package email_test

import (
	"bytes"
	"context"
	"html/template"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uctypes/messaging/email"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/uctest"
)

type data struct {
	Link       template.HTML
	UnsafeLink string
	Safe       template.HTML
	Unsafe     string
}

// NB: when this test starts to fail, it might mean that https://github.com/golang/go/issues/63586 has been
// fixed and we can remove our weird <a href=""> workaround in the email templates
// (this is adapted from https://play.golang.com/p/8Gm51YLf8gk which I wrote for that issue)
func TestHTMLTemplateEncodingBug(t *testing.T) {
	var tmp *template.Template
	var buf *bytes.Buffer
	var err error

	// encoding outside of an anchor tag works as expected
	simpleTemplate := `<h1>{{ .Link }}</h1>`
	tmp, err = template.New("test").Parse(simpleTemplate)
	assert.NoErr(t, err)

	buf = &bytes.Buffer{}
	assert.NoErr(t, tmp.Execute(buf, data{Link: "https://example.org?foo=bar&baz=quz"}))
	assert.Equal(t, buf.String(), `<h1>https://example.org?foo=bar&baz=quz</h1>`)

	// wrapping this link in an anchor tag fails because it gets encoded
	tt := `click <a href="{{ .Link }}">here</a>`
	tmp, err = template.New("test").Parse(tt)
	assert.NoErr(t, err)

	buf = &bytes.Buffer{}
	assert.NoErr(t, tmp.Execute(buf, data{Link: "https://example.org?foo=bar&baz=quz"}))
	workaroundTemplate := `click {{ .Link }}here</a>`
	tmp, err = template.New("test").Parse(workaroundTemplate)
	assert.NoErr(t, err)

	buf = &bytes.Buffer{}
	assert.NoErr(t, tmp.Execute(buf, data{Link: `<a href="https://example.org?foo=bar&baz=quz">`}))
	assert.Equal(t, buf.String(), `click <a href="https://example.org?foo=bar&baz=quz">here</a>`)
}

func TestHTMLEncoding(t *testing.T) {
	ctx := context.Background()

	c := &uctest.EmailClient{}
	to := "bobby@tables.org"

	getter := func(et message.ElementType) string {
		switch et {
		case message.EmailSender:
			return "me@me.com"
		case message.EmailSenderName:
			return "Me"
		case message.EmailSubjectTemplate:
			return "Test"
		case message.EmailTextTemplate:
			return "Text"
		case message.EmailHTMLTemplate:
			return `<h1>{{ .Safe }} @ {{ .Unsafe }} @ {{ .Link }} @ {{ .UnsafeLink }}</h1>`
		default:
			return ""
		}
	}

	d := data{
		Link:       "https://example.com?foo=bar&baz=qux",
		UnsafeLink: "https://example.com?foo=bar&baz=qux",
		Safe:       "<script>alert('hello')</script>",
		Unsafe:     "<script>alert('hello')</script>",
	}

	assert.NoErr(t, email.SendWithHTMLTemplate(ctx, c, to, getter, d))
	assert.Equal(t, c.HTMLBodies[0], `<h1><script>alert('hello')</script> @ &lt;script&gt;alert(&#39;hello&#39;)&lt;/script&gt; @ https://example.com?foo=bar&baz=qux @ https://example.com?foo=bar&amp;baz=qux</h1>`)
}
