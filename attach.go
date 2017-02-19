package docker

// attachOpts := types.ContainerAttachOptions{
// 	Stream: true,
// 	Stdin:  e.Stdin != nil,
// 	Stdout: true,
// 	Stderr: true,
// 	Logs:   true,
// }

// resp, errAttach := client.ContainerAttach(
// 	e.context,
// 	container.ID,
// 	attachOpts,
// )
// if errAttach != nil && errAttach != httputil.ErrPersistEOF {
// 	// ContainerAttach returns an ErrPersistEOF (connection closed)
// 	// means server met an error and put it in Hijacked connection
// 	// keep the error and read detailed error message from hijacked connection later
// 	return errors.Wrap(errAttach, "cannot attach to container")
// }

// strm := &stream{
// 	stdin:  e.Stdin,
// 	stdout: e.Stdout,
// 	stderr: e.Stderr,
// }
// cErr := promise.Go(func() error {
// 	defer resp.Close()
// 	errHijack := holdHijackedConnection(
// 		e.context,
// 		strm,
// 		isTty,
// 		e.Stdin,
// 		e.Stdout,
// 		e.Stderr,
// 		resp,
// 	)
// 	if errHijack == nil {
// 		return errAttach
// 	}
// 	return errHijack
// })

// e.wc = cErr
