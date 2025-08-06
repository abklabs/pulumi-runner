check:

.PHONY: check

.env-checked: bin/check-env
	./bin/check-env
	touch .env-checked

include .env-checked
