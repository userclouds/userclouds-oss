from django.urls import path

from contacts.userclouds.codegensdk import Purpose

from . import views

urlpatterns = [
    path("", views.HomePageView.as_view(), name="home"),
    path("detail/<uuid:pk>/", views.ContactDetailView.as_view(), name="detail"),
    path(
        "detail/<uuid:pk>/resolve-marketing/",
        views.ContactDetailView.as_view(resolve_purpose=Purpose.MARKETING),
        name="detail-resolve-marketing",
    ),
    path(
        "detail/<uuid:pk>/resolve-fraud/",
        views.ContactDetailView.as_view(resolve_purpose=Purpose.FRAUD),
        name="detail-resolve-fraud",
    ),
    path("contacts/create", views.ContactCreateView.as_view(), name="create"),
    path("contacts/update/<uuid:pk>", views.ContactUpdateView.as_view(), name="update"),
    path("contacts/delete/<uuid:pk>", views.ContactDeleteView.as_view(), name="delete"),
]
