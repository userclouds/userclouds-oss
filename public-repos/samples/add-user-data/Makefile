venv:
	scripts/ensure-python-venv.sh

check-env:
	scripts/check-env.sh



lint:
	.venv/bin/black .
	.venv/bin/isort --profile=black .
	.venv/bin/ruff check  --fix .
	find . -type f  -name "*.py" | grep -v .venv | xargs .venv/bin/pyupgrade  --py39-plus


 clean:
	find uctool/ -type d -name "__pycache__" -print0 | xargs rm -rf

