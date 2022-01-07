package core

import (
	"sync"
	"io"
	"io/ioutil"
	"archive/tar"
	"os"
	"fmt"
	"container/list"
	"time"
	"strings"
	"github.com/klauspost/compress/zstd"
	"compress/gzip"
	"github.com/ulikunitz/xz"
	"github.com/pierrec/lz4"
)

type CompressionMetadata struct {
	m         sync.Mutex
	Datafiles map[string]int64
	Compressor string
	Blobs     map[string][]string
	Manifests map[string]string
	BlobDoing map[string]int
}

func NewCompressionMetadata(compressor string) (*CompressionMetadata, error) {
	blobs := make(map[string][]string)
	manifests := make(map[string]string)
	datafiles := make(map[string]int64)
	blobDoing := make(map[string]int)
	return &CompressionMetadata{
		Blobs:     blobs,
		Manifests: manifests,
		Datafiles: datafiles,
		Compressor: compressor,
		BlobDoing: blobDoing,
	}, nil
}

func (c * CompressionMetadata) BlobExists(sha256 string) bool{
	c.m.Lock()
	defer c.m.Unlock()
	_, ok := c.Blobs[sha256]
	if ok {
		return true
	}
	return false
}

func (c * CompressionMetadata) BlobDone(sha256 string, ref string){
	c.m.Lock()
	defer c.m.Unlock()
	v, ok := c.Blobs[sha256]
	if !ok {
		v = make([]string, 0, 0)
	}
	c.Blobs[sha256] = append(v, ref)
}

func (c * CompressionMetadata) BlobStart(sha256 string, tid int) bool {
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

func (c * CompressionMetadata) ClearDoing(tid int) {
	c.m.Lock()
	defer c.m.Unlock()
	for k, v := range c.BlobDoing {
		if v == tid {
			delete(c.BlobDoing, k)
		}
	}
}

func (c * CompressionMetadata) AddImage(name string, manifest string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Manifests[name] = manifest
}

func (c * CompressionMetadata) AddDatafile(name string, num int64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Datafiles[name] = num
}

type SingleTarWriter struct{
	m sync.Mutex
	t *ImageCompressedTarWriter
	todo *list.List
	quit bool
}

func NewSingleTarWriter(filename string, compression string) (*SingleTarWriter, error){
	t, err := NewImageCompressedTarWriter(filename, compression)
	if err != nil {
		return nil, err
	}
	return &SingleTarWriter{
		todo : list.New(),
		quit : false,
		t: t,
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
			file, err := os.Open( filename )
			if err != nil {
				panic(err)
			}
			ff, err := file.Stat()
			if err != nil {
				panic(err)
			}
			s.t.AppendFileStream(ff.Name(), ff.Size(), file )
			s.t.Flush()
		}
	}
	s.t.Close()
}

func (s *SingleTarWriter) SetQuit() {
	s.quit = true
}

type ImageCompressedTarWriter struct {
	m sync.Mutex // synchronizes access to shared mutable state
	file *os.File
	tarWriter *tar.Writer
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
		err = fmt.Errorf("Unknown compression format: %s", compression)
	}

	tarWriter := tar.NewWriter(compressor)
	return &ImageCompressedTarWriter{
		tarWriter:    tarWriter,
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
	io.Copy(img.tarWriter, reader)
	reader.Close()
	return nil
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
		err = fmt.Errorf("Unknown compression format: %s", compression)
	}

	tarReader := tar.NewReader(compressor)
	return &ImageCompressedTarReader{
		tarReader:    tarReader,
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
			return nil, "" , 0, true, nil
		}
		if err != nil {
			return nil, "" , 0, false, err
		}
		if strings.HasPrefix(hdr.Name, hex) {
			return img.tarReader, hdr.Name, hdr.Size, false, nil
		}
	}
}

func (img *ImageCompressedTarReader) ReadFileStream(skip int) (io.Reader, string, int64, bool, error) {
	for i := 0 ; i < skip; i ++ {
		hdr, err := img.tarReader.Next()
		if hdr == nil {
			return nil, "" , 0, true, nil
		}
		if err != nil {
			return nil, "" , 0, false, err
		}
	}
	hdr, err := img.tarReader.Next()
	if hdr == nil {
		return nil, "" , 0, true, nil
	}
	if err != nil {
		return nil, "" , 0, false, err
	}
	return img.tarReader, hdr.Name, hdr.Size, false, nil
}