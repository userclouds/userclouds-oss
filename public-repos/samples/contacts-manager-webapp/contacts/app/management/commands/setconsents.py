from django.core.management.base import BaseCommand

from contacts.app.models import Contact
from contacts.userclouds.codegensdk import Purpose


class Command(BaseCommand):
    help = "Resets consents on existing contacts."

    def add_arguments(self, parser):
        parser.add_argument(
            "purposes_args",
            metavar="purposes",
            nargs="*",
            choices=[p.value for p in Purpose],
            type=str,
            help="Purposes (consents) to add to the contacts",
        )

    def handle(self, *args, purposes_args: list[str], **kwargs):
        purposes = [Purpose[p.upper()] for p in purposes_args]
        all_contacts = list(Contact.objects.all())
        self.stdout.write(
            self.style.NOTICE(f"Updating {len(all_contacts)} with {purposes_args}")
        )
        for cn in all_contacts:
            cn.save_with_purposes(purposes)
        self.stdout.write(self.style.SUCCESS("All done"))
