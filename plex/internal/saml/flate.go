package saml

import (
	"compress/flate"
	"errors"
	"io"

	"userclouds.com/infra/ucerr"
)

const flateUncompressLimit = 10 * 1024 * 1024 // 10MB

func newSaferFlateReader(r io.Reader) io.ReadCloser {
	return &saferFlateReader{r: flate.NewReader(r)}
}

type saferFlateReader struct {
	r     io.ReadCloser
	count int
}

func (r *saferFlateReader) Read(p []byte) (n int, err error) {
	if r.count+len(p) > flateUncompressLimit {
		return 0, ucerr.Errorf("flate: uncompress limit exceeded (%d bytes)", flateUncompressLimit)
	}
	n, err = r.r.Read(p)
	r.count += n

	// Reader returns EOF when it has read all the data, but `ioutil.ReadAll`
	// hides io.EOF via an == check, not an errors.Is check. So we need to not
	// wrap it or we get unexpected behavior. (note that the Reader interface
	// explicitly says that returning (n, io.EOF) is equivalent to returning (n, nil)
	// as long as the subsequent call returns (0, io.EOF), so we do some extra checks here
	// to help diagnose unexpected errors
	if errors.Is(err, io.EOF) && n == 0 {
		return n, io.EOF // lint: ucerr-ignore
	} else if !errors.Is(err, io.EOF) {
		return n, ucerr.Wrap(err)
	}
	return n, nil
}

func (r *saferFlateReader) Close() error {
	return ucerr.Wrap(r.r.Close())
}
