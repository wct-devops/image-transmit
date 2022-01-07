package main

import (
	. "github.com/wct-devops/image-transmit/core"
	"github.com/lxn/walk"
	"gopkg.in/yaml.v2"
	"strings"
	"strconv"
	"fmt"
	"time"
	"os"
	"path/filepath"
	"io/ioutil"
	log "github.com/cihub/seelog"
	"github.com/mcuadros/go-version"
)

type MyMainWindow struct {
	mainWindow       *walk.MainWindow
	labelOutput      *walk.Label
	labelInput       *walk.Label
	teInput          *walk.TextEdit
	teOutput         *walk.TextEdit
	btnSync          *walk.PushButton
	btnWatch         *walk.PushButton
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
	srcRepo          *Repo
	dstRepo          *Repo
	ctx              *TaskContext
	maxConn          int
	retries          int
	compressor       string
	singleFile       bool
	increment        bool
}

func (mw *MyMainWindow) BeginAction() bool {
	txtLen := len(mw.teOutput.Text())
	if txtLen > 1000000 {
		mw.teOutput.SetText(string(mw.teOutput.Text()[ txtLen-1000000:]))
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

func (mw *MyMainWindow) EndAction() () {
	if !conf.KeepTemp {
		mw.ctx.Temp.Clean()
	}
	mw.ctx.UpdateSecEnd(time.Now().Unix())
	if mw.ctx.Notify != nil {
		mw.ctx.Notify.Send( I18n.Sprintf("### Transmit Task End \n- Stat: %v", mw.ctx.GetStatus() ))
	}
	mw.ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}

func (mw *MyMainWindow) StatDatafiles(pathname string, filename string) error {
	for k := range mw.ctx.CompMeta.Datafiles {
		i, err := os.Stat(filepath.Join(pathname, k))
		if err != nil {
			return mw.ctx.Errorf(I18n.Sprintf("Stat data file failed: %v", err))
		}
		mw.ctx.CompMeta.AddDatafile(k, i.Size())
	}
	b, err := yaml.Marshal(mw.ctx.CompMeta)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	}
	metaFile := filepath.Join(pathname, filename+"_meta.yaml")
	err = ioutil.WriteFile(metaFile, b, os.ModePerm)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	} else {
		mw.ctx.Info(I18n.Sprintf("Create meta file: %s", metaFile))
	}
	return nil
}

func (mw *MyMainWindow) getInputList() [][] string {
	var list [][] string

	input := mw.teInput.Text()
	input = strings.ReplaceAll(input, "\t", "")

	if invalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r", ""), "\n", "")) {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Input Error"),
			I18n.Sprintf("Invalid char(s) found from the input, please check the text in the left edit box"),
			walk.MsgBoxIconStop)
		return nil
	}
	imgList := strings.Split(strings.ReplaceAll(input, "\r", ""), "\n")
	for _, imgName := range imgList {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		imgName = strings.TrimPrefix(
			strings.TrimPrefix(
				strings.TrimSpace(imgName), "http://"), "https://")
		urlList := strings.Split(imgName, "/")
		if strings.ContainsAny(urlList[0], ".") {
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

func (mw *MyMainWindow) Transmit() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
	if (err != nil) {
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
					c.GenerateOnlineTask(mw.srcRepo.Registry+"/"+strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password,
						mw.dstRepo.Registry+"/"+mw.dstRepo.Repository+"/"+urlList[len(urlList)-1], mw.dstRepo.User, mw.dstRepo.Password)
				} else {
					c.GenerateOnlineTask(mw.srcRepo.Registry+"/"+strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password,
						mw.dstRepo.Registry+"/"+strings.Join(urlList, "/"), mw.dstRepo.User, mw.dstRepo.Password)
				}
			}
			mw.ctx.UpdateTotalTask(c.TaskLen())
			c.Run()
			mw.EndAction()
		}()
	}
}

