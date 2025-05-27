from pathlib import Path

from django.conf import settings
from django.core.management.base import BaseCommand
from usercloudssdk.client import Client


class Command(BaseCommand):
    help = "Update UserClouds codegen file"

    _CODEGEN_PATH = Path("contacts/userclouds/codegensdk.py")

    def handle(self, *args, **kwargs):
        uc: Client = settings.UC_CLIENT
        uc.SaveUserstoreSDK(self._CODEGEN_PATH)
        self.stdout.write(self.style.SUCCESS("Done"))
