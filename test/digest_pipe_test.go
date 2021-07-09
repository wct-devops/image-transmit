package test

import (
	"testing"
	"io"
	"os"
	"crypto/sha256"
	"encoding/hex"
	"github.com/wct-devops/image-transmit/core"
)

func TestDigestPipe(t *testing.T) {

	file, err := os.Open("eeee0535bf3cec7a24bff2c6e97481afa3d37e2cdeff277c57cb5cbdb2fa9e92.temp.tar.gz")

	if err != nil {
		t.Log(err)
		return
	}

	hash := sha256.New()
	rsw := core.NewReaderSumWrapper(file)
	reader := io.TeeReader(rsw, hash)

	file2,_ := os.Create("test")
	io.Copy(file2, reader)
	file2.Close()


	d := hex.EncodeToString(hash.Sum(nil))
	t.Log(d)
	t.Log(d == "e3b3aa70783d6d4b1f4d59ff0235bfad9a7ba648dab4c2ba748c3436f7b84764")
	t.Log(rsw.Size)
	return

	//rsw := core.NewReaderSumWrapper(file)
	//reader := io.TeeReader(file, hash)
	//
	//go func() {
	//	t.Log("herere")
	//	//d, err := digest.FromReader(pr)
	//	t.Log("herere")
	//
	//	io.Copy(hash, pr)
	//	d := hex.EncodeToString(hash.Sum(nil))
	//	t.Log(d)
	//	t.Log(rsw.Size)
	//	//pw.Close()
	//	//pr.Close()
	//}()
	//
	//file2,_ := os.Create("test")
	//io.Copy(file2, reader)

	//

	//t.Log(rsw.Size)


}


