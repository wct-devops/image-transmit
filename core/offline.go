package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	log "github.com/cihub/seelog"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type OfflineDownTask struct {
	ctx *TaskContext
	url string
	is  *ImageSource
}

func NewOfflineDownTask(ctx *TaskContext, url string, is *ImageSource) Task {
	return &OfflineDownTask{
		ctx: ctx,
		url: url,
		is:  is,
	}
}

func (t *OfflineDownTask) Name() string {
	return t.url
}

func (t *OfflineDownTask) Callback(success bool, content string) {
}

func (t *OfflineDownTask) StatDown(size int64, duration time.Duration) {

}
func (t *OfflineDownTask) StatUp(size int64, duration time.Duration) {

}
func (t *OfflineDownTask) Status() string {
	return ""
}

func (t *OfflineDownTask) Run(tid int) error {
	srcUrl := fmt.Sprintf("%s/%s:%s", t.is.GetRegistry(), t.is.GetRepository(), t.is.GetTag())
	defer t.ctx.CompMeta.ClearDoing(tid)
	manifestByte, manifestType, err := t.is.GetManifest()
	if err != nil {
		return errors.New(I18n.Sprintf("Failed to get manifest from %s error: %v", srcUrl, err))
	}
	t.ctx.Info(I18n.Sprintf("Get manifest from %s", srcUrl))

	blobInfos, err := t.is.GetBlobInfos(manifestByte, manifestType)
	if err != nil {
		return errors.New(I18n.Sprintf("Get blob info from %s error: %v", srcUrl, err))
	}
	t.ctx.CompMeta.AddImage(t.url, string(manifestByte))

	for _, b := range blobInfos {
		begin := time.Now()
		var netBytes = b.Size
		if t.ctx.Cancel() {
			return errors.New(I18n.Sprintf("User cancelled..."))
		}

		if t.ctx.CompMeta.BlobExists(b.Digest.Hex()) || t.ctx.CompMeta.BlobStart(b.Digest.Hex(), tid) {
			t.ctx.Debug(I18n.Sprintf("Skip blob: %s", ShortenString(b.Digest.String(), 19)))
			continue
		}

		blob, size, err := t.is.GetABlob(b)
		if err != nil {
			return errors.New(I18n.Sprintf("Get blob %s(%v) from %s failed: %v", b.Digest.String(), FormatByteSize(size), srcUrl, err))
		}
		t.ctx.Debug(I18n.Sprintf("Get a blob %s(%v) from %s success", ShortenString(b.Digest.String(), 19), FormatByteSize(size), srcUrl))
		blobName := b.Digest.Hex() + GetBlobSuffix(b)

		if t.ctx.SquashfsTar != nil {
			if t.ctx.Cache != nil {
				matched, filename := t.ctx.Cache.Match(blobName, size)
				if !matched {
					r, w, _ := t.ctx.Cache.SaveStream(blobName, blob)
					err := t.ctx.SquashfsTar.AppendFileStream(blobName, size, r)
					w.Close()
					if err != nil {
						return errors.New(I18n.Sprintf("Save Stream file to cache failed: %v", err))
					}
				} else {
					blob.Close()
					r, err := t.ctx.Cache.Reuse(blobName)
					t.ctx.Debug(I18n.Sprintf("Reuse cache: %s", filename))
					netBytes = 0
					if err != nil {
						return errors.New(I18n.Sprintf("Read file from cache failed: %v", err))
					}
					err = t.ctx.SquashfsTar.AppendFileStream(blobName, size, r)
					if err != nil {
						return err
					}
				}
			} else {
				err = t.ctx.SquashfsTar.AppendFileStream(blobName, size, blob)
				if err != nil {
					return err
				}
			}
		} else if t.ctx.SingleWriter != nil {
			if t.ctx.Cache != nil {
				matched, filename := t.ctx.Cache.Match(blobName, size)
				if !matched {
					var err error
					filename, err = t.ctx.Cache.SaveFile(blobName, blob, size)
					if err != nil {
						return errors.New(I18n.Sprintf("Save Stream file to cache failed: %v", err))
					}
				} else {
					t.ctx.Debug(I18n.Sprintf("Reuse cache %s", filename))
					netBytes = 0
					blob.Close()
				}
				t.ctx.SingleWriter.PutFile(filename)
				t.ctx.Debug(I18n.Sprintf("Put file to archive: %s", filename))
			} else {
				filename, err := t.ctx.Temp.SaveFile(blobName, blob, size)
				if err != nil {
					return errors.New(I18n.Sprintf("Save Stream file to temp failed: %v", err))
				}
				t.ctx.SingleWriter.PutFile(filename)
				t.ctx.Debug(I18n.Sprintf("Put file to archive: %s", filename))
			}
		} else {
			tar := t.ctx.TarWriter[tid]
			defer tar.Flush()
			if t.ctx.Cache != nil {
				matched, filename := t.ctx.Cache.Match(blobName, size)
				if !matched {
					r, w, _ := t.ctx.Cache.SaveStream(blobName, blob)
					err = tar.AppendFileStream(blobName, size, r)
					w.Close()
					if err != nil {
						return err
					}
				} else {
					blob.Close()
					r, err := t.ctx.Cache.Reuse(blobName)
					t.ctx.Debug(I18n.Sprintf("Reuse cache: %s", filename))
					netBytes = 0
					if err != nil {
						return errors.New(I18n.Sprintf("Read file from cache failed: %v", err))
					}
					err = tar.AppendFileStream(blobName, size, r)
					if err != nil {
						return err
					}
				}
			} else {
				err = tar.AppendFileStream(blobName, size, blob)
				if err != nil {
					return err
				}
			}
		}
		if netBytes > 0 {
			t.ctx.StatDown(netBytes, time.Since(begin))
		}
		t.ctx.CompMeta.BlobDone(b.Digest.Hex(), t.url)
	}
	return nil
}

