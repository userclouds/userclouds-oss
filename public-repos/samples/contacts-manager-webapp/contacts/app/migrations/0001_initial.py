# Generated by Django 5.0.2 on 2024-02-09 19:23

import datetime

from django.db import migrations, models


class Migration(migrations.Migration):

    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="Contact",
            fields=[
                (
                    "contact_id",
                    models.UUIDField(editable=False, primary_key=True, serialize=False),
                ),
                ("name", models.CharField(max_length=20)),
                ("email", models.EmailField(max_length=100)),
                ("phone", models.CharField(max_length=20)),
                ("nickname", models.CharField(max_length=30)),
                ("image", models.ImageField(blank=True, upload_to="images/")),
                ("date_added", models.DateTimeField(default=datetime.datetime.utcnow)),
                ("is_deleted", models.BooleanField(default=False)),
            ],
        ),
    ]
