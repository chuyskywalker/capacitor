#!/bin/bash -x

go get github.com/mitchellh/gox

go get github.com/tcnksm/ghr

gox -output "dist/{{.OS}}_{{.Arch}}_{{.Dir}}"

ghr --username chuyskywalker --token $GITHUB_TOKEN --replace --prerelease --debug pre-release dist/
