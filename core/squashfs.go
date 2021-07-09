package core

import (
	"io"
	"strings"
	"fmt"
	"compress/gzip"
	"os"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
	"path/filepath"
	"os/exec"
	"runtime"
	sqfs "github.com/wangyumu/squashfs"
	"io/fs"
	log "github.com/cihub/seelog"
	"sync"
	"github.com/wangyumu/extract/v3"
	"context"
)

var (
	//EmptyLayerTarHex string = "a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
)

type SquashfsTar struct {
	tempPath string
	workPath string
	sfs      *sqfsFileSystem
	squashfsFileName string
}

func NewSquashfsTar(tempPath string, workPath string, squashfsFileName string) (*SquashfsTar,error){
	var sfs *sqfsFileSystem
	var err error
	if len(squashfsFileName) > 0 {
		sfs, err  = NewSqfsFileSystem(squashfsFileName, "")
		if err != nil {
			return nil, err
		}
	}
	return &SquashfsTar{
		tempPath: tempPath,
		workPath: workPath,
		sfs:      sfs,
		squashfsFileName: squashfsFileName,
	},nil
}

func MakeSquashfs(logger CtxLogger, path string, fs string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command( filepath.Join("squashfs", "mksquashfs"), path, fs)
	} else {
		cmd = exec.Command("mksquashfs", path, fs)
	}
	cmd.Stdout = NewStdoutWrapper(logger)
	cmd.Stderr = NewStderrWrapper(logger)
	return cmd.Run()
}

func TestSquashfs() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command( filepath.Join("squashfs", "mksquashfs"), "-version")
	} else {
		cmd = exec.Command("mksquashfs", "-version")
	}
	err := cmd.Run()
	if err == nil {
		return true
	} else {
		return false
	}
}

func TestTar() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command( filepath.Join("squashfs", "tar"), "--help")
	} else {
		cmd = exec.Command("tar", "--help")
	}
	err := cmd.Run()
	if err == nil {
		return true
	} else {
		return false
	}
}


func UnSquashfs(logger CtxLogger, path string, fs string, noCmd bool) error {
	if !TestSquashfs() || noCmd {
		file, err := os.Open(fs)
		if err != nil {
			return err
		}
		rdr, err := sqfs.NewSquashfsReader(file)
		if err != nil {
			return err
		}
		return rdr.ExtractTo(path)
	} else {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command(filepath.Join("squashfs", "unsquashfs"), "-d", path, fs )
		} else {
			cmd = exec.Command("unsquashfs", "-d", path, fs)
		}
		cmd.Stdout = NewStdoutWrapper(logger)
		cmd.Stderr = NewStderrWrapper(logger)
		return cmd.Run()
	}
}

func (w *SquashfsTar) AppendFileStream(blobName string, size int64, reader io.ReadCloser) error {
	if strings.HasSuffix(blobName, ".tar.gz") {
		err := w.DisassembleTarStream(strings.TrimSuffix(blobName, ".tar.gz"), size, reader)
		reader.Close()
		if err != nil {
			return err
		}
	} else {
		file, _ := os.Create(w.fullPathName(blobName))
		io.Copy(file, reader)
		file.Close()
		reader.Close()
	}
	return nil
}

func (w *SquashfsTar) GetFileStream(hex string) (io.Reader, error) {
	var reader io.Reader
	if w.sfs != nil {
		if _, err := w.sfs.Stat(hex + ".raw") ; err != nil && os.IsNotExist(err) {
			r, err := w.AssembleTarStream(hex)
			if err != nil {
				return nil, err
			}
			reader = r
		} else {
			file, err := w.sfs.Get(hex + ".raw")
			if err != nil {
				return nil, err
			}
			reader = file
		}
	} else {
		if _, err := os.Stat(w.fullPathName(hex + ".raw")); err != nil && os.IsNotExist( err ) {
			r, err := w.AssembleTarStream( hex )
			if err != nil {
				return nil, err
			}
			reader = r
		} else {
			file, err := os.Open(w.fullPathName( hex + ".raw"))
			if err != nil {
				return nil, err
			}
			reader = file
		}
	}
	return reader,nil
}

