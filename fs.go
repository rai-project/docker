package docker

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"

	"bitbucket.org/hwuligans/rai/pkg/amazon"
	"bitbucket.org/hwuligans/rai/pkg/archive"
	"bitbucket.org/hwuligans/rai/pkg/config"
)

func (c *Container) UploadToContainer(targetPath string, sourcePath string) error {

	if err := c.checkIsRunning(); err != nil {
		return err
	}

	f, err := os.Open(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open archive while uploading to container: %s", sourcePath)
	}
	defer f.Close()

	tr, err := os.Open(sourcePath)
	if err != nil {
		log.WithError(err).
			WithField("filename", sourcePath).
			Error("opening the file failed while trying to upload to container")
		return errors.Wrapf(err,
			"opening the file failed while trying to upload to container file = %s", sourcePath)
	}

	client := c.client
	err = client.UploadToContainer(c.ID, docker.UploadToContainerOptions{
		InputStream:          tr,
		Path:                 targetPath,
		NoOverwriteDirNonDir: true,
	})
	if err != nil {
		err = Error(err)
		msg := fmt.Sprintf("Failed to upload dir = %s to container at %s",
			sourcePath, targetPath)
		log.WithError(err).
			WithField("source", sourcePath).
			WithField("target", targetPath).
			Error(msg)
		return errors.Wrapf(err, msg)
	}
	log.WithField("source", sourcePath).
		WithField("target", targetPath).
		Debug("uploaded to container successfully")
	return nil
}

func (c *Container) UploadToContainerFromS3(targetPath string, key string) error {
	s3, err := amazon.NewS3("")
	if err != nil {
		msg := "Failed to upload s3 to container due to not " +
			"being able to create an s3 connection"
		log.WithError(err).Error(msg)
		return errors.Wrapf(err, msg)
	}

	tmpDir, err := ioutil.TempDir("", config.App.Name)
	if err != nil {
		msg := "Failed to create temporary directory for s3 to upload to container"
		log.WithError(err).
			WithField("temp_dir", tmpDir).
			Error(msg)
		return errors.Wrap(err, msg)
	}
	defer os.RemoveAll(tmpDir)

	var fileBaseName string
	if u, err := url.Parse(key); err != nil {
		fileBaseName = filepath.Base(u.Path)
	} else {
		fileBaseName = filepath.Base(key)
	}
	fileName := filepath.Join(tmpDir, fileBaseName)
	log.WithField("key", key).
		WithField("fileName", fileName).
		WithField("targetPath", targetPath).
		Debug("Downloading from S3")

	err = s3.Download(fileName, key)
	if err != nil {
		msg := fmt.Sprintf("Failed to upload s3 to container due to not "+
			"be able to download s3 with key=%v", key)
		log.WithError(err).Error(msg)
		return errors.Wrapf(err, msg)
	}

	return c.UploadToContainer(targetPath, fileName)
}

func (c *Container) DownloadFromContainer(targetPath string, sourcePath string) error {

	if err := c.checkIsRunning(); err != nil {
		return err
	}

	f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		msg := "Failed to download from container because was not able to open the targetPath file"
		log.WithError(err).
			WithField("source_path", sourcePath).
			WithField("target_path", targetPath).
			Error(msg)
		return errors.Wrap(err, msg)
	}
	defer f.Close()

	client := c.client

	err = client.DownloadFromContainer(c.ID, docker.DownloadFromContainerOptions{
		OutputStream: f,
		Path:         sourcePath,
	})

	if err != nil {
		err = Error(err)
		msg := fmt.Sprintf("Failed to download dir = %s from container to the directory %s on the host",
			sourcePath, targetPath)
		log.WithError(err).
			WithField("source", sourcePath).
			WithField("target", targetPath).
			Error(msg)
		return errors.Wrapf(err, msg)
	}
	log.WithField("source", sourcePath).
		WithField("target", targetPath).
		Debug("downloaded from container successfully")
	return nil
}

func (c *Container) DownloadFromContainerToS3(sourcePath string) (string, error) {
	s3, err := amazon.NewS3("")
	if err != nil {
		msg := "Failed to download from container to s3 due to not " +
			"being able to create an s3 connection"
		log.WithError(err).Error(msg)
		return "", errors.Wrap(err, msg)
	}

	tmpDir, err := ioutil.TempDir("", config.App.Name)
	if err != nil {
		msg := "Failed to create temporary directory for s3 to download from container"
		log.WithError(err).
			WithField("source_path", sourcePath).
			WithField("temp_dir", tmpDir).
			Error(msg)
		return "", errors.Wrap(err, msg)
	}
	defer os.RemoveAll(tmpDir)

	fileName := filepath.Join(tmpDir, c.ID+archive.FileExtension)

	err = c.DownloadFromContainer(fileName, sourcePath)
	if err != nil {
		msg := "Failed to download from container to s3."
		log.WithError(err).Error(msg)
		return "", errors.Wrap(err, msg)
	}

	key, err := s3.Upload(fileName, false)
	if err != nil {
		msg := "Failed to upload from container to s3 to due to not " +
			"be able to upload s3"
		log.WithError(err).Error(msg)
		return "", errors.Wrap(err, msg)
	}
	return key, nil
}

func (c *Container) checkIsRunning() error {
	client := c.client
	container, err := client.InspectContainer(c.ID)
	if err != nil {
		msg := "Failed to inspect container."
		log.WithError(err).WithField("container_id", c.ID).Error(msg)
		return errors.Wrapf(err, msg+" container_id = %s", c.ID)
	}
	if !container.State.Running {
		msg := "Expecting container to be running, but was not."
		log.WithField("container_id", c.ID).Error(msg)
		return errors.Wrapf(err, msg+" container_id = %s", c.ID)
	}

	return nil
}
