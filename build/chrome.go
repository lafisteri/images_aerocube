package build

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	hv "github.com/hashicorp/go-version"
)

const (
	chromeDriverBinary    = "chromedriver"
	newChromeDriverBinary = "chromedriver-linux64/chromedriver"
)

type Chrome struct {
	Requirements
}

func (c *Chrome) Build() error {
	pkgSrcPath, pkgVersion, err := c.BrowserSource.Prepare()
	if err != nil {
		return fmt.Errorf("invalid browser source: %v", err)
	}

	pkgTagVersion := extractVersion(pkgVersion)

	chromeDriverVersions, err := fetchChromeDriverVersions()
	if err != nil {
		return fmt.Errorf("fetch chromedriver versions: %v", err)
	}

	driverVersion, err := c.parseChromeDriverVersion(pkgTagVersion, chromeDriverVersions)
	if err != nil {
		return fmt.Errorf("parse chromedriver version: %v", err)
	}

	fmt.Printf("resolved chromedriver version for chrome %s -> %s\n", pkgTagVersion, driverVersion)

	// Build dev image
	devDestDir, err := tmpDir()
	if err != nil {
		return fmt.Errorf("create dev temporary dir: %v", err)
	}

	srcDir := "chrome/apt"

	if pkgSrcPath != "" {
		srcDir = "chrome/local"
		pkgDestDir := filepath.Join(devDestDir, srcDir)
		err := os.MkdirAll(pkgDestDir, 0755)
		if err != nil {
			return fmt.Errorf("create %v temporary dir: %v", pkgDestDir, err)
		}
		pkgDestPath := filepath.Join(pkgDestDir, "google-chrome.deb")
		err = os.Rename(pkgSrcPath, pkgDestPath)
		if err != nil {
			return fmt.Errorf("move package: %v", err)
		}
	}

	devImageTag := fmt.Sprintf("lafisteri/vnc_chrome_aerocube:%s", pkgTagVersion)
	devImageRequirements := Requirements{NoCache: c.NoCache, Tags: []string{devImageTag}}
	devImage, err := NewImage(srcDir, devDestDir, devImageRequirements)
	if err != nil {
		return fmt.Errorf("init dev image: %v", err)
	}
	devBuildArgs := []string{fmt.Sprintf("VERSION=%s", pkgVersion)}
	devBuildArgs = append(devBuildArgs, c.channelToBuildArgs()...)
	devImage.BuildArgs = devBuildArgs
	if pkgSrcPath != "" {
		devImage.FileServer = true
	}

	err = devImage.Build()
	if err != nil {
		return fmt.Errorf("build dev image: %v", err)
	}

	// Build main image
	destDir, err := tmpDir()
	if err != nil {
		return fmt.Errorf("create temporary dir: %v", err)
	}

	image, err := NewImage("chrome", destDir, c.Requirements)
	if err != nil {
		return fmt.Errorf("init image: %v", err)
	}
	image.BuildArgs = append(image.BuildArgs, fmt.Sprintf("VERSION=%s", pkgTagVersion))

	err = c.downloadChromeDriver(image.Dir, driverVersion, chromeDriverVersions)
	if err != nil {
		return fmt.Errorf("failed to download chromedriver: %v", err)
	}
	image.Labels = []string{fmt.Sprintf("driver=chromedriver:%s", driverVersion)}

	err = image.Build()
	if err != nil {
		return fmt.Errorf("build image: %v", err)
	}

	err = image.Test(c.TestsDir, "chrome", pkgTagVersion)
	if err != nil {
		return fmt.Errorf("test image: %v", err)
	}

	err = image.Push()
	if err != nil {
		return fmt.Errorf("push image: %v", err)
	}

	return nil
}

func (c *Chrome) channelToBuildArgs() []string {
	switch c.BrowserChannel {
	case "beta":
		return []string{"PACKAGE=google-chrome-beta", "INSTALL_DIR=chrome-beta"}
	case "dev":
		return []string{"PACKAGE=google-chrome-unstable", "INSTALL_DIR=chrome-unstable"}
	default:
		return []string{}
	}
}

