package main

import (
	. "github.com/wct-devops/image-transmit/core"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"io/ioutil"
	"strconv"
	"fmt"
	"time"
	"strings"
	"encoding/base64"
	"bytes"
	"image"
	"gopkg.in/yaml.v2"
	"image/png"
	"os"
	"path/filepath"
	log "github.com/cihub/seelog"
	"runtime"
)

var (
	home	   = "data"
	tempDir    = filepath.Join(home, "temp")
	squashfs   = true
	conf	    *YamlCfg
)

type Repo struct {
	Name       string `yaml:"name,omitempty"`
	User       string `yaml:"user"`
	Registry   string `yaml:"registry"`
	Password   string `yaml:"password"`
	Repository string `yaml:"repository,omitempty"`
}

type YamlCfg struct {
	SrcRepos [] Repo `yaml:"source,omitempty"`
	DstRepos [] Repo `yaml:"target,omitempty"`
	MaxConn int `yaml:"maxconn,omitempty"`
	Retries int `yaml:"retries,omitempty"`
	SingleFile bool `yaml:"singlefile,omitempty"`
	Compressor string `yaml:"compressor,omitempty"`
	Squashfs string `yaml:"squashfs,omitempty"`
	Cache LocalCache `yaml:"cache,omitempty"`
	Lang string  `yaml:"lang,omitempty"`
	KeepTemp bool `yaml:"keeptemp,omitempty"`
	OutPrefix string  `yaml:"outprefix,omitempty"`
}

