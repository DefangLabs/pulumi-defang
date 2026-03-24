cd.txt:
	defang -C ../../defang-mvp/fabric/cmd/cd/testdata estimate -P aws --debug 2>&1 | grep -e '^ - {' > cd.txt

oss.txt:
	AWS_PROFILE=defang-lab AWS_REGION=us-west-2 pulumi -C ../examples/crewai preview --json | jq '.steps[]' > oss.txt

type ?=
type_flag = $(if $(type),--type $(type))

.PHONY: compare
compare: | cd.txt oss.txt
	python3 compare.py cd.txt oss.txt $(type_flag)
