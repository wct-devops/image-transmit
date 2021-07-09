package core

import (
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"time"
	"strings"
	"io"
	"fmt"
	"github.com/pkg/errors"
)

var (
	// NoCache used to disable a blobinfocache
	NoCache = none.NoCache
)

// Task act as a action, it will pull a images from source to destination
type OnlineTask struct {
	source      *ImageSource
	destination *ImageDestination
	ctx *TaskContext
}

// NewTask creates a task
func NewOnlineTask(source *ImageSource, destination *ImageDestination, ctx *TaskContext) Task{
	return &OnlineTask{
		source:      source,
		destination: destination,
		ctx:      ctx,
	}
}

func (t *OnlineTask) Name() string {
	return fmt.Sprintf(	"%s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag())
}

// Run ids the main function of a task
func (t *OnlineTask) Run(idx int) error {
	// get manifest from source
	manifestByte, manifestType, err := t.source.GetManifest()
	if err != nil {
		return errors.New(I18n.Sprintf("Failed to get manifest from %s/%s:%s error: %v", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err))
	}
	t.ctx.Info(I18n.Sprintf("Get manifest from %s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag()))

	blobInfos, err := t.source.GetBlobInfos(manifestByte, manifestType)
	if err != nil {
		return errors.New(I18n.Sprintf("Get blob info from %s/%s:%s error: %v", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err))
	}

	// blob transformation
	for _, b := range blobInfos {
		blobExist, err := t.destination.CheckBlobExist(b)
		if err != nil {
			return errors.New(I18n.Sprintf("Check blob %s(%v) to %s/%s:%s exist error: %v", b.Digest.String(), FormatByteSize(b.Size), t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err))
		}

		if !blobExist {
			// pull a blob from source
			begin := time.Now().Unix()
			blob, size, err := t.source.GetABlob(b)
			if err != nil {
				return errors.New(I18n.Sprintf("Get blob %s(%v) from %s/%s:%s failed: %v", b.Digest.String(), FormatByteSize(b.Size), t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err))
			}
			t.ctx.Debug(I18n.Sprintf("Get a blob %s(%v) from %s/%s:%s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag()))

			if t.ctx.Cancel() {
				return errors.New(I18n.Sprintf("User cancelled..."))
			}

			b.Size = size
			var upReader io.ReadCloser
			var downSize int64
			var blobName string
			var wCloser io.WriteCloser
			var rCloser io.ReadCloser
			// skip the empty gzip layer or tar-split will failed, and many empty HEXs here, using size more safe
			if strings.HasSuffix(b.MediaType, "tar.gzip") && b.Size > 64 * 1024 {
				blobName = b.Digest.Hex() + ".tar.gz"
			} else {
				blobName = b.Digest.Hex() + ".raw"
			}
			if t.ctx.Cache != nil {
				match, _ := t.ctx.Cache.Match(blobName, size)
				if match {
					upReader, err = t.ctx.Cache.Reuse(blobName)
					t.ctx.Debug(I18n.Sprintf("Reuse cache: %s", blobName))
					downSize = 0
					if err != nil {
						return errors.New(I18n.Sprintf("Read file from cache failed %s", err))
					}
				} else {
					downSize = size
					upReader, wCloser = t.ctx.Cache.SaveStream(blobName, blob)
				}
				rCloser = blob
			} else {
				upReader = blob
				downSize = size
			}

			// push a blob to destination
			if err := t.destination.PutABlob(upReader, b); err != nil {
				return errors.New(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s failed: %v", b.Digest, b.Size, t.destination.GetRegistry(),t.destination.GetRepository(), t.destination.GetTag(), err))
			}
			t.ctx.Info(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag()))
			end := time.Now().Unix()

			if downSize > 0 {
				t.ctx.StatDown(downSize, end - begin)
			}
			t.ctx.StatUp(size, end - begin)

			if wCloser != nil {
				wCloser.Close()
			}
			if rCloser != nil {
				rCloser.Close()
			}

			if t.ctx.Cancel() {
				return errors.New(I18n.Sprintf("User cancelled..."))
			}
		} else {
			// print the log of ignored blob
			t.ctx.Info(I18n.Sprintf("Blob %s(%v) has been pushed to %s/%s:%s, will not be pulled",ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag()))
		}
	}

	//Push manifest list
	if manifestType == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(manifestByte)
		if err != nil {
			return err
		}

		var subManifestByte []byte

		// push manifest to destination
		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			subManifestByte, _, err = t.source.source.GetManifest(t.source.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return errors.New(I18n.Sprintf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v", manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture, err))
			}

			if err := t.destination.PushManifest(subManifestByte); err != nil {
				return errors.New(I18n.Sprintf("Put manifest to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err))
			}
		}

		// push manifest list to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return errors.New(I18n.Sprintf("Put manifestList to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err))
		}

		t.ctx.Info(I18n.Sprintf("Put manifestList to %s/%s:%s", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag()))

	} else {
		// push manifest to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return errors.New(I18n.Sprintf("Put manifest to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err))
		}
		t.ctx.Info(I18n.Sprintf("Put manifest to %s/%s:%s", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag()))
	}
	t.ctx.Info(I18n.Sprintf("Transmit successfully from %s/%s:%s to %s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag()))
	return nil
}

func ShortenString(str string, n int) string {
	if len(str) <= n {
		return str
	} else {
		return str[:n]
	}
}