func main() {
	MAINICO := string("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")
	mw := &MyMainWindow{}

	InitI18nPrinter("")
	var loggerCfg []byte
	if _, err := os.Stat( "logCfg.xml"); err == nil {
		loggerCfg, _ = ioutil.ReadFile("logCfg.xml")
	} else if _, err := os.Stat( filepath.Join(home, "logCfg.xml")); err == nil {
		loggerCfg, _ = ioutil.ReadFile(filepath.Join(home, "logCfg.xml"))
	}
	InitLogger(loggerCfg)

	conf = new(YamlCfg)

	var cfgFile []byte
	_, err := os.Stat( "cfg.yaml")
	if err != nil && os.IsNotExist(err) {
		_, err = os.Stat( filepath.Join(home,"cfg.yaml"))
		if err != nil && os.IsNotExist(err) {
			log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		} else {
			cfgFile, err = ioutil.ReadFile(filepath.Join(home,"cfg.yaml"))
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

	cfgFile, err = ioutil.ReadFile("cfg.yaml")
	if err != nil {
		log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
	}
	err = yaml.Unmarshal(cfgFile, conf)

	if len(conf.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			conf.Compressor = "tar"
		} else {
			conf.Compressor = "squashfs"
		}
	}

	if conf.Compressor != "squashfs"{
		squashfs = false
	}

	if len(conf.Lang) >1 {
		InitI18nPrinter(conf.Lang)
	}

	if err != nil {
		walk.MsgBox(nil,
			I18n.Sprintf("Configuration File Error"),
			fmt.Sprintf(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", err)),
			walk.MsgBoxIconStop)
		return
	}

	if len(conf.SrcRepos) < 1 || len(conf.DstRepos) < 1 {
		walk.MsgBox(nil,
			I18n.Sprintf("Configuration File Error"),
			I18n.Sprintf("Configuration File cfg.yaml incorrect, for instruction visit github.com/wct-devops/image-transmit"),
			walk.MsgBoxIconStop)
		return
	}

	mw.compressor = conf.Compressor
	mw.lmIncrement = NewIncrementListModel()

	mw.singleFile = conf.SingleFile
	mw.lmSingle = NewSingleListModel()

	mw.srcRepo = &conf.SrcRepos[0]
	mw.dstRepo = &conf.DstRepos[0]

	mw.lmSrcRepo = NewRepoListModel(conf.SrcRepos)
	mw.lmDstRepo = NewRepoListModel(conf.DstRepos)

	icon, _ := walk.NewIconFromImageForDPI(Base64ToImage(MAINICO), 96)

	MainWindow{
		AssignTo: &mw.mainWindow,
		Icon: icon,
		Title: I18n.Sprintf("Image Transmit-DragonBoat-WhaleCloud DevOps Team"),
		MinSize: Size{600, 400},
		Layout:  VBox{},
		//Icon: ico,
		Children: []Widget{
			Composite{
				Layout:  HBox{MarginsZero: true},
				MaxSize: Size{0, 20},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					Label{Text: I18n.Sprintf("Source:"), AssignTo: &mw.labelSrcRepo},
					ComboBox{
						AssignTo: &mw.cbSrcRepo,
						Model:    mw.lmSrcRepo,
						OnCurrentIndexChanged: mw.RepoCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  Destination:"), AssignTo: &mw.labelDstRepo},
					ComboBox{
						AssignTo: &mw.cbDstRepo,
						Model:    mw.lmDstRepo,
						OnCurrentIndexChanged: mw.RepoCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  MaxThreads:"), AssignTo: &mw.labelSrcRepo},
					LineEdit{
						MaxSize: Size{15, 0},
						MinSize: Size{15, 0},
						AssignTo: &mw.leMaxConn,
					},
					Label{Text: I18n.Sprintf("  Retries:"), AssignTo: &mw.labelSrcRepo},
					LineEdit{
						MaxSize: Size{15, 0},
						MinSize: Size{15, 0},
						AssignTo: &mw.leRetries,
					},
					Label{Text: I18n.Sprintf("  ArchiveMode:")},
					ComboBox{
						AssignTo: &mw.cbIncrement,
						Model:    mw.lmIncrement,
						MaxSize: Size{50, 0},
						MinSize: Size{50, 0},
						OnCurrentIndexChanged: mw.IncrementCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  SingleFile:")},
					ComboBox{
						MaxSize: Size{40, 0},
						MinSize: Size{40, 0},
						AssignTo: &mw.cbSingle,
						Model:    mw.lmSingle,
						OnCurrentIndexChanged: mw.SingleCurrentIndexChanged,
					},
					Label{Text: I18n.Sprintf("  LocalCache:")},
					Label{AssignTo: &mw.labelCache},
					HSpacer{},
				},
			},
			Composite{
				Layout:  HBox{MarginsZero: true},
				MaxSize: Size{200, 25},
				MinSize: Size{200, 25},
				Alignment: AlignHNearVNear,
				Children: []Widget{
					PushButton{
						Text: I18n.Sprintf("TRANSMIT"),
						AssignTo: &mw.btnSync,
						MinSize: Size{70, 22},
						MaxSize: Size{70, 22},
						OnClicked: func() {
							imgList := mw.getInputList()
							if imgList==nil || len(imgList) < 1 || !mw.BeginAction() {
								return
							}

							c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
							if (err!= nil){
								walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
								return
							}

							if imgList != nil {
								go func() {
									mw.btnSync.SetEnabled(false)
									defer mw.btnSync.SetEnabled(true)
									for _, urlList := range imgList {
										if mw.ctx.Cancel() {
											mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
											return
										}
										if mw.dstRepo.Repository != "" {
											c.GenerateOnlineTask(mw.srcRepo.Registry+"/" + strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password,
												mw.dstRepo.Registry + "/" + mw.dstRepo.Repository + "/" + urlList[len(urlList)-1], mw.dstRepo.User, mw.dstRepo.Password)
										} else {
											c.GenerateOnlineTask(mw.srcRepo.Registry+"/"+strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password,
												mw.dstRepo.Registry+"/"+ strings.Join(urlList, "/"), mw.dstRepo.User, mw.dstRepo.Password)
										}
									}
									mw.ctx.UpdateTotalTask(c.TaskLen())
									c.Run()
									mw.EndAction()
								}()
							}
						},
					},
					PushButton{
						Text: I18n.Sprintf("DOWNLOAD"),
						AssignTo: &mw.btnDownload,
						MinSize: Size{70, 22},
						MaxSize: Size{70, 22},
						OnClicked: func() {
							imgList := mw.getInputList()
							if imgList==nil || len(imgList) < 1 || !mw.BeginAction() {
								return
							}

							if mw.maxConn > len(imgList) {
								mw.maxConn = len(imgList)
							}
							c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
							if (err!= nil){
								walk.MsgBox(mw.mainWindow,I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
								return
							}

							pathname := filepath.Join(home, time.Now().Format("20060102"))
							_, err = os.Stat(pathname)
							if os.IsNotExist(err) {
								os.MkdirAll(pathname, os.ModePerm)
							}

							var workName string
							if mw.increment {
								workName = time.Now().Format("img_incr_200601021504")
							} else {
								workName = time.Now().Format("img_full_200601021504")
							}

							if len(conf.OutPrefix) > 0 {
								workName = conf.OutPrefix + "_" + workName
							}

							mw.ctx.CreateCompressionMetadata(mw.compressor)

							if mw.increment {
								dlg := new(walk.FileDialog)
								dlg.Title = I18n.Sprintf("Please select a history image meta file for increment")
								dlg.Filter = I18n.Sprintf("Image meta file (*meta.yaml)|*meta.yaml|all (*.*)|*.*")
								dlg.InitialDirPath = "."

								if ok, err := dlg.ShowOpen(mw.mainWindow); err != nil {
									//Error : File Open\r\n")
									mw.ctx.Errorf(I18n.Sprintf("Choose File Failed: %v", err))
									return
								} else if !ok { // Cancel
									return
								}
								mw.ctx.Info(I18n.Sprintf("Selected the history image meta file: %s", dlg.FilePath))
								b, err := ioutil.ReadFile(dlg.FilePath)
								if err != nil {
									walk.MsgBox(mw.mainWindow,I18n.Sprintf("ERROR"),
										fmt.Sprintf(I18n.Sprintf("Open file failed: %v"), err), walk.MsgBoxIconStop)
									return
								}
								cm := new(CompressionMetadata)
								err = yaml.Unmarshal(b, cm)
								if err != nil {
									walk.MsgBox(mw.mainWindow,I18n.Sprintf("Meta file error"),
										fmt.Sprintf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err)),  walk.MsgBoxIconStop)
									return
								}
								for k := range cm.Blobs {
									mw.ctx.CompMeta.BlobDone(k, fmt.Sprintf("https://last.img/skip/it:%s",filepath.Base(dlg.FilePath)))
								}
							}

							if squashfs {
								mw.ctx.Temp.SavePath(workName)
								mw.ctx.CreateSquashfsTar(tempDir, workName, "")
							} else {
								if mw.singleFile {
									mw.ctx.CreateSingleWriter(pathname, workName, mw.compressor)
								} else {
									mw.ctx.CreateTarWriter(pathname, workName, mw.compressor, mw.maxConn)
								}
							}

							go func() {
								mw.btnDownload.SetEnabled(false)
								for _, urlList := range imgList {
									if mw.ctx.Cancel() {
										mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
										return
									}
									c.GenerateOfflineDownTask(mw.srcRepo.Registry+"/" + strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password )
								}
								mw.ctx.UpdateTotalTask(c.TaskLen())
								c.Run()
								if mw.ctx.SingleWriter != nil {
									time.Sleep(1 * time.Second)
									mw.ctx.SingleWriter.SetQuit()
								} else {
									mw.ctx.CloseTarWriter()
								}

								if mw.ctx.SingleWriter == nil {
									if mw.ctx.SquashfsTar != nil {
										mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
										err := MakeSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, workName+ ".squashfs"))
										mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress End"))
										if err != nil {
											mw.ctx.Error(I18n.Sprintf("Mksquashfs compress failed with %v", err))
											walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
												I18n.Sprintf("Mksquashfs compress failed with %v", err), walk.MsgBoxIconStop)
											return
										} else {
											mw.ctx.CompMeta.AddDatafile( workName + ".squashfs", 0)
										}
									}
									mw.StatDatafiles(pathname, workName)
									mw.EndAction()
									mw.btnDownload.SetEnabled(true)
								}
							}()
							if mw.ctx.SingleWriter != nil {
								go func() {
									mw.ctx.SingleWriter.Run()
									mw.StatDatafiles(pathname, workName)
									mw.EndAction()
									mw.btnDownload.SetEnabled(true)
								}()
							}
						},
					},
					PushButton{
						Text: I18n.Sprintf("UPLOAD"),
						Alignment: AlignHNearVNear,
						AssignTo: &mw.btnUpload,
						MinSize: Size{60, 22},
						MaxSize: Size{60, 22},
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = I18n.Sprintf("Please choose an image meta file to upload")
							dlg.Filter = I18n.Sprintf("Image meta file (*meta.yaml)|*meta.yaml|all (*.*)|*.*")
							dlg.InitialDirPath = "."

							if ok, err := dlg.ShowOpen(mw.mainWindow); err != nil {
								//Error : File Open\r\n")
								mw.ctx.Errorf(I18n.Sprintf("Choose File Failed:%s"), err)
								return
							} else if !ok { // Cancel
								return
							}
							mw.ctx.Info(I18n.Sprintf("Selected image meta file to upload: %s", dlg.FilePath))
							b, err := ioutil.ReadFile(dlg.FilePath)
							if err != nil {
								walk.MsgBox(mw.mainWindow,I18n.Sprintf("ERROR"),
									fmt.Sprintf(I18n.Sprintf("Open file failed: %v", err)), walk.MsgBoxIconStop)
								return
							}
							cm := new(CompressionMetadata)
							yaml.Unmarshal(b, cm)

							pathname := filepath.Dir(dlg.FilePath)

							for k, v := range cm.Datafiles {
								f,err := os.Stat( filepath.Join(pathname, k) )
								if err != nil && os.IsNotExist(err) {
									mw.ctx.Errorf(I18n.Sprintf("Datafile %s missing", filepath.Join(pathname, k)))
									walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
										I18n.Sprintf("Some data files missing, please check the teLog"), walk.MsgBoxIconStop)
									return
								} else if f.Size() != v {
									mw.ctx.Errorf(I18n.Sprintf("Datafile %s mismatch in size, origin: %v, now: %v", filepath.Join(pathname, k), v, f.Size()))
									walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
										I18n.Sprintf("Some data files missing, please check the teLog"), walk.MsgBoxIconStop)
									return
								}
							}

							var srcImgUrlList []string
							for k := range cm.Manifests {
								srcImgUrlList = append(srcImgUrlList, k)
							}
							mw.teInput.SetText(strings.Join(srcImgUrlList,"\r\n"))
							text := fmt.Sprintf(I18n.Sprintf("Total %v images found, if need confirm, you can cancel and check the list in the left edit box", len(cm.Manifests)))

							//1-OK 2-Cancel
							if 1 == walk.MsgBox(mw.mainWindow, I18n.Sprintf("Start transmit now ?"), text, walk.MsgBoxOKCancel) {

								go func() {
									mw.BeginAction()
									mw.ctx.CompMeta = cm
									imgList := mw.getInputList()

									if mw.ctx.CompMeta.Compressor == "squashfs" {
										var filename string
										for k,_ := range cm.Datafiles {
											filename = k
										}
										workName := strings.TrimSuffix(filename , ".squashfs")
										if !TestSquashfs() || strings.Contains(conf.Squashfs, "stream" )     {
											err = mw.ctx.CreateSquashfsTar(tempDir, workName,  filepath.Join(pathname, filename) )
											if err != nil {
												walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
													I18n.Sprintf("Unsquashfs uncompress failed with %v", err), walk.MsgBoxIconStop)
												return
											}
										} else {
											mw.ctx.CreateSquashfsTar(tempDir, workName, "")
											mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
											if strings.Contains(conf.Squashfs, "nocmd" ) {
												err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName) , filepath.Join(pathname, filename), true)
											} else {
												err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName) , filepath.Join(pathname, filename), false)
												mw.ctx.Temp.SavePath(workName)
											}
											mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress End"))
											if err != nil {
												mw.ctx.Error(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
												walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
													I18n.Sprintf("Unsquashfs uncompress failed with %v", err), walk.MsgBoxIconStop)
												return
											}
										}
									}

									c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
									if (err != nil) {
										walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
										return
									}

									mw.btnUpload.SetEnabled(false)
									defer mw.btnUpload.SetEnabled(true)

									for i , urlList := range imgList {
										if mw.ctx.Cancel() {
											mw.ctx.Errorf("User cancelled...")
											return
										}
										if mw.dstRepo.Repository != "" {
											c.GenerateOfflineUploadTask( srcImgUrlList[i], mw.dstRepo.Registry + "/" + mw.dstRepo.Repository + "/" + urlList[len(urlList)-1], pathname, mw.dstRepo.User, mw.dstRepo.Password)
										} else {
											c.GenerateOfflineUploadTask( srcImgUrlList[i], mw.dstRepo.Registry + "/" + strings.Join(urlList, "/"), pathname, mw.dstRepo.User, mw.dstRepo.Password)
										}
									}

									mw.ctx.UpdateTotalTask(c.TaskLen())
									c.Run()
									mw.EndAction()
								}()
							}
						},
					},
					PushButton{
						Text: I18n.Sprintf("CANCEL"),
						Alignment: AlignHNearVNear,
						AssignTo: &mw.btnCancel,
						MinSize: Size{60, 22},
						MaxSize: Size{60, 22},
						OnClicked: func() {
							mw.ctx.SetCancel()
							mw.ctx.Info(I18n.Sprintf("User cancel it"))
						},
					},
					PushButton{
						Text: I18n.Sprintf("VERIFY"),
						Alignment: AlignHNearVNear,
						AssignTo: &mw.btnTest,
						MinSize: Size{60, 22},
						MaxSize: Size{60, 22},
						OnClicked: func() {
							imgList := mw.getInputList()
							if imgList != nil {
								var text = I18n.Sprintf( "Image List" ) + ":\r\n"
								if mw.srcRepo.Registry!= "" {
									text = text +I18n.Sprintf( "Source Repository" ) + ":\r\n"
									for _,urlList := range imgList {
										text = text + mw.srcRepo.Registry + "/" + strings.Join(urlList,"/") + "\r\n"
									}
								}
								if mw.dstRepo.Registry!= "" {
									text = text + I18n.Sprintf( "Destination Repository" ) + ":\r\n"
									for _,urlList := range imgList {
										if mw.dstRepo.Repository != "" {
											text = text + mw.dstRepo.Registry + "/" + mw.dstRepo.Repository+ "/" + urlList[len(urlList)-1] + "\r\n"
										} else {
											text = text + mw.dstRepo.Registry + "/" + strings.Join(urlList, "/") + "\r\n"
										}
									}
								}
								mw.ctx.Info(text)
							}
						},
					},
					Composite{
						Layout:  HBox{},
						MaxSize: Size{0, 22},
						MinSize: Size{0, 22},
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
						Layout:  VBox{MarginsZero: true},
						MaxSize: Size{200, 0},
						MinSize: Size{200, 0},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							Label{Text: I18n.Sprintf("Image List(seperated by lines): "), AssignTo: &mw.labelInput, Font: Font{ Bold: true} },
							TextEdit{AssignTo: &mw.teInput, HScroll:true, VScroll:true,  OnTextChanged: mw.updateWindowsNewLine},
						},
					},
					Composite{
						Layout:  VBox{MarginsZero: true},
						MaxSize: Size{0, 0},
						Alignment: AlignHNearVNear,
						Children: []Widget{
							Label{Text: I18n.Sprintf("Log Output: "), AssignTo: &mw.labelOutput , Font: Font{ Bold: true}},
							TextEdit{AssignTo: &mw.teOutput, ReadOnly: true, HScroll:false, VScroll:true, MaxLength:10000000},
						},
					},
				},
			},
		},
	}.Create()

	titleBrush, err := walk.NewSolidColorBrush( walk.RGB(255,245,177) )
	if err != nil {
		panic(err)
	}
	defer titleBrush.Dispose()

	statusBrush, err := walk.NewSolidColorBrush( walk.RGB(190,245,203) )
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
	if conf.Cache.Pathname != "" {
		keepDays := 7
		keepSize := 10
		if conf.Cache.KeepDays > 0 {
			keepDays = conf.Cache.KeepDays
		}
		if conf.Cache.KeepSize > 0 {
			keepSize = conf.Cache.KeepSize
		}
		lc = NewLocalCache(conf.Cache.Pathname , keepDays, keepSize)
		mw.labelCache.SetText(I18n.Sprintf("ON"))
	} else {
		mw.labelCache.SetText(I18n.Sprintf("OFF"))
	}

	lt := NewLocalTemp(tempDir)
	teLog := newGuiLogger(mw.teOutput)

	mw.ctx = NewTaskContext(teLog, lc, lt)
	mw.ctx.Reset()

	if conf.MaxConn > 0 {
		mw.leMaxConn.SetText(strconv.Itoa(conf.MaxConn))
	} else {
		mw.leMaxConn.SetText(strconv.Itoa(runtime.NumCPU()))
	}

	if conf.Retries > 0 {
		mw.leRetries.SetText(strconv.Itoa(conf.Retries))
	} else {
		mw.leRetries.SetText("2")
	}

	mw.mainWindow.Show()

	go func(){
		mw.cbSrcRepo.SetCurrentIndex(0)
		mw.cbDstRepo.SetCurrentIndex(0)
		mw.cbIncrement.SetCurrentIndex(0)

		if squashfs {
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
			time.Sleep( 1 * time.Second)
		}
	}()
	mw.mainWindow.Run()
}

