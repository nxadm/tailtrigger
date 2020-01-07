#!/usr/bin/env bash -e
cp tailtrigger.service /etc/systemd/system/
chmod 664 /etc/systemd/system/tailtrigger.service
systemctl daemon-reload
systemctl start tailtrigger.service
systemctl enable tailtrigger.service 
