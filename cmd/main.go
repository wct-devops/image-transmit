package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mcuadros/go-version"
	. "github.com/wct-devops/image-transmit/core"
	"gopkg.in/yaml.v2"
)

var (
	end       = false
	srcRepo   *Repo
	dstRepo   *Repo
	imgList   []string
	flConfSrc *string
	flConfDst *string
	flConfLst *string
	flConfInc *string
	flConfImg *string
	flConfOut *string
	flConfWat *bool
)

func main() {
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
			fmt.Println(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		} else {
			cfgFile, err = ioutil.ReadFile(filepath.Join(HOME, "cfg.yaml"))
			if err != nil {
				fmt.Println(I18n.Sprintf("Read cfg.yaml failed: %v", err))
			}
		}
	} else {
		cfgFile, err = ioutil.ReadFile("cfg.yaml")
		if err != nil {
			fmt.Println(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		}
	}

	if err != nil {
		return
	}

	err = yaml.Unmarshal(cfgFile, CONF)
	if err != nil {
		fmt.Print(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", err))
		os.Exit(1)
	}

	if CONF.MaxConn == 0 {
		CONF.MaxConn = runtime.NumCPU()
	}

	if CONF.Retries == 0 {
		CONF.Retries = 2
	}

	if CONF.Interval > 0 {
		INTERVAL = CONF.Interval
	}

	if len(CONF.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			CONF.Compressor = "tar"
		} else {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				CONF.Compressor = "squashfs"
			} else {
				CONF.Compressor = "tar"
			}
		}
	}

	if CONF.Compressor != "squashfs" {
		SQUASHFS = false
	} else {
		if runtime.GOOS != "windows" {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				// ok
			} else {
				fmt.Print(I18n.Sprintf("Squashfs condition check failed, we need root privilege(run as root or sudo) and squashfs-tools/tar installed\n"))
				return
			}
		}
	}

	if len(CONF.Lang) > 1 {
		InitI18nPrinter(CONF.Lang)
	}

	flConfSrc = flag.String("src", "", I18n.Sprintf("Source repository name, default: the first repo in cfg.yaml"))
	flConfDst = flag.String("dst", "", I18n.Sprintf("Destination repository name, default: the first repo in cfg.yaml"))
	flConfLst = flag.String("lst", "", I18n.Sprintf("Image list file, one image each line"))
	flConfInc = flag.String("inc", "", I18n.Sprintf("The referred image meta file(*meta.yaml) in increment mode"))
	flConfImg = flag.String("img", "", I18n.Sprintf("Image meta file to upload(*meta.yaml)"))
	flConfOut = flag.String("out", "", I18n.Sprintf("Output filename prefix"))
	flConfWat = flag.Bool("watch", false, I18n.Sprintf("Watch mode"))

	flag.Usage = func() {
		fmt.Println(I18n.Sprintf("Image Transmit-Ghang'e-WhaleCloud DevOps Team"))
		fmt.Print(I18n.Sprintf("%s [OPTIONS]\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("Examples: \n"))
		fmt.Print(I18n.Sprintf("            Save mode:           %s -src=nj -lst=img.lst\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Increment save mode: %s -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Transmit mode:       %s -src=nj -lst=img.lst -dst=gz\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Watch mode:          %s -src=nj -lst=img.lst -dst=gz --watch\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Upload mode:         %s -dst=gz -img=img_full_202106122344_meta.yaml [-lst=img.lst]\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("More description please refer to github.com/wct-devops/image-transmit\n"))
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*flConfSrc) > 0 {
		for _, v := range CONF.SrcRepos {
			if v.Name == *flConfSrc {
				srcRepo = &v
				break
			}
		}
		if srcRepo == nil {
			fmt.Print(I18n.Sprintf("Could not find repo: %s", *flConfSrc))
			return
		}
	}

	CONF.DstRepos = append(CONF.DstRepos, Repo{
		Name: "docker",
	}, Repo{
		Name: "ctr",
	})

	if len(*flConfDst) > 0 {
		for _, v := range CONF.DstRepos {
			if v.Name == *flConfDst {
				dstRepo = &v
				break
			}
		}
		if dstRepo == nil {
			fmt.Print(I18n.Sprintf("Could not find repo: %s", *flConfDst))
			return
		}
	}

	if len(*flConfOut) > 0 {
		CONF.OutPrefix = *flConfOut
	}

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
		lc = NewLocalCache(filepath.Join(HOME, CONF.Cache.Pathname), keepDays, keepSize)
	}

	lt := NewLocalTemp(TEMP_DIR)
	log := NewCmdLogger()

	ctx := NewTaskContext(log, lc, lt)
	ctx.Reset()
	if len(CONF.DingTalk) > 0 {
		ctx.Notify = NewDingTalkWapper(CONF.DingTalk)
	}

	if len(*flConfSrc) > 0 && len(*flConfDst) > 0 {
		err := readImgList(ctx)
		if err != nil {
			os.Exit(1)
		}
		if *flConfWat {
			BeginAction(ctx)
			watch(ctx)
		} else {
			BeginAction(ctx)
			transmit(ctx)
			EndAction(ctx)
		}
	} else if len(*flConfImg) > 0 && len(*flConfDst) > 0 {
		BeginAction(ctx)
		upload(ctx)
		EndAction(ctx)
	} else if len(*flConfSrc) > 0 {
		err := readImgList(ctx)
		if err != nil {
			os.Exit(1)
		}
		BeginAction(ctx)
		download(ctx)
		EndAction(ctx)
	} else {
		fmt.Println(I18n.Sprintf("Invalid args, please refer the help"))
		flag.Usage()
	}
}

func readImgList(ctx *TaskContext) error {
	if len(*flConfLst) > 0 {
		b, err := ioutil.ReadFile(*flConfLst)
		if err != nil {
			return ctx.Errorf(I18n.Sprintf("Read image list from file failed: %v", err))
		}
		getInputList(string(b))
	} else {
		var s string
		for {
			var l string
			_, err := fmt.Scanln(&l)
			if len(l) < 1 || err != nil {
				break
			}
			s = s + "\n" + l
		}

		if len(s) > 0 {
			getInputList(s)
		}
	}
	if len(imgList) < 1 {
		return ctx.Errorf(I18n.Sprintf("Empty image list"))
	}
	ctx.Info(I18n.Sprintf("Get %v images", len(imgList)))
	return nil
}

func getInputList(input string) {
	input = strings.ReplaceAll(input, "\t", "")
	if CheckInvalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r", ""), "\n", "")) {
		fmt.Println(I18n.Sprintf("Invalid chars in image list"))
		return
	}

	for _, imgName := range strings.Split(strings.ReplaceAll(input, "\r", ""), "\n") {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		imgList = append(imgList, imgName)
	}
}

func startReport(ctx *TaskContext) {
	go func() {
		for {
			if end {
				break
			}
			fmt.Println(ctx.GetStatus())
			time.Sleep(1 * time.Second)
		}
	}()
}

func transmit(ctx *TaskContext) error {
	c, err := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	if err != nil {
		ctx.Errorf("%v", err)
		return err
	}
	for _, rawURL := range imgList {
		src, dst := GenRepoUrl(srcRepo.Registry, dstRepo.Registry, dstRepo.Repository, rawURL)
		c.GenerateOnlineTask(src, srcRepo.User, srcRepo.Password, dst, dstRepo.User, dstRepo.Password)
	}
	ctx.UpdateTotalTask(c.TaskLen())
	startReport(ctx)
	c.Run()
	end = true
	return nil
}

func watch(ctx *TaskContext) error {
	c, err := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	if err != nil {
		ctx.Errorf("%v", err)
		return err
	}

	if imgList != nil {
		ctx.History, err = NewHistory(HIS_FILE)
		if err != nil {
			ctx.Errorf("%v", err)
			return err
		}
		for {
			for _, rawURL := range imgList {
				if ctx.Cancel() {
					ctx.Errorf(I18n.Sprintf("User cancelled..."))
					break
				}

				src, dst := GenRepoUrl(srcRepo.Registry, dstRepo.Registry, dstRepo.Repository, rawURL)
				srcURL, _ := NewRepoURL(src)
				dstURL, _ := NewRepoURL(dst)

				imageSourceSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", srcRepo.User, srcRepo.Password, !strings.HasPrefix(src, "https"))
				if err != nil {
					log.Error(err)
					return err
				}

				tags, err := imageSourceSrc.GetSourceRepoTags()
				if err != nil {
					c.PutAInvalidTask(src)
					ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
					return err
				}

				for _, tag := range tags {
					newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
					newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
					if version.Compare(tag, srcURL.GetTag(), "<") || ctx.History.Skip(newSrcUrl) {
						continue
					}

					newImgSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, srcRepo.User, srcRepo.Password, !strings.HasPrefix(src, "https"))
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
						continue
					}

					newImgDst, err := NewImageDestination(ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, dstRepo.User, dstRepo.Password, !strings.HasPrefix(dst, "https"))
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newDstUrl, err))
						continue
					}
					var callback func(bool, string)
					if ctx.Notify != nil {
						callback = func(result bool, content string) {
							if result {
								ctx.Notify.Send(I18n.Sprintf("### Transmit Success\n- Image: %v\n- Stat: %v", newDstUrl, content))
							} else {
								ctx.Notify.Send(I18n.Sprintf("### Transmit Failed\n- Image: %v\n- Error: %v", newDstUrl, content))
							}
						}
					}
					c.PutATask(NewOnlineTaskCallback(newImgSrc, newImgDst, ctx, callback))
					ctx.Info(I18n.Sprintf("Generated a task for %s to %s", newSrcUrl, newDstUrl))
				}
				imageSourceSrc.Close()
			}
			ctx.UpdateTotalTask(ctx.GetTotalTask() + c.TaskLen())
			c.Run()
			fmt.Println(ctx.GetStatus())
			select {
			case <-ctx.Context.Done():
				ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return nil
			case <-time.After(time.Duration(INTERVAL) * time.Second):
				continue
			}
		}
	}
	return nil
}

