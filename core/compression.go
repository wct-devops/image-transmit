package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/containers/image/v5/types"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4"
	"github.com/ulikunitz/xz"
)

type CompressionMetadata struct {
	m          sync.Mutex
	Datafiles  map[string]int64
	Compressor string
	Blobs      map[string][]string
	Manifests  map[string]string
	BlobDoing  map[string]int
}

func NewCompressionMetadata(compressor string) (*CompressionMetadata, error) {
	blobs := make(map[string][]string)
	manifests := make(map[string]string)
	datafiles := make(map[string]int64)
	blobDoing := make(map[string]int)
	return &CompressionMetadata{
		Blobs:      blobs,
		Manifests:  manifests,
		Datafiles:  datafiles,
		Compressor: compressor,
		BlobDoing:  blobDoing,
	}, nil
}

func (c *CompressionMetadata) BlobExists(sha256 string) bool {
	c.m.Lock()
	defer c.m.Unlock()
	_, ok := c.Blobs[sha256]
	return ok
}

func (c *CompressionMetadata) BlobDone(sha256 string, ref string) {
	c.m.Lock()
	defer c.m.Unlock()
	v, ok := c.Blobs[sha256]
	if !ok {
		v = make([]string, 0)
	}
	c.Blobs[sha256] = append(v, ref)
}

func (c *CompressionMetadata) BlobStart(sha256 string, tid int) bool {
	c.m.Lock()
	defer c.m.Unlock()
	_, ok := c.BlobDoing[sha256]
	if !ok {
		c.BlobDoing[sha256] = tid
		return false
	} else {
		return true
	}
}

func (c *CompressionMetadata) ClearDoing(tid int) {
	c.m.Lock()
	defer c.m.Unlock()
	for k, v := range c.BlobDoing {
		if v == tid {
			delete(c.BlobDoing, k)
		}
	}
}

func (c *CompressionMetadata) AddImage(name string, manifest string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Manifests[name] = manifest
}

func (c *CompressionMetadata) AddDatafile(name string, num int64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Datafiles[name] = num
}

type SingleTarWriter struct {
	m    sync.Mutex
	t    *ImageCompressedTarWriter
	d    *DockerSaver
	todo *list.List
	quit bool
}

func NewSingleTarWriter(ctx *TaskContext, filename string, compression string) (*SingleTarWriter, error) {
	var t *ImageCompressedTarWriter
	var d *DockerSaver
	var err error
	if CONF.DockerFile {
		d = NewDockerSaver(ctx, filename)
	} else {
		t, err = NewImageCompressedTarWriter(filename, compression)
		if err != nil {
			return nil, err
		}
	}

	return &SingleTarWriter{
		todo: list.New(),
		quit: false,
		t:    t,
		d:    d,
	}, nil
}

func (s *SingleTarWriter) PutFile(filename string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.todo.PushBack(filename)
}

func (s *SingleTarWriter) takeFile() (string, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	file := s.todo.Front()
	if file == nil {
		return "", true
	}
	s.todo.Remove(file)
	return file.Value.(string), false
}

func (s *SingleTarWriter) Run() {
	for {
		filename, empty := s.takeFile()
		if empty {
			if s.quit {
				break
			} else {
				time.Sleep(1 * time.Second)
				continue
			}
		} else {
			file, err := os.Open(filename)
			if err != nil {
				panic(err)
			}
			ff, err := file.Stat()
			if err != nil {
				panic(err)
			}
			if s.t != nil {
				err = s.t.AppendFileStream(ff.Name(), ff.Size(), file)
				s.t.Flush()
			} else if s.d != nil {
				hex := ff.Name()[0:strings.Index(ff.Name(), ".")]
				if strings.HasSuffix(ff.Name(), ".json") {
					err = s.d.AppendFileStream(ff.Name(), ff.Size(), file)
				} else if strings.HasSuffix(ff.Name(), ".tar.gz") {
					err = s.d.AppendFileStream(hex+"/"+"layer.tar.gz", ff.Size(), file)
				} else {
					err = s.d.AppendFileStream(hex+"/"+"layer.raw", ff.Size(), file)
				}
			}
			if err != nil {
				panic(err)
			}
		}
	}
	if s.t != nil {
		s.t.Close()
	}
}

func (s *SingleTarWriter) SaveDockerMeta(cm *CompressionMetadata) {
	if s.d == nil {
		return
	}
	for k, v := range cm.Manifests {
		m := Manifest{}
		manifestByte := []byte(v)
		json.Unmarshal(manifestByte, &m)
		s.d.AppendMeta(&m, k)
	}
	s.d.Close()
}

