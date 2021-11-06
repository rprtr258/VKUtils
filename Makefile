VKUTILS:=go run main.go --token $(shell cat .env)

.PHONY: help
help: # show list of all commands
	@grep -E '^[a-zA-Z_-]+:.*?# .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?# "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: todo
todo: # show list of all todos left in code
	@rg 'TODO' --glob '**/*.go' || echo 'All done!'

.PHONY: lint
lint: # run linter
	golangci-lint run --exclude-use-default=false --disable-all --enable=revive --enable=deadcode --enable=errcheck --enable=govet --enable=ineffassign --enable=structcheck --enable=typecheck --enable=varcheck --enable=asciicheck --enable=bidichk --enable=bodyclose --enable=containedctx --enable=contextcheck --enable=cyclop --enable=decorder --enable=depguard --enable=dogsled --enable=dupl --enable=durationcheck --enable=errchkjson --enable=errname --enable=errorlint --enable=execinquery --enable=exhaustive --enable=exhaustruct --enable=exportloopref --enable=forbidigo --enable=forcetypeassert --enable=funlen --enable=gochecknoglobals --enable=gochecknoinits --enable=gocognit --enable=goconst --enable=gocritic --enable=gocyclo --enable=godot --enable=godox --enable=goerr113 --enable=gofmt --enable=gofumpt --enable=goimports --enable=gomnd --enable=gomoddirectives --enable=gomodguard --enable=goprintffuncname --enable=gosec --enable=grouper --enable=ifshort --enable=importas --enable=lll --enable=maintidx --enable=makezero --enable=misspell --enable=nestif --enable=nilerr --enable=nilnil --enable=noctx --enable=nolintlint --enable=nosprintfhostport --enable=paralleltest --enable=prealloc --enable=predeclared --enable=promlinter --enable=rowserrcheck --enable=sqlclosecheck --enable=tenv --enable=testpackage --enable=thelper --enable=tparallel --enable=unconvert --enable=unparam --enable=wastedassign --enable=whitespace --enable=wrapcheck

.PHONY: run-reposts
run-reposts: WALL_URL?=https://vk.com/wall-149859311_975
run-reposts: # run finding reposts
	@$(VKUTILS) reposts -u $(WALL_URL)

.PHONY: run-dumpwall
run-dumpwall: GROUP_URL?=https://vk.com/tsgnmdmvn
run-dumpwall: # run dumping group wall
	@$(VKUTILS) dumpwall -u $(GROUP_URL)

.PHONY: run-count
run-count: # run counting intersection between friends and group members
	@$(VKUTILS) count --friends 168715495 --groups -187839235