func (mw *MyMainWindow) Watch() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
	if (err != nil) {
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
		return
	}

	if imgList != nil {
		go func() {
			mw.btnWatch.SetEnabled(false)
			defer func() {
				mw.EndAction()
				mw.btnWatch.SetEnabled(true)
				mw.ctx.History = nil
			}()

			mw.ctx.History, err = NewHistory(hisFile)
			if err != nil {
				log.Error(err)
				panic(err)
			}

			for {
				for _, urlList := range imgList {
					if mw.ctx.Cancel() {
						mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
						break;
					}

					var srcUrl, dstUrl string
					srcUrl = mw.srcRepo.Registry + "/" + strings.Join(urlList, "/")
					if mw.dstRepo.Repository != "" {
						dstUrl = mw.dstRepo.Registry + "/" + mw.dstRepo.Repository + "/" + urlList[len(urlList)-1]
					} else {
						dstUrl = mw.dstRepo.Registry + "/" + strings.Join(urlList, "/")
					}

					srcURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(srcUrl, "https://"), "http://"))
					dstURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(dstUrl, "https://"), "http://"))

					imageSourceSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(srcUrl, "https"))
					if err != nil {
						log.Error(err)
						return
					}

					tags, err := imageSourceSrc.GetSourceRepoTags()
					if err != nil {
						c.PutAInvalidTask(srcUrl)
						mw.ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
						continue
					}

					for _, tag := range tags {
						newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
						newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
						if version.Compare(tag, srcURL.GetTag(), "<") || mw.ctx.History.Skip(newSrcUrl) {
							continue
						}

						newImgSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(srcUrl, "https"))
						if err != nil {
							c.PutAInvalidTask(newSrcUrl)
							mw.ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
							continue
						}

						newImgDst, err := NewImageDestination(mw.ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, mw.dstRepo.User, mw.dstRepo.Password, !strings.HasPrefix(dstUrl, "https"))
						if err != nil {
							c.PutAInvalidTask(newSrcUrl)
							mw.ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newDstUrl, err))
							continue
						}
						var callback func(bool, string)
						if mw.ctx.Notify != nil {
							callback = func(result bool, content string) {
								if result {
									mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Success\n- Image: %v\n- Stat: %v", newDstUrl, content))
								} else {
									mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Failed\n- Image: %v\n- Error: %v", newDstUrl, content))
								}
							}
						}
						c.PutATask(NewOnlineTaskCallback(newImgSrc, newImgDst, mw.ctx, callback))
						mw.ctx.Info(I18n.Sprintf("Generated a task for %s to %s", newSrcUrl, newDstUrl))
					}
					imageSourceSrc.Close()
				}
				mw.ctx.UpdateTotalTask(mw.ctx.GetTotalTask() + c.TaskLen())
				c.Run()
				select {
				case <-mw.ctx.Context.Done():
					mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
					return
				case <-time.After(time.Duration(interval) * time.Second):
					c.ClearInvalidTask()
					continue
				}
			}
		}()
	}
}

func (mw *MyMainWindow) Download() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	if mw.maxConn > len(imgList) {
		mw.maxConn = len(imgList)
	}
	c, err := Newlient(mw.maxConn, mw.retries, mw.ctx)
	if (err != nil) {
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
		return
	}

	var prefixPathname string
	var prefixFilename string
	if len(conf.OutPrefix) > 0 {
		prefixPathIdx := strings.LastIndex(conf.OutPrefix, string(os.PathSeparator))
		if prefixPathIdx > 0 {
			prefixPathname = conf.OutPrefix[0 : prefixPathIdx]
			prefixFilename = conf.OutPrefix[prefixPathIdx + 1 : ]
		}
	}

	pathname := filepath.Join(home, time.Now().Format("20060102"), prefixPathname)
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

	if len(prefixFilename) > 0 {
		workName = prefixFilename + "_" + workName
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
			walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
				fmt.Sprintf(I18n.Sprintf("Open file failed: %v"), err), walk.MsgBoxIconStop)
			return
		}
		cm := new(CompressionMetadata)
		err = yaml.Unmarshal(b, cm)
		if err != nil {
			walk.MsgBox(mw.mainWindow, I18n.Sprintf("Meta file error"),
				fmt.Sprintf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err)), walk.MsgBoxIconStop)
			return
		}
		for k := range cm.Blobs {
			mw.ctx.CompMeta.BlobDone(k, fmt.Sprintf("https://last.img/skip/it:%s", filepath.Base(dlg.FilePath)))
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
			c.GenerateOfflineDownTask(mw.srcRepo.Registry+"/"+strings.Join(urlList, "/"), mw.srcRepo.User, mw.srcRepo.Password)
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
				err := MakeSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, workName+".squashfs"))
				mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress End"))
				if err != nil {
					mw.ctx.Error(I18n.Sprintf("Mksquashfs compress failed with %v", err))
					walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
						I18n.Sprintf("Mksquashfs compress failed with %v", err), walk.MsgBoxIconStop)
					return
				} else {
					mw.ctx.CompMeta.AddDatafile(workName+".squashfs", 0)
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
}

