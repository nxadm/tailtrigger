package main

import (
	"regexp"
)

/* Functions called from outside this file */
func (fileWatch *FileWatch) watch() {
	record := ""
	for line := range fileWatch.FileTail.Lines {
		/* Create records */
		if fileWatch.RecDelimRx != nil &&
			!fileWatch.RecDelimRx.MatchString(line.Text+"\n") {
			record = record + line.Text + "\n"
			continue
		}
		if record == "" {
			record = line.Text + "\n"
		}

		/* Triggers */
		for _, trigger := range fileWatch.Triggers {
			matched, matchedVars := captureGroupsToMap(trigger.MatchRx, record)
			if matched {
				//go trigger.act(matchedVars)
				trigger.act(matchedVars)
			}
		}
		record = ""
	}
}

/* Functions ony called from this file */
func captureGroupsToMap(
	rx *regexp.Regexp, text string) (bool, map[string]string) {

	match := rx.FindStringSubmatch(text)
	if match == nil {
		return false, nil
	}

	result := make(map[string]string)
	for i, name := range rx.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return true, result
}
