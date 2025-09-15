#!/bin/bash
echo "Parent process started"
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1 &
ping 127.0.0.1 -c 100 >/dev/null 2>&1
