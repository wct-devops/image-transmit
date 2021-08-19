package core

import (
	"fmt"
	"time"
	"os"
	"runtime"
	"strconv"
	log "github.com/cihub/seelog"
	"io"
	"bytes"
	"github.com/pkg/errors"
)

type CtxLogger interface {
	Debug(logStr string)
	Info(logStr string)
	Error(logStr string)
	Errorf(string, ...interface{}) error
}

type CmdLogger struct {
}

type StdoutWrapper struct {
	buf *bytes.Buffer
	logger CtxLogger
}

func NewStdoutWrapper(logger CtxLogger) io.Writer {
	return &StdoutWrapper{
		buf : new(bytes.Buffer),
		logger: logger,
	}
}

func (r *StdoutWrapper) Write(p []byte) (int, error) {
	r.buf.Write(p)
	if bytes.Contains(p, []byte("\n")) {
		if runtime.GOOS == "windows" && r.logger != nil {
			r.logger.Info(r.buf.String())
		} else {
			log.Info(r.buf.String())
		}
		r.buf.Reset()
	}
	if runtime.GOOS == "windows" && r.logger != nil {
		return len(p), nil
	} else {
		return os.Stdout.Write(p)
	}
}

type StderrWrapper struct {
	buf *bytes.Buffer
	logger CtxLogger
}

func NewStderrWrapper(logger CtxLogger) io.Writer {
	return &StderrWrapper{
		buf : new(bytes.Buffer),
		logger: logger,
	}
}

func (r *StderrWrapper) Write(p []byte) (int, error) {
	r.buf.Write(p)
	if bytes.Contains(p, []byte("\n")) {
		if runtime.GOOS == "windows" && r.logger != nil {
			r.logger.Error(r.buf.String())
		} else {
			log.Error(r.buf.String())
		}
		r.buf.Reset()
	}
	if runtime.GOOS == "windows" && r.logger != nil {
		return len(p), nil
	} else {
		return os.Stderr.Write(p)
	}
}

func NewCmdLogger() CtxLogger {
	return &CmdLogger{}
}

func (t *CmdLogger) Debug(logStr string){
	log.Debug(logStr)
}

func (t *CmdLogger) Info(logStr string) {
	fmt.Println( time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr)
	log.Info(logStr)
}

func (t *CmdLogger) Error(logStr string) {
	os.Stderr.Write( []byte( time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\n" ))
	log.Error(logStr)
}

func (t *CmdLogger) Errorf(format string, args ...interface{}) error {
	var errStr string
	if len(args) > 0 {
		errStr = fmt.Sprintf(format, args)
	} else {
		errStr = format
	}
	os.Stderr.Write( []byte( time.Now().Format("[2006-01-02 15:04:05]") + " " + errStr + "\n" ))
	log.Error(errStr)
	return errors.New(errStr)
}

func FormatByteSize(sizeInByte int64) (size string) {
	if sizeInByte < 1024 {
		return fmt.Sprintf("%.1fB", float64(sizeInByte)/float64(1))
	} else if sizeInByte < (1024 * 1024) {
		return fmt.Sprintf("%.1fKB", float64(sizeInByte)/float64(1024))
	} else if sizeInByte < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.1fMB", float64(sizeInByte)/float64(1024*1024))
	} else if sizeInByte < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.1fGB", float64(sizeInByte)/float64(1024*1024*1024))
	} else if sizeInByte < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.1fTB", float64(sizeInByte)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.1fEB", float64(sizeInByte)/float64(1024*1024*1024*1024*1024))
	}
}

func FormatSeconds(seconds int64) string {
	day := seconds / (24 * 3600)
	hour := (seconds - day * 3600 * 24) / 3600
	minute := (seconds - day * 24 * 3600 - hour * 3600) / 60
	second := seconds - day * 24 * 3600 - hour * 3600 - minute * 60

	var str string
	if day > 0 {
		str = str + strconv.FormatInt(day,10) + I18n.Sprintf("D")
	}
	if hour > 0 {
		str = str + strconv.FormatInt(hour,10) + I18n.Sprintf("H")
	}
	if minute > 0 {
		str = str + strconv.FormatInt(minute,10) + I18n.Sprintf("M")
	}
	if second > 0 {
		str = str + strconv.FormatInt(second,10) + I18n.Sprintf("S")
	}
	return str
}

func InitLogger(logConfig []byte) {
	if len(logConfig) == 0 {
		logConfig = []byte(`
<seelog>
    <outputs formatid="main">
		<rollingfile type="size" filename="./data/log.txt" maxsize="10240000" maxrolls="5"/>
    </outputs>
    <formats>
        <format id="main" format="%Date(2006-01-02 15:04:05.000) [%File.%Line] %LEV %Msg%n"/>
    </formats>
</seelog>
`)
	}
	logger, _ := log.LoggerFromConfigAsBytes(logConfig)
	log.ReplaceLogger(logger)
}

