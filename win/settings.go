package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	. "github.com/wct-devops/image-transmit/core"
)

type settingsDialog struct {
	dlg           *walk.Dialog
	srcList       *walk.ListBox
	dstList       *walk.ListBox
	srcRepos      []Repo
	dstRepos      []Repo
	leMaxConn     *walk.LineEdit
	leRetries     *walk.LineEdit
	leInterval    *walk.LineEdit
	cbIncrement   *walk.ComboBox
	lmIncrement   *ListStrModel
	cbSingle      *walk.CheckBox
	cbDocker      *walk.CheckBox
	cbSkipTLS     *walk.CheckBox
	cbCacheEnable *walk.CheckBox
	leCachePath   *walk.LineEdit
	leCacheDays   *walk.LineEdit
	leCacheSize   *walk.LineEdit
}

func (mw *MyMainWindow) Settings() {
	sd := &settingsDialog{
		srcRepos:    append([]Repo{}, CONF.SrcRepos...),
		dstRepos:    append([]Repo{}, CONF.DstRepos...),
		lmIncrement: NewIncrementListModel(),
	}

	maxConn := CONF.MaxConn
	if maxConn == 0 {
		maxConn = runtime.NumCPU()
	}
	retries := CONF.Retries
	if retries == 0 {
		retries = 2
	}
	cacheDays := CONF.Cache.KeepDays
	if cacheDays <= 0 {
		cacheDays = 7
	}
	cacheSize := CONF.Cache.KeepSize
	if cacheSize <= 0 {
		cacheSize = 10
	}
	cachePath := CONF.Cache.Pathname
	if cachePath == "" {
		cachePath = "cache"
	}

	repoButtons := func(target bool) Widget {
		return Composite{
			Layout: HBox{MarginsZero: true},
			Children: []Widget{
				PushButton{Text: I18n.Sprintf("Add"), OnClicked: func() { sd.addRepo(target) }},
				PushButton{Text: I18n.Sprintf("Edit"), OnClicked: func() { sd.editRepo(target) }},
				PushButton{Text: I18n.Sprintf("Delete"), OnClicked: func() { sd.deleteRepo(target) }},
			},
		}
	}

	err := Dialog{
		AssignTo: &sd.dlg,
		Title:    I18n.Sprintf("Settings"),
		MinSize:  Size{Width: 660, Height: 480},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					GroupBox{
						Title:  I18n.Sprintf("Source Repositories"),
						Layout: VBox{},
						Children: []Widget{
							ListBox{AssignTo: &sd.srcList, MinSize: Size{Width: 270, Height: 150}},
							repoButtons(false),
						},
					},
					GroupBox{
						Title:  I18n.Sprintf("Target Repositories"),
						Layout: VBox{},
						Children: []Widget{
							ListBox{AssignTo: &sd.dstList, MinSize: Size{Width: 270, Height: 150}},
							repoButtons(true),
						},
					},
				},
			},
			GroupBox{
				Title:  I18n.Sprintf("General"),
				Layout: VBox{},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							Label{Text: I18n.Sprintf("Max threads:")},
							LineEdit{AssignTo: &sd.leMaxConn, Text: strconv.Itoa(maxConn), MaxSize: Size{Width: 60, Height: 0}},
							Label{Text: I18n.Sprintf("Retries:")},
							LineEdit{AssignTo: &sd.leRetries, Text: strconv.Itoa(retries), MaxSize: Size{Width: 60, Height: 0}},
							Label{Text: I18n.Sprintf("Watch interval (s):")},
							LineEdit{AssignTo: &sd.leInterval, Text: strconv.Itoa(INTERVAL), MaxSize: Size{Width: 60, Height: 0}},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							Label{Text: I18n.Sprintf("Archive mode:")},
							ComboBox{AssignTo: &sd.cbIncrement, Model: sd.lmIncrement, MinSize: Size{Width: 80, Height: 0}, MaxSize: Size{Width: 80, Height: 0}},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{AssignTo: &sd.cbSingle, Text: I18n.Sprintf("Single file"), Checked: CONF.SingleFile},
							CheckBox{AssignTo: &sd.cbDocker, Text: I18n.Sprintf("Docker compatible format"), Checked: CONF.DockerFile},
							CheckBox{AssignTo: &sd.cbSkipTLS, Text: I18n.Sprintf("Skip TLS verify"), Checked: CONF.SkipTlsVerify},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{AssignTo: &sd.cbCacheEnable, Text: I18n.Sprintf("Enable local cache"), Checked: CONF.Cache.Pathname != "", OnCheckedChanged: sd.cacheEnabledChanged},
							Label{Text: I18n.Sprintf("Cache directory:")},
							LineEdit{AssignTo: &sd.leCachePath, Text: cachePath, MinSize: Size{Width: 160, Height: 0}},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							Label{Text: I18n.Sprintf("Keep days:")},
							LineEdit{AssignTo: &sd.leCacheDays, Text: strconv.Itoa(cacheDays), MinSize: Size{Width: 50, Height: 0}, MaxSize: Size{Width: 50, Height: 0}},
							Label{Text: I18n.Sprintf("Keep size (G):")},
							LineEdit{AssignTo: &sd.leCacheSize, Text: strconv.Itoa(cacheSize), MinSize: Size{Width: 50, Height: 0}, MaxSize: Size{Width: 50, Height: 0}},
							HSpacer{},
						},
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: I18n.Sprintf("OK"), MinSize: Size{Width: 80, Height: 24}, OnClicked: sd.accept},
					PushButton{Text: I18n.Sprintf("Cancel"), MinSize: Size{Width: 80, Height: 24}, OnClicked: func() { sd.dlg.Cancel() }},
				},
			},
		},
	}.Create(mw.mainWindow)
	if err != nil {
		walk.MsgBox(mw.mainWindow, I18n.Sprintf("ERROR"), fmt.Sprint(err), walk.MsgBoxIconStop)
		return
	}

	if SQUASHFS {
		sd.cbSingle.SetChecked(true)
		sd.cbSingle.SetEnabled(false)
	}
	if CONF.Increment {
		sd.cbIncrement.SetCurrentIndex(1)
	} else {
		sd.cbIncrement.SetCurrentIndex(0)
	}
	sd.cacheEnabledChanged()
	sd.refreshLists()

	if sd.dlg.Run() == walk.DlgCmdOK {
		mw.reloadRepoModels()
		mw.refreshSettingsDisplay()
		mw.applyCache()
		mw.ctx.Info(I18n.Sprintf("Settings saved"))
	}
}