func (s *SingleTarWriter) SetQuit() {
	s.quit = true
}

type ImageCompressedTarWriter struct {
	file       *os.File
	tarWriter  *tar.Writer
	compressor io.WriteCloser
}

func NewImageCompressedTarWriter(filename string, compression string) (*ImageCompressedTarWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	var compressor io.WriteCloser
	switch compression {
	case "zstd":
		compressor, err = zstd.NewWriter(file)
	case "gzip":
		compressor = gzip.NewWriter(file)
	case "xz":
		conf := xz.WriterConfig{CheckSum: xz.CRC32}
		if err := conf.Verify(); err != nil {
			return nil, err
		}
		compressor, err = conf.NewWriter(file)
	case "lz4":
		compressor = lz4.NewWriter(file)
	case "tar":
		compressor = file
	default:
		err = fmt.Errorf("unknown compression format: %s", compression)
	}

	if err != nil {
		return nil, err
	}

	tarWriter := tar.NewWriter(compressor)
	return &ImageCompressedTarWriter{
		tarWriter:  tarWriter,
		compressor: compressor,
	}, nil
}

func (img *ImageCompressedTarWriter) Cleanup() {
	_ = img.tarWriter.Close()
	if img.compressor != img.file {
		_ = img.compressor.Close()
	}
	_ = img.file.Close()
}

func (img *ImageCompressedTarWriter) Flush() {
	_ = img.tarWriter.Flush()
	_ = img.file.Sync()
}

func (img *ImageCompressedTarWriter) Close() error {
	if err := img.tarWriter.Close(); err != nil {
		return err
	}
	if img.compressor != img.file {
		if err := img.compressor.Close(); err != nil {
			return err
		}
	}
	return img.file.Close()
}

func (img *ImageCompressedTarWriter) AppendFileStream(filename string, size int64, reader io.ReadCloser) error {
	hdr := &tar.Header{
		Name: filename,
		Size: size,
		Mode: tar.TypeReg,
	}
	img.tarWriter.WriteHeader(hdr)
	ws, err := io.Copy(img.tarWriter, reader)
	reader.Close()
	if err == nil && ws != size {
		err = fmt.Errorf("file %s content size mismatch, %v VS %v, network or file system problem", filename, ws, size)
	}
	return err
}

type ImageCompressedTarReader struct {
	file       *os.File
	tarReader  *tar.Reader
	compressor io.ReadCloser
}

func NewImageCompressedTarReader(filename string, compression string) (*ImageCompressedTarReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	var compressor io.ReadCloser
	switch compression {
	case "zstd":
		decoder, err := zstd.NewReader(file)
		if err != nil {
			return nil, err
		}
		compressor = decoder.IOReadCloser()
	case "gzip":
		compressor, err = gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
	case "xz":
		conf := xz.ReaderConfig{SingleStream: true}
		if err := conf.Verify(); err != nil {
			return nil, err
		}
		decoder, err := conf.NewReader(file)
		if err != nil {
			return nil, err
		}
		compressor = ioutil.NopCloser(decoder)
	case "lz4":
		compressor = ioutil.NopCloser(lz4.NewReader(file))
	case "tar":
		compressor = file
	default:
		err = fmt.Errorf("unknown compression format: %s", compression)
	}

	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(compressor)
	return &ImageCompressedTarReader{
		tarReader:  tarReader,
		compressor: compressor,
	}, nil
}

func (img *ImageCompressedTarReader) Close() error {
	if img.compressor != img.file {
		if err := img.compressor.Close(); err != nil {
			return err
		}
	}
	return img.file.Close()
}

func (img *ImageCompressedTarReader) ReadFileStreamByName(hex string) (io.Reader, string, int64, bool, error) {
	for {
		hdr, err := img.tarReader.Next()
		if hdr == nil {
			return nil, "", 0, true, nil
		}
		if err != nil {
			return nil, "", 0, false, err
		}
		if strings.HasPrefix(hdr.Name, hex) {
			return img.tarReader, hdr.Name, hdr.Size, false, nil
		}
	}
}

func (img *ImageCompressedTarReader) ReadFileStream(skip int) (io.Reader, string, int64, bool, error) {
	for i := 0; i < skip; i++ {
		hdr, err := img.tarReader.Next()
		if hdr == nil {
			return nil, "", 0, true, nil
		}
		if err != nil {
			return nil, "", 0, false, err
		}
	}
	hdr, err := img.tarReader.Next()
	if hdr == nil {
		return nil, "", 0, true, nil
	}
	if err != nil {
		return nil, "", 0, false, err
	}
	return img.tarReader, hdr.Name, hdr.Size, false, nil
}