func (mw *MyMainWindow) Upload() {
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
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
			fmt.Sprintf(I18n.Sprintf("Open file failed: %v", err)), walk.MsgBoxIconStop)
		return
	}
	cm := new(CompressionMetadata)
	yaml.Unmarshal(b, cm)

	pathname := filepath.Dir(dlg.FilePath)

	for k, v := range cm.Datafiles {
		f, err := os.Stat(filepath.Join(pathname, k))
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
	mw.teInput.SetText(strings.Join(srcImgUrlList, "\r\n"))
	text := fmt.Sprintf(I18n.Sprintf("Total %v images found, if need confirm, you can cancel and check the list in the left edit box", len(cm.Manifests)))

	//1-OK 2-Cancel
	if 1 == walk.MsgBox(mw.mainWindow, I18n.Sprintf("Start transmit now ?"), text, walk.MsgBoxOKCancel) {
		go func() {
			mw.BeginAction()
			mw.ctx.CompMeta = cm
			imgList := mw.getInputList()

			if mw.ctx.CompMeta.Compressor == "squashfs" {
				var filename string
				for k, _ := range cm.Datafiles {
					filename = k
				}
				workName := strings.TrimSuffix(filename, ".squashfs")
				if !TestSquashfs() || strings.Contains(conf.Squashfs, "stream") {
					err = mw.ctx.CreateSquashfsTar(tempDir, workName, filepath.Join(pathname, filename))
					if err != nil {
						walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
							I18n.Sprintf("Unsquashfs uncompress failed with %v", err), walk.MsgBoxIconStop)
						return
					}
				} else {
					mw.ctx.CreateSquashfsTar(tempDir, workName, "")
					mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
					if strings.Contains(conf.Squashfs, "nocmd") {
						err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, filename), true)
					} else {
						err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, filename), false)
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

			for i, urlList := range imgList {
				if mw.ctx.Cancel() {
					mw.ctx.Error("User cancelled...")
					return
				}
				if mw.dstRepo.Repository != "" {
					c.GenerateOfflineUploadTask(srcImgUrlList[i], mw.dstRepo.Registry+"/"+mw.dstRepo.Repository+"/"+urlList[len(urlList)-1], pathname, mw.dstRepo.User, mw.dstRepo.Password)
				} else {
					c.GenerateOfflineUploadTask(srcImgUrlList[i], mw.dstRepo.Registry+"/"+strings.Join(urlList, "/"), pathname, mw.dstRepo.User, mw.dstRepo.Password)
				}
			}

			mw.ctx.UpdateTotalTask(c.TaskLen())
			c.Run()
			mw.EndAction()
		}()
	}
}

func (mw *MyMainWindow) Verify() {
	imgList := mw.getInputList()
	if imgList != nil {
		var text = I18n.Sprintf("Image List") + ":\r\n"
		if mw.srcRepo.Registry != "" {
			text = text + I18n.Sprintf("Source Repository") + ":\r\n"
			for _, urlList := range imgList {
				text = text + mw.srcRepo.Registry + "/" + strings.Join(urlList, "/") + "\r\n"
			}
		}
		if mw.dstRepo.Registry != "" {
			text = text + I18n.Sprintf("Destination Repository") + ":\r\n"
			for _, urlList := range imgList {
				if mw.dstRepo.Repository != "" {
					text = text + mw.dstRepo.Registry + "/" + mw.dstRepo.Repository + "/" + urlList[len(urlList)-1] + "\r\n"
				} else {
					text = text + mw.dstRepo.Registry + "/" + strings.Join(urlList, "/") + "\r\n"
				}
			}
		}
		mw.ctx.Info(text)
	}
}