/*

func (t *OfflineUploadTask) Run(tid int) error {
	manifestJson, _ := t.ctx.CompMeta.Manifests[t.url]
	m := Manifest{}
	manifestByte := []byte(manifestJson)
	err := json.Unmarshal(manifestByte, &m)
	if err != nil {
		return fmt.Errorf(I18n.Sprintf("Manifest format error: %v, manifest: %s", err, manifestJson))
	}

	var blobs []types.BlobInfo
	blobs = append(blobs, m.Config )
	for _, l := range m.Layers {
		blobs = append(blobs , l)
	}

	for _, b := range blobs {
		blobExist, err := t.ids.CheckBlobExist(b)
		if err != nil {
			return fmt.Errorf(I18n.Sprintf("Check blob %s(%v) to %s/%s:%s exist error: %v", b.Digest.String(), FormatByteSize(b.Size), t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag(), err))
		}
		if blobExist {
			t.ctx.Debug(I18n.Sprintf("Blob %s(%v) has been pushed to %s/%s:%s, will not be pulled", ShortenString(b.Digest.String(), 19), FormatByteSize(b.Size), t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag()))
		} else {
			found, err := t.uploadBlob(b)
			// Retry if gzip output changed
			// Error uploading layer to http://10.45.46.109/v2/test/addon-resizer/blobs/uploads/563f398a-b7de-450a-99e1-83ef8e88a467?... digest invalid: provided digest did not match uploaded content
			// Patch "http://10.45.46.109/v2/test/addon-resizer/blobs/uploads/c837bfc0-6810-4751-99fa-e62a7accd765?...": net/http: HTTP/1.x transport connection broken: http: ContentLength=675812 with Body length 699642
			if err != nil {
				str := fmt.Sprint(err)
				if t.ctx.SquashfsTar != nil && ( strings.Contains(str, "transport connection broken: http: ContentLength") ||
					strings.Contains(str, "digest invalid: provided digest did not match uploaded content")) {
					t.ctx.log.Info(I18n.Sprintf("Error %v will be ignored", err))
					continue
				} else {
					return err
				}
			}

			if !found {
				return fmt.Errorf(I18n.Sprintf("Blob not found in datafiles: %s", b.Digest.Hex()))
			}
			if t.ctx.Cancel() {
				return fmt.Errorf(I18n.Sprintf("User cancelled..."))
			}
		}
	}

	// Retry the layers that gzip tar mismatch with origin gzip tar
	for o, n := range t.gzRetries {
		b := *n
		blobExist, err :=  t.ids.CheckBlobExist(b)
		if err != nil {
			return fmt.Errorf(I18n.Sprintf("Check blob %s(%v) to %s/%s:%s exist error: %v", b.Digest.String(), FormatByteSize(b.Size), t.ids.GetRegistry(),t.ids.GetRepository(), t.ids.GetTag(), err))
		}
		if blobExist {
			t.ctx.Debug(I18n.Sprintf("Blob %s(%v) has been pushed to %s/%s:%s, will not be pulled",ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.ids.GetRegistry(),t.ids.GetRepository(), t.ids.GetTag()))
		} else {
			reader, err := t.ctx.SquashfsTar.GetFileStream(o.Digest.Hex())
			if err != nil {
				return err
			}
			netBytes := n.Size
			begin := time.Now().Unix()
			err = t.ids.PutABlob(ioutil.NopCloser(reader), b)
			end := time.Now().Unix()
			if err != nil {
				return fmt.Errorf(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s failed: %v", b.Digest, b.Size, t.ids.GetRegistry(),t.ids.GetRepository(), t.ids.GetTag(), err))
			} else {
				t.ctx.Debug(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag()))
				t.ctx.StatUp(netBytes, end - begin)
			}
		}

		start := bytes.Index(manifestByte, []byte(o.Digest.String()))
		begIdx := bytes.LastIndex(manifestByte[0:start], []byte{'{'} )
		endIdx := bytes.Index(manifestByte[start:], []byte{'}'} )
		oldBytes := manifestByte[begIdx : start + endIdx]
		newBytes := bytes.ReplaceAll(oldBytes, []byte(o.Digest.String()), []byte(n.Digest.String()))
		newBytes =  bytes.ReplaceAll(newBytes, []byte(fmt.Sprintf(": %v", o.Size )), []byte(fmt.Sprintf(": %v", n.Size )))
		manifestByte = bytes.ReplaceAll(manifestByte, oldBytes, newBytes)
	}

	if err := t.ids.PushManifest(manifestByte); err != nil {
		return fmt.Errorf(I18n.Sprintf("Put manifest to %s/%s:%s error: %v", t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag(), err))
	}
	t.ctx.Info(I18n.Sprintf("Put manifest to %s/%s:%s", t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag()))

	return nil
}

func (t *OfflineUploadTask) uploadBlob(b types.BlobInfo) (found bool, err error){
	for k := range t.ctx.CompMeta.Datafiles {
		var reader io.Reader
		var netBytes int64
		var layerHash hash.Hash
		var rsw *ReaderSumWrapper
		if t.ctx.SquashfsTar != nil {
			rawRdr, err := t.ctx.SquashfsTar.GetFileStream(b.Digest.Hex())
			if err != nil {
				return false, err
			}
			layerHash = sha256.New()
			rsw = NewReaderSumWrapper(rawRdr)
			reader = io.TeeReader(rsw, layerHash)
		} else {
			r, err := NewImageCompressedTarReader(filepath.Join(t.path, k), t.ctx.CompMeta.Compressor)
			defer r.Close()
			if err != nil {
				return false, err
			}
			rdr, name, size, eof, err := r.ReadFileStreamByName(b.Digest.Hex())
			if eof {
				continue
			}
			if err != nil {
				return false, err
			}
			if size != b.Size {
				return false, fmt.Errorf(I18n.Sprintf("Blob %s size mismatch, size in meta: %v, size in tar: %v", name, b.Size, size))
			}
			reader = rdr
			netBytes = size
		}

		if reader == nil {
			continue
		}

		begin := time.Now().Unix()
		err = t.ids.PutABlob(ioutil.NopCloser(reader), b)
		end := time.Now().Unix()

		if layerHash != nil {
			d, _ := digest.Parse("sha256:" + hex.EncodeToString(layerHash.Sum(nil)))
			if b.Digest.Hex() != d.Hex() {
				log.Infof("Update digest from %v to %v", b.Digest.Hex(), d.Hex())
				log.Infof("Update digest from %v to %v", b.Size, rsw.Size)
				n := new(types.BlobInfo)
				tmpBytes, _ := json.Marshal(b)
				json.Unmarshal(tmpBytes, n)
				n.Digest = d
				n.Size = rsw.Size
				t.gzRetries[&b] = n
			}
		}

		if err != nil {
			return false, fmt.Errorf(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s failed: %v", b.Digest, b.Size, t.ids.GetRegistry(),t.ids.GetRepository(), t.ids.GetTag(), err))
		} else {
			t.ctx.Debug(I18n.Sprintf("Put blob %s(%v) to %s/%s:%s success", ShortenString(b.Digest.String(),19), FormatByteSize(b.Size), t.ids.GetRegistry(), t.ids.GetRepository(), t.ids.GetTag()))
			t.ctx.StatUp(netBytes, end - begin)
			found = true
			break
		}
	}
	return found, nil
}

 */