package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
)

const (
	// DefaultDockerfileName is the Default filename with Docker commands, read by docker build
	DefaultDockerfileName string = "Dockerfile"

	// HeaderSize is the size in bytes of a tar header
	HeaderSize = 512
)

// IsArchive checks for the magic bytes of a tar or any supported compression
// algorithm.
func IsArchive(header []byte) bool {
	compression := archive.DetectCompression(header)
	if compression != archive.Uncompressed {
		return true
	}
	r := tar.NewReader(bytes.NewBuffer(header))
	_, err := r.Next()
	return err == nil
}

// getContextFromReader will read the contents of the given reader as either a
// Dockerfile or tar archive. Returns a tar archive used as a context and a
// path to the Dockerfile inside the tar.
func getContextFromReader(r io.ReadCloser, dockerfileName string) (out io.ReadCloser, relDockerfile string, err error) {
	buf := bufio.NewReader(r)

	magic, err := buf.Peek(HeaderSize)
	if err != nil && err != io.EOF {
		return nil, "", errors.Errorf("failed to peek context header from STDIN: %v", err)
	}

	if IsArchive(magic) {
		return ioutils.NewReadCloserWrapper(buf, func() error { return r.Close() }), dockerfileName, nil
	}

	if dockerfileName == "-" {
		return nil, "", errors.New("build context is not an archive")
	}

	// Input should be read as a Dockerfile.
	tmpDir, err := ioutil.TempDir("", "docker-build-context-")
	if err != nil {
		return nil, "", errors.Errorf("unable to create temporary context directory: %v", err)
	}

	f, err := os.Create(filepath.Join(tmpDir, DefaultDockerfileName))
	if err != nil {
		return nil, "", err
	}
	_, err = io.Copy(f, buf)
	if err != nil {
		f.Close()
		return nil, "", err
	}

	if err := f.Close(); err != nil {
		return nil, "", err
	}
	if err := r.Close(); err != nil {
		return nil, "", err
	}

	tar, err := archive.Tar(tmpDir, archive.Uncompressed)
	if err != nil {
		return nil, "", err
	}

	return ioutils.NewReadCloserWrapper(tar, func() error {
		err := tar.Close()
		os.RemoveAll(tmpDir)
		return err
	}), DefaultDockerfileName, nil

}