func download(ctx *TaskContext) error {
	if CONF.MaxConn > len(imgList) {
		CONF.MaxConn = len(imgList)
	}
	c, _ := NewClient(CONF.MaxConn, CONF.Retries, ctx)

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
	_, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		os.MkdirAll(pathname, os.ModePerm)
	}

	var workName string
	if len(*flConfInc) > 1 {
		workName = time.Now().Format("img_incr_200601021504")
	} else {
		workName = time.Now().Format("img_full_200601021504")
	}

	if len(prefixFilename) > 0 {
		workName = prefixFilename + "_" + workName
	}

	ctx.CreateCompressionMetadata(CONF.Compressor)

	if len(*flConfInc) > 0 {
		b, err := ioutil.ReadFile(*flConfInc)
		if err != nil {
			return fmt.Errorf(I18n.Sprintf("Open file failed: %v", err))
		}
		cm := new(CompressionMetadata)
		err = yaml.Unmarshal(b, cm)
		if err != nil {
			return fmt.Errorf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err))
		}
		for k := range cm.Blobs {
			ctx.CompMeta.BlobDone(k, fmt.Sprintf("https://last.img/skip/it:%s", filepath.Base(*flConfInc)))
		}
	}

	if SQUASHFS {
		ctx.Temp.SavePath(workName)
		ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
	} else {
		if CONF.SingleFile {
			ctx.CreateSingleWriter(pathname, workName, "tar")
		} else {
			ctx.CreateTarWriter(pathname, workName, "tar", CONF.MaxConn)
		}
	}
	for _, rawURL := range imgList {
		src, _ := GenRepoUrl(srcRepo.Registry, "", "", rawURL)
		c.GenerateOfflineDownTask(src, srcRepo.User, srcRepo.Password)
	}
	startReport(ctx)
	ctx.UpdateTotalTask(c.TaskLen())
	c.Run()
	end = true
	if ctx.SingleWriter != nil {
		ctx.SingleWriter.SetQuit()
		ctx.SingleWriter.Run()
		ctx.SingleWriter.SaveDockerMeta(ctx.CompMeta)
	} else {
		ctx.CloseTarWriter()
	}

	if ctx.SquashfsTar != nil {
		ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
		err := MakeSquashfs(ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, workName+".squashfs"))
		ctx.Info(I18n.Sprintf("Mksquashfs Compress End"))
		if err != nil {
			ctx.Error(I18n.Sprintf("Mksquashfs compress failed with %v", err))
			return err
		} else {
			ctx.CompMeta.AddDatafile(workName+".squashfs", 0)
		}
	}

	WriteMetaFile(ctx, pathname, workName)
	return nil
}