func (sd *settingsDialog) cacheEnabledChanged() {
	enabled := sd.cbCacheEnable.Checked()
	sd.leCachePath.SetEnabled(enabled)
	sd.leCacheDays.SetEnabled(enabled)
	sd.leCacheSize.SetEnabled(enabled)
}

func (sd *settingsDialog) refreshLists() {
	srcIdx := sd.srcList.CurrentIndex()
	dstIdx := sd.dstList.CurrentIndex()
	sd.srcList.SetModel(NewRepoListModel(sd.srcRepos))
	sd.dstList.SetModel(NewRepoListModel(sd.dstRepos))
	if srcIdx >= len(sd.srcRepos) {
		srcIdx = len(sd.srcRepos) - 1
	}
	if dstIdx >= len(sd.dstRepos) {
		dstIdx = len(sd.dstRepos) - 1
	}
	sd.srcList.SetCurrentIndex(srcIdx)
	sd.dstList.SetCurrentIndex(dstIdx)
}

func (sd *settingsDialog) addRepo(target bool) {
	repo := Repo{}
	if !sd.editRepoDialog(&repo, target, I18n.Sprintf("Add Repository")) {
		return
	}
	if target {
		sd.dstRepos = append(sd.dstRepos, repo)
	} else {
		sd.srcRepos = append(sd.srcRepos, repo)
	}
	sd.refreshLists()
	if target {
		sd.dstList.SetCurrentIndex(len(sd.dstRepos) - 1)
	} else {
		sd.srcList.SetCurrentIndex(len(sd.srcRepos) - 1)
	}
}

func (sd *settingsDialog) editRepo(target bool) {
	list, repos := sd.srcList, sd.srcRepos
	if target {
		list, repos = sd.dstList, sd.dstRepos
	}
	i := list.CurrentIndex()
	if i < 0 || i >= len(repos) {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Please select a repository first"), walk.MsgBoxIconStop)
		return
	}
	repo := repos[i]
	if !sd.editRepoDialog(&repo, target, I18n.Sprintf("Edit Repository")) {
		return
	}
	repos[i] = repo
	sd.refreshLists()
	list.SetCurrentIndex(i)
}

