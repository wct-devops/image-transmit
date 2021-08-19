package main

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"os"
	"strings"
	"time"
	"gopkg.in/yaml.v2"
	. "github.com/wct-devops/image-transmit/core"
	"runtime"
	log "github.com/cihub/seelog"
	"github.com/mcuadros/go-version"
)

var (
	home      = "data"
	tempDir   = filepath.Join(home, "temp")
	hisFile   = filepath.Join(home, "history.yaml")
	conf      = new(YamlCfg)
	increment = false
	squashfs  = true
	end       = false
	interval  = 60
	srcRepo   *Repo
	dstRepo   *Repo
	imgList   [][]string
	flConfSrc *string
	flConfDst *string
	flConfLst *string
	flConfInc *string
	flConfImg *string
	flConfOut *string
	flConfWat *bool
)

type Repo struct {
	Name       string `yaml:"name"`
	User       string `yaml:"user"`
	Registry   string `yaml:"registry"`
	Password   string `yaml:"password"`
	Repository string `yaml:"repository,omitempty"`
}

type YamlCfg struct {
	SrcRepos   [] Repo          `yaml:"source",omitempty"`
	DstRepos   [] Repo          `yaml:"target",omitempty"`
	MaxConn    int              `yaml:"maxconn,omitempty"`
	Retries    int              `yaml:"retries,omitempty"`
	SingleFile bool             `yaml:"singlefile,omitempty"`
	Compressor string           `yaml:"compressor,omitempty"`
	Squashfs   string           `yaml:"squashfs,omitempty"`
	Cache      LocalCache       `yaml:"cache,omitempty"`
	Lang       string           `yaml:"lang,omitempty"`
	KeepTemp   bool             `yaml:"keeptemp,omitempty"`
	OutPrefix  string           `yaml:"outprefix,omitempty"`
	Interval   int              `yaml:"interval,omitempty"`
	DingTalk   []DingTalkAccess `yaml:"dingtalk,omitempty"`
}

