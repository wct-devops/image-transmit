package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/pkg/errors"
	. "github.com/wct-devops/image-transmit/core"
	"gopkg.in/yaml.v2"
)

func main() {
	MAINICO := string("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")
	mw := &MyMainWindow{}

	InitI18nPrinter("")
	var loggerCfg []byte
	if _, err := os.Stat("logCfg.xml"); err == nil {
		loggerCfg, _ = ioutil.ReadFile("logCfg.xml")
	} else if _, err := os.Stat(filepath.Join(HOME, "logCfg.xml")); err == nil {
		loggerCfg, _ = ioutil.ReadFile(filepath.Join(HOME, "logCfg.xml"))
	}
	InitLogger(loggerCfg)

	CONF = new(YamlCfg)

	var cfgFile []byte
	_, err := os.Stat("cfg.yaml")
	if err != nil && os.IsNotExist(err) {
		_, err = os.Stat(filepath.Join(HOME, "cfg.yaml"))
		if err != nil && os.IsNotExist(err) {
			log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		} else {
			cfgFile, err = ioutil.ReadFile(filepath.Join(HOME, "cfg.yaml"))
			if err != nil {
				log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
			}
		}
	} else {
		cfgFile, err = ioutil.ReadFile("cfg.yaml")
		if err != nil {
			log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		}
	}

	err = yaml.Unmarshal(cfgFile, CONF)

	if len(CONF.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			CONF.Compressor = "tar"
		} else {
			CONF.Compressor = "squashfs"
		}
	}

	if CONF.Compressor != "squashfs" {
		SQUASHFS = false
	}

	if len(CONF.Lang) > 1 {
		InitI18nPrinter(CONF.Lang)
	}

	if err != nil {
		walk.MsgBox(nil,
			I18n.Sprintf("Configuration File Error"),
			fmt.Sprint(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", err)),
			walk.MsgBoxIconStop)
		return
	}

	if len(CONF.SrcRepos) < 1 || len(CONF.DstRepos) < 1 {
		walk.MsgBox(nil,
			I18n.Sprintf("Configuration File Error"),
			I18n.Sprintf("Configuration File cfg.yaml incorrect, for instruction visit github.com/wct-devops/image-transmit"),
			walk.MsgBoxIconStop)
		return
	}

	if CONF.Interval > 0 {
		INTERVAL = CONF.Interval
	}

	mw.compressor = CONF.Compressor
	mw.lmIncrement = NewIncrementListModel()

	mw.singleFile = CONF.SingleFile
	mw.lmSingle = NewSingleListModel()

	mw.srcRepo = &CONF.SrcRepos[0]
	mw.dstRepo = &CONF.DstRepos[0]

	mw.lmSrcRepo = NewRepoListModel(CONF.SrcRepos)
	mw.lmDstRepo = NewRepoListModel(CONF.DstRepos)

	icon, _ := walk.NewIconFromImageForDPI(Base64ToImage(MAINICO), 96)

	MainWindow{
		AssignTo: &mw.mainWindow,
		Icon:     icon,
		Title:    I18n.Sprintf("Image Transmit-Ghang'e-WhaleCloud DevOps Team"),
		MinSize:  Size{Width: 600, Height: 400},
		Layout:   VBox{},
		//Icon: ico,
		Children: []Widget{
			Composite{
				Layout:    HBox{MarginsZero: true},
				MaxSize:   Size{Width: 0, Height: 20},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					Label{Text: I18n.Sprintf("Source:"), AssignTo: &mw.labelSrcRepo},
					ComboBox{
						AssignTo:              &mw.cbSrcRepo,
						Model:                 mw.lmSrcRepo,
						OnCurrentIndexChanged: mw.RepoCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  Destination:"), AssignTo: &mw.labelDstRepo},
					ComboBox{
						AssignTo:              &mw.cbDstRepo,
						Model:                 mw.lmDstRepo,
						OnCurrentIndexChanged: mw.RepoCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  MaxThreads:"), AssignTo: &mw.labelSrcRepo},
					LineEdit{
						MaxSize:  Size{Width: 15, Height: 0},
						MinSize:  Size{Width: 15, Height: 0},
						AssignTo: &mw.leMaxConn,
					},
					Label{Text: I18n.Sprintf("  Retries:"), AssignTo: &mw.labelSrcRepo},
					LineEdit{
						MaxSize:  Size{Width: 15, Height: 0},
						MinSize:  Size{Width: 15, Height: 0},
						AssignTo: &mw.leRetries,
					},
					Label{Text: I18n.Sprintf("  ArchiveMode:")},
					ComboBox{
						AssignTo:              &mw.cbIncrement,
						Model:                 mw.lmIncrement,
						MaxSize:               Size{Width: 50, Height: 0},
						MinSize:               Size{Width: 50, Height: 0},
						OnCurrentIndexChanged: mw.IncrementCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  SingleFile:")},
					ComboBox{
						MaxSize:               Size{Width: 40, Height: 0},
						MinSize:               Size{Width: 40, Height: 0},
						AssignTo:              &mw.cbSingle,
						Model:                 mw.lmSingle,
						OnCurrentIndexChanged: mw.SingleCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  LocalCache:")},
					Label{AssignTo: &mw.labelCache},
					HSpacer{},
				},
			},
			Composite{
				Layout:    HBox{MarginsZero: true},
				MaxSize:   Size{Width: 200, Height: 25},
				MinSize:   Size{Width: 200, Height: 25},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					PushButton{
						Text:      I18n.Sprintf("TRANSMIT"),
						AssignTo:  &mw.btnSync,
						MinSize:   Size{Width: 70, Height: 22},
						MaxSize:   Size{Width: 70, Height: 22},
						OnClicked: mw.Transmit,
					},
					PushButton{
						Text:      I18n.Sprintf("WATCH"),
						AssignTo:  &mw.btnWatch,
						MinSize:   Size{Width: 70, Height: 22},
						MaxSize:   Size{Width: 70, Height: 22},
						OnClicked: mw.Watch,
					},
					PushButton{
						Text:      I18n.Sprintf("DOWNLOAD"),
						AssignTo:  &mw.btnDownload,
						MinSize:   Size{Width: 70, Height: 22},
						MaxSize:   Size{Width: 70, Height: 22},
						OnClicked: mw.Download,
					},
					PushButton{
						Text:      I18n.Sprintf("UPLOAD"),
						Alignment: AlignHNearVNear,
						AssignTo:  &mw.btnUpload,
						MinSize:   Size{Width: 60, Height: 22},
						MaxSize:   Size{Width: 60, Height: 22},
						OnClicked: mw.Upload,
					},
					PushButton{
						Text:      I18n.Sprintf("CANCEL"),
						Alignment: AlignHNearVNear,
						AssignTo:  &mw.btnCancel,
						MinSize:   Size{Width: 60, Height: 22},
						MaxSize:   Size{Width: 60, Height: 22},
						OnClicked: func() {
							mw.ctx.CancelFunc()
							mw.ctx.Info(I18n.Sprintf("User cancel it"))
						},
					},
					PushButton{
						Text:      I18n.Sprintf("VERIFY"),
						Alignment: AlignHNearVNear,
						AssignTo:  &mw.btnTest,
						MinSize:   Size{Width: 60, Height: 22},
						MaxSize:   Size{Width: 60, Height: 22},
						OnClicked: mw.Verify,
					},
					Composite{
						Layout:    HBox{},
						MaxSize:   Size{Width: 0, Height: 22},
						MinSize:   Size{Width: 0, Height: 22},
						Alignment: AlignHNearVCenter,
						Children: []Widget{
							Label{Text: I18n.Sprintf("Status: "), AssignTo: &mw.labelStatusTitle},
							Label{Text: "-----------", AssignTo: &mw.labelStatus},
						},
					},
					HSpacer{},
				},
			},
			HSplitter{
				Children: []Widget{
					Composite{
						Layout:    VBox{MarginsZero: true},
						MaxSize:   Size{Width: 200, Height: 0},
						MinSize:   Size{Width: 200, Height: 0},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							Label{Text: I18n.Sprintf("Image List(seperated by lines): "), AssignTo: &mw.labelInput, Font: Font{Bold: true}},
							TextEdit{AssignTo: &mw.teInput, HScroll: true, VScroll: true, OnTextChanged: mw.updateWindowsNewLine},
						},
					},
					Composite{
						Layout:    VBox{MarginsZero: true},
						MaxSize:   Size{Width: 0, Height: 0},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							Label{Text: I18n.Sprintf("Log Output: "), AssignTo: &mw.labelOutput, Font: Font{Bold: true}},
							TextEdit{AssignTo: &mw.teOutput, ReadOnly: true, HScroll: false, VScroll: true, MaxLength: 10000000},
						},
					},
				},
			},
		},
	}.Create()

	titleBrush, err := walk.NewSolidColorBrush(walk.RGB(255, 245, 177))
	if err != nil {
		panic(err)
	}
	defer titleBrush.Dispose()

	statusBrush, err := walk.NewSolidColorBrush(walk.RGB(190, 245, 203))
	if err != nil {
		panic(err)
	}
	defer statusBrush.Dispose()

	/*
		mw.labelSrcRepo.SetBackground(titleBrush)
		mw.labelDstRepo.SetBackground(titleBrush)
		mw.labelInput.SetBackground(titleBrush)
		mw.labelOutput.SetBackground(titleBrush)
		mw.labelStatusTitle.SetBackground(titleBrush)
	*/

	mw.labelStatus.SetBackground(titleBrush)

	var lc *LocalCache
	if CONF.Cache.Pathname != "" {
		keepDays := 7
		keepSize := 10
		if CONF.Cache.KeepDays > 0 {
			keepDays = CONF.Cache.KeepDays
		}
		if CONF.Cache.KeepSize > 0 {
			keepSize = CONF.Cache.KeepSize
		}
		lc = NewLocalCache(CONF.Cache.Pathname, keepDays, keepSize)
		mw.labelCache.SetText(I18n.Sprintf("ON"))
	} else {
		mw.labelCache.SetText(I18n.Sprintf("OFF"))
	}

	lt := NewLocalTemp(TEMP_DIR)
	teLog := newGuiLogger(mw.teOutput)
	mw.ctx = NewTaskContext(teLog, lc, lt)

	if len(CONF.DingTalk) > 0 {
		mw.ctx.Notify = NewDingTalkWapper(CONF.DingTalk)
	}

	mw.ctx.Reset()

	if CONF.MaxConn > 0 {
		mw.leMaxConn.SetText(strconv.Itoa(CONF.MaxConn))
	} else {
		mw.leMaxConn.SetText(strconv.Itoa(runtime.NumCPU()))
	}

	if CONF.Retries > 0 {
		mw.leRetries.SetText(strconv.Itoa(CONF.Retries))
	} else {
		mw.leRetries.SetText("2")
	}

	mw.mainWindow.Show()

	go func() {
		mw.cbSrcRepo.SetCurrentIndex(0)
		mw.cbDstRepo.SetCurrentIndex(0)
		mw.cbIncrement.SetCurrentIndex(0)

		if SQUASHFS {
			mw.cbSingle.SetCurrentIndex(0)
			mw.cbSingle.SetEnabled(false)
		}

		for i, v := range mw.lmSingle.items {
			b, _ := strconv.ParseBool(v.value)

			if mw.singleFile == b {
				mw.cbSingle.SetCurrentIndex(i)
			}
		}

		for {
			mw.labelStatus.SetText(mw.ctx.GetStatus())
			//mw.teOutput.ScrollToCaret()
			time.Sleep(1 * time.Second)
		}
	}()
	mw.mainWindow.Run()
}