func (w *SquashfsTar) DisassembleTarStream(hex string, size int64, reader io.ReadCloser) error {
	compressor, _ := gzip.NewReader(reader)
	mf, err := os.OpenFile(w.fullPathName(hex + "_tar-split.json.gz"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer mf.Close()
	mfz := gzip.NewWriter(mf)
	defer mfz.Close()
	metaPacker := storage.NewJSONPacker(mfz)
	rdr, err := asm.NewInputTarStream(compressor, metaPacker, nil)
	if err != nil {
		return fmt.Errorf("TarSplit json write failed: %v", err)
	}
	/*
	arch := archive.NewDefaultArchiver()
	options := &archive.TarOptions{
		UIDMaps: arch.IDMapping.UIDs(),
		GIDMaps: arch.IDMapping.GIDs(),
	}
	return arch.Untar(rdr, w.fullPathName(hex), options)
	*/
	//

	if runtime.GOOS == "windows" {
		return extract.Tar(context.Background(), rdr, w.fullPathName(hex), nil)
	} else {
		os.MkdirAll(w.fullPathName(hex), os.ModePerm)
		cmd := exec.Command("tar", "-C", w.fullPathName(hex), "-x", "--no-same-owner", "--no-same-permissions", "--no-xattrs", "--warning=no-timestamp", "--no-selinux", "--no-acls")
		cmdWriter, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		wait := new(sync.Mutex)
		go func(){
			wait.Lock()
			defer wait.Unlock()
			cmd.Stdout = NewStdoutWrapper(nil)
			cmd.Stderr = NewStderrWrapper(nil)
			err = cmd.Run()
			if err != nil {
				log.Error(err)
			}
		}()
		io.Copy(cmdWriter, rdr)
		cmdWriter.Close()
		mfz.Close()
		mf.Close()
		wait.Lock()
		defer wait.Unlock()
		return err
	}
}

func (w *SquashfsTar) AssembleTarStream(hex string) (io.Reader, error) {
	pr, pw := io.Pipe()
	go func() {
		gw := gzip.NewWriter(pw)
		r,err := w.TarSplitReader(hex)
		if err != nil {
			log.Errorf("Unexpected err: %v", err)
			return
		}
		metaUnPacker := storage.NewJSONUnpacker(r)
		var fg storage.FileGetter
		var layerfs *sqfsFileSystem
		if w.sfs != nil {
			layerfs, err  := NewSqfsFileSystem(w.squashfsFileName, hex)
			if err != nil {
				log.Errorf("Create %s fs failed: %v",hex , err)
				return
			}
			fg = layerfs.GetFileGetter()
		} else {
			fg = storage.NewPathFileGetter(w.fullPathName(hex))
		}
		err = asm.WriteOutputTarStream(fg, metaUnPacker, gw)
		if err != nil {
			log.Errorf("Unexpected err: %v", err)
			gw.Close()
			pw.CloseWithError(err)
		} else {
			gw.Close()
			pw.Close()
		}
		if layerfs != nil {
			layerfs.Close()
		}
	}()
	return pr,nil
}

func (w *SquashfsTar) TarSplitReader(hex string) (io.ReadCloser, error) {
	var fz io.ReadCloser
	var err error
	if w.sfs != nil {
		fz, err = w.sfs.Get(hex + "_tar-split.json.gz")
	} else {
		fz, err = os.Open(filepath.Join(w.tempPath, w.workPath, hex + "_tar-split.json.gz"))
	}
	if err != nil {
		return nil, err
	}
	f, err := gzip.NewReader(fz)
	if err != nil {
		fz.Close()
		return nil, err
	}
	return NewReadCloserWrapper(f, func() error {
		f.Close()
		return fz.Close()
	}), nil
}

func (w *SquashfsTar) fullPathName(path string) string {
	return filepath.Join(w.tempPath, w.workPath, path)
}

func NewSqfsFileSystem(sqfsFile string, home string) (*sqfsFileSystem, error) {
	file, err := os.Open(sqfsFile)
	if err != nil {
		return nil, err
	}
	rdr, err := sqfs.NewSquashfsReader(file)
	if err != nil {
		return nil, err
	}
	return &sqfsFileSystem{file: file,  fs: rdr.FS, home: home}, nil
}

type sqfsFileSystem struct {
	file *os.File
	fs sqfs.FS
	home string
}

func (sfs *sqfsFileSystem) Get(filename string) (io.ReadCloser, error) {
	if len(sfs.home) > 0 {
		return sfs.fs.Open(sfs.home + "/" + filename)
	}
	return sfs.fs.Open(filename)
}

func (sfs *sqfsFileSystem) Stat(filename string) (fs.FileInfo, error) {
	if len(sfs.home) > 0 {
		return sfs.fs.Stat(sfs.home + "/" + filename)
	}
	return sfs.fs.Stat(filename)
}

func (sfs *sqfsFileSystem) GetFileGetter() (storage.FileGetter) {
	return sfs
}

func (sfs *sqfsFileSystem) Close() {
	sfs.file.Close()
}

type writeCloserWrapper struct {
	io.Writer
	closer func() error
}

func (r *writeCloserWrapper) Close() error {
	return r.closer()
}

func NewWriteCloserWrapper(r io.Writer, closer func() error) io.WriteCloser {
	return &writeCloserWrapper{
		Writer: r,
		closer: closer,
	}
}

type readCloserWrapper struct {
	io.Reader
	closer func() error
}

func (r *readCloserWrapper) Close() error {
	return r.closer()
}

func NewReadCloserWrapper(r io.Reader, closer func() error) io.ReadCloser {
	return &readCloserWrapper{
		Reader: r,
		closer: closer,
	}
}