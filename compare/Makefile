SHELL=/bin/bash -o pipefail

type ?=
type_flag = $(if $(type),--type $(type))

.PHONY: compare
compare: | cd.txt oss.txt
	python3 compare.py cd.txt oss.txt $(type_flag)

.PHONE: clean
clean:
	rm cd.txt oss.txt

cd.txt: ../../samples/samples/crewai/compose.yaml
	defang -C ../../samples/samples/crewai estimate -P aws --debug 2>&1 | grep -e '^ - {' > $@

oss.txt: ../examples/crewai/main.go
	AWS_PROFILE=defang-lab AWS_REGION=us-west-2 pulumi -C ../examples/crewai -s dev preview --json | jq '.steps[]' > $@