func Base64ToImage(str string) image.Image {
	ddd, _ := base64.StdEncoding.DecodeString(str)
	bbb := bytes.NewBuffer(ddd)
	m, _, _ := image.Decode(bbb)
	png.Encode(bbb, m)
	return m
}

func (mw *MyMainWindow) RepoCurrentIndexChanged() {
	if mw.cbSrcRepo.CurrentIndex() > -1 {
		mw.srcRepo = &mw.lmSrcRepo.items[mw.cbSrcRepo.CurrentIndex()].value
	}
	if mw.cbDstRepo.CurrentIndex() > -1 {
		mw.dstRepo = &mw.lmDstRepo.items[mw.cbDstRepo.CurrentIndex()].value
	}
}

func (mw *MyMainWindow) IncrementCurrentIndexChanged() {
	mw.increment, _ = strconv.ParseBool(mw.lmIncrement.items[mw.cbIncrement.CurrentIndex()].value)
}

func (mw *MyMainWindow) SingleCurrentIndexChanged() {
	mw.singleFile, _ = strconv.ParseBool(mw.lmSingle.items[mw.cbSingle.CurrentIndex()].value)
}

func (mw *MyMainWindow) updateWindowsNewLine() {
	input := mw.teInput.Text()
	input = strings.Replace(input, "\r\n", "\n", -1)
	input = strings.Replace(input, "\n", "\r\n", -1)
	mw.teInput.SetText(input)
}

