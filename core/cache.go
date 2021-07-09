package core

import (
	"io/ioutil"
	"time"
	"sort"
	"os"
	"path/filepath"
	"io"
	"fmt"
)

type LocalCache struct {
	Pathname string `yaml:"pathname"`
	KeepDays int  `yaml:"keepdays"`
	KeepSize int `yaml:"keepsize"`
}

func NewLocalCache(pathname string, keepDays int, keepSize int) *LocalCache {
	_, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		os.MkdirAll(pathname, os.ModePerm)
	}
	return &LocalCache{
		Pathname: pathname,
		KeepDays: keepDays,
		KeepSize: keepSize,
	}
}

func (c *LocalCache) Match(blobName string, size int64) (bool, string) {
	files, _ := ioutil.ReadDir(c.Pathname)
	for _, f := range files {
		if (f.Name() == blobName && f.Size() == size ){
			return true, filepath.Join( c.Pathname, blobName)
		}
	}
	return false, ""
}

func (c *LocalCache) Reuse(blobName string) (io.ReadCloser, error) {
	filename := filepath.Join( c.Pathname, blobName)
	current := time.Now().Local()
	os.Chtimes(filename, current, current)
	file, err := os.Open( filename )
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *LocalCache) SaveStream(blobName string, reader io.Reader) (io.ReadCloser, io.WriteCloser) {
	file, err := os.Create( filepath.Join( c.Pathname, blobName ) )
	os.OpenFile(filepath.Join( c.Pathname, blobName ), os.O_CREATE, 777)
	if err != nil {
		fmt.Errorf("Create file error:%s , filename: %s", err, filepath.Join( c.Pathname, blobName )  )
		return ioutil.NopCloser(reader), nil
	}
	return ioutil.NopCloser(io.TeeReader(reader, file)), file
}

func (c *LocalCache) SaveFile(filename string, reader io.ReadCloser) (string, error)  {
	fullFilename := filepath.Join( c.Pathname, filename )
	file, err := os.Create(fullFilename)
	defer file.Close()
	if err != nil {
		return fullFilename, err
	}
	_, err = io.Copy(file, reader)
	reader.Close()
	return fullFilename, err
}

func (c *LocalCache) Clean() {
	files, _ := ioutil.ReadDir(c.Pathname)
	var deletedNum int
	if c.KeepDays > 0 {
		current := time.Now().Local().Second()
		sort.Slice(files, func(i, j int) bool { return files[i].ModTime().Unix() < files[j].ModTime().Unix() })
		for i, f := range files {
			if current - f.ModTime().Local().Second() > 14400 * c.KeepDays {
				os.Remove( filepath.Join( c.Pathname, f.Name() ))
			} else {
				deletedNum = i
				break
			}
		}
	}

	if c.KeepSize > 0 {
		var totalSize int64
		sort.Slice(files, func(i, j int) bool { return files[i].ModTime().Unix() > files[j].ModTime().Unix() })
		for i, f := range files {
			if i == len(files) - deletedNum  {
				break
			}
			totalSize = totalSize + f.Size()
			if totalSize > int64(c.KeepSize* 1024 * 1024 * 1024 ) {
				os.Remove( filepath.Join( c.Pathname, f.Name() ))
			}
		}
	}
}
