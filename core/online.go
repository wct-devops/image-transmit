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
	source       *ImageSource
	destination  *ImageDestination
	ctx          *TaskContext
	srcUrl       string
	dstUrl       string
	callbackFunc func(bool, string)
	byteDown     int64
	byteUp       int64
	timeDown	 time.Duration
	timeUp	     time.Duration
}

// NewTask creates a task
func NewOnlineTask(source *ImageSource, destination *ImageDestination, ctx *TaskContext) Task{
	return NewOnlineTaskCallback(source, destination, ctx, nil)
}

func NewOnlineTaskCallback(source *ImageSource, destination *ImageDestination, ctx *TaskContext, callback func(bool, string)) Task{
	srcUrl := fmt.Sprintf(	"%s/%s:%s", source.GetRegistry(), source.GetRepository(), source.GetTag())
	dstUrl := fmt.Sprintf(	"%s/%s:%s", destination.GetRegistry(), destination.GetRepository(), destination.GetTag())
	return &OnlineTask{
		source:       source,
		destination:  destination,
		ctx:          ctx,
		srcUrl:       srcUrl,
		dstUrl:       dstUrl,
		callbackFunc: callback,
		byteDown:     0,
		byteUp:       0,
		timeDown: 1,
		timeUp: 1,
	}
}

func (t *OnlineTask) Name() string {
	return t.srcUrl
}

func (t *OnlineTask) Callback(success bool, content string) {
	if t.callbackFunc != nil {
		t.callbackFunc(success, content)
	}
}

func (t *OnlineTask) StatDown(size int64, duration time.Duration) {
	t.byteDown = t.byteDown + size
	t.timeDown = t.timeDown + duration
}
func (t *OnlineTask) StatUp(size int64, duration time.Duration) {
	t.byteUp = t.byteUp + size
	t.timeUp = t.timeUp + duration
}
func (t *OnlineTask) Status() string {
	return I18n.Sprintf("Speed:^%s/s v%s/s Total:^%s v%s", FormatByteSize(int64(float64(t.byteDown)/(float64(t.timeDown)/float64(time.Second)))), FormatByteSize(int64(float64(t.byteUp)/(float64(t.timeUp)/float64(time.Second)))), FormatByteSize(t.byteDown), FormatByteSize(t.byteUp))
}

// Run ids the main function of a task
func (t *OnlineTask) Run(idx int) error {
	// get manifest from source
	manifestByte, manifestType, err := t.source.GetManifest()
	if err != nil {
		return errors.New(I18n.Sprintf("Failed to get manifest from %s error: %v", t.srcUrl, err))
	}
	t.ctx.Info(I18n.Sprintf("Get manifest from %s", t.srcUrl))

	blobInfos, err := t.source.GetBlobInfos(manifestByte, manifestType)
	if err != nil {
		return errors.New(I18n.Sprintf("Get blob info from %s error: %v", t.srcUrl, err))
	}

	// blob transformation
	for _, b := range blobInfos {
		blobExist, err := t.destination.CheckBlobExist(b)
		if err != nil {
			return errors.New(I18n.Sprintf("Check blob %s(%v) to %s exist error: %v", b.Digest.String(), FormatByteSize(b.Size), t.srcUrl, err))
		}

		if !blobExist {
			// pull a blob from source
			begin := time.Now()
			blob, size, err := t.source.GetABlob(b)
			if err != nil {
				return errors.New(I18n.Sprintf("Get blob %s(%v) from %s failed: %v", b.Digest.String(), FormatByteSize(b.Size), t.srcUrl, err))
			}
			t.ctx.Debug(I18n.Sprintf("Get a blob %s(%v) from %s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.srcUrl))

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
				return errors.New(I18n.Sprintf("Put blob %s(%v) to %s failed: %v", b.Digest, b.Size, t.destination.GetRegistry(),t.destination.GetRepository(), t.destination.GetTag(), err))
			}
			t.ctx.Info(I18n.Sprintf("Put blob %s(%v) to %s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.dstUrl))

			duration := time.Now().Sub(begin)

			if downSize > 0 {
				t.ctx.StatDown(downSize, duration)
				t.StatDown(downSize, duration)
			}
			t.ctx.StatUp(size, duration)
			t.StatUp(size, duration)

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
			t.ctx.Info(I18n.Sprintf("Blob %s(%v) has been pushed to %s, will not be pulled",ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.dstUrl))
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
				return errors.New(I18n.Sprintf("Put manifest to %s error: %v", t.dstUrl, err))
			}
		}

		// push manifest list to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return errors.New(I18n.Sprintf("Put manifestList to %s error: %v", t.dstUrl, err))
		}

		t.ctx.Info(I18n.Sprintf("Put manifestList to %s", t.dstUrl))

	} else {
		// push manifest to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return errors.New(I18n.Sprintf("Put manifest to %s error: %v", t.dstUrl, err))
		}
		t.ctx.Info(I18n.Sprintf("Put manifest to %s", t.dstUrl))
	}

	t.ctx.Info(I18n.Sprintf("Transmit successfully from %s to %s", t.srcUrl, t.dstUrl))
	if t.ctx.History != nil {
		t.ctx.History.Add( t.srcUrl )
	}
	return nil
}

func ShortenString(str string, n int) string {
	if len(str) <= n {
		return str
	} else {
		return str[:n]
	}
}