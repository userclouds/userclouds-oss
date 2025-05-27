venv:
	scripts/ensure-python-venv.sh

check-env:
	scripts/check-env.sh

migrate: venv check-env
	.venv/bin/python3 manage.py migrate

dev: venv check-env
	.venv/bin/python3 manage.py runserver

lint:
	.venv/bin/black contacts/
	.venv/bin/isort --profile=black contacts/
	.venv/bin/ruff check  --fix contacts/
	find contacts/ -type f  -name "*.py" | grep -v .venv | xargs .venv/bin/pyupgrade  --py39-plus

download-ucconfig:
	scripts/download-ucconfig.sh

apply-config: check-env
	bin/ucconfig apply config/uc-manifest.yaml

bootstrap: download-ucconfig apply-config migrate dev

 clean:
	find contacts/ -type d -name "__pycache__" -print0 | xargs rm -rf

update-codegen:
	.venv/bin/python3 manage.py updatecodegen
	make lint

container: clean
	docker build --platform=linux/amd64 -t contacts-manager-webapp:v0 .

container-run:
	docker run -p 8000:8000 --env USERCLOUDS_TENANT_URL --env USERCLOUDS_CLIENT_ID --env USERCLOUDS_CLIENT_SECRET -it contacts-manager-webapp:v0

reset-db: check-env
	rm -f data/db.sqlite3
	.venv/bin/python3 manage.py migrate
	.venv/bin/python3 manage.py deleteucusers
