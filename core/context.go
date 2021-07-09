package core

import (
	"fmt"
	"time"
	"strconv"
	"path/filepath"
)

type TaskContext struct{
	log          CtxLogger
	download     int64
	kbDown       int64
	kbUp         int64
	secDown      int64
	secUp        int64
	secStart     int64
	secEnd       int64
	parallelism  int
	failedTask   int
	invalidTask  int
	totalTask    int
	waitTask     int
	statChan     chan int
	cancelled    bool
	Cache        *LocalCache
	Temp         *LocalTemp
	TarWriter    []*ImageCompressedTarWriter
	SingleWriter *SingleTarWriter
	CompMeta     *CompressionMetadata
	SquashfsTar  *SquashfsTar
}

func NewTaskContext(log CtxLogger, lc *LocalCache, lt *LocalTemp) *TaskContext {
	return &TaskContext{
		log:      log,
		statChan: make(chan int, 1),
		Cache:    lc,
		Temp:     lt,
	}
}

func (t *TaskContext) GetLogger() CtxLogger {
	return t.log
}

func (t *TaskContext) Info(logStr string) {
	t.log.Info(logStr)
}

func (t *TaskContext) Debug(logStr string) {
	t.log.Debug(logStr)
}

func (t *TaskContext) Error(logStr string) {
	t.log.Error(logStr)
}

func (t *TaskContext) Errorf(format string, args ...interface{}) error {
	if len(args) > 0 {
		return t.log.Errorf(format, args)
	} else {
		return t.log.Errorf(format)
	}
}

func (t *TaskContext) StatDown(size int64, seconds int64) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.kbDown = t.kbDown + size
	t.secDown = t.secDown + seconds
}

func (t *TaskContext) StatUp(size int64, seconds int64) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.kbUp = t.kbUp + size
	t.secUp = t.secUp + seconds
}

func (t *TaskContext) UpdateCurrentConn(n int) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.parallelism = t.parallelism + n
}

func (t *TaskContext) Reset() {
	t.kbDown = 0
	t.kbUp = 0
	t.secDown = 1
	t.secUp = 1
	t.secStart = 0
	t.secEnd = 0
	t.parallelism = 0
	t.failedTask = 0
	t.invalidTask = 0
	t.totalTask = 0
	t.waitTask = 0
	t.cancelled = false
	t.SingleWriter = nil
	t.TarWriter = nil
	t.CompMeta = nil
	t.SquashfsTar = nil
}

func (t *TaskContext) CloseTarWriter() {
	for _, i := range t.TarWriter {
		i.Close()
	}
	t.TarWriter = nil
}

func (t *TaskContext) CreateSquashfsTar(tempPath string, workPath string, squashfsFileName string) error {
	var err error
	t.SquashfsTar, err = NewSquashfsTar(tempPath, workPath, squashfsFileName)
	return err
}

func (t *TaskContext) CreateSingleWriter(pathname string, filename string, compression string) error{
	var err error
	tarName := filename + "." + compression
	t.Info(I18n.Sprintf("Create data file: %s", tarName ))
	t.CompMeta.AddDatafile(tarName, 0)
	t.SingleWriter, err = NewSingleTarWriter( filepath.Join(pathname, tarName) , compression)
	return  err
}

func (t *TaskContext) CreateCompressionMetadata(compressor string) error{
	var err error
	t.CompMeta, err = NewCompressionMetadata(compressor)
	return err
}

func (t *TaskContext) CreateTarWriter(pathname string, filename string, compression string, num int) error{
	t.TarWriter = make([]*ImageCompressedTarWriter, num)
	for i := range t.TarWriter {
		tarName := filename + "_" + strconv.Itoa(i) + "." + compression
		t.Info(I18n.Sprintf("Create data file: %s", tarName ))
		t.CompMeta.AddDatafile(tarName, 0)
		tar, err := NewImageCompressedTarWriter( filepath.Join(pathname, tarName), compression)
		if err != nil {
			return err
		}
		t.TarWriter[i] = tar
	}
	return nil
}

func (t *TaskContext) UpdateFailedTask(n int) {
	t.failedTask = n
}

func (t *TaskContext) UpdateInvalidTask(n int) {
	t.invalidTask = n
}

func (t *TaskContext) UpdateWaitTask(n int) {
	t.waitTask = n
}

func (t *TaskContext) UpdateTotalTask(n int) {
	t.totalTask = n
}

func (t *TaskContext) UpdateSecStart(n int64) {
	t.secStart = n
}

func (t *TaskContext) UpdateSecEnd(n int64) {
	t.secEnd = n
}

func (t *TaskContext) SetCancel() {
	t.cancelled = true
}

func (t *TaskContext) Cancel() bool {
	return t.cancelled
}

func  (t *TaskContext) GetStatus() string {
	var totalSec int64 = 0
	if t.secStart > 0 {
		if t.secEnd > 0 {
			totalSec = t.secEnd - t.secStart
		} else {
			totalSec = time.Now().Unix() - t.secStart
		}
	}
	return fmt.Sprintf(I18n.Sprintf("Invalid:%v Total:%v Success:%v Failed:%v Doing:%v Down:%s/s Up:%s/s, Total Down:%s Up:%s Time:%s",
		t.invalidTask, t.totalTask, t.totalTask - t.waitTask - t.failedTask - t.parallelism, t.failedTask,
		t.parallelism, FormatByteSize(t.kbDown/t.secDown), FormatByteSize(t.kbUp/t.secUp), FormatByteSize(t.kbDown), FormatByteSize(t.kbUp),FormatSeconds(totalSec)))
}
