from django.core.management.base import BaseCommand
from faker import Faker

from contacts.app.models import Contact
from contacts.userclouds.codegensdk import Purpose


class Command(BaseCommand):
    help = "Adds contacts."

    def add_arguments(self, parser):
        parser.add_argument(
            "num_of_contacts",
            type=int,
            default=1,
            help="Number of contacts to add",
        )
        parser.add_argument(
            "purposes_args",
            metavar="purposes",
            nargs="*",
            choices=[p.value for p in Purpose],
            type=str,
            help="Purposes (consents) to add to the contacts",
        )

    def handle(self, *args, num_of_contacts: int, purposes_args: list[str], **kwargs):
        purposes = {Purpose[p.upper()] for p in purposes_args}
        fk = Faker()
        before_count = Contact.objects.count()
        for _ in range(num_of_contacts):
            contact = Contact(
                name=fk.name(),
                email=fk.email(),
                phone=fk.basic_phone_number(),
                nickname=fk.user_name(),
            )
            contact.save_with_userclouds(purposes=purposes)
        self.stdout.write(
            self.style.SUCCESS(
                f"Added {num_of_contacts}, before count: {before_count}, after count: {Contact.objects.count()} "
            )
        )