func (sd *settingsDialog) deleteRepo(target bool) {
	list, repos := sd.srcList, sd.srcRepos
	if target {
		list, repos = sd.dstList, sd.dstRepos
	}
	i := list.CurrentIndex()
	if i < 0 || i >= len(repos) {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Please select a repository first"), walk.MsgBoxIconStop)
		return
	}
	name := repos[i].Name
	if name == "" {
		name = repos[i].Registry
	}
	if walk.MsgBox(sd.dlg, I18n.Sprintf("Delete"), I18n.Sprintf("Delete repository %s?", name), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != 6 {
		return
	}
	if target {
		sd.dstRepos = append(repos[:i], repos[i+1:]...)
	} else {
		sd.srcRepos = append(repos[:i], repos[i+1:]...)
	}
	sd.refreshLists()
}

func (sd *settingsDialog) editRepoDialog(repo *Repo, target bool, title string) bool {
	var dlg *walk.Dialog
	var leName, leRegistry, leUser, lePassword, leProxy, leRepository *walk.LineEdit

	children := []Widget{
		Label{Text: I18n.Sprintf("Name:")},
		LineEdit{AssignTo: &leName, Text: repo.Name, CueBanner: I18n.Sprintf("optional")},
		Label{Text: I18n.Sprintf("Registry:")},
		LineEdit{AssignTo: &leRegistry, Text: repo.Registry, CueBanner: "http://10.0.0.1:5000"},
		Label{Text: I18n.Sprintf("Username:")},
		LineEdit{AssignTo: &leUser, Text: repo.User},
		Label{Text: I18n.Sprintf("Password:")},
		LineEdit{AssignTo: &lePassword, Text: repo.Password, PasswordMode: true},
		Label{Text: I18n.Sprintf("Proxy:")},
		LineEdit{AssignTo: &leProxy, Text: repo.Proxy, CueBanner: I18n.Sprintf("optional, e.g. socks5://127.0.0.1:1080")},
	}
	if target {
		children = append(children,
			Label{Text: I18n.Sprintf("Repository prefix:")},
			LineEdit{AssignTo: &leRepository, Text: repo.Repository, CueBanner: I18n.Sprintf("optional")},
		)
	}
	children = append(children, Composite{
		ColumnSpan: 2,
		Layout:     HBox{MarginsZero: true},
		Children: []Widget{
			HSpacer{},
			PushButton{Text: I18n.Sprintf("OK"), MinSize: Size{Width: 80, Height: 24}, OnClicked: func() {
				registry := strings.TrimSpace(leRegistry.Text())
				if registry == "" {
					walk.MsgBox(dlg, title, I18n.Sprintf("Registry must not be empty"), walk.MsgBoxIconStop)
					return
				}
				proxy := strings.TrimSpace(leProxy.Text())
				if proxy != "" {
					p, err := NormalizeProxy(proxy)
					if err != nil {
						walk.MsgBox(dlg, title, fmt.Sprint(err), walk.MsgBoxIconStop)
						return
					}
					proxy = p
				}
				repo.Name = strings.TrimSpace(leName.Text())
				repo.Registry = registry
				repo.User = leUser.Text()
				repo.Password = lePassword.Text()
				repo.Proxy = proxy
				if target {
					repo.Repository = strings.TrimSpace(leRepository.Text())
				}
				dlg.Accept()
			}},
			PushButton{Text: I18n.Sprintf("Cancel"), MinSize: Size{Width: 80, Height: 24}, OnClicked: func() { dlg.Cancel() }},
		},
	})

	err := Dialog{
		AssignTo: &dlg,
		Title:    title,
		MinSize:  Size{Width: 440, Height: 0},
		Layout:   Grid{Columns: 2},
		Children: children,
	}.Create(sd.dlg)
	if err != nil {
		walk.MsgBox(sd.dlg, I18n.Sprintf("ERROR"), fmt.Sprint(err), walk.MsgBoxIconStop)
		return false
	}
	return dlg.Run() == walk.DlgCmdOK
}

func (sd *settingsDialog) accept() {
	maxConn, err := strconv.Atoi(strings.TrimSpace(sd.leMaxConn.Text()))
	if err != nil || maxConn < 1 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Invalid number: %s", sd.leMaxConn.Text()), walk.MsgBoxIconStop)
		return
	}
	retries, err := strconv.Atoi(strings.TrimSpace(sd.leRetries.Text()))
	if err != nil || retries < 1 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Invalid number: %s", sd.leRetries.Text()), walk.MsgBoxIconStop)
		return
	}
	interval, err := strconv.Atoi(strings.TrimSpace(sd.leInterval.Text()))
	if err != nil || interval < 0 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Invalid number: %s", sd.leInterval.Text()), walk.MsgBoxIconStop)
		return
	}
	cacheDays, err := strconv.Atoi(strings.TrimSpace(sd.leCacheDays.Text()))
	if err != nil || cacheDays < 0 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Invalid number: %s", sd.leCacheDays.Text()), walk.MsgBoxIconStop)
		return
	}
	cacheSize, err := strconv.Atoi(strings.TrimSpace(sd.leCacheSize.Text()))
	if err != nil || cacheSize < 0 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Invalid number: %s", sd.leCacheSize.Text()), walk.MsgBoxIconStop)
		return
	}
	if len(sd.srcRepos) == 0 {
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("At least one source repository is required"), walk.MsgBoxIconStop)
		return
	}

	cachePath := ""
	if sd.cbCacheEnable.Checked() {
		cachePath = strings.TrimSpace(sd.leCachePath.Text())
		if cachePath == "" {
			cachePath = "cache"
		}
	}

	snapshot := *CONF
	CONF.SrcRepos = sd.srcRepos
	CONF.DstRepos = sd.dstRepos
	CONF.MaxConn = maxConn
	CONF.Retries = retries
	CONF.Interval = interval
	CONF.SingleFile = sd.cbSingle.Checked()
	CONF.DockerFile = sd.cbDocker.Checked()
	CONF.SkipTlsVerify = sd.cbSkipTLS.Checked()
	CONF.Increment = sd.cbIncrement.CurrentIndex() == 1
	CONF.Cache = LocalCache{Pathname: cachePath, KeepDays: cacheDays, KeepSize: cacheSize}
	if interval > 0 {
		INTERVAL = interval
	}
	if err := SaveConfig(); err != nil {
		*CONF = snapshot
		walk.MsgBox(sd.dlg, I18n.Sprintf("Settings"), I18n.Sprintf("Save cfg.yaml failed: %v", err), walk.MsgBoxIconStop)
		return
	}
	sd.dlg.Accept()
}

