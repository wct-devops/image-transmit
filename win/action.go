package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/lxn/walk"
	"github.com/mcuadros/go-version"
	. "github.com/wct-devops/image-transmit/core"
	"gopkg.in/yaml.v2"
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
	cmUpload         *CompressionMetadata
	pathUpload       string
	maxConn          int
	retries          int
	compressor       string
	singleFile       bool
	increment        bool
}

func (mw *MyMainWindow) BeginAction() bool {
	txtLen := len(mw.teOutput.Text())
	if txtLen > 1000000 {
		mw.teOutput.SetText(string(mw.teOutput.Text()[txtLen-1000000:]))
	}
	maxConn, err := strconv.Atoi(mw.leMaxConn.Text())
	if err != nil {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Verify input failed"),
			fmt.Sprint(I18n.Sprintf("Failed to set 'MaxThreads' with error: %v", err)),
			walk.MsgBoxIconStop)
		return true
	}
	mw.maxConn = maxConn

	retries, err := strconv.Atoi(mw.leRetries.Text())
	if err != nil {
		walk.MsgBox(mw.mainWindow,
			I18n.Sprintf("Verify input failed"),
			fmt.Sprint(I18n.Sprintf("Failed to set 'Retries' with error: %v", err)),
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

func (mw *MyMainWindow) EndAction() {
	if !CONF.KeepTemp {
		mw.ctx.Temp.Clean()
	}
	mw.ctx.UpdateSecEnd(time.Now().Unix())
	if mw.ctx.Notify != nil {
		mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Task End \n- Stat: %v", mw.ctx.GetStatus()))
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

func (mw *MyMainWindow) getInputList() []string {
	var list []string

	input := mw.teInput.Text()
	input = strings.ReplaceAll(input, "\t", "")

	if CheckInvalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r", ""), "\n", "")) {
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
		list = append(list, imgName)
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

func (mw *MyMainWindow) Transmit() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
		return
	}

	if imgList != nil {
		go func() {
			mw.btnSync.SetEnabled(false)
			defer mw.btnSync.SetEnabled(true)
			for _, rawURL := range imgList {
				if mw.ctx.Cancel() {
					mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
					return
				}

				src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
				c.GenerateOnlineTask(src, mw.srcRepo.User, mw.srcRepo.Password,
					dst, mw.dstRepo.User, mw.dstRepo.Password)

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

	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
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

			mw.ctx.History, err = NewHistory(HIS_FILE)
			if err != nil {
				log.Error(err)
				panic(err)
			}

			for {
				for _, rawURL := range imgList {
					if mw.ctx.Cancel() {
						mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
						break
					}

					src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)

					srcURL, _ := NewRepoURL(src)
					dstURL, _ := NewRepoURL(dst)

					imageSourceSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(src, "https"))
					if err != nil {
						log.Error(err)
						return
					}

					tags, err := imageSourceSrc.GetSourceRepoTags()
					if err != nil {
						c.PutAInvalidTask(src)
						mw.ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
						continue
					}

					for _, tag := range tags {
						newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
						newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
						if version.Compare(tag, srcURL.GetTag(), "<") || mw.ctx.History.Skip(newSrcUrl) {
							continue
						}

						newImgSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(newSrcUrl, "https"))
						if err != nil {
							c.PutAInvalidTask(newSrcUrl)
							mw.ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
							continue
						}

						newImgDst, err := NewImageDestination(mw.ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, mw.dstRepo.User, mw.dstRepo.Password, !strings.HasPrefix(newDstUrl, "https"))
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
				case <-time.After(time.Duration(INTERVAL) * time.Second):
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
	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
		return
	}

	var prefixPathname string
	var prefixFilename string
	if len(CONF.OutPrefix) > 0 {
		prefixPathIdx := strings.LastIndex(CONF.OutPrefix, string(os.PathSeparator))
		if prefixPathIdx > 0 {
			prefixPathname = CONF.OutPrefix[0:prefixPathIdx]
			prefixFilename = CONF.OutPrefix[prefixPathIdx+1:]
		}
	}

	pathname := filepath.Join(HOME, time.Now().Format("20060102"), prefixPathname)
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
				fmt.Sprint(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err)), walk.MsgBoxIconStop)
			return
		}
		for k := range cm.Blobs {
			mw.ctx.CompMeta.BlobDone(k, fmt.Sprintf("https://last.img/skip/it:%s", filepath.Base(dlg.FilePath)))
		}
	}

	if SQUASHFS {
		mw.ctx.Temp.SavePath(workName)
		mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
	} else {
		if mw.singleFile {
			mw.ctx.CreateSingleWriter(pathname, workName, mw.compressor)
		} else {
			mw.ctx.CreateTarWriter(pathname, workName, mw.compressor, mw.maxConn)
		}
	}

	go func() {
		mw.btnDownload.SetEnabled(false)
		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return
			}
			src, _ := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOfflineDownTask(src, mw.srcRepo.User, mw.srcRepo.Password)
		}
		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		if mw.ctx.SingleWriter != nil {
			time.Sleep(1 * time.Second)
			mw.ctx.SingleWriter.SetQuit()
		} else if mw.ctx.TarWriter != nil {
			mw.ctx.CloseTarWriter()
		}

		if mw.ctx.SingleWriter == nil {
			if mw.ctx.SquashfsTar != nil {
				mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
				err := MakeSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, workName+".squashfs"))
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
			mw.ctx.SingleWriter.SaveDockerMeta(mw.ctx.CompMeta)
			mw.StatDatafiles(pathname, workName)
			mw.EndAction()
			mw.btnDownload.SetEnabled(true)
		}()
	}
}

