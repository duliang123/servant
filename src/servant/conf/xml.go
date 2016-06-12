package conf
import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"path"
	"math"
	"bytes"
	"path/filepath"
	"fmt"
)

type XConfig struct {
	XMLName    xml.Name    `xml:"config"`
	Server     XServer     `xml:"server"`
	Users      []XUser     `xml:"user"`
	Commands   []XCommands `xml:"commands"`
	Files      []XFiles    `xml:"files"`
	Databases  []XDatabase `xml:"database"`
	Timers     []XTimer    `xml:"timer"`
	Daemons    []XDaemon   `xml:"daemon"`

}

type XServer struct {
	Listen  string      `xml:"listen"`
	Auth    XAuth       `xml:"auth"`
	Log     string      `xml:"log"`
}

type XAuth struct {
	Enabled       bool     `xml:"enabled,attr"`
	MaxTimeDelta  uint32   `xml:"maxTimeDelta"`
}

type XUser struct {
	Name      string           `xml:"id,attr"`
	Hosts     []string         `xml:"host"`
	Key       string           `xml:"key"`
	Files     []XUserFiles     `xml:"files"`
	Commands  []XUserCommands  `xml:"commands"`
	Databases []XUserDatabases `xml:"databases"`
}

type XCommands struct {
	Name     string      `xml:"id,attr"`
	Commands []XCommand  `xml:"command"`
}

type XCommand struct {
	Name         string  `xml:"id,attr"`
	Lang         string	 `xml:"lang,attr"`
	Code         string  `xml:"code"`
	Timeout      uint32  `xml:"timeout,attr"`
	User         string  `xml:"runas,attr"`
	Background   bool    `xml:"background,attr"`
	Validator    []XValidator `xml:"validator"`
	Lock         XLock   `xml:"lock"`
}

type XDatabase struct {
	Name    string    `xml:"id,attr"`
	Driver  string    `xml:"driver,attr"`
	Dsn     string    `xml:"dsn,attr"`
	Queries []XQuery  `xml:"query"`
}

type XQuery struct {
	Name      string   `xml:"id,attr"`
	Sqls      []string `xml:"sql"`
	Validator []XValidator `xml:"validator"`
}

type XLock struct {
	Name     string  `xml:"id,attr"`
	Timeout  uint    `xml:"timeout,attr"`
	Wait     bool    `xml:"wait,attr"`
}

type XFiles struct {
	Name   string       `xml:"id,attr"`
	Dirs   []XDir       `xml:"dir"`
}

type XDir struct {
	Name      string    `xml:"id,attr"`
	Root      string    `xml:"root"`
	Allows    []string  `xml:"allow"`
	Patterns  []string  `xml:"pattern"`
	Validator []XValidator `xml:"validator"`
}

type XTimer struct {
	Name      string `xml:"id,attr"`
	Lang      string `xml:"lang,attr"`
	Code      string `xml:"code"`
	User      string `xml:"runas,attr"`
	Tick      int    `xml:"tick,attr"`
	Deadline  uint32 `xml:"deadline,attr"`
}

type XDaemon struct {
	Name      string `xml:"id,attr"`
	Lang      string `xml:"lang,attr"`
	Code      string `xml:"code"`
	User      string `xml:"runas,attr"`
	Retries   int    `xml:"retries,attr"`
	Live      int    `xml:"live,attr"`
}

type XUserFiles struct {
	Name   string   `xml:"id,attr"`
}

type XUserCommands struct {
	Name   string   `xml:"id,attr"`
}

type XUserDatabases struct {
	Name   string   `xml:"id,attr"`
}

type XValidator struct {
	Name     string `xml:"name,attr"`
	//class  string
	Pattern  string `xml:",chardata"`
}

