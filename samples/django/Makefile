venv:
	scripts/ensure-python-venv.sh

migrate: venv
	.venv/bin/python3 manage.py migrate

dev: venv
	.venv/bin/python3 manage.py runserver

lint:
	.venv/bin/black backstage/
	.venv/bin/isort --profile=black backstage/
	.venv/bin/ruff check  --fix backstage/
	find backstage/ -type f  -name "*.py" | grep -v .venv | xargs .venv/bin/pyupgrade  --py39-plus

reset-db: 
	rm -f data/db.sqlite3
	.venv/bin/python3 manage.py migrate