func (mw *MyMainWindow) Upload() {
	var cm *CompressionMetadata

	newUpload := true
	if mw.cmUpload != nil {
		text := fmt.Sprint(I18n.Sprintf("Continue upload(Yes) or create a new upload(No) or Cancel ?"))
		ret := walk.MsgBox(mw.mainWindow, I18n.Sprintf("Continue the upload ?"), text, walk.MsgBoxYesNoCancel)
		if ret == 6 {
			newUpload = false
			cm = mw.cmUpload
		} else if ret == 2 {
			return
		}
	}

	if newUpload {
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
				fmt.Sprint(I18n.Sprintf("Open file failed: %v", err)), walk.MsgBoxIconStop)
			return
		}
		cm = new(CompressionMetadata)
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

		mw.pathUpload = pathname
		mw.cmUpload = cm

		var srcImgUrlList []string
		for k := range cm.Manifests {
			srcImgUrlList = append(srcImgUrlList, k)
		}
		mw.teInput.SetText(strings.Join(srcImgUrlList, "\r\n"))

		text := fmt.Sprint(I18n.Sprintf("Total %v images found, if need update image name or tag, you can cancel and modify the list in the left edit box or we will start upload by default", len(cm.Manifests)))
		if walk.MsgBox(mw.mainWindow, I18n.Sprintf("Start transmit now ?"), text, walk.MsgBoxOKCancel) == 2 {
			return
		}
	}

	go func() {
		var err error
		mw.BeginAction()
		mw.ctx.CompMeta = cm
		imgList := mw.getInputList()

		if mw.ctx.CompMeta.Compressor == "squashfs" {
			var filename string
			for k := range cm.Datafiles {
				filename = k
			}
			workName := strings.TrimSuffix(filename, ".squashfs")
			if !TestSquashfs() || strings.Contains(CONF.Squashfs, "stream") {
				err = mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, filepath.Join(mw.pathUpload, filename))
				if err != nil {
					walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"),
						I18n.Sprintf("Unsquashfs uncompress failed with %v", err), walk.MsgBoxIconStop)
					return
				}
			} else {
				mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
				mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
				if strings.Contains(CONF.Squashfs, "nocmd") {
					err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(mw.pathUpload, filename), true)
				} else {
					err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(mw.pathUpload, filename), false)
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

		c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
		if err != nil {
			walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprintf("%s", err), walk.MsgBoxIconStop)
			return
		}

		mw.btnUpload.SetEnabled(false)
		defer mw.btnUpload.SetEnabled(true)

		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Error("User cancelled...")
				return
			}
			src, dst := GenRepoUrl("", mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOfflineUploadTask(src, dst, mw.pathUpload, mw.dstRepo.User, mw.dstRepo.Password)
		}

		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		mw.EndAction()
	}()
}

func (mw *MyMainWindow) Verify() {
	imgList := mw.getInputList()
	if imgList != nil {
		var text = I18n.Sprintf("Image List") + ":\r\n"
		text = text + I18n.Sprintf("Source Repository") + ", " + I18n.Sprintf("Destination Repository") + "\r\n"
		for _, rawURL := range imgList {
			src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			text = text + src + ", " + dst + "\r\n"
		}
		mw.ctx.Info(text)
	}
}
