from contextlib import ExitStack
import logging

import django
from django.db import connections
from django.db.backends.utils import CursorDebugWrapper

from ..models import Employee

django_version = django.get_version()
logger = logging.getLogger(__name__)


class SqlCommenter:
    """
    Middleware to append a comment to each database query with details about
    the framework and the execution context.
    """

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        with ExitStack() as stack:
            for db_alias in connections:
                stack.enter_context(
                    connections[db_alias].execute_wrapper(QueryWrapper(request))
                )
            return self.get_response(request)


class QueryWrapper:
    def __init__(self, request):
        self.request = request

    def __call__(self, execute, sql, params, many, context):
        with_framework = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_FRAMEWORK", True
        )
        with_controller = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_CONTROLLER", True
        )
        with_route = getattr(django.conf.settings, "SQLCOMMENTER_WITH_ROUTE", True)
        with_app_name = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_APP_NAME", False
        )
        with_opencensus = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_OPENCENSUS", False
        )
        with_opentelemetry = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_OPENTELEMETRY", False
        )
        with_db_driver = getattr(
            django.conf.settings, "SQLCOMMENTER_WITH_DB_DRIVER", False
        )

        if with_opencensus and with_opentelemetry:
            logger.warning(
                "SQLCOMMENTER_WITH_OPENCENSUS and SQLCOMMENTER_WITH_OPENTELEMETRY were enabled. "
                "Only use one to avoid unexpected behavior"
            )

        db_driver = context["connection"].settings_dict.get("ENGINE", "")
        resolver_match = self.request.resolver_match

        user = None
        user_geo = None

        if (
            "auth_user" not in sql
            and "django_session" not in sql
            and "employee" not in sql
        ):
            user = (
                self.request.user.username
                if self.request.user.is_authenticated
                else None
            )
            if user:
                user_geo = Employee.objects.get(user=self.request.user).location

        sql = add_sql_comment(
            sql,
            # Information about the controller.
            controller=(
                resolver_match.view_name if resolver_match and with_controller else None
            ),
            # route is the pattern that matched a request with a controller i.e. the regex
            # See https://docs.djangoproject.com/en/stable/ref/urlresolvers/#django.urls.ResolverMatch.route
            # getattr() because the attribute doesn't exist in Django < 2.2.
            route=(
                getattr(resolver_match, "route", None)
                if resolver_match and with_route
                else None
            ),
            # app_name is the application namespace for the URL pattern that matches the URL.
            # See https://docs.djangoproject.com/en/stable/ref/urlresolvers/#django.urls.ResolverMatch.app_name
            app_name=(
                (resolver_match.app_name or None)
                if resolver_match and with_app_name
                else None
            ),
            # Framework centric information.
            framework=("django:%s" % django_version) if with_framework else None,
            # Information about the database and driver.
            db_driver=db_driver if with_db_driver else None,
            user=user,
            geo=user_geo,
        )

        print(sql, "\n")

        # TODO: MySQL truncates logs > 1024B so prepend comments
        # instead of statements, if the engine is MySQL.
        # See:
        #  * https://github.com/basecamp/marginalia/issues/61
        #  * https://github.com/basecamp/marginalia/pull/80

        # Add the query to the query log if debugging.
        if isinstance(context["cursor"], CursorDebugWrapper):
            context["connection"].queries_log.append(sql)

        return execute(sql, params, many, context)


#!/usr/bin/python
#
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import sys

if sys.version_info.major <= 2:
    import urllib

    url_quote_fn = urllib.quote
else:
    import urllib.parse

    url_quote_fn = urllib.parse.quote

KEY_VALUE_DELIMITER = ","


def generate_sql_comment(**meta):
    """
    Return a SQL comment with comma delimited key=value pairs created from
    **meta kwargs.
    """
    if not meta:  # No entries added.
        return ""

    # Sort the keywords to ensure that caching works and that testing is
    # deterministic. It eases visual inspection as well.

    return (
        " /*"
        + KEY_VALUE_DELIMITER.join(
            "{}={!r}".format(url_quote(key), url_quote(value))
            for key, value in sorted(meta.items())
            if value is not None
        )
        + "*/"
    )


def add_sql_comment(sql, **meta):
    comment = generate_sql_comment(**meta)
    sql = sql.rstrip()
    if sql[-1] == ";":
        sql = sql[:-1] + comment + ";"
    else:
        sql = sql + comment
    return sql


def url_quote(s):
    if not isinstance(s, (str, bytes)):
        return s
    quoted = url_quote_fn(s)
    # Since SQL uses '%' as a keyword, '%' is a by-product of url quoting
    # e.g. foo,bar --> foo%2Cbar
    # thus in our quoting, we need to escape it too to finally give
    #      foo,bar --> foo%%2Cbar
    return quoted.replace("%", "%%")
