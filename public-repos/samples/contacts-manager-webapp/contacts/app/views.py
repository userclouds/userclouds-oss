import logging
import uuid
from typing import Any

from django.conf import settings
from django.contrib import messages
from django.db.models.base import Model as Model
from django.db.models.query import QuerySet
from django.http import Http404
from django.shortcuts import get_object_or_404, redirect
from django.views.generic import DetailView, ListView
from django.views.generic.edit import CreateView, DeleteView, UpdateView

from contacts.userclouds.codegensdk import Purpose

from .models import (
    Contact,
    get_consents_for_user,
    get_raw_contact,
    get_raw_contacts_qc,
    resolve_email_for_purpose,
)

_logger = logging.getLogger(__name__)


def get_uc():
    return settings.UC_CLIENT


class HomePageView(ListView):
    template_name = "index.html"
    model = Contact
    context_object_name = "contacts"

    def get_context_data(self, **kwargs: Any) -> dict[str, Any]:
        data = super().get_context_data(**kwargs)
        mode = self.request.GET.get("mode", "tokenized")
        if mode == "deleted":
            data["contacts_title"] = "Deleted Contacts"
        elif mode == "raw":
            data["contacts_title"] = "Raw Contacts [from UC]"
        else:
            data["contacts_title"] = "Contacts [from app DB]"
        return data

    def get_queryset(self):
        mode = self.request.GET.get("mode", "tokenized")
        _logger.info(f"request mode: {mode}")
        if mode == "raw":
            return get_raw_contacts_qc()
        return super().get_queryset().filter(is_deleted=mode == "deleted")


class ContactDetailView(DetailView):
    template_name = "detail.html"
    model = Contact
    context_object_name = "contact"
    resolve_purpose = None

    def get_object(self, queryset: QuerySet[Any] | None = None) -> Contact:
        if self.resolve_purpose is not None:
            contact_id = self.kwargs[self.pk_url_kwarg]
            cn = get_contact_with_resolved_email_or_404(
                contact_id, self.resolve_purpose
            )
            if not cn.is_email_resolved:
                messages.error(
                    self.request,
                    f"Token resolution for {self.resolve_purpose.value} failed",
                )

            return cn
        return super().get_object(queryset)


class ContactCreateView(CreateView):
    model = Contact
    template_name = "create.html"
    fields = ["name", "nickname", "email", "phone", "image"]

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        _add_available_consents(context)
        return context

    def form_valid(self, form):
        instance: Contact = form.save(commit=False)
        purposes = _get_selected_consents(form.data)
        instance.save_with_userclouds(purposes=purposes)
        messages.success(self.request, "Your contact has been successfully created!")
        return redirect("home")


class ContactUpdateView(UpdateView):
    model = Contact
    template_name = "update.html"
    fields = ["name", "nickname", "email", "phone", "image"]

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        consents = get_consents_for_user(self.object.contact_id)
        _add_available_consents(context, existing_consents=consents)
        return context

    def form_valid(self, form):
        contact: Contact = form.save(commit=False)
        purposes = _get_selected_consents(form.data)
        contact.save_with_userclouds(purposes=purposes)
        messages.success(self.request, "Your contact has been successfully updated!")
        return redirect("detail", contact.contact_id)

    def get_object(self, queryset=None):
        pk = self.kwargs.get(self.pk_url_kwarg)
        return get_contact_or_404(pk)


class ContactDeleteView(DeleteView):
    model = Contact
    template_name = "delete.html"
    success_url = "/"

    def form_valid(self, form):
        messages.success(self.request, "Your contact has been successfully deleted!")
        return super().form_valid(form)

    # def get_object(self, queryset=None):
    #     pk = self.kwargs.get(self.pk_url_kwarg)
    #     return get_contact_or_404(pk)


def get_contact_or_404(pk):
    cn = get_raw_contact(pk)
    if not cn:
        raise Http404("Contact not found")
    return cn


def get_contact_with_resolved_email_or_404(
    contact_id: uuid.UUID, purpose: Purpose
) -> Contact:
    cn = get_object_or_404(Contact, pk=contact_id)
    resolve_email_for_purpose(cn, purpose)
    return cn


def _add_available_consents(
    context: dict, existing_consents: set[Purpose] | None = None
) -> None:
    existing_consents = existing_consents or set()
    context["available_consents"] = [
        (p.name, p.value.capitalize, p in existing_consents) for p in Purpose
    ]


def _get_selected_consents(form_data: dict) -> set[Purpose]:
    consents: set[Purpose] = set()
    for key in form_data:
        if not key.startswith("consent."):
            continue
        *_, purpose_name = key.partition(".")
        if not purpose_name:
            continue
        try:
            purpose = Purpose[purpose_name]
        except KeyError:
            _logger.warning(f"Unknown purpose: '{purpose_name}' (field='{key}')")
            # we should probably fail the request, but just log for now
            continue
        consents.add(purpose)
    return consents
