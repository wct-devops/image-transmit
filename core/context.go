package core

import (
	"fmt"
	"time"
	"strconv"
	"path/filepath"
	"context"
)

type TaskContext struct{
	log          CtxLogger
	download     int64
	byteDown     int64
	byteUp       int64
	timeDown     time.Duration
	timeUp       time.Duration
	secStart     int64
	secEnd       int64
	parallelism  int
	failedTask   int
	invalidTask  int
	totalTask    int
	waitTask     int
	statChan     chan int
	Cache        *LocalCache
	Temp         *LocalTemp
	History      *History
	TarWriter    []*ImageCompressedTarWriter
	SingleWriter *SingleTarWriter
	CompMeta     *CompressionMetadata
	SquashfsTar  *SquashfsTar
	Context      context.Context
	CancelFunc   context.CancelFunc
	Notify       Notify
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
	return t.log.Errorf(format, args...)
}

func (t *TaskContext) StatDown(size int64, duration time.Duration) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.byteDown = t.byteDown + size
	t.timeDown = t.timeDown + duration
}

func (t *TaskContext) StatUp(size int64, duration time.Duration) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.byteUp = t.byteUp + size
	t.timeUp = t.timeUp + duration
}

func (t *TaskContext) UpdateCurrentConn(n int) {
	t.statChan <- 1
	defer func() {
		<-t.statChan
	}()
	t.parallelism = t.parallelism + n
}

func (t *TaskContext) Reset() {
	t.byteDown = 0
	t.byteUp = 0
	t.timeDown = 1
	t.timeUp = 1
	t.secStart = 0
	t.secEnd = 0
	t.parallelism = 0
	t.failedTask = 0
	t.invalidTask = 0
	t.totalTask = 0
	t.waitTask = 0
	t.SingleWriter = nil
	t.TarWriter = nil
	t.CompMeta = nil
	t.SquashfsTar = nil
	t.Context, t.CancelFunc = context.WithCancel(context.Background())
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

func (t *TaskContext) GetTotalTask() int {
	return t.totalTask
}

func (t *TaskContext) Cancel() bool {
	return t.Context.Err() != nil
}

func (t *TaskContext) UpdateSecStart(n int64) {
	t.secStart = n
}

func (t *TaskContext) UpdateSecEnd(n int64) {
	t.secEnd = n
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
	return fmt.Sprintf(I18n.Sprintf("Invalid:%v All:%v OK:%v Err:%v Doing:%v Speed:^%s/s v%s/s Total:^%s v%s Time:%s",
		t.invalidTask, t.totalTask, t.totalTask - t.waitTask - t.failedTask - t.parallelism, t.failedTask,
		t.parallelism, FormatByteSize(int64(float64(t.byteDown)/(float64(t.timeDown)/float64(time.Second)))), FormatByteSize(int64(float64(t.byteUp)/(float64(t.timeUp)/float64(time.Second)))), FormatByteSize(t.byteDown), FormatByteSize(t.byteUp),FormatSeconds(totalSec)))
}