type OfflineUploadTask struct {
	ctx       *TaskContext
	ids       *ImageDestination
	url       string
	path      string
	gzRetries map[*types.BlobInfo]*types.BlobInfo
}

func NewOfflineUploadTask(ctx *TaskContext, ids *ImageDestination, url string, path string) Task {
	return &OfflineUploadTask{
		ctx:       ctx,
		ids:       ids,
		url:       url,
		path:      path,
		gzRetries: make(map[*types.BlobInfo]*types.BlobInfo),
	}
}

func (t *OfflineUploadTask) Name() string {
	return t.url
}

func (t *OfflineUploadTask) Callback(bool, string) {
}

func (t *OfflineUploadTask) StatDown(size int64, duration time.Duration) {

}
func (t *OfflineUploadTask) StatUp(size int64, duration time.Duration) {

}
func (t *OfflineUploadTask) Status() string {
	return ""
}

func (t *OfflineUploadTask) Run(tid int) error {
	manifestJson := t.ctx.CompMeta.Manifests[t.url]
	m := Manifest{}
	manifestByte := []byte(manifestJson)
	err := json.Unmarshal(manifestByte, &m)
	if err != nil {
		return fmt.Errorf(I18n.Sprintf("Manifest format error: %v, manifest: %s", err, manifestJson))
	}

	var blobs []types.BlobInfo
	blobs = append(blobs, m.Config)
	blobs = append(blobs, m.Layers...)

	var dockerSaver *DockerSaver
	if t.ctx.DockerTarget != "" {
		dockerSaver = NewDockerSaver(t.ctx, t.ctx.DockerTarget)
	}

	var dstUrl string
	for i, b := range blobs {
		blobExist := false
		var err error
		if t.ids != nil {
			dstUrl = fmt.Sprintf("%s/%s:%s", t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag())
			blobExist, err = t.ids.CheckBlobExist(b)
			if err != nil {
				return fmt.Errorf(I18n.Sprintf("Check blob %s(%v) to %s exist error: %v", b.Digest.String(), FormatByteSize(b.Size), dstUrl, err))
			}
		}
		if blobExist {
			t.ctx.Debug(I18n.Sprintf("Blob %s(%v) has been pushed to %s, will not be pulled", ShortenString(b.Digest.String(), 19), FormatByteSize(b.Size), dstUrl))
		} else {
			var found bool = false
			for k := range t.ctx.CompMeta.Datafiles {
				var reader io.Reader
				var netBytes int64
				if t.ctx.SquashfsTar != nil {
					rawRdr, err := t.ctx.SquashfsTar.GetFileStream(b.Digest.Hex())
					if err != nil {
						return err
					}
					layerHash := sha256.New()
					rsw := NewReaderSumWrapper(rawRdr)
					io.Copy(layerHash, rsw)

					reader, err = t.ctx.SquashfsTar.GetFileStream(b.Digest.Hex())
					if err != nil {
						return err
					}

					d, _ := digest.Parse("sha256:" + hex.EncodeToString(layerHash.Sum(nil)))
					if b.Digest.Hex() != d.Hex() && dockerSaver == nil { // TODO avoid some failure cases, but not know why inconsist happened
						log.Warnf("Update digest from %v to %v", b.Digest.Hex(), d.Hex())
						log.Warnf("Update digest from %v to %v", b.Size, rsw.Size)
						n := new(types.BlobInfo)
						tmpBytes, _ := json.Marshal(b)
						json.Unmarshal(tmpBytes, n)
						n.Digest = d
						n.Size = rsw.Size
						start := bytes.Index(manifestByte, []byte(b.Digest.String()))
						begIdx := bytes.LastIndex(manifestByte[0:start], []byte{'{'})
						endIdx := bytes.Index(manifestByte[start:], []byte{'}'})
						oldBytes := manifestByte[begIdx : start+endIdx]
						newBytes := bytes.ReplaceAll(oldBytes, []byte(b.Digest.String()), []byte(n.Digest.String()))
						newBytes = bytes.ReplaceAll(newBytes, []byte(fmt.Sprintf(": %v", b.Size)), []byte(fmt.Sprintf(": %v", n.Size)))
						manifestByte = bytes.ReplaceAll(manifestByte, oldBytes, newBytes)
						b = *n
					}
				} else {
					r, err := NewImageCompressedTarReader(filepath.Join(t.path, k), t.ctx.CompMeta.Compressor)
					if err != nil {
						return err
					}
					defer r.Close()
					rdr, name, size, eof, err := r.ReadFileStreamByName(b.Digest.Hex())
					if eof {
						continue
					}
					if err != nil {
						return err
					}
					if size != b.Size {
						return fmt.Errorf(I18n.Sprintf("Blob %s size mismatch, size in meta: %v, size in tar: %v", name, b.Size, size))
					}
					reader = rdr
					netBytes = size
				}

				if reader == nil {
					continue
				}

				if dockerSaver != nil {
					if i == 0 {
						dockerSaver.AppendFileStream(b.Digest.Hex()+".json", b.Size, reader)
					} else {
						dockerSaver.AppendFileStream(b.Digest.Hex()+"/layer"+GetBlobSuffix(b), b.Size, reader)
					}
					found = true
					break
				} else {
					begin := time.Now()
					err = t.ids.PutABlob(ioutil.NopCloser(reader), b)
					if err != nil {
						return fmt.Errorf(I18n.Sprintf("Put blob %s(%v) to %s failed: %v", b.Digest, b.Size, t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag(), err))
					} else {
						t.ctx.Debug(I18n.Sprintf("Put blob %s(%v) to %s success", ShortenString(b.Digest.String(), 19), FormatByteSize(b.Size), dstUrl))
						t.ctx.StatUp(netBytes, time.Since(begin))
						found = true
						break
					}
				}
			}
			if dockerSaver == nil && !found {
				return fmt.Errorf(I18n.Sprintf("Blob not found in datafiles: %s", b.Digest.Hex()))
			}
			if t.ctx.Cancel() {
				return fmt.Errorf(I18n.Sprintf("User cancelled..."))
			}
		}
	}

	if dockerSaver == nil {
		if err := t.ids.PushManifest(manifestByte); err != nil {
			return fmt.Errorf(I18n.Sprintf("Put manifest to %s error: %v", dstUrl, err))
		}
		t.ctx.Info(I18n.Sprintf("Put manifest to %s", dstUrl))
	} else {
		dockerSaver.AppendMeta(&m, t.url)
		dockerSaver.Close()
	}

	return nil
}

type Manifest struct {
	Config types.BlobInfo   `json:"config"`
	Layers []types.BlobInfo `json:"layers"`
}

type ReaderSumWrapper struct {
	reader io.Reader
	Size   int64
}

func NewReaderSumWrapper(reader io.Reader) *ReaderSumWrapper {
	return &ReaderSumWrapper{
		reader: reader,
	}
}

func (r *ReaderSumWrapper) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.Size = r.Size + int64(n)
	return n, err
}