type MyMainWindow struct {
	mainWindow       *walk.MainWindow
	labelOutput      *walk.Label
	labelInput       *walk.Label
	teInput          *walk.TextEdit
	teOutput         *walk.TextEdit
	btnSync          *walk.PushButton
	btnCancel        *walk.PushButton
	btnDownload      *walk.PushButton
	btnUpload        *walk.PushButton
	btnTest          *walk.PushButton
	labelSrcRepo     *walk.Label
	labelDstRepo     *walk.Label
	cbSrcRepo        *walk.ComboBox
	cbDstRepo        *walk.ComboBox
	cbIncrement      *walk.ComboBox
	cbSingle         *walk.ComboBox
	labelStatusTitle *walk.Label
	labelStatus      *walk.Label
	labelCache       *walk.Label
	lmSrcRepo        *ListRepoModel
	lmDstRepo        *ListRepoModel
	lmIncrement      *ListStrModel
	lmSingle         *ListStrModel
	leMaxConn        *walk.LineEdit
	leRetries        *walk.LineEdit
	srcRepo          * Repo
	dstRepo          * Repo
	ctx              *TaskContext
	maxConn          int
	retries          int
	compressor       string
	singleFile       bool
	increment		 bool
}

func(mw *MyMainWindow) BeginAction() bool{
	txtLen := len(mw.teOutput.Text())
	if txtLen > 1000000 {
		mw.teOutput.SetText(string(mw.teOutput.Text()[ txtLen - 1000000 :]))
	}
	maxConn, err := strconv.Atoi(mw.leMaxConn.Text())
	if err != nil {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Verify input failed"),
			fmt.Sprintf(I18n.Sprintf("Failed to set 'MaxThreads' with error: %v", err)),
			walk.MsgBoxIconStop)
		return true
	}
	mw.maxConn = maxConn

	retries, err := strconv.Atoi(mw.leRetries.Text())
	if err != nil {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Verify input failed"),
			fmt.Sprintf(I18n.Sprintf("Failed to set 'Retries' with error: %v", err)),
			walk.MsgBoxIconStop)
		return true
	}
	mw.retries = retries

	mw.ctx.Info(I18n.Sprintf("==============BEGIN=============="))
	mw.ctx.Info(I18n.Sprintf("Transmit params: max threads: %v, max retries: %v", mw.maxConn, retries))
	mw.ctx.Reset()
	mw.ctx.UpdateSecStart(time.Now().Unix())
	return true
}

