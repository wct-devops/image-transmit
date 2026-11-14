package core

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	HOME      = "data"
	TEMP_DIR  = filepath.Join(HOME, "temp")
	HIS_FILE  = filepath.Join(HOME, "history.yaml")
	SQUASHFS  = true
	CONF      *YamlCfg
	CONF_PATH = "cfg.yaml"
	INTERVAL  = 60
)

// SaveConfig writes CONF back to the file it was loaded from.
func SaveConfig() error {
	b, err := yaml.Marshal(CONF)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(CONF_PATH, b, 0644)
}

func RepoID(r Repo) string {
	if r.Name != "" {
		return r.Name
	}
	return r.Registry
}

func FindRepoIndex(repos []Repo, id string) int {
	if id == "" {
		return 0
	}
	for i, r := range repos {
		if RepoID(r) == id {
			return i
		}
	}
	return 0
}

type Repo struct {
	Name       string `yaml:"name,omitempty"`
	User       string `yaml:"user"`
	Registry   string `yaml:"registry"`
	Password   string `yaml:"password"`
	Repository string `yaml:"repository,omitempty"`
	Proxy      string `yaml:"proxy,omitempty"`
}

type YamlCfg struct {
	SrcRepos      []Repo           `yaml:"source,omitempty"`
	DstRepos      []Repo           `yaml:"target,omitempty"`
	SrcSelected   string           `yaml:"src_selected,omitempty"`
	DstSelected   string           `yaml:"dst_selected,omitempty"`
	MaxConn       int              `yaml:"maxconn,omitempty"`
	Retries       int              `yaml:"retries,omitempty"`
	SingleFile    bool             `yaml:"singlefile,omitempty"`
	DockerFile    bool             `yaml:"dockerfile,omitempty"`
	Increment     bool             `yaml:"increment,omitempty"`
	Compressor    string           `yaml:"compressor,omitempty"`
	Squashfs      string           `yaml:"squashfs,omitempty"`
	Cache         LocalCache       `yaml:"cache,omitempty"`
	Lang          string           `yaml:"lang,omitempty"`
	KeepTemp      bool             `yaml:"keeptemp,omitempty"`
	OutPrefix     string           `yaml:"outprefix,omitempty"`
	Interval      int              `yaml:"interval,omitempty"`
	DingTalk      []DingTalkAccess `yaml:"dingtalk,omitempty"`
	SkipTlsVerify bool             `yaml:"skiptlsverify,omitempty"`
}

func CheckInvalidChar(text string) bool {
	f := func(r rune) bool {
		return r < ' ' || r > '~'
	}
	return strings.IndexFunc(text, f) != -1
}

// "Http" endpoint or Skip TLS Verify
func InsecureTarget(endpoint string) bool {
	return !strings.HasPrefix(endpoint, "https") || CONF.SkipTlsVerify
}

// we support two cases:
// hub.docker.com/myrepo/img:tag
// https://hub.docker.com/myrepo/img:tag
// hub.docker.com/myrepo/img:tag -> newname:newtag
// hub.docker.com/myrepo/img:tag -> mynewrepo/newname:newtag
// hub.docker.com/myrepo/img:tag -> newhub.docker.com/mynewrepo/newname:newtag
func GenRepoUrl(srcReg string, dstReg string, dstRepo string, rawURL string) (src string, dst string) {

	var rawSrcURL, rawDstURL string
	var rename bool
	if strings.Contains(rawURL, "->") {
		t := strings.Split(rawURL, "->")
		rawSrcURL = t[0]
		rawDstURL = t[1]
		rename = true
	} else {
		rawSrcURL = rawURL
		rawDstURL = rawURL
		rename = false
	}

	if srcReg == "" { //upload mode we keep the original URL
		src = strings.TrimSpace(rawSrcURL)
	}

	rawSrcURL =
		strings.TrimPrefix(
			strings.TrimPrefix(
				strings.TrimPrefix(
					strings.TrimSpace(rawSrcURL), "http://"), "https://"), "/")
	rawDstURL =
		strings.TrimPrefix(
			strings.TrimPrefix(
				strings.TrimPrefix(
					strings.TrimSpace(rawDstURL), "http://"), "https://"), "/")

	segSrcList := strings.Split(rawSrcURL, "/")
	if strings.ContainsAny(segSrcList[0], ".") { // omit the registry
		segSrcList = segSrcList[1:]
	}

	segDstList := strings.Split(rawDstURL, "/")
	if strings.ContainsAny(segDstList[0], ".") {
		segDstList = segDstList[1:]
	}

	if src == "" {
		src = srcReg + "/" + strings.Join(segSrcList, "/")
	}

	if dstRepo != "" {
		if rename {
			if len(segDstList) > 1 {
				dstRepo = "" // override the dest repo
			}
		} else {
			segDstList = segDstList[1:] // remove the old repo
		}
	}

	if dstRepo == "" {
		dst = dstReg + "/" + strings.Join(segDstList, "/")
	} else {
		dst = dstReg + "/" + dstRepo + "/" + strings.Join(segDstList, "/")
	}

	return src, dst
}
