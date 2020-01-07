package main

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nxadm/tail"
	"gopkg.in/yaml.v3"
)

var sampleConfigFile = `---
# A valid configuration consists of blocks headed by the filename to be
# watched (absolute path or relative to this configuration file). By default,
# files are read line by line, i.e. separated by a new line). If the format
# of the file consists of multi-line records (e.g. a LDAP audit log), you can
# enable matching on the record level by supplying a 'record-delimiter' regex.
# In record mode it may be a good idea to start the regex with '(?ms)'. 'm'
# enables multi-line mode (^ and $ match begin/end line in addition to
# begin/end text) and 's' lets '.' also match '\n'.
#
# Each configuration block contains named 'triggers'. In turn, each trigger
# contains a 'match-regex'. When matched named 'actions' wil be watch. Each
# action of a certain type ('local', 'rest') and has corresponding attributes.
# Type 'local' runs local programs and needs a 'watch-template'. Type 'rest'
# send a REST request and needs a 'url-template', a 'http-verb' (default POST)
# and optionally a 'json-template', a Basic Auth 'user' and 'pass'.
#
# The regexes and templates must be provided in the syntax of the Go language
# (see link below). The use of *-template instead of literal values are meant
# to allow the use of the values of named captures in the 'match-regex'. When
# named captures are used (?P<name>), templates will receive the value of the
# captures as '.name', e.g. '{{ .username }}'. The configuration file itself
# is valid YAML, so muli-tine constructs (next line + indent, '>', '|') can
# be used when providing long regexes or templates.
#
# Regex and template values starting with '@' (you need to quote these strings
# in YAML) are expanded to the contents of the file they reference. Their path
# can be absolute or relative to this configuration file. Like on the YAML
# configuration, the last newline is stripped. This option can be useful for
# big templates.
#
# Links:
# https://golang.org/pkg/regexp/syntax/
# https://golang.org/pkg/text/template/
# https://yaml-multiline.info/
'audit_db.log':
  record-delimiter: '^#'
  triggers:
    password-lock:
      match-regex:
        (?ms)(?P<dn>^dn:\s+.+?)\n.+?replace:\s+pwdAccountLockedTime\npwdAccountLockedTime:\s+(?P<datetime>\d{14}Z)
      actions:
        syslog:
          type: local
          run-template: "logger -t INFO {{ .dn }} locked at {{ .datetime }}"
        remote-server-1:
          type: rest
          url-template: 'http://localhost/v1/foo?action=lock?date={{ .datetime }}'
          user: foo
          pass: bar
    password-unlock:
      match-regex:
        (?ms)(?P<dn>^dn:\s+.+?)\n.+?delete:\s+pwdAccountLockedTime\n.+?modifyTimestamp:\s+(?P<datetime>\d{14}Z)
      actions:
        syslog:
          type: local
          run-template: "logger -t INFO {{ .dn }} unlocked at {{ .datetime }}"
        remote-server-1:
          type: rest
          url-template: 'http://localhost/v1/foo?action=unlock?date={{ .datetime }}'
          user: foo
          pass: bar
# Other file blocks ...
`
var configErr []error

type YAMLConfig map[string]FileWatch

type FileWatch struct {
	FileName         string
	FileTail         *tail.Tail
	RecDelimRx       *regexp.Regexp
	RecDelimRxString string              `yaml:"record-delimiter,omitempty"`
	Triggers         map[string]*Trigger `yaml:"triggers,flow"`
}

type Trigger struct {
	Name, FileWatchName string
	MatchRx             *regexp.Regexp
	MatchRxStr          string             `yaml:"match-regex,omitempty"`
	Actions             map[string]*Action `yaml:"actions,flow"`
}

type Action struct {
	Name, BasicAuth, FileWatchName, TriggerName string
	Type                                        string `yaml:"type,omitempty"`
	RunTemplate                                 string `yaml:"run-template,omitempty"`
	URLTemplate                                 string `yaml:"url-template,omitempty"`
	HTTPVerb                                    string `yaml:"http-verb,omitempty"`
	JSONTemplate                                string `yaml:"json-template,omitempty"`
	User                                        string `yaml:"user,omitempty"`
	Pass                                        string `yaml:"pass,omitempty"`
}

