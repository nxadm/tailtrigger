
# tailtrigger

Trigger actions by matching regexes in files

## Run

```text
$ tailtrigger -h
tailtrigger, 0.5.2.
Trigger actions by matching regexes in logfiles.
See https://github.com/nxadm/tailtrigger for more information.
Author: Claudio Ramirez <pub.claudio@gmail.com>

Usage:
  tailtrigger [-c <configuration file>] [-d] [-t seconds]
  tailtrigger [-s]
  tailtrigger [-h]
  tailtrigger [-v]


Parameters:
  -c  | --config        : Configuration file [default: config.yaml].
  -d  | --debug         : Log extra runtime information.
  -t  | --timeout       : timeout seconds for actions [default: 0]
  -s  | --sample-config : Print a sample configuration.
  -h  | --help          : This help message.
  -v  | --version       : Version message.

```

## Configuration

```yaml
---
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

```
