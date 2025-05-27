from django.conf import settings
from django.core.management.base import BaseCommand
from usercloudssdk.client import Client


class Command(BaseCommand):
    help = "Deletes all users from UserClouds."

    def handle(self, *args, **kwargs):
        uc: Client = settings.UC_CLIENT
        users = uc.ListUsers()
        self.stdout.write(self.style.NOTICE(f"Deleting {len(users)} users"))
        for user in users:
            uc.DeleteUser(user.id)
        self.stdout.write(self.style.SUCCESS("Done"))
