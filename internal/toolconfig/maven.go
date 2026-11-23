package toolconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const managedMirrorID = "dev-toolbox"

var mavenPresets = []Preset{
	{ID: "aliyun", Label: "阿里云镜像", Value: "https://maven.aliyun.com/repository/public"},
	{ID: "tencent", Label: "腾讯云镜像", Value: "https://mirrors.cloud.tencent.com/nexus/repository/maven-public/"},
	{ID: "huawei", Label: "华为云镜像", Value: "https://repo.huaweicloud.com/repository/maven/"},
}

// settingsPath 默认 ~/.m2/settings.xml；DEVTOOLBOX_MAVEN_SETTINGS 供测试隔离。
func settingsPath() string {
	if override := os.Getenv("DEVTOOLBOX_MAVEN_SETTINGS"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".m2", "settings.xml")
}

var managedBlockRe = regexp.MustCompile(`(?s)<mirror>\s*<id>` + managedMirrorID + `</id>.*?</mirror>`)
var anyMirrorRe = regexp.MustCompile(`(?s)<mirror>.*?</mirror>`)
var mirrorURLRe = regexp.MustCompile(`(?s)<url>(.*?)</url>`)

func extractManagedURL(data string) string {
	block := managedBlockRe.FindString(data)
	if block == "" {
		return ""
	}
	if m := mirrorURLRe.FindStringSubmatch(block); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func readMaven() (ToolConfigReport, error) {
	path := settingsPath()
	data := ""
	if b, err := os.ReadFile(path); err == nil {
		data = string(b)
	}
	managed := extractManagedURL(data)
	others := 0
	for _, block := range anyMirrorRe.FindAllString(data, -1) {
		if !strings.Contains(block, "<id>"+managedMirrorID+"</id>") {
			others++
		}
	}
	note := "仅托管 id=" + managedMirrorID + " 的 mirror；你已有的其它 mirror 不受影响。写入前会备份 settings.xml。"
	if others > 0 {
		note = fmt.Sprintf("另有 %d 个用户 mirror 不受影响。", others) + note
	}
	return ToolConfigReport{Tool: "maven", Items: []ConfigItem{{
		Key: "mirror.url", Label: "中央仓库镜像", Value: managed, Editable: true,
		Source:  "~/.m2/settings.xml（托管 id=" + managedMirrorID + "）",
		Presets: mavenPresets, Note: note,
	}}}, nil
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func managedBlock(url string) string {
	return "  <mirror>\n    <id>" + managedMirrorID + "</id>\n    <mirrorOf>central</mirrorOf>\n    <name>DevToolbox managed mirror</name>\n    <url>" + xmlEscape(url) + "</url>\n  </mirror>\n"
}

// writeMavenMirror 幂等 upsert/移除托管 mirror；首次写前备份原文件。
func writeMavenMirror(url string, unset bool) error {
	path := settingsPath()
	if path == "" {
		return fmt.Errorf("TOOL_CONFIG_WRITE_FAILED: cannot locate settings.xml")
	}
	data := ""
	existed := false
	if b, err := os.ReadFile(path); err == nil {
		data = string(b)
		existed = true
	}
	var next string
	switch {
	case unset:
		next = managedBlockRe.ReplaceAllString(data, "")
	case managedBlockRe.MatchString(data):
		next = managedBlockRe.ReplaceAllString(data, managedBlock(url))
	case strings.Contains(data, "</mirrors>"):
		next = strings.Replace(data, "</mirrors>", managedBlock(url)+"</mirrors>", 1)
	case strings.Contains(data, "</settings>"):
		next = strings.Replace(data, "</settings>", "  <mirrors>\n"+managedBlock(url)+"  </mirrors>\n</settings>", 1)
	default:
		next = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<settings>\n  <mirrors>\n" + managedBlock(url) + "  </mirrors>\n</settings>\n"
	}
	if existed {
		backup := path + ".devtoolbox.bak"
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			if b, rerr := os.ReadFile(path); rerr == nil {
				_ = os.WriteFile(backup, b, 0o600)
			}
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("TOOL_CONFIG_WRITE_FAILED: %v", err)
	}
	if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
		return fmt.Errorf("TOOL_CONFIG_WRITE_FAILED: %v", err)
	}
	return nil
}
