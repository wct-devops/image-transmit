package core

import (
	"io/ioutil"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"gopkg.in/yaml.v2"
)

type History struct {
	imageHistory map[string]string
	fileName     string
	hisChan      chan int
}

func NewHistory(fileName string) (*History, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	imageHistory := make(map[string]string)
	if len(file) > 0 {
		err = yaml.Unmarshal(file, imageHistory)
		if err != nil {
			return nil, err
		}
	}

	return &History{
		imageHistory: imageHistory,
		fileName:     fileName,
		hisChan:      make(chan int, 1),
	}, nil
}

func (h *History) Add(url string) {
	h.hisChan <- 1
	defer func() {
		<-h.hisChan
	}()
	h.imageHistory[url] = time.Now().Format("20060102150405")
	b, err := yaml.Marshal(h.imageHistory)
	if err != nil {
		log.Error(err)
	}
	err = ioutil.WriteFile(h.fileName, b, os.ModePerm)
	if err != nil {
		log.Error(err)
	}
}

func (h *History) Skip(url string) bool {
	h.hisChan <- 1
	defer func() {
		<-h.hisChan
	}()
	_, ok := h.imageHistory[url]
	if !ok {
		return false
	} else {
		return true
	}
}