type ListRepoItem struct {
	name  string
	value Repo
}

type ListRepoModel struct {
	walk.ListModelBase
	items []ListRepoItem
}

func (m *ListRepoModel) ItemCount() int {
	return len(m.items)
}

func (m *ListRepoModel) Value(index int) interface{} {
	return m.items[index].name
}

func NewRepoListModel(repos []Repo) *ListRepoModel {
	m := &ListRepoModel{items: make([]ListRepoItem, len(repos))}
	for i, v := range repos {
		if len(v.Name) > 0 {
			m.items[i] = ListRepoItem{name: v.Name, value: v}
		} else {
			if v.Repository != "" {
				m.items[i] = ListRepoItem{name: v.Registry + "-" + v.Repository, value: v}
			} else {
				m.items[i] = ListRepoItem{name: v.Registry, value: v}
			}
		}
	}
	return m
}

type ListStrItem struct {
	name  string
	value string
}

type ListStrModel struct {
	walk.ListModelBase
	items []ListStrItem
}

func (m *ListStrModel) ItemCount() int {
	return len(m.items)
}

func (m *ListStrModel) Value(index int) interface{} {
	return m.items[index].name
}

func NewIncrementListModel() *ListStrModel {
	m := &ListStrModel{items: make([]ListStrItem, 2)}
	m.items[0] = ListStrItem{name: I18n.Sprintf("FULL"), value: "false"}
	m.items[1] = ListStrItem{name: I18n.Sprintf("INCR"), value: "true"}
	return m
}

