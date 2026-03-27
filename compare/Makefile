SHELL=/bin/bash -o pipefail

type ?=
all ?=
type_flag = $(if $(type),--type $(type))
all_flag = $(if $(all),--all)
PULUMI_STACK ?= example
EXAMPLE ?= gcp-go

export PULUMI_CONFIG_PASSPHRASE=foobar

.PHONY: compare
compare: | cd.txt oss.txt
	python3 compare.py cd.txt oss.txt $(type_flag) $(all_flag)

cd.txt: ../examples/$(EXAMPLE)/compose.yaml
	defang -C ../examples/$(EXAMPLE) cd preview -s $(PULUMI_STACK) --json 2>&1 | grep 'cd pulumi {' |cut -d' ' -f4- > cd.txt

oss.txt: ../examples/$(EXAMPLE)/main.go
	pulumi -C ../examples/$(EXAMPLE) preview --json | jq '.steps[]' > oss.txt

.PHONY: clean_oss
clean_oss:
	rm -f oss.txt

.PHONY: clean_cd
clean_cd:
	rm -f cd.txt

.PHONY: clean
clean: clean_oss clean_cd
