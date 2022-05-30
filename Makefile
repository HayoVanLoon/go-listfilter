
TMP_DIR=tmp

test:
	go test	\
		-bench . \
		-race

profile:
	mkdir -p $(TMP_DIR)
	TS=$(shell date +%s) && \
	go test \
		-test.bench . \
		-cpuprofile $(TMP_DIR)/cpu-$$TS.prof \
		-o $(TMP_DIR)/$(shell basename $(shell pwd))-$$TS.test && \
	pprof -http :6060 -focus '.+listfilter.+'  $(TMP_DIR)/cpu-$$TS.prof