func(mw *MyMainWindow) StatDatafiles(pathname string, filename string) error{
	for k := range mw.ctx.CompMeta.Datafiles {
		i, err := os.Stat(filepath.Join( pathname , k))
		if err != nil {
			return mw.ctx.Errorf(I18n.Sprintf("Stat data file failed: %v", err))
		}
		mw.ctx.CompMeta.AddDatafile(k, i.Size())
	}
	b, err := yaml.Marshal(mw.ctx.CompMeta)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	}
	metaFile := filepath.Join( pathname , filename + "_meta.yaml")
	err = ioutil.WriteFile(metaFile, b, os.ModePerm)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	} else  {
		mw.ctx.Info(I18n.Sprintf("Create meta file: %s", metaFile))
	}
	return nil
}

func(mw *MyMainWindow) EndAction() (){
	if !conf.KeepTemp {
		mw.ctx.Temp.Clean()
	}
	mw.ctx.UpdateSecEnd(time.Now().Unix())
	mw.ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}

func (mw *MyMainWindow) getInputList() [][] string {
	var list [][] string

	input := mw.teInput.Text()
	input = strings.ReplaceAll(input, "\t", "")
	if invalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r",""),"\n", "")){
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Input Error"),
			I18n.Sprintf("Invalid char(s) found from the input, please check the text in the left edit box"),
			walk.MsgBoxIconStop)
		return nil
	}
	imgList := strings.Split(strings.ReplaceAll(input, "\r", ""),"\n")
	for _ , imgName := range imgList {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		imgName = strings.TrimPrefix(
			strings.TrimPrefix(
				strings.TrimSpace(imgName) , "http://" ), "https://")
		urlList := strings.Split(imgName, "/")
		if strings.ContainsAny(urlList[0],".") {
			urlList = urlList[1:]
		}
		list = append(list, urlList)
	}
	if len(list) < 1 {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Input Error"),
			I18n.Sprintf("The image list is empty, please input on the left edit box, one image each line"),
			walk.MsgBoxIconStop)
		return nil
	}
	return list
}