func XConfigFromData(data []byte, entities map[string]string) (*XConfig, error) {
	ret := XConfig{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Entity = entities
	err := decoder.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func XConfigFromReader(reader io.Reader, entities map[string]string) (*XConfig, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return XConfigFromData(data, entities)
}

func XConfigFromFile(path string, entities map[string]string) (*XConfig, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return XConfigFromReader(reader, entities)
}

func (conf *XConfig) ToConfig() *Config {
	ret := Config{}
	conf.IntoConfig(&ret)
	return &ret
}

func (conf *XConfig) IntoConfig(ret *Config) {
	if ret.Server.Listen == "" {
		ret.Server = Server{
			Listen: conf.Server.Listen,
		}
		ret.Auth = Auth {
			Enabled:      conf.Server.Auth.Enabled,
			MaxTimeDelta: conf.Server.Auth.MaxTimeDelta,
		}
		ret.Log = conf.Server.Log
	}

	ret.Files = make(map[string]*Files)
	for _, file := range(conf.Files) {
		fname := file.Name
		ret.Files[fname] = &Files{
			Dirs: make(map[string]*Dir),
		}
		for _, xdir := range(file.Dirs) {
			dname := xdir.Name
			dir := &Dir{
				Root: path.Clean(strings.TrimSpace(xdir.Root)),
				Allows: make([]string, 0, 4),
				Patterns: make([]string, 0, 4),
				Validators: xvalidatorsToValidators(xdir.Validator),
			}
			for _, method := range(xdir.Allows) {
				dir.Allows = append(dir.Allows, strings.ToUpper(strings.TrimSpace(method)))
			}
			for _, pattern := range(xdir.Patterns) {
				dir.Patterns = append(dir.Patterns, strings.TrimSpace(pattern))
			}
			ret.Files[fname].Dirs[dname] = dir
		}
	}
	ret.Commands = make(map[string]*Commands)
	for _, commands := range(conf.Commands) {
		csname := commands.Name
		ret.Commands[csname] = &Commands{
			Commands: make(map[string]*Command),
		}
		for _, command := range(commands.Commands) {
			cname := command.Name
			if command.Timeout == 0 {
				command.Timeout = math.MaxUint32
			}
			if command.Lock.Timeout == 0 {
				command.Lock.Timeout = math.MaxUint32
			}
			ret.Commands[csname].Commands[cname] = &Command{
				Code: strings.TrimSpace(command.Code),
				Lang: command.Lang,
				User: command.User,
				Timeout: command.Timeout,
				Background: command.Background,
				Lock: Lock {
					Name: strings.TrimSpace(command.Lock.Name),
					Timeout: command.Lock.Timeout,
					Wait: command.Lock.Wait,
				},
				Validators: xvalidatorsToValidators(command.Validator),
			}
		}
	}
	ret.Databases = make(map[string]*Database)
	for _, database := range(conf.Databases) {
		dname := database.Name
		ret.Databases[dname] = &Database{
			Dsn: database.Dsn,
			Driver: database.Driver,
			Queries: make(map[string]*Query),
		}
		for _, query := range(database.Queries) {
			ret.Databases[dname].Queries[query.Name] = &Query{
				Sqls: query.Sqls,
				Validators: xvalidatorsToValidators(query.Validator),
			}
		}
	}

	ret.Daemons = make(map[string]*Daemon)
	for _, daemon := range(conf.Daemons) {
		if daemon.Live <= 0 {
			daemon.Live = math.MaxUint32
		}
		ret.Daemons[daemon.Name] = &Daemon{
			Code: daemon.Code,
			Lang: daemon.Lang,
			User: daemon.User,
			Live: daemon.Live,
			Retries: daemon.Retries,
		}
	}

	ret.Timers = make(map[string]*Timer)
	for _, timer := range(conf.Timers) {
		if timer.Deadline <= 0 {
			timer.Deadline = math.MaxUint32
		}
		ret.Timers[timer.Name] = &Timer{
			Code: timer.Code,
			Lang: timer.Lang,
			User: timer.User,
			Tick: timer.Tick,
			Deadline: timer.Deadline,
		}
	}

	ret.Users = make(map[string]*User)
	for _, user := range(conf.Users) {
		uname := user.Name
		u := &User{
			Key: strings.TrimSpace(user.Key),
			Hosts: make([]string, len(user.Hosts)),
		}
		for j := range(user.Hosts) {
			u.Hosts[j] = strings.TrimSpace(user.Hosts[j])
		}
		u.Allows = make(map[string][]string)
		u.Allows["commands"] = make([]string, 0, 2)
		u.Allows["files"] = make([]string, 0, 2)
		u.Allows["databases"] = make([]string, 0, 2)
		for _, command := range(user.Commands) {
			u.Allows["commands"] = append(u.Allows["commands"], command.Name)
		}
		for _, file := range(user.Files) {
			u.Allows["files"] = append(u.Allows["files"], file.Name)
		}
		for _, database := range(user.Databases) {
			u.Allows["databases"] = append(u.Allows["databases"], database.Name)
		}
		ret.Users[uname] = u
	}
}
/*
type LoadConfigError struct {
	paths  []string
	errors []error
}*/


func xvalidatorsToValidators(xs []XValidator) Validators {
	ret := make(map[string]Validator)
	for _, x := range xs {
		ret[x.Name] = Validator{
			Name: x.Name,
			Pattern: x.Pattern,
		}
	}
	return ret
}

type LoadConfigError struct {
	Path string
	Err error
}
func (self LoadConfigError) Error() string {
	return fmt.Sprintf("load %s failed: %s", self.Path, self.Err.Error())
}

func LoadXmlConfig(files, dirs []string, params map[string]string) (config Config, err error) {
	for _, confPath := range files {
		xconf, err := XConfigFromFile(confPath, params)
		if err != nil {
			return config, LoadConfigError{ Path: confPath, Err: err }
		}
		xconf.IntoConfig(&config)
	}
	for _, confDirPath := range dirs {
		filesInfo, err := ioutil.ReadDir(confDirPath)
		if err != nil {
			return config, LoadConfigError{ Path: confDirPath, Err: err }
		}
		for _, fileInfo := range filesInfo {
			filename := fileInfo.Name()
			if strings.HasSuffix(fileInfo.Name(), ".conf") {
				if fileInfo.IsDir() {
					continue
				}
				confPath := filepath.Join(confDirPath, filename)
				xconf, err := XConfigFromFile(confPath, params)
				if err != nil {
					return config, LoadConfigError{ Path: confPath, Err: err }
				}
				xconf.IntoConfig(&config)
			}
		}
	}
	return
}