func (mw *MyMainWindow) reloadRepoModels() {
	srcIdx := mw.cbSrcRepo.CurrentIndex()
	dstIdx := mw.cbDstRepo.CurrentIndex()
	mw.lmSrcRepo = NewRepoListModel(CONF.SrcRepos)
	mw.lmDstRepo = NewRepoListModel(CONF.DstRepos)
	mw.cbSrcRepo.SetModel(mw.lmSrcRepo)
	mw.cbDstRepo.SetModel(mw.lmDstRepo)
	if srcIdx < 0 || srcIdx >= len(CONF.SrcRepos) {
		srcIdx = 0
	}
	mw.cbSrcRepo.SetCurrentIndex(srcIdx)
	if len(CONF.DstRepos) > 0 {
		if dstIdx < 0 || dstIdx >= len(CONF.DstRepos) {
			dstIdx = 0
		}
		mw.cbDstRepo.SetCurrentIndex(dstIdx)
		mw.dstRepo = &CONF.DstRepos[dstIdx]
	} else {
		mw.dstRepo = nil
	}
	mw.updateDstButtons()
}

func (mw *MyMainWindow) refreshSettingsDisplay() {
	maxConn := CONF.MaxConn
	if maxConn < 1 {
		maxConn = runtime.NumCPU()
	}
	retries := CONF.Retries
	if retries < 1 {
		retries = 2
	}
	mw.labelMaxConn.SetText(strconv.Itoa(maxConn))
	mw.labelRetries.SetText(strconv.Itoa(retries))

	mw.singleFile = CONF.SingleFile
	if mw.singleFile {
		mw.labelSingle.SetText(I18n.Sprintf("YES"))
	} else {
		mw.labelSingle.SetText(I18n.Sprintf("NO"))
	}

	mw.increment = CONF.Increment
	if mw.increment {
		mw.labelIncrement.SetText(I18n.Sprintf("INCR"))
	} else {
		mw.labelIncrement.SetText(I18n.Sprintf("FULL"))
	}

	if CONF.Cache.Pathname != "" {
		mw.labelCache.SetText(I18n.Sprintf("ON"))
	} else {
		mw.labelCache.SetText(I18n.Sprintf("OFF"))
	}
}

func (mw *MyMainWindow) applyCache() {
	if CONF.Cache.Pathname == "" {
		mw.ctx.Cache = nil
		return
	}
	keepDays := CONF.Cache.KeepDays
	if keepDays <= 0 {
		keepDays = 7
	}
	keepSize := CONF.Cache.KeepSize
	if keepSize <= 0 {
		keepSize = 10
	}
	mw.ctx.Cache = NewLocalCache(CONF.Cache.Pathname, keepDays, keepSize)
}