-include .makefiles/Makefile
-include .makefiles/pkg/protobuf/v2/Makefile
-include .makefiles/pkg/protobuf/v2/with-primo.mk
-include .makefiles/pkg/go/v1/Makefile
-include .makefiles/pkg/vscode/v1/Makefile

@PHONY: run-example
run-example: $(GO_DEBUG_DIR)/example
	$^ $(args)

.makefiles/%:
	@curl -sfL https://makefiles.dev/v1 | bash /dev/stdin "$@"
