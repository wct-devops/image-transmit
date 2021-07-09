package core

import (
	"io"
	"os"
	"path/filepath"
	"container/list"
	log "github.com/cihub/seelog"
)

type LocalTemp struct {
	files    * list.List
	tempPath string
}

func NewLocalTemp(pathname string) *LocalTemp {
	_, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		os.MkdirAll(pathname, os.ModePerm)
	}
	return &LocalTemp{
		tempPath: pathname,
		files :   list.New(),
	}
}

func (t *LocalTemp) SavePath(path string) (string, error){
	fullPathName := filepath.Join( t.tempPath, path )
	t.files.PushBack(fullPathName)
	return fullPathName, os.MkdirAll(fullPathName, os.ModePerm)
}

func (t *LocalTemp) SaveFile(filename string, reader io.ReadCloser) (string, error)  {
	fullFilename := filepath.Join( t.tempPath, filename )
	file, err := os.Create(fullFilename)
	defer file.Close()
	if err != nil {
		return fullFilename, err
	}
	_, err = io.Copy(file, reader)
	reader.Close()
	if err == nil {
		t.files.PushBack(fullFilename)
	}
	return fullFilename, err
}

func (t *LocalTemp) Clean() {
	for e := t.files.Front(); e != nil; e = e.Next() {
		f := e.Value.(string)
		i, err := os.Stat(f)
		if err != nil {
			log.Errorf("Unexpect %v", err)
			continue
		}
		if i.IsDir() {
			os.RemoveAll(f)
		} else {
			os.Remove(f)
		}
	}
}
