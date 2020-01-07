#!/bin/bash -e
go build
echo > README.md
echo "# tailtrigger" >> README.md
echo >> README.md
echo "Trigger actions by matching regexes in files" >> README.md
echo >> README.md
echo "## Run" >> README.md
echo >> README.md
echo '```text' >> README.md
echo "$ tailtrigger -h" >> README.md
./tailtrigger -h >> README.md
echo '```' >> README.md
echo >> README.md
echo "## Configuration" >> README.md
echo >> README.md
echo '```yaml' >> README.md
./tailtrigger -s >> README.md
echo '```' >> README.md
