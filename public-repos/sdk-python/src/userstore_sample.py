from __future__ import annotations

import asyncio
import json
import os
import random
import uuid
import warnings

from usercloudssdk.asyncclient import AsyncClient
from usercloudssdk.client import Client
from usercloudssdk.constants import (
    PAGINATION_CURSOR_BEGIN,
    PAGINATION_CURSOR_END,
    PAGINATION_SORT_ASCENDING,
    PAGINATION_SORT_DESCENDING,
    ColumnIndexType,
    DataLifeCycleState,
    DataType,
    PolicyType,
    Region,
    TransformType,
)
from usercloudssdk.data_types import ColumnDataTypeBoolean, ColumnDataTypeString
from usercloudssdk.models import (
    Accessor,
    AccessPolicy,
    AccessPolicyComponent,
    AccessPolicyTemplate,
    Column,
    ColumnConstraints,
    ColumnDataType,
    ColumnField,
    ColumnInputConfig,
    ColumnOutputConfig,
    ColumnRetentionDuration,
    CompositeAttributes,
    CompositeField,
    Mutator,
    Purpose,
    ResourceID,
    RetentionDuration,
    Transformer,
    UpdateColumnRetentionDurationRequest,
    UpdateColumnRetentionDurationsRequest,
    UserSelectorConfig,
)
from usercloudssdk.policies import (
    AccessPolicyOpen,
    NormalizerOpen,
    TransformerPassThrough,
)
from usercloudssdk.uchttpclient import (
    create_default_uc_http_async_client,
    create_default_uc_http_client,
    create_no_ssl_http_async_client,
    create_no_ssl_http_client,
)

client_id = "<REPLACE ME>"
client_secret = "<REPLACE ME>"
url = "<REPLACE ME>"

# This sample shows you how to create new columns in the user store and create access
# policies governing access to the data inside those columns. It also shows you how to
# create, delete and execute accessors and mutators. To learn more about these
# concepts, see docs.userclouds.com.


# Use unique names for all userstore entities that we create so that they do not conflict with
# any existing userstore entities in the tenant being used, and so that they can be cleaned
# up after execution of the sample app.
class Names:
    accessorMarketing: str
    accessorPagination: str
    accessorPhoneToken: str
    accessorSecurity: str
    accessorSupport: str
    accessPolicy: str
    accessPolicyTemplate: str
    columnBillingAddress: str
    columnEmail: str
    columnPhone: str
    columnShippingAddresses: str
    columnTemp: str
    dataTypeAddress: str
    mutatorEmail: str
    mutatorPhoneAddress: str
    prefix: str
    purposeMarketing: str
    purposeSecurity: str
    purposeSupport: str
    purposeTemp: str
    transformerPhoneLogging: str
    transformerPhoneSecurity: str
    transformerPhoneSupport: str
    userEmail: str

    def __init__(
        self,
    ) -> None:
        prefix = f"PySample{random.randint(1, 100000)}"
        self.accessorMarketing = f"{prefix}MarketingAccessor"
        self.accessorPagination = f"{prefix}PaginationAccessor"
        self.accessorPhoneToken = f"{prefix}PhoneTokenAccessor"
        self.accessorSecurity = f"{prefix}SecurityAccessor"
        self.accessorSupport = f"{prefix}SupportAccessor"
        self.accessPolicy = f"{prefix}AccessPolicy"
        self.accessPolicyTemplate = f"{prefix}AccessPolicyTemplate"
        self.columnBillingAddress = f"{prefix}BillingAddress"
        self.columnEmail = f"{prefix}Email"
        self.columnPhone = f"{prefix}Phone"
        self.columnShippingAddresses = f"{prefix}ShippingAddresses"
        self.columnTemp = f"{prefix}Temp"
        self.dataTypeAddress = f"{prefix}Address"
        self.mutatorEmail = f"{prefix}EmailMutator"
        self.mutatorPhoneAddress = f"{prefix}PhoneAddressMutator"
        self.prefix = prefix
        self.purposeMarketing = f"{prefix}Marketing"
        self.purposeSecurity = f"{prefix}Security"
        self.purposeSupport = f"{prefix}Support"
        self.purposeTemp = f"{prefix}Temp"
        self.transformerPhoneLogging = f"{prefix}PhoneLoggingTransformer"
        self.transformerPhoneSecurity = f"{prefix}PhoneSecurityTransformer"
        self.transformerPhoneSupport = f"{prefix}PhoneSupportTransformer"
        self.userEmail = f"{prefix}User@example.org"