func NewSingleListModel() *ListStrModel {
	m := &ListStrModel{items: make([]ListStrItem, 2)}
	m.items[0] = ListStrItem{name: I18n.Sprintf("YES"), value: "true"}
	m.items[1] = ListStrItem{name: I18n.Sprintf("NO"), value: "false"}
	return m
}

type GuiLogger struct {
	te      *walk.TextEdit
	logChan chan int
}

func newGuiLogger(te *walk.TextEdit) CtxLogger {
	return &GuiLogger{
		te:      te,
		logChan: make(chan int, 1),
	}
}

func (l *GuiLogger) Info(logStr string) {
	l.logChan <- 1
	defer func() {
		<-l.logChan
	}()
	l.te.AppendText(time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\r\n")
	log.Info(logStr)
}

func (l *GuiLogger) Error(logStr string) {
	l.logChan <- 1
	defer func() {
		<-l.logChan
	}()
	l.te.AppendText(time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\r\n")
	log.Error(logStr)
}

func (l *GuiLogger) Debug(logStr string) {
	log.Debug(fmt.Sprint(logStr))
}

func (l *GuiLogger) Errorf(format string, args ...interface{}) error {
	l.logChan <- 1
	defer func() {
		<-l.logChan
	}()
	var errStr string
	if len(args) > 0 {
		errStr = fmt.Sprintf(format, args...)
	} else {
		errStr = format
	}

	l.te.AppendText(time.Now().Format("[2006-01-02 15:04:05]<ERROR>") + " " + errStr + "\r\n")
	log.Error(errStr)
	return errors.New(errStr)
}
