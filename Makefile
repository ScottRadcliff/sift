.DEFAULT_GOAL := help
PYTHON ?= python3

.PHONY: help test portable

help:
	@echo "test      Run Python behavior and integration tests (Python 3.11+)"
	@echo "portable  Verify a copied skill installation in isolation"

test:
	PYTHONPATH=skills/sift/scripts/_vendor $(PYTHON) -m unittest discover -s tests/migration -v

portable:
	$(PYTHON) tests/migration/check_portable_skill.py
