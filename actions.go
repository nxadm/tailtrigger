package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

type RunInfo struct {
	Cmd, CmdOutput, ErrCategory, URL, JSON, Body string
	HasRun, isErr, Success                       bool
	HTTPStatus                                   int
	Err                                          error
}

/* Functions called from outside this file */
func (trigger *Trigger) act(matchedVars map[string]string) {
	for _, action := range trigger.Actions {
		//go action.run(matchedVars)
		action.run(matchedVars)
	}
}

/* Functions ony called from this file */
func fillTemplate(templateStr string, values map[string]string) (string, error) {
	var buffer bytes.Buffer
	tmpl, err := template.New("tmpl").Parse(templateStr)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(&buffer, values)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func printActInfo(action *Action, runInfo RunInfo) {
	switch {
	case action.Type == "local" && runInfo.CmdOutput == "":
		runInfo.CmdOutput = "<none>"
	case action.Type == "rest" && runInfo.Body == "":
		runInfo.Body = "<none>"
	}

	switch {
	case runInfo.Err != nil && runInfo.HasRun && debug:
		if runInfo.Cmd != "" {
			log.Printf("[Error] [%s]/[%s]/[%s]: success => false, %s error => %s, output => %s\n",
				action.FileWatchName, action.TriggerName, action.Name,
				runInfo.ErrCategory, runInfo.Err, runInfo.CmdOutput)
		} else {
			log.Printf("[Error] [%s]/[%s]/[%s]: HTTP status => %d, %s error => %s, output => %s\n",
				action.FileWatchName, action.TriggerName, action.Name,
				runInfo.HTTPStatus, runInfo.ErrCategory, runInfo.Err, runInfo.Body)
		}
	case runInfo.Err != nil && runInfo.HasRun:
		if runInfo.Cmd != "" {
			log.Printf("[Error] [%s]/[%s]/[%s]: success => false, %s error => %s\n",
				action.FileWatchName, action.TriggerName, action.Name,
				runInfo.ErrCategory, runInfo.Err)
		} else {
			log.Printf("[Error] [%s]/[%s]/[%s]: HTTP status => %d, %s error => %s\n",
				action.FileWatchName, action.TriggerName, action.Name,
				runInfo.HTTPStatus, runInfo.ErrCategory, runInfo.Err)
		}
	case runInfo.Err != nil:
		log.Printf("[Error] [%s]/[%s]/[%s]: success => false, %s error => %s\n",
			action.FileWatchName, action.TriggerName, action.Name, runInfo.ErrCategory, runInfo.Err)
	case !runInfo.HasRun && debug:
		if runInfo.Cmd != "" {
			log.Printf("Running [%s]/[%s]/[%s]: command => %s\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.Cmd)
		} else {
			if runInfo.JSON == "" {
				runInfo.JSON = "<none>"
			}
			log.Printf("Running [%s]/[%s]/[%s]: url => %s, json => %s\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.URL, runInfo.JSON)
		}
	case !runInfo.HasRun:
		log.Printf("Running [%s]/[%s]/[%s]\n",
			action.FileWatchName, action.TriggerName, action.Name)
	case runInfo.HasRun && debug:
		if runInfo.Cmd != "" {
			log.Printf("Result [%s]/[%s]/[%s]: success => %t, output => %s\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.Success, runInfo.CmdOutput)
		} else {
			log.Printf("Running [%s]/[%s]/[%s]: HTTP status => %d, body => %s\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.HTTPStatus, runInfo.Body)
		}
	case runInfo.HasRun:
		if runInfo.Cmd != "" {
			log.Printf("Result [%s]/[%s]/[%s]: success => %t\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.Success)
		} else {
			log.Printf("Running [%s]/[%s]/[%s]: HTTP status => %d\n",
				action.FileWatchName, action.TriggerName, action.Name, runInfo.HTTPStatus)
		}
	}
}

func (action *Action) run(matchedVars map[string]string) {
	runInfo := RunInfo{}
	switch action.Type {
	case "local":
		runCmd(action, matchedVars, runInfo)
	case "rest":
		runRest(action, matchedVars, runInfo)
	}
}

func runCmd(action *Action, matchedVars map[string]string, runInfo RunInfo) {
	cmdStr, err := fillTemplate(action.RunTemplate, matchedVars)
	runInfo.Cmd = cmdStr
	if err != nil {
		runInfo.Err = err
		runInfo.ErrCategory = "template"
		printActInfo(action, runInfo)
		return
	}
	printActInfo(action, runInfo)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	output, err :=
		exec.CommandContext(ctx, "sh", "-c", cmdStr).CombinedOutput()
	runInfo.CmdOutput = strings.TrimSpace(strings.Replace(string(output), "\n", "␤", -1))
	if err != nil {
		runInfo.Err = err
		runInfo.ErrCategory = "runtime"
		runInfo.Success = false
	} else {
		runInfo.Success = true
	}
	runInfo.HasRun = true
	printActInfo(action, runInfo)
}

func runRest(action *Action, matchedVars map[string]string, runInfo RunInfo) {
	var jsonStr string
	var body io.Reader

	if action.JSONTemplate != "" {
		jsonStrTmp, err := fillTemplate(action.JSONTemplate, matchedVars)
		if err != nil {
			runInfo.Err = err
			runInfo.ErrCategory = "template"
			printActInfo(action, runInfo)
			return
		}
		jsonStr = jsonStrTmp
	}
	if jsonStr == "" {
		body = nil
	} else {
		body = strings.NewReader(jsonStr)
	}

	url, err := fillTemplate(action.URLTemplate, matchedVars)
	if err != nil {
		runInfo.Err = err
		runInfo.ErrCategory = "template"
		printActInfo(action, runInfo)
		return
	}
	req, err := http.NewRequest(action.HTTPVerb, url, body)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	if err != nil {
		runInfo.Err = err
		runInfo.ErrCategory = "connection"
		printActInfo(action, runInfo)
		return
	}

	if action.BasicAuth != "" {
		req.Header.Add("Authorization", action.BasicAuth)
	}

	req.Header.Set("Content-Type", "application/json")

	var netClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := netClient.Do(req.WithContext(ctx))
	//resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		runInfo.Err = err
		runInfo.ErrCategory = "connection"
		printActInfo(action, runInfo)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	runInfo.Body = strings.TrimSpace(strings.Replace(string(bodyBytes), "\n", "␤", -1))
	runInfo.HasRun = true
	switch {
	case resp.StatusCode >= 400 && resp.StatusCode < 600:
		runInfo.HTTPStatus = resp.StatusCode
		runInfo.ErrCategory = "REST"
		runInfo.Err = errors.New(runInfo.Body)
		runInfo.Success = false
	default:
		runInfo.HTTPStatus = resp.StatusCode
		runInfo.Success = true
	}
	printActInfo(action, runInfo)
}
