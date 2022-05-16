package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kohirens/stdlib"
)

const (
	PS       = string(os.PathSeparator)
	DIR_MODE = 0774
)

var (
	appConfig = &Config{}
	flagStore *flagStorage
	errMsgs   = [...]string{
		"please specify a path (or URL) to a template",
		"enter a local path to output the app",
		"the following error occurred trying to get the app data directory: %q",
		"path/URL to template is not in the allow-list",
		"template download aborted; I'm coded to NOT do anything when HTTP status is %q and status code is %d",
		"please specify a path to an answer file that exist",
	}
	programName string
)

func init() {
	// Use `path/filepath.Base` for cross-platform compatibility.
	programName = filepath.Base(os.Args[0])
	fs, err := defineFlags(programName, flag.ContinueOnError)
	if err != nil {
		return
	}

	flagStore = fs
}

func main() {
	var mainErr error

	defer func() {
		if mainErr != nil {
			fmt.Print("\nfatal error detected: ")
			log.Fatalln(mainErr)
		}
		os.Exit(0)
	}()

	mainErr = flagMain()
	if mainErr != nil {
		return
	}

	verboseF(verboseLvlWarn, "running program %q", programName)
	verboseF(verboseLvlInfo, "verbose level: %v", verbosityLevel)

	appDataDir, mainErr := stdlib.AppDataDir()
	if mainErr != nil {
		return
	}

	// Make a directory for tmpltoapp to store data.
	appDataDir = appDataDir + PS + "tmpltoapp"
	mainErr = os.MkdirAll(appDataDir, DIR_MODE)
	if mainErr != nil {
		return
	}

	// Make a configuration file when there is not one.
	configFile := appDataDir + PS + "config.json"
	mainErr = initConfigFile(configFile)
	if mainErr != nil {
		return
	}

	mainErr = settings(configFile, appConfig)

	verboseF(3, "configured runtime options %v", appConfig)

	if mainErr != nil {
		return
	}

	isUrl, isAllowed := urlIsAllowed(appConfig.tplPath, appConfig.AllowedUrls)
	if isUrl && !isAllowed {
		mainErr = fmt.Errorf(errMsgs[3])
		return
	}
	verboseF(1, "isUrl %v", isUrl)

	appConfig.cacheDir = appDataDir + PS + "cache"
	mainErr = os.MkdirAll(appConfig.cacheDir, DIR_MODE)
	if mainErr != nil {
		mainErr = fmt.Errorf("could not make cache directory, error: %s", mainErr.Error())
		return
	}

	tmplPathType, err3 := getPathType(appConfig.tmplType)
	if err3 != nil {
		mainErr = err3
		return
	}

	if tmplPathType == "zip" {
		client := http.Client{}
		zipFile, iErr := download(appConfig.tplPath, appConfig.cacheDir, &client)
		if iErr != nil {
			mainErr = iErr
			return
		}

		appConfig.tmpl, iErr = extract(zipFile)
		if iErr != nil {
			mainErr = iErr
			return
		}
	}

	if tmplPathType == "local" {
		appConfig.tmpl = filepath.Clean(appConfig.tplPath)
	}

	if tmplPathType == "git" {
		var repo, commitHash string
		var err2 error

		repoDir := appConfig.cacheDir + PS + getRepoDir(appConfig.tplPath)
		infof("repoDir = %q\n", repoDir)

		// Do a pull when the repo already exists. This will fail if it downloaded a zip.
		if stdlib.DirExist(repoDir) {
			repo, commitHash, err2 = gitCheckout(repoDir, appConfig.branch)
		} else {
			repo, commitHash, err2 = gitClone(appConfig.tplPath, repoDir, appConfig.branch)
		}

		//infof("repo = %q; %q", repo, commitHash)
		fmt.Printf("repo = %q; %q", repo, commitHash)
		if err2 != nil {
			mainErr = err2
			return
		}
		appConfig.tmpl = repo
	}

	verboseF(3, "appConfig = %v", appConfig)

	if !stdlib.DirExist(appConfig.tmpl) {
		mainErr = fmt.Errorf("invalid template directory %q", appConfig.tmpl)
		return
	}

	fec, err1 := stdlib.NewFileExtChecker(&appConfig.ExcludeFileExtensions, &appConfig.IncludeFileExtensions)
	if err1 != nil {
		mainErr = fmt.Errorf("error instantiating file extension checker: %v", err1.Error())
	}

	// Require template directories to have a specific file in order to be processed to prevent processing directories unintentionally.
	tmplManifestFile := appConfig.tmpl + PS + TMPL_MANIFEST
	tmplManifest, errX := readTemplateJson(tmplManifestFile)
	if errX != nil {
		mainErr = fmt.Errorf(errs.missingTmplJson, TMPL_MANIFEST, tmplManifestFile, errX.Error())
		return
	}

	appConfig.Questions = *tmplManifest
	appConfig.answers, mainErr = loadAnswers(appConfig.answersPath)
	if mainErr != nil {
		return
	}

	if e := getInput(&appConfig.Questions, &appConfig.answers, os.Stdin); e != nil {
		mainErr = fmt.Errorf(errs.gettingAnswers, e.Error())
	}

	//missingAnswrs := checkAnswrsToQuestions()

	mainErr = parseDir(appConfig.tmpl, appConfig.appPath, appConfig.answers, fec, tmplManifest.Excludes)
}

// Check to see if a URL is in the allowed list to download template from.
func urlIsAllowed(loc string, urls []string) (isUrl, isAllowed bool) {
	isUrl = strings.HasPrefix(loc, "https://")
	isAllowed = false

	if isUrl {
		for _, url := range urls {
			if strings.HasPrefix(loc, url) {
				isAllowed = true
				break
			}
		}
	}

	return
}
