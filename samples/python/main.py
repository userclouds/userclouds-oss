from functools import wraps

from flask import Flask, jsonify, Response, redirect, request, session

from authlib.integrations.flask_client import OAuth

BASE_URL = "https://sample.tenant.userclouds.com"

app = Flask(__name__)
app.secret_key = "REPLACE_ME_WITH_SOMETHING_SECRET"
oauth = OAuth(app)
provider = oauth.register(
    "userclouds",
    client_id="5f107e226353791560f93164a09f7e0f",
    client_secret="2ftQe4RU7aR/iStpcFf3gfiUjnsbWGFY0C9aWkPzDqT14eyp23ysKuI6iBbOAW/O",
    api_base_url=BASE_URL,
    access_token_url=BASE_URL + "/oidc/token",
    authorize_url=BASE_URL + "/oidc/authorize",
    client_kwargs={
        "scope": "openid profile email",
    },
)


def requires_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        if "user_id" not in session:
            session["redirect"] = request.path
            return redirect("/login")
        return f(*args, **kwargs)

    return decorated


@app.route("/")
def index() -> Response:
    response = 'Hello world ... want to know a <a href="/private">secret</a>?'
    return response


@app.route("/private")
@requires_auth
def private() -> Response:
    response = "I'm a secret, " + session["user_id"] + ". <a href='/logout'>Logout</a>"
    return response


@app.route("/login")
def login():
    return provider.authorize_redirect(redirect_uri="http://localhost:8080/callback")


@app.route("/logout")
def logout():
    session.clear()
    return redirect("/")


@app.route("/callback")
def callback():
    provider.authorize_access_token()
    ui = provider.get("oidc/userinfo").json()
    session["user_id"] = ui["sub"]

    redir = "/"
    if "redirect" in session:
        redir = session["redirect"]
        session.pop("redirect")

    return redirect(redir)


if __name__ == "__main__":
    app.run(host="localhost", port="8080")
