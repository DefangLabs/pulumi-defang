cd.txt:
	defang -C ../../defang-mvp/fabric/cmd/cd/testdata estimate -P aws --debug 2>&1 | grep -e '^ - {' > cd.txt

oss.txt:
	AWS_PROFILE=defang-lab AWS_REGION=us-west-2 pulumi -C ../examples/crewai preview --json | jq '.steps[]' > oss.txt

.PHONY: compare compare-type
compare: | cd.txt oss.txt
	python3 compare.py cd.txt oss.txt

type ?=
compare-type: | cd.txt oss.txt
	python3 compare.py cd.txt oss.txt --type $(type)
