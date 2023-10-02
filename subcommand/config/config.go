package config

import (
	"flag"
	"fmt"
	"github.com/kohirens/stdlib/log"
	"github.com/kohirens/stdlib/path"
	"github.com/kohirens/tmpltoapp/internal/cli"
	"github.com/kohirens/tmpltoapp/internal/msg"
	"github.com/kohirens/tmpltoapp/internal/press"
	"regexp"
	"strings"
)

type Arguments struct {
	Setting string // config setting
	Method  string // Method to call
	Value   string // value to update config setting
}

const (
	Summary = "Update or retrieve a configuration value"
	Name    = "config"
)

var (
	args  = &Arguments{}
	flags *flag.FlagSet
	help  bool
)

func Init() *flag.FlagSet {
	flags = flag.NewFlagSet(Name, flag.ExitOnError)

	flags.BoolVar(&help, "help", false, UsageMessages["help"])

	return flags
}

func ParseInput(ca []string) error {
	if e := flags.Parse(ca); e != nil {
		return fmt.Errorf(msg.Stderr.ParsingConfigArgs, e.Error())
	}

	if help {
		return nil
	}

	if len(ca) < 2 {
		return fmt.Errorf(msg.Stderr.InvalidNoArgs)
	}

	args.Method = ca[0]
	args.Setting = ca[1]

	if len(ca) > 2 {
		args.Value = ca[2]
	}

	if args.Method == "set" && len(ca) != 3 {
		return fmt.Errorf(Stderr.ConfigValueNotSet)
	}

	log.Dbugf("args.Method = %v\n", args.Method)
	log.Dbugf("args.Setting = %v\n", args.Setting)
	log.Dbugf("args.Value = %v\n", args.Value)

	return nil
}

// Run Set or get a config setting.
func Run(ca []string, appName string) error {
	if e := ParseInput(ca); e != nil {
		return e
	}

	if help {
		flags.Usage()
		return nil
	}

	log.Dbugf("args.Method = %v\n", args.Method)
	log.Dbugf("args.Key = %v\n", args.Setting)

	appDataDir, err1 := press.BuildAppDataPath(appName)
	if err1 != nil {
		return err1
	}

	cp := appDataDir + cli.PS + press.ConfigFileName

	switch args.Method {
	case "set":
		log.Dbugf("args.Value = %v\n", args.Value)
		return set(args.Setting, args.Value, cp)

	case "get":
		return get(args.Setting, cp)

	default:
		return fmt.Errorf(Stderr.InvalidConfigMethod, args.Method)
	}
}

// get the value of a user setting.
func get(key string, cp string) error {
	var val interface{}

	sc, err1 := press.LoadConfig(cp)
	if err1 != nil {
		return err1
	}

	switch key {
	case "CacheDir":
		val = sc.CacheDir
		log.Logf("%v", val)
		break

	case "ExcludeFileExtensions":
		v2 := fmt.Sprintf("%v", val)
		ok, _ := regexp.Match("^[a-zA-Z0-9-.]+(?:,[a-zA-Z0-9-.]+)*", []byte(v2))
		if !ok {
			return fmt.Errorf(Stderr.BadExcludeFileExt, val)
		}
		val = strings.Join(*sc.ExcludeFileExtensions, ",")
		log.Logf("%v", val)
		break

	default:
		return fmt.Errorf("no setting %v found", key)
	}

	return nil
}

// set the value of a user setting
func set(key, val string, cp string) error {
	sc := &press.ConfigSaveData{}

	switch key {
	case "CacheDir":
		sc.CacheDir = val
		log.Dbugf("setting CacheDir = %q", val)
		break

	case "ExcludeFileExtensions":
		tmp := strings.Split(val, ",")
		sc.ExcludeFileExtensions = &tmp
		log.Dbugf("adding exclusions %q to config", val)
		break

	default:
		return fmt.Errorf("no %q setting found", key)
	}

	if !path.Exist(cp) {
		_, e := press.InitConfig(cp)
		return e
	}

	return press.SaveConfig(cp, sc)
}