func (c *Chrome) parseChromeDriverVersion(pkgVersion string, chromeDriverVersions map[string]string) (string, error) {
	version := c.DriverVersion
	if version != LatestVersion {
		fmt.Printf("using explicitly requested chromedriver version: %s\n", version)
		return version, nil
	}

	// 1. exact match
	if _, ok := chromeDriverVersions[pkgVersion]; ok {
		return pkgVersion, nil
	}

	// 2. same MAJOR.MINOR.BUILD, highest patch
	buildPrefix := buildVersion(pkgVersion) + "."

	var buildMatches []string
	for mv := range chromeDriverVersions {
		if strings.HasPrefix(mv, buildPrefix) {
			buildMatches = append(buildMatches, mv)
		}
	}

	if len(buildMatches) > 0 {
		sortVersionsDesc(buildMatches)
		return buildMatches[0], nil
	}

	// 3. same MAJOR, highest available
	majorPrefix := majorVersion(pkgVersion) + "."

	var majorMatches []string
	for mv := range chromeDriverVersions {
		if strings.HasPrefix(mv, majorPrefix) {
			majorMatches = append(majorMatches, mv)
		}
	}

	if len(majorMatches) > 0 {
		sortVersionsDesc(majorMatches)
		return majorMatches[0], nil
	}

	return "", fmt.Errorf("could not find compatible chromedriver in Chrome for Testing for chrome %s", pkgVersion)
}

func sortVersionsDesc(versions []string) {
	sort.SliceStable(versions, func(i, j int) bool {
		lv, lerr := hv.NewVersion(versions[i])
		rv, rerr := hv.NewVersion(versions[j])

		if lerr != nil && rerr != nil {
			return versions[i] > versions[j]
		}
		if lerr != nil {
			return false
		}
		if rerr != nil {
			return true
		}
		return lv.GreaterThan(rv)
	})
}

func (c *Chrome) downloadChromeDriver(dir string, version string, chromeDriverVersions map[string]string) error {
	u, ok := chromeDriverVersions[version]
	if !ok {
		return fmt.Errorf("chromedriver version %s not found in Chrome for Testing index", version)
	}

	fmt.Printf("downloading chromedriver version=%s url=%s\n", version, u)

	outputPath, err := downloadDriver(u, newChromeDriverBinary, dir)
	if err != nil {
		return fmt.Errorf("download chromedriver: %v", err)
	}

	err = os.Rename(outputPath, filepath.Join(dir, chromeDriverBinary))
	if err != nil {
		return fmt.Errorf("rename chromedriver: %v", err)
	}

	return nil
}

func fetchChromeDriverVersions() (map[string]string, error) {
	const versionsURL = "https://googlechromelabs.github.io/chrome-for-testing/known-good-versions-with-downloads.json"

	resp, err := http.Get(versionsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch chrome versions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch chrome versions: unexpected status code %d", resp.StatusCode)
	}

	var cv ChromeVersions
	err = json.NewDecoder(resp.Body).Decode(&cv)
	if err != nil {
		return nil, fmt.Errorf("decode json: %v", err)
	}

	ret := make(map[string]string)
	const platformLinux64 = "linux64"
	const chromeDriver = "chromedriver"

	for _, v := range cv.Versions {
		version := v.Version
		if cd, ok := v.Downloads[chromeDriver]; ok {
			for _, d := range cd {
				if d.URL != "" && d.Platform == platformLinux64 {
					ret[version] = d.URL
					break
				}
			}
		}
	}

	return ret, nil
}

type ChromeVersions struct {
	Versions []ChromeVersion `json:"versions"`
}

type ChromeVersion struct {
	Version   string                      `json:"version"`
	Downloads map[string][]ChromeDownload `json:"downloads"`
}

type ChromeDownload struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}