func main() {
	InitI18nPrinter("")

	var loggerCfg []byte
	if _, err := os.Stat("logCfg.xml"); err == nil {
		loggerCfg, _ = ioutil.ReadFile("logCfg.xml")
	} else if _, err := os.Stat(filepath.Join(home, "logCfg.xml")); err == nil {
		loggerCfg, _ = ioutil.ReadFile(filepath.Join(home, "logCfg.xml"))
	}
	InitLogger(loggerCfg)

	var cfgFile []byte
	_, err := os.Stat("cfg.yaml")
	if err != nil && os.IsNotExist(err) {
		_, err = os.Stat(filepath.Join(home, "cfg.yaml"))
		if err != nil && os.IsNotExist(err) {
			fmt.Println(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		} else {
			cfgFile, err = ioutil.ReadFile(filepath.Join(home, "cfg.yaml"))
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

	err = yaml.Unmarshal(cfgFile, conf)
	if err != nil {
		fmt.Printf(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", err))
		os.Exit(1)
	}

	if conf.MaxConn == 0 {
		conf.MaxConn = runtime.NumCPU()
	}

	if conf.Retries == 0 {
		conf.Retries = 2
	}

	if conf.Interval > 0 {
		interval = conf.Interval
	}

	if len(conf.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			conf.Compressor = "tar"
		} else {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				conf.Compressor = "squashfs"
			} else {
				conf.Compressor = "tar"
			}
		}
	}

	if conf.Compressor != "squashfs" {
		squashfs = false
	} else {
		if runtime.GOOS != "windows" {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				// ok
			} else {
				fmt.Printf(I18n.Sprintf("Squashfs condition check failed， we need root privilege(run as root or sudo) and squashfs-tools/tar installed\n"))
				return
			}
		}
	}

	if len(conf.Lang) > 1 {
		InitI18nPrinter(conf.Lang)
	}

	flConfSrc = flag.String("src", "", I18n.Sprintf("Source repository name, default: the first repo in cfg.yaml"))
	flConfDst = flag.String("dst", "", I18n.Sprintf("Destination repository name, default: the first repo in cfg.yaml"))
	flConfLst = flag.String("lst", "", I18n.Sprintf("Image list file, one image each line"))
	flConfInc = flag.String("inc", "", I18n.Sprintf("The referred image meta file(*meta.yaml) in increment mode"))
	flConfImg = flag.String("img", "", I18n.Sprintf("Image meta file to upload(*meta.yaml)"))
	flConfOut = flag.String("out", "", I18n.Sprintf("Output filename prefix"))
	flConfWat = flag.Bool("watch", false, I18n.Sprintf("Watch mode"))

	flag.Usage = func() {
		fmt.Println(I18n.Sprintf("Image Transmit-EastWind-WhaleCloud DevOps Team"))
		fmt.Printf(I18n.Sprintf("%s [OPTIONS]\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("Examples: \n"))
		fmt.Printf(I18n.Sprintf("            Save mode:           %s -src=nj -lst=img.lst\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("            Increment save mode: %s -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("            Transmit mode:       %s -src=nj -lst=img.lst -dst=gz\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("            Watch mode:          %s -src=nj -lst=img.lst -dst=gz --watch\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("            Upload mode:         %s -dst=gz -img=img_full_202106122344_meta.yaml\n", os.Args[0]))
		fmt.Printf(I18n.Sprintf("More description please refer to github.com/wct-devops/image-transmit\n"))
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*flConfSrc) > 0 {
		for _, v := range (conf.SrcRepos) {
			if v.Name == *flConfSrc {
				srcRepo = &v
				break
			}
		}
		if srcRepo == nil {
			fmt.Printf(I18n.Sprintf("Could not find repo: %s", *flConfSrc))
			return
		}
	}

	conf.DstRepos = append(conf.DstRepos, Repo{
		Name: "docker",
	})

	if len(*flConfDst) > 0 {
		for _, v := range (conf.DstRepos) {
			if v.Name == *flConfDst {
				dstRepo = &v
				break
			}
		}
		if dstRepo == nil {
			fmt.Printf(I18n.Sprintf("Could not find repo: %s", *flConfDst))
			return
		}
	}

	if len(*flConfOut) > 0 {
		conf.OutPrefix = *flConfOut
	}

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
		lc = NewLocalCache(filepath.Join(home, conf.Cache.Pathname), keepDays, keepSize)
	}

	lt := NewLocalTemp(tempDir)
	log := NewCmdLogger()

	ctx := NewTaskContext(log, lc, lt)
	ctx.Reset()
	if len(conf.DingTalk) > 0 {
		ctx.Notify = NewDingTalkWapper(conf.DingTalk)
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
	if invalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r", ""), "\n", "")) {
		fmt.Println(I18n.Sprintf("Invalid chars in image list"))
		return
	}
	imgStrArr := strings.Split(strings.ReplaceAll(input, "\r", ""), "\n")
	for _, imgName := range imgStrArr {
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
		imgList = append(imgList, urlList)
	}
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
	c, err := Newlient(conf.MaxConn, conf.Retries, ctx)
	if (err != nil) {
		ctx.Errorf("%v", err)
		return err
	}
	for _, urlList := range imgList {
		if dstRepo.Repository != "" {
			c.GenerateOnlineTask(srcRepo.Registry+"/"+strings.Join(urlList, "/"), srcRepo.User, srcRepo.Password,
				dstRepo.Registry+"/"+dstRepo.Repository+"/"+urlList[len(urlList)-1], dstRepo.User, dstRepo.Password)
		} else {
			c.GenerateOnlineTask(srcRepo.Registry+"/"+strings.Join(urlList, "/"), srcRepo.User, srcRepo.Password,
				dstRepo.Registry+"/"+strings.Join(urlList, "/"), dstRepo.User, dstRepo.Password)
		}
	}
	ctx.UpdateTotalTask(c.TaskLen())
	startReport(ctx)
	c.Run()
	end = true
	return nil
}

func watch(ctx *TaskContext) error {
	c, err := Newlient(conf.MaxConn, conf.Retries, ctx)
	if (err != nil) {
		ctx.Errorf("%v", err)
		return err
	}

	if imgList != nil {
		ctx.History, err = NewHistory(hisFile)
		if err != nil {
			ctx.Errorf("%v", err)
			return err
		}
		for {
			for _, urlList := range imgList {
				if ctx.Cancel() {
					ctx.Errorf(I18n.Sprintf("User cancelled..."))
					break;
				}
				var srcUrl, dstUrl string
				srcUrl = srcRepo.Registry + "/" + strings.Join(urlList, "/")
				if dstRepo.Repository != "" {
					dstUrl = dstRepo.Registry + "/" + dstRepo.Repository + "/" + urlList[len(urlList)-1]
				} else {
					dstUrl = dstRepo.Registry + "/" + strings.Join(urlList, "/")
				}

				srcURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(srcUrl, "https://"), "http://"))
				dstURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(dstUrl, "https://"), "http://"))

				imageSourceSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", srcRepo.User, srcRepo.Password, !strings.HasPrefix(srcUrl, "https"))
				if err != nil {
					log.Error(err)
					return err
				}

				tags, err := imageSourceSrc.GetSourceRepoTags()
				if err != nil {
					c.PutAInvalidTask(srcUrl)
					ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
					return err
				}

				for _, tag := range tags {
					newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
					newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
					if version.Compare(tag, srcURL.GetTag(), "<") || ctx.History.Skip(newSrcUrl) {
						continue
					}

					newImgSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, srcRepo.User, srcRepo.Password, !strings.HasPrefix(srcUrl, "https"))
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
						continue
					}

					newImgDst, err := NewImageDestination(ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, dstRepo.User, dstRepo.Password, !strings.HasPrefix(dstUrl, "https"))
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
			case <-time.After(time.Duration(interval) * time.Second):
				continue
			}
		}
	}
	return nil
}