func invalidChar(text string) bool {
	f := func(r rune) bool {
		return r < ' ' || r > '~'
	}
	if strings.IndexFunc(text, f) != -1 {
		return true
	}
	return false
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
	mw.increment , _ = strconv.ParseBool(mw.lmIncrement.items[mw.cbIncrement.CurrentIndex()].value)
}

func (mw *MyMainWindow) SingleCurrentIndexChanged() {
	mw.singleFile, _ = strconv.ParseBool(mw.lmSingle.items[mw.cbSingle.CurrentIndex()].value)
}

func (mw *MyMainWindow)updateWindowsNewLine(){
	input := mw.teInput.Text()
	input = strings.Replace(input, "\r\n", "\n", -1 )
	input = strings.Replace(input, "\n", "\r\n", -1 )
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
			m.items[i]= ListRepoItem{name: v.Name, value: v }
		} else {
			if v.Repository != "" {
				m.items[i]= ListRepoItem{name: v.Registry + "-" + v.Repository, value: v }
			} else {
				m.items[i]= ListRepoItem{name: v.Registry, value: v }
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
	m := &ListStrModel{items: make([]ListStrItem, 2 )}
	m.items[0] = ListStrItem{name: I18n.Sprintf("FULL"), value: "false" }
	m.items[1] = ListStrItem{name: I18n.Sprintf("INCR"), value: "true" }
	return m
}

func NewSingleListModel() *ListStrModel {
	m := &ListStrModel{items: make([]ListStrItem, 2 )}
	m.items[0] = ListStrItem{name: I18n.Sprintf("YES"), value: "true" }
	m.items[1] = ListStrItem{name: I18n.Sprintf("NO"), value: "false" }
	return m
}

type GuiLogger struct {
	te	*walk.TextEdit
	logChan    chan int
}

func newGuiLogger(te *walk.TextEdit) CtxLogger {
	return &GuiLogger{
		te: te ,
		logChan: make(chan int, 1),
	}
}

func (l *GuiLogger) Info(logStr string) {
	l.logChan <- 1
	defer func() {
		<-l.logChan
	}()
	l.te.AppendText( time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\r\n" )
	log.Info(logStr)
}

func (l *GuiLogger) Error(logStr string) {
	l.logChan <- 1
	defer func() {
		<-l.logChan
	}()
	l.te.AppendText( time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\r\n" )
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
	errStr := fmt.Sprintf(format, args)
	l.te.AppendText( time.Now().Format("[2006-01-02 15:04:05]<ERROR>") + " " + fmt.Sprintf(format, args...) + "\r\n")
	log.Error(errStr)
	return fmt.Errorf(format, args...)
}
