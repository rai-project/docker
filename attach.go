package docker

import (
	"net/http/httputil"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/rai-project/utils/promise"
)

func (c *Container) Attach() error {
	client := c.client
	attachOpts := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  client.options.stdin != nil,
		Stdout: client.options.stdout != nil,
		Stderr: client.options.stderr != nil,
		Logs:   true,
	}

	resp, errAttach := client.ContainerAttach(
		c.options.context,
		c.ID,
		attachOpts,
	)
	if errAttach != nil && errAttach != httputil.ErrPersistEOF {
		// ContainerAttach returns an ErrPersistEOF (connection closed)
		// means server met an error and put it in Hijacked connection
		// keep the error and read detailed error message from hijacked connection later
		return errors.Wrap(errAttach, "cannot attach to container")
	}
	if errAttach != nil {
		return errAttach
	}

	strm := &stream{
		stdin:  client.options.stdin,
		stdout: client.options.stdout,
		stderr: client.options.stderr,
	}
	cErr := promise.Go(func() error {
		defer resp.Close()
		errHijack := holdHijackedConnection(
			c.options.context,
			strm,
			c.options.containerConfig.Tty,
			client.options.stdin,
			client.options.stdout,
			client.options.stderr,
			resp,
		)
		if errHijack == nil {
			return errAttach
		}
		return errHijack
	})
	if cErr != nil {
		return <-cErr
	}
	return nil
}
