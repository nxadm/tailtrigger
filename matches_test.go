package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

//
//import (
//	"regexp"
//	"testing"
//	"time"
//)
//
func TestWatch(t *testing.T) {
	//dn: uid=r000000001,ou=people,dc=something,dc=be
	//changetype: modify
	//replace: pwdAccountLockedTime
	//pwdAccountLockedTime: 20170424064252Z
	fileWatch := FileWatch{RecDelimRxString: "^#"}
	fileWatch.RecDelimRxString = "^#"
	matchRxStr := `(?ms)(?P<dn>^dn:\s+.+?)`
	action := Action{Name: "testact", Type: "local", RunTemplate: `echo {{ .dn }}`}
	actions := make(map[string]*Action)
	actions[action.Name] = &action
	trigger := Trigger{Name: "test", MatchRxStr: matchRxStr, Actions: actions}
	triggers := make(map[string]*Trigger)
	triggers[trigger.Name] = &trigger
	fileWatch.Triggers = triggers
	config := make(map[string]FileWatch)
	config["audit_db.log"] = fileWatch
	absConfigFile, _ := filepath.Abs("t/config.yaml")
	fileWatches, _ := generate(config, absConfigFile)
	fmt.Printf("%+v\n", fileWatches)
	go fileWatches[0].watch()
	time.Sleep(time.Second * 2)
}
