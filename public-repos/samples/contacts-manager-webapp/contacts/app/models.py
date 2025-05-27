from __future__ import annotations

import logging
import uuid

from django.conf import settings
from django.db import models
from django.utils import timezone
from usercloudssdk.client import Client
from usercloudssdk.models import ResourceID

from contacts.userclouds.codegensdk import Purpose, UsersObject

_logger = logging.getLogger(__name__)


def get_uc() -> Client:
    return settings.UC_CLIENT


class Contact(models.Model):
    contact_id = models.UUIDField(primary_key=True, editable=False)
    name = models.CharField(max_length=20)
    email = models.EmailField(max_length=100)
    phone = models.CharField(max_length=20)
    nickname = models.CharField(max_length=30)
    image = models.ImageField(upload_to="images/", blank=True)
    date_added = models.DateTimeField(default=timezone.now)
    is_deleted = models.BooleanField(default=False)

    @property
    def id(self):
        return self.contact_id

    @property
    def is_email_resolved(self) -> bool:
        return getattr(self, "_raw_email_resolved", False)

    def __str__(self) -> str:
        return f"{self.name}"

    def to_user_object(self) -> UsersObject:
        return UsersObject(
            id=self.contact_id,
            name=self.name,
            email=self.email,
            phone_number=self.phone,
            nickname=self.nickname,
            # picture=self.image.url,
        )

    def save_with_userclouds(self, purposes: set[Purpose]) -> None:
        cl = get_uc()
        is_new = self.contact_id is None
        if self.contact_id is None:
            # new user
            uid = cl.CreateUser()
            _logger.info(f"UC: Created user {uid=}")
            self.contact_id = uid
            consents_to_remove = set()
            consents_to_add = purposes
        else:
            uid = self.contact_id
            existing_consents = get_consents_for_user(uid)
            consents_to_remove = existing_consents - purposes
            consents_to_add = purposes - existing_consents
        user = self.to_user_object()
        resp = cl.UpdateUserForPurposes(consents_to_add, user, uid)
        _logger.info(f"UC: UpdateUserForPurposes user {uid=}  {purposes=}- {resp=}")
        RemoveUserConsents(cl, user, consents_to_remove)
        tokens = cl.GetTokenizedPII(uid)[0]
        self.email = tokens.email
        self.phone = tokens.phone_number
        super().save(force_insert=is_new, force_update=not is_new)

    def delete(self):
        if self.is_deleted:
            return
        _logger.info(f"UC: Deleting user {self.contact_id=}")
        cl = get_uc()
        # If we delete the data in UC, with the current implementation, accessors won't work so we won't be able to resolve tokenized email
        # so Instead of deleting the user, we will just revoke all consents except for the deletion purpose
        # cl.DeleteUser(self.contact_id)
        existing_consents = get_consents_for_user(self.contact_id)
        consents_to_remove = existing_consents - {Purpose.FRAUD}
        user = get_raw_contact(self.contact_id).to_user_object()
        RemoveUserConsents(cl, user, consents_to_remove)
        self.is_deleted = True
        self.nickname = "REDACTED"
        self.name = "REDACTED"
        super().save(force_insert=False, force_update=True)


def RemoveUserConsents(
    cl: Client, user: UsersObject, consents_to_remove: set[Purpose]
) -> None:
    if not consents_to_remove:
        return
    purpose_deletions = list(map(lambda p: {"name": p.value}, consents_to_remove))
    row_data = {
        "email": {"value": user.email, "purpose_deletions": purpose_deletions},
        "name": {"value": user.name, "purpose_deletions": purpose_deletions},
        "phone_number": {
            "value": user.phone_number,
            "purpose_deletions": purpose_deletions,
        },
        "nickname": {"value": user.nickname, "purpose_deletions": purpose_deletions},
        "picture": {"value": user.picture, "purpose_deletions": purpose_deletions},
    }
    cl.ExecuteMutator("45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc", {}, [user.id], row_data)
    _logger.info(f"UC: RemoveUserConsents user {user.id=}  {consents_to_remove=}")


def user_to_contact(user: UsersObject, is_email_resolved: bool) -> Contact:
    # image_url
    cn = Contact(
        contact_id=user.id,
        name=user.name,
        email=user.email,
        phone=user.phone_number,
        nickname=user.nickname,
        # image=user.picture,
        is_deleted=False,
    )
    if is_email_resolved:
        cn._raw_email_resolved = True
    return cn


def get_raw_contacts_qc():
    client = get_uc()
    ids = list(Contact.objects.all().values_list("contact_id", flat=True))
    users = client.GetUsers(ids)
    return UCQuerySet([user_to_contact(u, is_email_resolved=True) for u in users])


def get_raw_contact(user_id: uuid.UUID) -> Contact | None:
    client = get_uc()
    users = client.GetUsers([user_id])
    if not users:
        return None
    return user_to_contact(users[0], is_email_resolved=True)


class UCQuerySet:
    def __init__(self, contacts):
        self._contacts = contacts

    def __iter__(self):
        return iter(self._contacts)


def resolve_email_for_purpose(contact: Contact, purpose: Purpose) -> None:
    cl = get_uc()
    tokenized_email = contact.email
    purposes_rids = [ResourceID(name=purpose.value)]
    resp = cl.ResolveTokens([tokenized_email], context={}, purposes=purposes_rids)
    if len(resp) != 1:
        raise ValueError(f"Expected 1 resolved value, got {len(resp)}")
    resolved_value = resp[0]["data"]
    if not resolved_value:
        _logger.warning(f"Failed to resolve email {tokenized_email} for {purpose=}")
        contact._raw_email_resolved = False
    else:
        _logger.info(
            f"Resolved user: {contact.contact_id} email for {purpose=}: {tokenized_email} -> {resolved_value}"
        )
        contact.email = resolved_value
        contact._raw_email_resolved = True


def get_consents_for_user(user_id: uuid.UUID) -> set[Purpose]:
    client = get_uc()
    cols = [ResourceID(name="email")]
    # cols = [ResourceID(name="email"), ResourceID(name="phone_number")]
    consents = client.GetConsentedPurposesForUser(user_id, columns=cols)
    if len(consents) != 1:
        raise ValueError(f"Expected 1 column, got {len(consents)}")
    purposes = {Purpose(cp.name) for cp in consents[0].consented_purposes}
    _logger.info(f"User {user_id} consents: {purposes}")
    return purposes