func upload(ctx *TaskContext) error {
	b, err := ioutil.ReadFile(*flConfImg)
	if err != nil {
		return ctx.Errorf(I18n.Sprintf("Open file failed: %v", err))
	}
	cm := new(CompressionMetadata)
	err = yaml.Unmarshal(b, cm)
	if err != nil {
		return ctx.Errorf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err))
	}
	pathname := filepath.Dir(*flConfImg)

	ctx.CompMeta = cm

	for k, v := range cm.Datafiles {
		f, err := os.Stat(filepath.Join(pathname, k))
		if err != nil && os.IsNotExist(err) {
			return ctx.Errorf(I18n.Sprintf("Datafile %s missing", filepath.Join(pathname, k)))

		} else if f.Size() != v {
			return ctx.Errorf(I18n.Sprintf("Datafile %s mismatch in size, origin: %v, now: %v", filepath.Join(pathname, k), v, f.Size()))
		}
	}

	var srcImgUrlList []string
	for k := range cm.Manifests {
		srcImgUrlList = append(srcImgUrlList, k)
	}
	ctx.Info(I18n.Sprintf("The img file contains %v images:\n%s", len(cm.Manifests), strings.Join(srcImgUrlList, "\n")))

	if len(*flConfLst) > 0 {
		readImgList(ctx)
	} else {
		getInputList(strings.Join(srcImgUrlList, "\n")) // if no input list then take the original
	}

	if ctx.CompMeta.Compressor == "squashfs" {
		var filename string
		for k := range cm.Datafiles {
			filename = k
		}
		workName := strings.TrimSuffix(filename, ".squashfs")
		if !TestSquashfs() || strings.Contains(CONF.Squashfs, "stream") {
			err = ctx.CreateSquashfsTar(TEMP_DIR, workName, filepath.Join(pathname, filename))
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
			}
		} else {
			ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
			ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
			if strings.Contains(CONF.Squashfs, "nocmd") {
				err = UnSquashfs(ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, filename), true)
			} else {
				err = UnSquashfs(ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, filename), false)
				ctx.Temp.SavePath(workName)
			}
			ctx.Info(I18n.Sprintf("Unsquashfs uncompress End"))
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
			}
		}
	}

	c, _ := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	for _, rawURL := range imgList {
		src, dst := GenRepoUrl("", dstRepo.Registry, dstRepo.Repository, rawURL)
		if dstRepo.Name == "docker" || dstRepo.Name == "ctr" {
			ctx.DockerTarget = dstRepo.Name
			c.GenerateOfflineUploadTask(src, "", pathname, dstRepo.User, dstRepo.Password)
		} else {
			c.GenerateOfflineUploadTask(src, dst, pathname, dstRepo.User, dstRepo.Password)
		}
	}
	ctx.UpdateTotalTask(c.TaskLen())
	startReport(ctx)
	c.Run()
	end = true
	return nil
}

