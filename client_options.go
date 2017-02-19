package docker

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/docker/docker/cli/command"
	"github.com/docker/go-connections/tlsconfig"
)

type ClientOptions struct {
	host       string
	tlsConfig  *tls.Config
	apiVersion string
	stderr     *command.OutStream
	stdout     *command.OutStream
	stdin      *command.InStream
	context    context.Context
}

type ClientOption func(*ClientOptions)

func NewClientOptions() *ClientOptions {
	opts := &ClientOptions{
		host:       Config.Host,
		apiVersion: Config.APIVersion,
		stderr:     command.NewOutStream(ioutil.Discard),
		stdout:     command.NewOutStream(ioutil.Discard),
		stdin:      nil,
		context:    context.Background(),
	}
	if com.IsDir(Config.CertPath) {
		TLSConfig(Config.CertPath, Config.TLSVerify)(opts)
	}
	return opts
}

func Host(s string) ClientOption {
	return func(o *ClientOptions) {
		o.host = s
	}
}

func TLSConfig(certPath string, verify bool) ClientOption {
	return func(o *ClientOptions) {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(certPath, "ca.pem"),
			CertFile:           filepath.Join(certPath, "cert.pem"),
			KeyFile:            filepath.Join(certPath, "key.pem"),
			InsecureSkipVerify: verify,
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			log.WithError(err).
				WithField("cert_path", certPath).
				WithField("verify", verify).
				Error("Failed to create tls configuration")
			return
		}
		o.tlsConfig = tlsc
	}
}

func APIVersion(s string) ClientOption {
	return func(o *ClientOptions) {
		o.apiVersion = s
	}
}

func Stderr(stderr io.Writer) ClientOption {
	return func(o *ClientOptions) {
		o.stderr = command.NewOutStream(stderr)
	}
}

func Stdout(stdout io.Writer) ClientOption {
	return func(o *ClientOptions) {
		o.stdout = command.NewOutStream(stdout)
	}
}

func Stdin(stdin io.Reader) ClientOption {
	return func(o *ClientOptions) {
		o.stdin = command.NewInStream(ioutil.NopCloser(stdin))
	}
}