func download(ctx *TaskContext) error {
	if conf.MaxConn > len(imgList) {
		conf.MaxConn = len(imgList)
	}
	c, _ := Newlient(conf.MaxConn, conf.Retries, ctx)

	pathname := filepath.Join(home, time.Now().Format("20060102"))
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

	if len(conf.OutPrefix) > 0 {
		workName = conf.OutPrefix + "_" + workName
	}

	ctx.CreateCompressionMetadata(conf.Compressor)

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

	if squashfs {
		ctx.Temp.SavePath(workName)
		ctx.CreateSquashfsTar(tempDir, workName, "")
	} else {
		if conf.SingleFile {
			ctx.CreateSingleWriter(pathname, workName, "tar")
		} else {
			ctx.CreateTarWriter(pathname, workName, "tar", conf.MaxConn)
		}
	}
	for _, urlList := range imgList {
		c.GenerateOfflineDownTask(srcRepo.Registry+"/"+strings.Join(urlList, "/"), srcRepo.User, srcRepo.Password)
	}
	startReport(ctx)
	ctx.UpdateTotalTask(c.TaskLen())
	c.Run()
	end = true
	if ctx.SingleWriter != nil {
		ctx.SingleWriter.SetQuit()
		ctx.SingleWriter.Run()
	} else {
		ctx.CloseTarWriter()
	}

	if ctx.SquashfsTar != nil {
		ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
		err := MakeSquashfs(ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, workName+".squashfs"))
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
	getInputList(strings.Join(srcImgUrlList, "\n"))

	if ctx.CompMeta.Compressor == "squashfs" {
		var filename string
		for k, _ := range cm.Datafiles {
			filename = k
		}
		workName := strings.TrimSuffix(filename, ".squashfs")
		if !TestSquashfs() || strings.Contains(conf.Squashfs, "stream") {
			err = ctx.CreateSquashfsTar(tempDir, workName, filepath.Join(pathname, filename))
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
			}
		} else {
			ctx.CreateSquashfsTar(tempDir, workName, "")
			ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
			if strings.Contains(conf.Squashfs, "nocmd") {
				err = UnSquashfs(ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, filename), true)
			} else {
				err = UnSquashfs(ctx.GetLogger(), filepath.Join(tempDir, workName), filepath.Join(pathname, filename), false)
				ctx.Temp.SavePath(workName)
			}
			ctx.Info(I18n.Sprintf("Unsquashfs uncompress End"))
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
			}
		}
	}

	c, _ := Newlient(conf.MaxConn, conf.Retries, ctx)
	for i, urlList := range imgList {
		if dstRepo.Repository != "" {
			c.GenerateOfflineUploadTask(srcImgUrlList[i], dstRepo.Registry+"/"+dstRepo.Repository+"/"+urlList[len(urlList)-1], pathname, dstRepo.User, dstRepo.Password)
		} else if dstRepo.Name == "docker" {
			c.GenerateOfflineUploadTask(srcImgUrlList[i], "", pathname, dstRepo.User, dstRepo.Password)
		} else {
			c.GenerateOfflineUploadTask(srcImgUrlList[i], dstRepo.Registry+"/"+strings.Join(urlList, "/"), pathname, dstRepo.User, dstRepo.Password)
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
	ctx.Info(I18n.Sprintf("Transmit params: max threads: %v, max retries: %v", conf.MaxConn, conf.Retries))
	ctx.UpdateSecStart(time.Now().Unix())
	return true
}

func EndAction(ctx *TaskContext) () {
	if !conf.KeepTemp {
		ctx.Temp.Clean()
	}
	ctx.UpdateSecEnd(time.Now().Unix())
	if ctx.Notify != nil {
		ctx.Notify.Send(I18n.Sprintf("### Transmit Task End \n- Stat: %v", ctx.GetStatus()))
	}
	ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}