func WriteMetaFile(ctx *TaskContext, pathname string, filename string) error {
	for k := range ctx.CompMeta.Datafiles {
		i, err := os.Stat(filepath.Join(pathname, k))
		if err != nil {
			return ctx.Errorf(I18n.Sprintf("Stat data file failed: %v", err))
		}
		ctx.CompMeta.AddDatafile(k, i.Size())
	}
	b, err := yaml.Marshal(ctx.CompMeta)
	if err != nil {
		return ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	}
	metaFile := filepath.Join(pathname, filename+"_meta.yaml")
	err = ioutil.WriteFile(metaFile, b, os.ModePerm)
	if err != nil {
		return ctx.Errorf(I18n.Sprintf("Save meta yaml file failed:%s", err))
	} else {
		ctx.Info(I18n.Sprintf("Create meta file: %s", metaFile))
	}
	return nil
}

func BeginAction(ctx *TaskContext) bool {
	ctx.Info(I18n.Sprintf("==============BEGIN=============="))
	ctx.Info(I18n.Sprintf("Transmit params: max threads: %v, max retries: %v", CONF.MaxConn, CONF.Retries))
	ctx.UpdateSecStart(time.Now().Unix())
	return true
}

func EndAction(ctx *TaskContext) {
	if !CONF.KeepTemp {
		ctx.Temp.Clean()
	}
	ctx.UpdateSecEnd(time.Now().Unix())
	if ctx.Notify != nil {
		ctx.Notify.Send(I18n.Sprintf("### Transmit Task End \n- Stat: %v", ctx.GetStatus()))
	}
	ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}