// Save image with docker tar format
type DockerSaver struct {
	cmdWriter    io.WriteCloser
	tarWriter    *tar.Writer
	ctx          *TaskContext
	wait         *sync.Mutex
	repositories map[string]map[string]string
	manifests    [](map[string]interface{})
}

func NewDockerSaver(ctx *TaskContext, target string) *DockerSaver {
	var cmdWriter io.WriteCloser
	var tarWriter *tar.Writer
	var err error
	wait := new(sync.Mutex)
	repositories := make(map[string]map[string]string)

	if target == "docker" || target == "ctr" {
		var cmd *exec.Cmd
		if target == "docker" {
			cmd = exec.Command("docker", "load")
		} else {
			cmd = exec.Command("ctr", "image", "import", "/dev/stdin")
		}

		cmdWriter, err = cmd.StdinPipe()
		if err != nil {
			log.Error(err)
			panic(err)
		}

		go func() {
			wait.Lock()
			defer wait.Unlock()
			cmd.Stdout = NewStdoutWrapper(ctx.log)
			cmd.Stderr = NewStderrWrapper(ctx.log)
			err := cmd.Run()
			if err != nil {
				log.Error(err)
			}
		}()
		tarWriter = tar.NewWriter(cmdWriter)
	} else {
		cmdWriter, err = os.Create(target)
		if err != nil {
			log.Error(err)
			panic(err)
		}
		tarWriter = tar.NewWriter(cmdWriter)
	}

	return &DockerSaver{
		cmdWriter:    cmdWriter,
		tarWriter:    tarWriter,
		ctx:          ctx,
		wait:         wait,
		repositories: repositories,
	}
}

func (d *DockerSaver) Close() {
	mb, _ := json.Marshal(d.manifests)
	d.AppendFileStream("manifest.json", int64(len(mb)), ioutil.NopCloser(bytes.NewReader(mb)))
	rb, _ := json.Marshal(d.repositories)
	d.AppendFileStream("repositories", int64(len(rb)), ioutil.NopCloser(bytes.NewReader(rb)))
	d.tarWriter.Close()
	d.cmdWriter.Close()
	d.wait.Lock()
	defer d.wait.Unlock()
}

func (d *DockerSaver) AppendMeta(m *Manifest, url string) {
	imgUrl := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	manifest_json := make(map[string]interface{})
	manifest_json["Config"] = m.Config.Digest.Hex() + ".json"
	manifest_json["RepoTags"] = []string{imgUrl}
	layers := []string{}
	for _, b := range m.Layers {
		layers = append(layers, b.Digest.Hex()+"/layer"+GetBlobSuffix(b))
	}
	manifest_json["Layers"] = layers
	d.manifests = append(d.manifests, manifest_json)
	imageName := imgUrl[:strings.LastIndex(imgUrl, ":")]
	imageTag := imgUrl[strings.LastIndex(imgUrl, ":"):]
	//d.repositories[imageName] = map[string]string{imageTag: m.Layers[len(m.Layers)-1].Digest.Hex() + strconv.Itoa(len(m.Layers)-1)}
	d.repositories[imageName] = map[string]string{imageTag: m.Layers[len(m.Layers)-1].Digest.Hex()}
}

func (d *DockerSaver) AppendFileStream(filename string, size int64, reader io.Reader) error {
	hdr := &tar.Header{
		Name: filename,
		Size: size,
		Mode: tar.TypeReg,
	}
	d.tarWriter.WriteHeader(hdr)
	ws, err := io.Copy(d.tarWriter, reader)
	if err == nil && ws != size {
		err = fmt.Errorf("file %s content size mismatch, %v VS %v, network or file system problem", filename, ws, size)
	}
	return err
}

func GetBlobSuffix(b types.BlobInfo) string {
	// skip some empty gzip layers or tar-split will failed, and lots of empty HEXs here, using size more safe
	if strings.HasSuffix(b.MediaType, "tar.gzip") && b.Size > 32 {
		return ".tar.gz"
	} else if strings.HasSuffix(b.MediaType, "tar") {
		return ".tar"
	} else if strings.HasSuffix(b.MediaType, "json") {
		return ".json"
	} else {
		return ".raw"
	}
}
