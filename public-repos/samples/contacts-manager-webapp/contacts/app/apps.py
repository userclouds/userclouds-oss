from django.apps import AppConfig


class ContactsAppConfig(AppConfig):
    name = "contacts.app"
    models_module = "contacts.app.models"
    label = "contacts_manager"
    verbose_name = "Contacts Manager"