/* Functions called from outside this file */
func importConfigFile(configFile string) ([]FileWatch, []error) {
	data, err := ioutil.ReadFile(configFile)
	checkConfigErr(err)

	var config YAMLConfig
	err = yaml.Unmarshal(data, &config)
	checkConfigErr(err)

	absConfigFile, err := filepath.Abs(configFile)
	if err != nil {
		log.Printf("[WARNING] Can nog determine the path for " + configFile)
		absConfigFile = configFile
	}
	return generate(config, filepath.Dir(absConfigFile))
}

/* Functions ony called from this file */
func generate(config YAMLConfig, configFileDir string) ([]FileWatch, []error) {
	var fileWatches []FileWatch
	for fileName, fileWatch := range config {
		/* top-level parameters and global objects */
		fileWatch.FileName = getAbsPath(fileName, configFileDir)
		t, err := tail.TailFile(fileWatch.FileName, tail.Config{
			Follow: true,
			ReOpen: true,
		})
		checkConfigErr(err)
		fileWatch.FileTail = t

		fileWatch.RecDelimRx = compileRx(
			expandContents(fileWatch.RecDelimRxString),
			"\"record-delimiter\" can not be set [file: "+fileName+"]", true)

		/* triggers */
		for triggerName, trigger := range fileWatch.Triggers {
			trigger.Name = triggerName
			trigger.FileWatchName = fileWatch.FileName
			trigger.MatchRx = compileRx(
				expandContents(trigger.MatchRxStr),
				"\"match-regex\" missing [file: "+fileName+", trigger: "+triggerName+"]",
				false)
			for actionName, action := range trigger.Actions {
				action.Name = actionName
				action.TriggerName = triggerName
				action.FileWatchName = fileWatch.FileName
				action.generate()
			}
		}

		// Construct the return value
		fileWatches = append(fileWatches, fileWatch)
	}
	return fileWatches, configErr
}

func (action *Action) generate() {
	switch action.Type {
	case "local":
		if action.RunTemplate == "" {
			err := errors.New(
				"\"run-template\" missing [file: " + action.FileWatchName +
					", trigger: " + action.TriggerName +
					", action: " + action.Name + "]")
			checkConfigErr(err)
		} else {
			action.RunTemplate = expandContents(action.RunTemplate)
		}
	case "rest":
		if action.URLTemplate == "" {
			err := errors.New(
				"\"url-template\" missing [file: " + action.FileWatchName +
					", trigger: " + action.TriggerName +
					", action: " + action.Name + "]")
			checkConfigErr(err)
		} else {
			action.URLTemplate = expandContents(action.URLTemplate)
		}
		if action.HTTPVerb == "" {
			action.HTTPVerb = "POST"
		}
		if action.User != "" && action.Pass != "" {
			auth := action.User + ":" + action.Pass
			action.BasicAuth = "Basic " +
				base64.StdEncoding.EncodeToString([]byte(auth))
		}
		action.JSONTemplate = expandContents(action.JSONTemplate)
	}
}

func checkConfigErr(err error) {
	if err != nil {
		configErr = append(configErr, err)
	}
}

func compileRx(rxStr, errStr string, allowEmpty bool) *regexp.Regexp {
	var rx *regexp.Regexp
	switch {
	case rxStr == "" && !allowEmpty:
		configErr =
			append(configErr, errors.New(errStr))
	case rxStr == "":
	default:
		rxTmp, err := regexp.Compile(rxStr)
		checkConfigErr(err)
		rx = rxTmp

	}
	return rx
}

func getAbsPath(fileName, configFileDir string) string {
	if filepath.IsAbs(fileName) {
		return fileName
	}
	return filepath.Join(filepath.Join(configFileDir), fileName)
}

func expandContents(value string) string {
	if !strings.HasPrefix(value, "@") {
		return value
	}
	contents, err := ioutil.ReadFile(strings.TrimLeft(value, "@"))
	checkConfigErr(err)
	switch err {
	case nil:
		// trim new line on unix and windows
		return strings.TrimSuffix(
			strings.TrimSuffix(string(contents), "\n"), "\r")
	default:
		return ""
	}
}