def setup(
    client: Client, names: Names
) -> tuple[tuple[Accessor, ...], tuple[Mutator, ...]]:
    # illustrate CRUD for columns
    col = client.CreateColumn(
        Column(
            id=None,
            name=names.columnTemp,
            data_type=ColumnDataTypeBoolean,
            type=DataType.BOOLEAN,
            is_array=False,
            default_value="",
            index_type=ColumnIndexType.INDEXED,
        ),
        if_not_exists=True,
    )
    client.ListColumns()
    col = client.GetColumn(col.id)
    col.name = f"Different{col.name}"
    client.UpdateColumn(col)
    client.DeleteColumn(col.id)

    # create a custom us address data type
    client.CreateColumnDataType(
        ColumnDataType(
            id=None,
            name=names.dataTypeAddress,
            description="a US style address",
            composite_attributes=CompositeAttributes(
                include_id=False,
                fields=[
                    CompositeField(
                        data_type=ColumnDataTypeString, name="Street_Address"
                    ),
                    CompositeField(data_type=ColumnDataTypeString, name="City"),
                    CompositeField(data_type=ColumnDataTypeString, name="State"),
                    CompositeField(data_type=ColumnDataTypeString, name="Zip"),
                ],
            ),
        ),
        if_not_exists=True,
    )

    # create phone number, shipping addresses, billing address, and email columns
    phone = client.CreateColumn(
        Column(
            id=None,
            name=names.columnPhone,
            data_type=ColumnDataTypeString,
            type=DataType.STRING,
            is_array=False,
            default_value="",
            index_type=ColumnIndexType.INDEXED,
        ),
        if_not_exists=True,
    )

    client.CreateColumn(
        Column(
            id=None,
            name=names.columnShippingAddresses,
            data_type=ResourceID(name=names.dataTypeAddress),
            type=DataType.COMPOSITE,
            is_array=True,
            default_value="",
            index_type=ColumnIndexType.INDEXED,
            constraints=ColumnConstraints(
                immutable_required=False,
                partial_updates=True,
                unique_id_required=False,
                unique_required=True,
                fields=[
                    ColumnField(type=DataType.STRING, name="Street_Address"),
                    ColumnField(type=DataType.STRING, name="City"),
                    ColumnField(type=DataType.STRING, name="State"),
                    ColumnField(type=DataType.STRING, name="Zip"),
                ],
            ),
        ),
        if_not_exists=True,
    )

    client.CreateColumn(
        Column(
            id=None,
            name=names.columnEmail,
            data_type=ColumnDataTypeString,
            type=DataType.STRING,
            is_array=False,
            default_value="",
            index_type=ColumnIndexType.INDEXED,
        ),
        if_not_exists=True,
    )

    client.CreateColumn(
        Column(
            id=None,
            name=names.columnBillingAddress,
            data_type=ResourceID(name=names.dataTypeAddress),
            type=DataType.COMPOSITE,
            is_array=False,
            default_value="",
            index_type=ColumnIndexType.INDEXED,
            constraints=ColumnConstraints(
                immutable_required=False,
                partial_updates=False,
                unique_id_required=False,
                unique_required=False,
                fields=[
                    ColumnField(type=DataType.STRING, name="Street_Address"),
                    ColumnField(type=DataType.STRING, name="City"),
                    ColumnField(type=DataType.STRING, name="State"),
                    ColumnField(type=DataType.STRING, name="Zip"),
                ],
            ),
        ),
        if_not_exists=True,
    )

    # illustrate CRUD for purposes
    purpose = client.CreatePurpose(
        Purpose(
            id=None,
            name=names.purposeTemp,
            description="a temporary purpose",
        ),
        if_not_exists=True,
    )
    client.ListPurposes()
    purpose = client.GetPurpose(purpose.id)
    purpose.description = "a different description"
    client.UpdatePurpose(purpose)
    client.DeletePurpose(purpose.id)

    # create purposes for security, support and marketing
    security = client.CreatePurpose(
        Purpose(
            id=None,
            name=names.purposeSecurity,
            description="Allows access to the data in the columns for security purposes",
        ),
        if_not_exists=True,
    )

    support = client.CreatePurpose(
        Purpose(
            id=None,
            name=names.purposeSupport,
            description="Allows access to the data in the columns for support purposes",
        ),
        if_not_exists=True,
    )

    client.CreatePurpose(
        Purpose(
            id=None,
            name=names.purposeMarketing,
            description="Allows access to the data in the columns for marketing purposes",
        ),
        if_not_exists=True,
    )

    # configure retention durations for soft-deleted data

    # retrieve and delete any pre-existing phone_number soft-deleted retention durations
    phone_rds = client.GetSoftDeletedRetentionDurationsOnColumn(
        phone.id,
    )
    for rd in phone_rds.retention_durations:
        if rd.id != uuid.UUID(int=0):
            client.DeleteSoftDeletedRetentionDurationOnColumn(phone.id, rd.id)

    # retain soft-deleted phone_number values with a support purpose for 3 months
    column_rd_update = UpdateColumnRetentionDurationsRequest(
        [
            ColumnRetentionDuration(
                duration_type=DataLifeCycleState.SOFT_DELETED,
                duration=RetentionDuration(unit="month", duration=3),
                column_id=phone.id,
                purpose_id=support.id,
            ),
        ],
    )
    client.UpdateSoftDeletedRetentionDurationsOnColumn(
        columnID=phone.id,
        req=column_rd_update,
    )

    # retrieve and delete pre-existing default soft-deleted retention duration on tenant
    try:
        default_duration = client.GetDefaultSoftDeletedRetentionDurationOnTenant()
        if default_duration.retention_duration.id != uuid.UUID(int=0):
            client.DeleteSoftDeletedRetentionDurationOnTenant(
                default_duration.retention_duration.id
            )
    except Exception:
        pass

    # retain all soft-deleted values for any column or purpose for 1 week by default
    tenant_rd_update = UpdateColumnRetentionDurationRequest(
        ColumnRetentionDuration(
            duration_type=DataLifeCycleState.SOFT_DELETED,
            duration=RetentionDuration(unit="week", duration=1),
        ),
    )

    client.CreateSoftDeletedRetentionDurationOnTenant(
        tenant_rd_update,
    )

    # retrieve and delete any pre-existing security purpose soft-deleted retention duration
    try:
        purpose_rd = client.GetDefaultSoftDeletedRetentionDurationOnPurpose(
            security.id,
        )
        if purpose_rd.retention_duration.id != uuid.UUID(int=0):
            client.DeleteSoftDeletedRetentionDurationOnPurpose(
                security.id, purpose_rd.retention_duration.id
            )
    except Exception:
        pass

    # retain soft-deleted values for any column with a security purpose for 1 year by default
    purpose_rd_update = UpdateColumnRetentionDurationRequest(
        ColumnRetentionDuration(
            duration_type=DataLifeCycleState.SOFT_DELETED,
            duration=RetentionDuration(unit="year", duration=1),
            purpose_id=security.id,
        ),
    )

    client.CreateSoftDeletedRetentionDurationOnPurpose(
        security.id,
        purpose_rd_update,
    )

    # retrieve phone_number soft-deleted retention durations after configuration
    phone_rds = client.GetSoftDeletedRetentionDurationsOnColumn(
        phone.id,
    )
    print(f"phone_number retention durations post-configuration: {phone_rds}\n")

    # Create an access policy that allows access to the data in the columns for security
    # and support purposes
    apt = AccessPolicyTemplate(
        name=names.accessPolicyTemplate,
        function="""function policy(context, params) {
            return params.teams.includes(context.client.team);
        }"""
        + f" // {names.accessPolicyTemplate}",
    )
    apt = client.CreateAccessPolicyTemplate(apt, if_not_exists=True)

    ap = AccessPolicy(
        name=names.accessPolicy,
        policy_type=PolicyType.COMPOSITE_AND,
        components=[
            AccessPolicyComponent(
                template=ResourceID(name=names.accessPolicyTemplate),
                template_parameters='{"teams": ["security_team", "support_team"]}',
            )
        ],
    )
    ap = client.CreateAccessPolicy(ap, if_not_exists=True)

    # Create a transformer that transforms the data in the columns for security and
    # support teams
    phone_transformer_function = r"""
function transform(data, params) {
    if (params.team == "security_team") {
        return data;
    } else if (params.team == "support_team") {
        phone = /^(\d{3})-(\d{3})-(\d{4})$/.exec(data);
        if (phone) {
            return "XXX-XXX-"+phone[3];
        } else {
            return "<invalid phone number>";
        }
    }
    return "";
}"""

    support_phone_transformer = Transformer(
        id=None,
        name=names.transformerPhoneSupport,
        input_data_type=ColumnDataTypeString,
        input_type=DataType.STRING,
        output_data_type=ColumnDataTypeString,
        output_type=DataType.STRING,
        reuse_existing_token=False,
        transform_type=TransformType.TRANSFORM,
        function=f"{phone_transformer_function} // {names.transformerPhoneSupport}",
        parameters='{"team": "support_team"}',
    )
    support_phone_transformer = client.CreateTransformer(
        support_phone_transformer, if_not_exists=True
    )

    security_phone_transformer = Transformer(
        id=None,
        name=names.transformerPhoneSecurity,
        input_data_type=ColumnDataTypeString,
        input_type=DataType.STRING,
        output_data_type=ColumnDataTypeString,
        output_type=DataType.STRING,
        reuse_existing_token=False,
        transform_type=TransformType.TRANSFORM,
        function=f"{phone_transformer_function} // {names.transformerPhoneSecurity}",
        parameters='{"team": "security_team"}',
    )
    security_phone_transformer = client.CreateTransformer(
        security_phone_transformer, if_not_exists=True
    )

    phone_tokenizing_transformer_function = r"""
function id(len) {
        var s = "0123456789";
        return Array(len).join().split(',').map(function() {
            return s.charAt(Math.floor(Math.random() * s.length));
        }).join('');
    }
    function validate(str) {
        return (str.length === 10);
    }
    function transform(data, params) {
      // Strip non numeric characters if present
      orig_data = data;
      data = data.replace(/\D/g, '');
      if (data.length === 11 ) {
        data = data.substr(1, 11);
      }
      if (!validate(data)) {
            throw new Error('Invalid US Phone Number Provided');
      }
      return '1' + id(10);
}"""

    logging_phone_transformer = Transformer(
        id=None,
        name=names.transformerPhoneLogging,
        input_data_type=ColumnDataTypeString,
        input_type=DataType.STRING,
        output_data_type=ColumnDataTypeString,
        output_type=DataType.STRING,
        reuse_existing_token=True,  # Set this is to false to get a unique token every time this transformer is called vs getting same token on every call
        transform_type=TransformType.TOKENIZE_BY_VALUE,
        function=f"{phone_tokenizing_transformer_function} // {names.transformerPhoneLogging}",
        parameters='{"team": "security_team"}',
    )

    logging_phone_transformer = client.CreateTransformer(
        logging_phone_transformer, if_not_exists=True
    )
    # Accessors are configurable APIs that allow a client to retrieve data from the user
    # store. Accessors are intended to be use-case specific. They enforce data usage
    # policies and minimize outbound data from the store for their given use case.

    # Selectors are used to filter the set of users that are returned by an accessor.
    # They are essentially SQL WHERE clauses and are configured per-accessor /
    # per-mutator referencing column IDs of the userstore.

    # Here we create accessors for two example teams: (1) security team and (2) support
    # team

    acc_support = Accessor(
        id=None,
        name=names.accessorSupport,
        description="Accessor for support team",
        columns=[
            ColumnOutputConfig(
                column=ResourceID(name=names.columnPhone),
                transformer=ResourceID(id=support_phone_transformer.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnShippingAddresses),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="created"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="id"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnBillingAddress),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
        ],
        access_policy=ResourceID(id=ap.id),
        selector_config=UserSelectorConfig("{id} = ?"),
        purposes=[ResourceID(name=names.purposeSupport)],
        data_life_cycle_state=DataLifeCycleState.LIVE,
    )
    acc_support = client.CreateAccessor(acc_support, if_not_exists=True)

    # illustrate updating accessor, getting, and listing accessors
    acc_support.description = "New description"
    acc_support = client.UpdateAccessor(acc_support)
    acc_support = client.GetAccessor(acc_support.id)
    client.ListAccessors()

    acc_security = Accessor(
        id=None,
        name=names.accessorSecurity,
        description="Accessor for security team",
        columns=[
            ColumnOutputConfig(
                column=ResourceID(name=names.columnPhone),
                transformer=ResourceID(id=security_phone_transformer.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnShippingAddresses),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="created"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="id"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnBillingAddress),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
        ],
        access_policy=ResourceID(id=ap.id),
        selector_config=UserSelectorConfig(
            f"{{{names.columnShippingAddresses}}}->>'street_address' LIKE (?) AND {{{names.columnPhone}}} = (?)"
        ),
        purposes=[ResourceID(name=names.purposeSecurity)],
        data_life_cycle_state=DataLifeCycleState.LIVE,
    )
    acc_security = client.CreateAccessor(acc_security, if_not_exists=True)

    acc_marketing = Accessor(
        id=None,
        name=names.accessorMarketing,
        description="Accessor for marketing team",
        columns=[
            ColumnOutputConfig(
                column=ResourceID(name=names.columnPhone),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnShippingAddresses),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="created"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name="id"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnBillingAddress),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
        ],
        access_policy=ResourceID(id=AccessPolicyOpen.id),
        selector_config=UserSelectorConfig("{id} = ?"),
        purposes=[ResourceID(name=names.purposeMarketing)],
        data_life_cycle_state=DataLifeCycleState.LIVE,
    )
    acc_marketing = client.CreateAccessor(acc_marketing, if_not_exists=True)

    acc_phone_token = Accessor(
        id=None,
        name=names.accessorPhoneToken,
        description="Accessor for getting phone number token for security team",
        columns=[
            ColumnOutputConfig(
                column=ResourceID(name=names.columnPhone),
                transformer=ResourceID(id=logging_phone_transformer.id),
                token_access_policy=ResourceID(id=AccessPolicyOpen.id),
            ),
        ],
        access_policy=ResourceID(id=AccessPolicyOpen.id),
        selector_config=UserSelectorConfig("{id} = ?"),
        purposes=[ResourceID(name=names.purposeSecurity)],
        data_life_cycle_state=DataLifeCycleState.LIVE,
    )
    acc_phone_token = client.CreateAccessor(acc_phone_token, if_not_exists=True)

    acc_pagination = Accessor(
        id=None,
        name=names.accessorPagination,
        description="Accessor for illustrating pagination",
        columns=[
            ColumnOutputConfig(
                column=ResourceID(name="id"),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
            ColumnOutputConfig(
                column=ResourceID(name=names.columnEmail),
                transformer=ResourceID(id=TransformerPassThrough.id),
            ),
        ],
        access_policy=ResourceID(id=AccessPolicyOpen.id),
        selector_config=UserSelectorConfig("{id} = ANY(?)"),
        purposes=[ResourceID(name="operational")],
        data_life_cycle_state=DataLifeCycleState.LIVE,
    )
    acc_pagination = client.CreateAccessor(acc_pagination, if_not_exists=True)

    # Mutators are configurable APIs that allow a client to write data to the User
    # Store. Mutators (setters) can be thought of as the complement to accessors
    # (getters). Here we create mutator to update the user's phone number, shipping
    # addresses, and billing address, and another mutator for updating email address.
    mut_phone_address = Mutator(
        id=None,
        name=names.mutatorPhoneAddress,
        description="Mutator for updating phone number and addresses",
        columns=[
            ColumnInputConfig(
                column=ResourceID(name=names.columnPhone),
                normalizer=ResourceID(id=NormalizerOpen.id),
            ),
            ColumnInputConfig(
                column=ResourceID(name=names.columnShippingAddresses),
                normalizer=ResourceID(id=NormalizerOpen.id),
            ),
            ColumnInputConfig(
                column=ResourceID(name=names.columnBillingAddress),
                normalizer=ResourceID(id=NormalizerOpen.id),
            ),
        ],
        access_policy=ResourceID(id=AccessPolicyOpen.id),
        selector_config=UserSelectorConfig("{id} = ?"),
    )
    mut_phone_address = client.CreateMutator(mut_phone_address, if_not_exists=True)
    # illustrate updating, getting, and listing mutators
    mut_phone_address.description = "A new description"
    mut_phone_address = client.UpdateMutator(mut_phone_address)
    mut_phone_address = client.GetMutator(mut_phone_address.id)
    client.ListMutators()

    mut_email = Mutator(
        id=None,
        name=names.mutatorEmail,
        description="Mutator for updating email",
        columns=[
            ColumnInputConfig(
                column=ResourceID(name=names.columnEmail),
                normalizer=ResourceID(id=NormalizerOpen.id),
            ),
        ],
        access_policy=ResourceID(id=AccessPolicyOpen.id),
        selector_config=UserSelectorConfig("{id} = ?"),
    )
    mut_email = client.CreateMutator(mut_email, if_not_exists=True)
    return (
        acc_support,
        acc_security,
        acc_marketing,
        acc_phone_token,
        acc_pagination,
    ), (
        mut_phone_address,
        mut_email,
    )


def userstore_example(
    *,
    client: Client,
    user_region: Region | str,
    names: Names,
    accessors: tuple[Accessor, ...],
    mutators: tuple[Mutator, ...],
) -> None:
    assert len(accessors) == 5
    assert len(mutators) == 2
    mut_phone_address, mut_email = mutators
    acc_support, acc_security, acc_marketing, acc_phone_token, acc_pagination = (
        accessors
    )
    # Just make sure we unpacked stuff correctly
    assert mut_phone_address.name == names.mutatorPhoneAddress
    assert mut_email.name == names.mutatorEmail
    assert acc_support.name == names.accessorSupport
    assert acc_security.name == names.accessorSecurity
    assert acc_marketing.name == names.accessorMarketing
    assert acc_phone_token.name == names.accessorPhoneToken
    assert acc_pagination.name == names.accessorPagination

    # create a user
    uid = client.CreateUser()

    # retrieve the user the "old way" (not using accessors) just for illustration
    user = client.GetUser(uid)

    # update the user using the "old way" (not using mutators) just for illustration
    profile = user.profile
    profile[names.columnEmail] = names.userEmail
    profile[names.columnPhone] = "123-456-7890"
    client.UpdateUser(uid, profile)

    # retrieve the user the "old way" (not using accessors) just for illustration
    user = client.GetUser(uid)
    print(f"old way: user's details are {user.profile}\n")

    client.DeleteUser(uid)

    # create and initialize a user with email address
    uid = client.CreateUserWithMutator(
        mutator_id=mut_email.id,
        context={},
        row_data={
            names.columnEmail: {
                "value": names.userEmail,
                "purpose_additions": [
                    {"Name": "operational"},
                ],
            },
        },
        region=user_region,
    )

    # set the user's info using the mutator
    client.ExecuteMutator(
        mutator_id=mut_phone_address.id,
        context={},
        selector_values=[uid],
        row_data={
            names.columnPhone: {
                "value": "123-456-7890",
                "purpose_additions": [
                    {"Name": names.purposeSecurity},
                    {"Name": names.purposeSupport},
                    {"Name": "operational"},
                ],
            },
            names.columnShippingAddresses: {
                "value_additions": '[{"state":"IL", "street_address":"742 Evergreen \
                        Terrace", "city":"Springfield", "zip":"62704"}, {"state":"CA", "street_address":"123 \
                        Main St", "city":"Pleasantville", "zip":"94566"}]',
                "purpose_additions": [
                    {"Name": names.purposeSecurity},
                    {"Name": names.purposeSupport},
                    {"Name": "operational"},
                ],
            },
            names.columnBillingAddress: {
                "value": '{"street_address":"742 Evergreen Terrace","city":"Springfield","state":"IL","zip":"62704"}',
                "purpose_additions": [
                    {"Name": names.purposeSecurity},
                    {"Name": names.purposeSupport},
                    {"Name": "operational"},
                ],
            },
        },
    )

    # now retrieve the user's info using the accessor with the right context
    resolved = client.ExecuteAccessor(
        accessor_id=acc_support.id,
        context={"team": "support_team"},
        selector_values=[uid],
    )
    # expect ['["XXX-XXX-7890","<home address hidden>"]']
    print(f"support context: user's details are {resolved}\n")

    resolved = client.ExecuteAccessor(
        accessor_id=acc_security.id,
        context={"team": "security_team"},
        selector_values=["%Evergreen%", "123-456-7890"],
    )
    # expect full details
    print(f"security context: user's details are {resolved}\n")

    resolved = client.ExecuteAccessor(
        accessor_id=acc_marketing.id,
        context={"team": "marketing_team"},
        selector_values=[uid],
    )
    # expect [] (due to team mismatch in access policy)
    print(f"marketing context: user's details are {resolved}\n")

    resolved = client.ExecuteAccessor(
        accessor_id=acc_phone_token.id,
        context={"team": "security_team"},
        selector_values=[uid],
    )
    # expect to get back a token
    token = json.loads(resolved["data"][0])[names.columnPhone]
    print(f"user's phone token (first call) {token}\n")

    resolved = client.ExecuteAccessor(
        accessor_id=acc_phone_token.id,
        context={"team": "security_team"},
        selector_values=[uid],
    )

    # expect to get back the same token so it can be used in logs as a unique identifier if desired
    token = json.loads(resolved["data"][0])[names.columnPhone]
    print(f"user's phone token (repeat call) {token}\n")

    # resolving the token to the original phone number
    value = client.ResolveTokens(
        [token], {"team": "security_team"}, [ResourceID(name=names.purposeSecurity)]
    )
    print(f"user's phone token resolved  {value}\n")

    # delete token so we can delete the transformer during cleanup

    if not client.DeleteToken(token):
        warnings.warn(f"failed to delete token - {token}")

    if not client.DeleteUser(uid):
        warnings.warn(f"failed to delete user - {uid}")

    ## demonstrate accessor pagination

    # create test users

    test_user_ids: list[uuid.UUID] = []
    for i in range(50):
        user_email = f"{names.prefix}User_{i:0>2}@foo.org"
        test_user_id = client.CreateUserWithMutator(
            mutator_id=mut_email.id,
            context={},
            row_data={
                names.columnEmail: {
                    "value": user_email,
                    "purpose_additions": [
                        {"Name": "operational"},
                    ],
                },
            },
            region=user_region,
        )
        test_user_ids.append(test_user_id)

    # ascending forward pagination

    paginationExecute(
        client=client,
        names=names,
        accessor_id=acc_pagination.id,
        description="forward ascending",
        sort_order=PAGINATION_SORT_ASCENDING,
        user_ids=test_user_ids,
        starting_after=PAGINATION_CURSOR_BEGIN,
    )

    # descending forward pagination

    paginationExecute(
        client=client,
        names=names,
        accessor_id=acc_pagination.id,
        description="forward descending",
        sort_order=PAGINATION_SORT_DESCENDING,
        user_ids=test_user_ids,
        starting_after=PAGINATION_CURSOR_BEGIN,
    )

    # ascending backward pagination

    paginationExecute(
        client=client,
        names=names,
        accessor_id=acc_pagination.id,
        description="backward ascending",
        sort_order=PAGINATION_SORT_ASCENDING,
        user_ids=test_user_ids,
        ending_before=PAGINATION_CURSOR_END,
    )

    # ascending backward pagination

    paginationExecute(
        client=client,
        names=names,
        accessor_id=acc_pagination.id,
        description="backward descending",
        sort_order=PAGINATION_SORT_DESCENDING,
        user_ids=test_user_ids,
        ending_before=PAGINATION_CURSOR_END,
    )

    # delete test users

    for test_user_id in test_user_ids:
        client.DeleteUser(test_user_id)


def paginationExecute(
    client: Client,
    names: Names,
    accessor_id: uuid.UUID,
    description: str,
    sort_order: str,
    user_ids: list[uuid.UUID],
    starting_after: str | None = None,
    ending_before: str | None = None,
):
    cursor: str | None = starting_after
    if ending_before is not None:
        cursor = ending_before

    print(f"\nexecuting {description} pagination accessor with cursor '{cursor}'\n")

    resp = client.ExecuteAccessor(
        accessor_id=accessor_id,
        context={},
        selector_values=[user_ids],
        limit=25,
        starting_after=starting_after,
        ending_before=ending_before,
        sort_key=f"{names.columnEmail},id",
        sort_order=sort_order,
    )

    # verify that the pagination response fields are all populated
    assert resp["next"] is not None
    assert resp["prev"] is not None
    assert resp["has_next"] is not None
    assert resp["has_prev"] is not None

    rows = resp["data"]

    resultDesc = f"- returned {len(rows)} rows"
    if resp["has_next"]:
        assert resp["next"] != PAGINATION_CURSOR_END
        nextCursor = resp["next"]
        resultDesc = f"{resultDesc}, Next = '{nextCursor}'"
    if resp["has_prev"]:
        assert resp["prev"] != PAGINATION_CURSOR_BEGIN
        prevCursor = resp["prev"]
        resultDesc = f"{resultDesc}, Prev = '{prevCursor}'"

    print(f"{resultDesc}\n")

    for row in rows:
        print(f"-- {row}")

    if starting_after is not None:
        if resp["has_next"]:
            paginationExecute(
                client=client,
                names=names,
                accessor_id=accessor_id,
                description=description,
                sort_order=sort_order,
                user_ids=user_ids,
                starting_after=resp["next"],
            )
    elif ending_before is not None:
        if resp["has_prev"]:
            paginationExecute(
                client=client,
                names=names,
                accessor_id=accessor_id,
                description=description,
                sort_order=sort_order,
                user_ids=user_ids,
                ending_before=resp["prev"],
            )


def cleanup(
    client: Client,
    names: Names,
):
    # delete the created retention durations
    clean_retention_durations(client=client, names=names)

    # delete the created userstore entities
    for m in client.ListMutators():
        if m.name.startswith(names.prefix):
            if not client.DeleteMutator(m.id):
                warnings.warn(f"Failed to delete mutator - {m.id=} - {m.name=}")

    for a in client.ListAccessors():
        if a.name.startswith(names.prefix):
            if not client.DeleteAccessor(a.id):
                warnings.warn(f"Failed to delete accessor - {a.id=} - {a.name=}")

    for t in client.ListTransformers():
        if t.name.startswith(names.prefix):
            if not client.DeleteTransformer(t.id):
                warnings.warn(f"Failed to delete transformer - {t.id=} - {t.name=}")

    for ap in client.ListAccessPolicies():
        if ap.name.startswith(names.prefix):
            if not client.DeleteAccessPolicy(ap.id, 0):
                warnings.warn(f"Failed to delete access policy - {ap.id=} - {ap.name=}")

    for apt in client.ListAccessPolicyTemplates():
        if apt.name.startswith(names.prefix):
            if not client.DeleteAccessPolicyTemplate(apt.id, 0):
                warnings.warn(
                    f"Failed to delete access policy template - {apt.id=} - {apt.name=}"
                )

    for c in client.ListColumns():
        if c.name.startswith(names.prefix):
            if not client.DeleteColumn(c.id):
                warnings.warn(f"Failed to delete column - {c.id=} - {c.name=}")

    for p in client.ListPurposes():
        if p.name.startswith(names.prefix):
            if not client.DeletePurpose(p.id):
                warnings.warn(f"Failed to delete purpose - {p.id=} - {p.name=}")

    for cdt in client.ListColumnDataTypes():
        if cdt.name.startswith(names.prefix):
            if not client.DeleteColumnDataType(cdt.id):
                warnings.warn(f"Failed to delete data type - {cdt.id=} - {cdt.name=}")


def clean_retention_durations(client: Client, names: Names) -> None:
    column_id = next(
        (col.id for col in client.ListColumns() if col.name == names.columnPhone),
        None,
    )
    purpose_id = next(
        (
            purpose.id
            for purpose in client.ListPurposes()
            if purpose.name == names.purposeSecurity
        ),
        None,
    )

    if column_id:
        created_rds = client.GetSoftDeletedRetentionDurationsOnColumn(
            column_id
        ).retention_durations
    else:
        created_rds: list[RetentionDuration] = []
        warnings.warn(f"Failed to find column - {names.columnPhone}")
    created_rds.append(
        client.GetDefaultSoftDeletedRetentionDurationOnTenant().retention_duration
    )
    if purpose_id:
        created_rds.append(
            client.GetDefaultSoftDeletedRetentionDurationOnPurpose(
                purpose_id
            ).retention_duration
        )
    else:
        warnings.warn(f"Failed to find purpose - {names.purposeSecurity}")

    for rd in created_rds:
        if rd.id == uuid.UUID(int=0):
            continue
        if rd.column_id == uuid.UUID(int=0):
            if rd.purpose_id == uuid.UUID(int=0):
                if not client.DeleteSoftDeletedRetentionDurationOnTenant(rd.id):
                    warnings.warn(
                        f"Failed to delete default retention duration on tenant - {rd.id=}"
                    )
            else:
                if not client.DeleteSoftDeletedRetentionDurationOnPurpose(
                    rd.purpose_id, rd.id
                ):
                    warnings.warn(
                        f"Failed to delete default retention duration on purpose - {rd.id=} - {rd.purpose_id=}"
                    )
        else:
            if not client.DeleteSoftDeletedRetentionDurationOnColumn(
                rd.column_id, rd.id
            ):
                warnings.warn(
                    f"Failed to delete default retention duration on column - {rd.id=} - {rd.column_id=}"
                )


def run_userstore_sample(*, client: Client, user_region: Region | str) -> None:
    # set up the userstore with the right columns, policies, accessors, mutators,
    # and retention durations
    names = Names()
    accessors, mutators = setup(client=client, names=names)
    # run the example
    userstore_example(
        client=client,
        user_region=user_region,
        names=names,
        accessors=accessors,
        mutators=mutators,
    )
    cleanup(client=client, names=names)


async def print_userstore_stats(client_async: Client):
    tasks = [
        client_async.ListColumnDataTypesAsync(),
        client_async.ListColumnsAsync(),
        client_async.ListPurposesAsync(),
        client_async.ListAccessPolicyTemplatesAsync(),
        client_async.ListAccessPoliciesAsync(),
        client_async.ListTransformersAsync(),
        client_async.ListAccessorsAsync(),
        client_async.ListMutatorsAsync(),
    ]
    results = await asyncio.gather(*tasks)
    print("Userstore stats:")
    print(f"num data types: {len(results[0])}")
    print(f"num columns: {len(results[1])}")
    print(f"num purposes: {len(results[2])}")
    print(f"num access policy templates: {len(results[3])}")
    print(f"num access policies: {len(results[4])}")
    print(f"num transformers: {len(results[5])}")
    print(f"num accessors: {len(results[6])}")
    print(f"num mutators: {len(results[7])}")


if __name__ == "__main__":
    disable_ssl_verify = (
        os.environ.get("DEV_ONLY_DISABLE_SSL_VERIFICATION", "") == "true"
    )
    user_region = os.environ.get("UC_REGION", Region.AWS_US_EAST_1)
    client = Client(
        url=url,
        client_id=client_id,
        client_secret=client_secret,
        client_factory=(
            create_no_ssl_http_client
            if disable_ssl_verify
            else create_default_uc_http_client
        ),
        session_name=os.environ.get("UC_SESSION_NAME"),
    )
    run_userstore_sample(client=client, user_region=user_region)

    client_async = AsyncClient(
        url=url,
        client_id=client_id,
        client_secret=client_secret,
        client_factory=(
            create_no_ssl_http_async_client
            if disable_ssl_verify
            else create_default_uc_http_async_client
        ),
        session_name=os.environ.get("UC_SESSION_NAME"),
    )

    asyncio.run(print_userstore_stats(client_async))
